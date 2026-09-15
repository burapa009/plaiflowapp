package exportcontract

import (
	"encoding/json"
	"errors"
)

type Request struct {
	Version string          `json:"version"`
	Payload json.RawMessage `json:"payload"`
}

type Result struct {
	Version string `json:"version"`
	JobRef  string `json:"job_ref"`
}

func ParseRequest(data []byte) (Request, error) {
	var value Request
	if err := json.Unmarshal(data, &value); err != nil || value.Version != "v1" || !json.Valid(value.Payload) {
		return Request{}, errors.New("invalid v1 export request")
	}
	return value, nil
}

func ParseResult(data []byte) (Result, error) {
	var value Result
	if err := json.Unmarshal(data, &value); err != nil || value.Version != "v1" || value.JobRef == "" {
		return Result{}, errors.New("invalid v1 export result")
	}
	return value, nil
}
