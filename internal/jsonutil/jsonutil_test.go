package jsonutil

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeRejectsOversizedJSONAndTrailingData(t *testing.T) {
	for _, input := range []string{`{"value":"` + strings.Repeat("a", 30) + `"}`, `{} ` + strings.Repeat(" ", 30)} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(input))
		var value map[string]any
		if err := Decode(r, 10, &value); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("expected size error, got %v", err)
		}
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{} {}`))
	var value map[string]any
	if err := Decode(r, 100, &value); err == nil {
		t.Fatal("accepted multiple JSON values")
	}
}
