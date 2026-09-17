package line

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const messagingAPI = "https://api.line.me"

type RichMenuPublisher struct {
	Client  *http.Client
	BaseURL string
}

func RichMenuDefinition(webBaseURL string) ([]byte, error) {
	base, err := url.Parse(strings.TrimRight(webBaseURL, "/"))
	if err != nil || base.Scheme != "https" || base.Host == "" {
		return nil, errors.New("rich menu web URL must be HTTPS")
	}
	labels := []string{"ภาพรวม", "งาน", "คู่ค้า", "นำเข้า", "ส่งออก", "ตั้งค่า"}
	paths := []string{"dashboard", "tasks", "vendors", "imports", "exports", "settings"}
	columns := []struct{ x, width int }{{0, 834}, {834, 833}, {1667, 833}}
	areas := make([]map[string]any, 0, 6)
	for index, path := range paths {
		column := columns[index%3]
		y := (index / 3) * 843
		areas = append(areas, map[string]any{
			"bounds": map[string]int{"x": column.x, "y": y, "width": column.width, "height": 843},
			"action": map[string]string{"type": "uri", "label": labels[index], "uri": strings.TrimRight(webBaseURL, "/") + "/open/" + path},
		})
	}
	return json.Marshal(map[string]any{
		"size": map[string]int{"width": 2500, "height": 1686}, "selected": true,
		"name": "PlaiFlow Main", "chatBarText": "เมนู PlaiFlow", "areas": areas,
	})
}

func (p RichMenuPublisher) Publish(ctx context.Context, token, webBaseURL string, image []byte) (string, error) {
	if token == "" || len(image) == 0 {
		return "", errors.New("rich menu token and image are required")
	}
	if len(image) > 1<<20 {
		return "", errors.New("rich menu image must be a PNG no larger than 1 MB")
	}
	configuration, err := png.DecodeConfig(bytes.NewReader(image))
	if err != nil || configuration.Width != 2500 || configuration.Height != 1686 {
		return "", errors.New("rich menu image must be a 2500x1686 PNG")
	}
	definition, err := RichMenuDefinition(webBaseURL)
	if err != nil {
		return "", err
	}
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = messagingAPI
	}
	var created struct {
		RichMenuID string `json:"richMenuId"`
	}
	if err := richMenuRequest(ctx, client, token, http.MethodPost, base+"/v2/bot/richmenu", "application/json", definition, &created); err != nil {
		return "", fmt.Errorf("create rich menu: %w", err)
	}
	if created.RichMenuID == "" {
		return "", errors.New("create rich menu: LINE returned no ID")
	}
	if err := richMenuRequest(ctx, client, token, http.MethodPost, base+"/v2/bot/richmenu/"+url.PathEscape(created.RichMenuID)+"/content", "image/png", image, nil); err != nil {
		return "", fmt.Errorf("upload rich menu: %w", err)
	}
	if err := richMenuRequest(ctx, client, token, http.MethodPost, base+"/v2/bot/user/all/richmenu/"+url.PathEscape(created.RichMenuID), "application/json", nil, nil); err != nil {
		return "", fmt.Errorf("set default rich menu: %w", err)
	}
	return created.RichMenuID, nil
}

func richMenuRequest(ctx context.Context, client *http.Client, token, method, endpoint, contentType string, body []byte, result any) error {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", contentType)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("LINE returned %s", response.Status)
	}
	if result != nil {
		return json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(result)
	}
	return nil
}
