package worker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"path"
	"strings"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/inbound"
	lineadapter "plaiflow/api/internal/line"
)

type LINEDocumentResolver interface {
	ResolveLINEDocument(context.Context, inbound.Event) (string, string, error)
}

func NewLINEDocumentProcessor(service *document.Service, resolver LINEDocumentResolver, downloader lineadapter.ContentDownloader) ProcessFunc {
	return func(ctx context.Context, event inbound.Event) (inbound.Status, string, error) {
		message, ok := lineadapter.ParseDocumentMessage(event.Payload)
		if !ok {
			return inbound.Processed, "No document message", nil
		}
		if service == nil || resolver == nil || downloader == nil {
			return inbound.Retryable, "Document intake is not configured", errors.New("LINE document intake is not configured")
		}
		organizationID, userID, err := resolver.ResolveLINEDocument(ctx, event)
		if err != nil {
			return inbound.Ignored, "LINE sender is not mapped to one organization", nil
		}
		body, err := downloader.Download(ctx, message.ID)
		if errors.Is(err, lineadapter.ErrContentUnavailable) {
			return inbound.Ignored, "LINE message content is unavailable; sender must resend", nil
		}
		if err != nil {
			return inbound.Retryable, "LINE content download failed", err
		}
		defer body.Close()
		filename := safeFilename(message.FileName, message.Type, message.ID)
		attemptID, err := uuidString()
		if err != nil {
			return inbound.Retryable, "LINE document attempt could not be created", err
		}
		result, err := service.Accept(ctx, document.AcceptInput{
			OrganizationID: organizationID, ActorUserID: userID, AttemptID: attemptID,
			OriginKey: "line:" + event.Provider + ":" + event.Channel + ":" + message.ID,
			Filename:  filename, Channel: "LINE", Now: time.Now().UTC(),
		}, body)
		if errors.Is(err, document.ErrQuota) || errors.Is(err, document.ErrTrashed) || errors.Is(err, document.ErrUnsupportedType) || errors.Is(err, document.ErrTooLarge) || errors.Is(err, document.ErrMalware) {
			return inbound.Ignored, "LINE document was rejected", nil
		}
		if err != nil {
			return inbound.Retryable, "LINE document intake failed", err
		}
		if result.Duplicate {
			return inbound.Processed, "LINE document already exists", nil
		}
		return inbound.Processed, "LINE document accepted", nil
	}
}

func safeFilename(value, messageType, messageID string) string {
	value = path.Base(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"))
	if value == "." || value == "/" || value == "" || len(value) > 240 {
		ext := ".bin"
		if messageType == "image" {
			ext = ".jpg"
		}
		return "line-" + messageID + ext
	}
	return value
}

func uuidString() (string, error) {
	value := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, value); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return hex.EncodeToString(value[:4]) + "-" + hex.EncodeToString(value[4:6]) + "-" + hex.EncodeToString(value[6:8]) + "-" + hex.EncodeToString(value[8:10]) + "-" + hex.EncodeToString(value[10:]), nil
}
