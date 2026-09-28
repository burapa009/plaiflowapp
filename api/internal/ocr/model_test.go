package ocr

import "testing"

func TestStructuredResultValidation(t *testing.T) {
	input := Input{SHA256: "digest", ModelVersion: ModelVersion, PreprocessingVersion: PreprocessingVersion}
	line := Line{
		ID: "p1_line_0001", PageNumber: 1, Text: "ยอดรวม 100", Confidence: .9,
		BBox:           &BoundingBox{10, 20, 50, 40},
		NormalizedBBox: &NormalizedBoundingBox{.1, .2, .5, .4},
		Polygon:        [][2]float64{{.1, .2}, {.5, .2}, {.5, .4}, {.1, .4}},
		PixelPolygon:   [][2]float64{{10, 20}, {50, 20}, {50, 40}, {10, 40}},
		Center:         &Point{30, 30}, Width: 40, Height: 20, ReadingOrder: 1,
	}
	result := Result{SchemaVersion: 2, InputSHA256: input.SHA256, ModelVersion: input.ModelVersion, PreprocessingVersion: input.PreprocessingVersion,
		Pages: []Page{{Number: 1, Width: 100, Height: 100, Text: line.Text, AverageConfidence: .9, LineCount: 1, Lines: []Line{line}}}}
	if err := result.Validate(input); err != nil {
		t.Fatal(err)
	}
	result.Pages[0].Lines[0].NormalizedBBox.XMin = .9
	if err := result.Validate(input); err == nil {
		t.Fatal("accepted wrong normalized coordinate")
	}
	result.SchemaVersion = 1
	result.Pages[0].Lines[0].ID = ""
	if err := result.Validate(input); err != nil {
		t.Fatalf("legacy artifact rejected: %v", err)
	}
}
