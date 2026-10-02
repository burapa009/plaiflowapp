package classification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type DocumentClassifier interface {
	Classify(context.Context, Input) (Result, error)
}

// HTTPClassifier expects an OpenAI-compatible chat endpoint supplied by the operator.
type HTTPClassifier struct {
	URL, Model, Token string
	Client            *http.Client
}

const systemPrompt = `คุณเป็นระบบ AI สำหรับจำแนกเอกสารทางบัญชีของประเทศไทย เลือกประเภทจากรายการที่กำหนดโดยดู OCR และพิกัด ห้ามเดาข้อมูลที่ไม่มีในเอกสาร หากหลักฐานไม่พอให้คืน unknown คืน JSON เท่านั้นในรูป {"document_type":"...","confidence":0.0,"candidate_types":[],"signals":[]} ห้ามคำนวณยอดบัญชีหรือเปลี่ยนข้อความ OCR.`

func (c HTTPClassifier) Classify(ctx context.Context, in Input) (Result, error) {
	input, err := json.Marshal(in)
	if err != nil || len(input) > 1<<20 {
		return Result{}, errors.New("classification input too large")
	}
	types, _ := json.Marshal(Labels)
	requestBody, _ := json.Marshal(map[string]any{"model": c.Model, "temperature": 0,
		"response_format": map[string]string{"type": "json_object"}, "messages": []map[string]string{
			{"role": "system", "content": systemPrompt + "\nAllowed types and Thai labels: " + string(types)},
			{"role": "user", "content": string(input)},
		}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(requestBody))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Result{}, errors.New("classifier upstream unavailable")
	}
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&payload); err != nil || len(payload.Choices) != 1 {
		return Result{}, errors.New("invalid classifier response")
	}
	var out Result
	decoder := json.NewDecoder(strings.NewReader(payload.Choices[0].Message.Content))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&out); err != nil || !Valid(out.DocumentType) || out.Confidence < 0 || out.Confidence > 1 || len(out.Signals) > 12 || len(out.CandidateTypes) > 5 || out.DocumentType != "unknown" && len(out.Signals) == 0 {
		return Result{}, errors.New("invalid classifier JSON")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return Result{}, errors.New("invalid classifier JSON")
	}
	if out.DocumentType == "unknown" {
		out.Confidence = 0
	}
	for _, candidate := range out.CandidateTypes {
		if !Valid(candidate.Type) || candidate.Confidence < 0 || candidate.Confidence > 1 {
			return Result{}, errors.New("invalid classifier candidate")
		}
	}
	out.Method, out.ClassifierVersion, out.Accounting = "rule_llm", c.Model, AccountingFor(out.DocumentType)
	return out, nil
}
