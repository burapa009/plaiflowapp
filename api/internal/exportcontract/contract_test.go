package exportcontract

import (
	"os"
	"testing"
)

func TestSharedFixtures(t *testing.T) {
	request, err := os.ReadFile("../../../contracts/export/v1/request.json")
	if err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile("../../../contracts/export/v1/result.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRequest(request); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseResult(result); err != nil {
		t.Fatal(err)
	}
}
