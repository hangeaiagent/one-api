package controller

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestRelayErrorHandlerGoogleArray(t *testing.T) {
	body := `[{
  "error": {
    "code": 503,
    "message": "This model is currently experiencing high demand.",
    "status": "UNAVAILABLE"
  }
}
]`
	resp := &http.Response{StatusCode: 503, Body: io.NopCloser(bytes.NewBufferString(body))}
	e := RelayErrorHandler(resp)
	if e.StatusCode != 503 {
		t.Fatalf("status = %d", e.StatusCode)
	}
	if e.Error.Message != "This model is currently experiencing high demand." {
		t.Fatalf("message = %q", e.Error.Message)
	}
	if e.Error.Code != "UNAVAILABLE" || e.Error.Type != "upstream_error" {
		t.Fatalf("code/type = %v/%s", e.Error.Code, e.Error.Type)
	}
}

func TestRelayErrorHandlerOpenAIObjectUnchanged(t *testing.T) {
	body := `{"error":{"message":"bad key","type":"invalid_request_error","code":"invalid_api_key"}}`
	resp := &http.Response{StatusCode: 401, Body: io.NopCloser(bytes.NewBufferString(body))}
	e := RelayErrorHandler(resp)
	if e.Error.Message != "bad key" || e.Error.Code != "invalid_api_key" {
		t.Fatalf("got %+v", e.Error)
	}
}

func TestRelayErrorHandlerGarbageKeepsFallback(t *testing.T) {
	resp := &http.Response{StatusCode: 502, Body: io.NopCloser(bytes.NewBufferString("<html>bad gateway</html>"))}
	e := RelayErrorHandler(resp)
	if e.Error.Code != "bad_response_status_code" || e.Error.Message != "" {
		t.Fatalf("got %+v", e.Error)
	}
}
