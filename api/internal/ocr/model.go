package ocr

import (
 "context"
 "errors"
 "math"
 "strings"
 "time"

 "plaiflow/api/internal/job"
)

const MaxResultBytes = 8 << 20
const ModelVersion = "paddleocr-3.7.0-ppocrv5-th-v1"
const PreprocessingVersion = "v1"

type Line struct {
 Text string `json:"text"`
 Confidence float64 `json:"confidence"`
 Polygon [][2]float64 `json:"polygon"`
}
type Page struct {
 Number int `json:"page_number"`
 Width int `json:"width"`
 Height int `json:"height"`
 Rotation int `json:"rotation"`
 DurationMS int64 `json:"duration_ms"`
 Text string `json:"text"`
 Lines []Line `json:"lines"`
}
type Result struct {
 SchemaVersion int `json:"schema_version"`
 InputSHA256 string `json:"input_sha256"`
 ModelVersion string `json:"model_version"`
 PreprocessingVersion string `json:"preprocessing_version"`
 DurationMS int64 `json:"duration_ms"`
 Pages []Page `json:"pages"`
}
func (r Result) Validate(input Input) error {
 if r.SchemaVersion != 1 || r.InputSHA256 != input.SHA256 || r.ModelVersion != input.ModelVersion || r.PreprocessingVersion != input.PreprocessingVersion || len(r.Pages)<1 || len(r.Pages)>20 || r.DurationMS<0 || r.DurationMS>900000 { return errors.New("invalid OCR result") }
 for i,p := range r.Pages {
  if p.Number!=i+1 || p.Width<1 || p.Height<1 || p.Width>25000000 || p.Height>25000000 || int64(p.Width)*int64(p.Height)>25000000 || p.Rotation<0 || p.Rotation>270 || p.Rotation%90!=0 || len(p.Lines)>5000 || p.DurationMS<0 || p.DurationMS>45000 { return errors.New("invalid OCR page") }
  texts:=make([]string,0,len(p.Lines))
  for _,l:=range p.Lines {
   if len(l.Text)>16384 || math.IsNaN(l.Confidence) || l.Confidence<0 || l.Confidence>1 || len(l.Polygon)!=4 { return errors.New("invalid OCR line") }
   for _,point:=range l.Polygon { for _,v:=range point { if math.IsNaN(v)||math.IsInf(v,0)||v<0||v>1 { return errors.New("invalid OCR polygon") } } }
   texts=append(texts,l.Text)
  }
  if strings.Join(texts,"\n")!=p.Text { return errors.New("invalid OCR page text") }
 }
 return nil
}
type Input struct {
 OrganizationID string `json:"organization_id"`
 DocumentID string `json:"document_id"`
 SHA256 string `json:"sha256"`
 MIME string `json:"mime"`
 Size int64 `json:"size"`
 StorageKey string `json:"-"`
 ModelVersion string `json:"model_version"`
 PreprocessingVersion string `json:"preprocessing_version"`
}
type State struct {
 JobID string `json:"job_id"`
 Status string `json:"status"`
 FailureCode string `json:"failure_code,omitempty"`
 PageCount int `json:"page_count"`
 ObjectKey string `json:"-"`
}
type Store interface {
 OCRInput(context.Context, job.LeaseCommand, string) (Input,error)
 CompleteOCR(context.Context,job.LeaseCommand,string,Result,string,string,int64) (string,error)
 OCRState(context.Context,string,string,string) (State,error)
 RetryOCR(context.Context,string,string,string,time.Time) (State,error)
}
