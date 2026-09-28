package ocr

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"plaiflow/api/internal/job"
)

const MaxResultBytes = 8 << 20
const ModelVersion = "paddleocr-3.7.0-ppocrv5-th-v2"
const PreprocessingVersion = "v2"

type Line struct {
	ID             string                 `json:"id,omitempty"`
	PageNumber     int                    `json:"page_number,omitempty"`
	Text           string                 `json:"text"`
	Confidence     float64                `json:"confidence"`
	LowConfidence  bool                   `json:"is_low_confidence,omitempty"`
	BBox           *BoundingBox           `json:"bbox,omitempty"`
	NormalizedBBox *NormalizedBoundingBox `json:"normalized_bbox,omitempty"`
	Polygon        [][2]float64           `json:"polygon"`
	PixelPolygon   [][2]float64           `json:"pixel_polygon,omitempty"`
	Center         *Point                 `json:"center,omitempty"`
	Width          int                    `json:"width,omitempty"`
	Height         int                    `json:"height,omitempty"`
	ReadingOrder   int                    `json:"reading_order,omitempty"`
}
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type BoundingBox struct {
	XMin int `json:"x_min"`
	YMin int `json:"y_min"`
	XMax int `json:"x_max"`
	YMax int `json:"y_max"`
}
type NormalizedBoundingBox struct {
	XMin float64 `json:"x_min"`
	YMin float64 `json:"y_min"`
	XMax float64 `json:"x_max"`
	YMax float64 `json:"y_max"`
}
type Page struct {
	Number             int     `json:"page_number"`
	Width              int     `json:"width"`
	Height             int     `json:"height"`
	Rotation           int     `json:"rotation"`
	DurationMS         int64   `json:"duration_ms"`
	Text               string  `json:"text"`
	AverageConfidence  float64 `json:"average_confidence,omitempty"`
	LowConfidenceCount int     `json:"low_confidence_count,omitempty"`
	LineCount          int     `json:"line_count,omitempty"`
	Lines              []Line  `json:"lines"`
}
type Result struct {
	SchemaVersion        int    `json:"schema_version"`
	InputSHA256          string `json:"input_sha256"`
	ModelVersion         string `json:"model_version"`
	PreprocessingVersion string `json:"preprocessing_version"`
	DurationMS           int64  `json:"duration_ms"`
	Pages                []Page `json:"pages"`
}

func (r Result) Validate(input Input) error {
	if (r.SchemaVersion != 1 && r.SchemaVersion != 2) || r.InputSHA256 != input.SHA256 || r.ModelVersion != input.ModelVersion || r.PreprocessingVersion != input.PreprocessingVersion || len(r.Pages) < 1 || len(r.Pages) > 20 || r.DurationMS < 0 || r.DurationMS > 900000 {
		return errors.New("invalid OCR result")
	}
	for i, p := range r.Pages {
		if p.Number != i+1 || p.Width < 1 || p.Height < 1 || p.Width > 25000000 || p.Height > 25000000 || int64(p.Width)*int64(p.Height) > 25000000 || p.Rotation < 0 || p.Rotation > 270 || p.Rotation%90 != 0 || len(p.Lines) > 5000 || p.DurationMS < 0 || p.DurationMS > 45000 {
			return errors.New("invalid OCR page")
		}
		if r.SchemaVersion == 2 && (p.LineCount != len(p.Lines) || math.IsNaN(p.AverageConfidence) || p.AverageConfidence < 0 || p.AverageConfidence > 1 || p.LowConfidenceCount < 0 || p.LowConfidenceCount > p.LineCount) {
			return errors.New("invalid OCR page statistics")
		}
		texts := make([]string, 0, len(p.Lines))
		for lineNumber, l := range p.Lines {
			if len(l.Text) > 16384 || math.IsNaN(l.Confidence) || l.Confidence < 0 || l.Confidence > 1 || len(l.Polygon) != 4 {
				return errors.New("invalid OCR line")
			}
			for _, point := range l.Polygon {
				for _, v := range point {
					if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
						return errors.New("invalid OCR polygon")
					}
				}
			}
			if r.SchemaVersion == 2 {
				if l.ID != fmt.Sprintf("p%d_line_%04d", i+1, lineNumber+1) || l.PageNumber != i+1 || l.ReadingOrder != lineNumber+1 || l.BBox == nil || l.NormalizedBBox == nil || l.Center == nil || len(l.PixelPolygon) != 4 || l.Width != l.BBox.XMax-l.BBox.XMin || l.Height != l.BBox.YMax-l.BBox.YMin || l.BBox.XMin < 0 || l.BBox.YMin < 0 || l.BBox.XMax > p.Width || l.BBox.YMax > p.Height || l.Width < 1 || l.Height < 1 {
					return errors.New("invalid OCR geometry")
				}
				b := l.NormalizedBBox
				if b.XMin < 0 || b.YMin < 0 || b.XMax > 1 || b.YMax > 1 || math.Abs(b.XMin-float64(l.BBox.XMin)/float64(p.Width)) > 0.0001 || math.Abs(b.YMin-float64(l.BBox.YMin)/float64(p.Height)) > 0.0001 || math.Abs(b.XMax-float64(l.BBox.XMax)/float64(p.Width)) > 0.0001 || math.Abs(b.YMax-float64(l.BBox.YMax)/float64(p.Height)) > 0.0001 || l.Center.X != float64(l.BBox.XMin+l.BBox.XMax)/2 || l.Center.Y != float64(l.BBox.YMin+l.BBox.YMax)/2 {
					return errors.New("invalid OCR normalized geometry")
				}
				for _, point := range l.PixelPolygon {
					if math.IsNaN(point[0]) || math.IsNaN(point[1]) || math.IsInf(point[0], 0) || math.IsInf(point[1], 0) {
						return errors.New("invalid OCR pixel polygon")
					}
				}
			}
			texts = append(texts, l.Text)
		}
		if strings.Join(texts, "\n") != p.Text {
			return errors.New("invalid OCR page text")
		}
	}
	return nil
}

type Input struct {
	OrganizationID       string `json:"organization_id"`
	DocumentID           string `json:"document_id"`
	SHA256               string `json:"sha256"`
	MIME                 string `json:"mime"`
	Size                 int64  `json:"size"`
	StorageKey           string `json:"-"`
	ModelVersion         string `json:"model_version"`
	PreprocessingVersion string `json:"preprocessing_version"`
}
type State struct {
	JobID       string `json:"job_id"`
	Status      string `json:"status"`
	Enabled     bool   `json:"enabled"`
	FailureCode string `json:"failure_code,omitempty"`
	PageCount   int    `json:"page_count"`
	ObjectKey   string `json:"-"`
}
type Store interface {
	OCRInput(context.Context, job.LeaseCommand, string) (Input, error)
	CompleteOCR(context.Context, job.LeaseCommand, string, Result, string, string, int64) (string, error)
	OCRState(context.Context, string, string, string) (State, error)
	RetryOCR(context.Context, string, string, string, time.Time) (State, error)
	ReprocessOCR(context.Context, string, string, string, string, int, string, string, string, time.Time) (State, error)
}
