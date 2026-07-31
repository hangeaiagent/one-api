package gemini

import (
	"encoding/json"
	"strings"
	"testing"
)

const groundingSample = `{
  "webSearchQueries": ["2026 Euro winner"],
  "groundingChunks": [
    {"web": {"uri": "https://vertexaisearch.cloud.google.com/id/abc", "title": "example.com"}}
  ],
  "groundingSupports": [
    {"segment": {"startIndex": 0, "endIndex": 10, "text": "hello"}, "groundingChunkIndices": [0], "confidenceScores": [0.9]}
  ]
}`

func unmarshalResp(t *testing.T, body string) *ChatResponse {
	t.Helper()
	var r ChatResponse
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &r
}

func TestResponseGeminiChat2OpenAI_NoGrounding(t *testing.T) {
	resp := unmarshalResp(t, `{
		"candidates": [{
			"content": {"role":"model", "parts":[{"text":"hi"}]},
			"finishReason": "STOP"
		}]
	}`)
	out := responseGeminiChat2OpenAI(resp)
	if out.Metadata != nil {
		t.Fatalf("expected nil Metadata, got %v", out.Metadata)
	}
	b, _ := json.Marshal(out)
	if strings.Contains(string(b), `"metadata"`) {
		t.Fatalf("metadata should be omitted from JSON, got: %s", b)
	}
}

func TestResponseGeminiChat2OpenAI_WithGrounding(t *testing.T) {
	resp := unmarshalResp(t, `{
		"candidates": [{
			"content": {"role":"model", "parts":[{"text":"hi"}]},
			"finishReason": "STOP",
			"groundingMetadata": `+groundingSample+`
		}]
	}`)
	out := responseGeminiChat2OpenAI(resp)
	if out.Metadata == nil {
		t.Fatal("expected Metadata to be set")
	}
	grounding, ok := out.Metadata["grounding"]
	if !ok {
		t.Fatal("expected metadata.grounding key")
	}
	// Round-trip: parsed grounding should deep-equal the original raw payload.
	var gotGrounding, wantGrounding any
	if err := json.Unmarshal(grounding, &gotGrounding); err != nil {
		t.Fatalf("unmarshal grounding: %v", err)
	}
	if err := json.Unmarshal([]byte(groundingSample), &wantGrounding); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	gotJSON, _ := json.Marshal(gotGrounding)
	wantJSON, _ := json.Marshal(wantGrounding)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("grounding not preserved:\n got:  %s\n want: %s", gotJSON, wantJSON)
	}
	// Alias must also be present.
	if _, ok := out.Metadata["google_grounding"]; !ok {
		t.Fatal("expected metadata.google_grounding alias")
	}
}

func TestStreamResponseGeminiChat2OpenAI_LastChunk(t *testing.T) {
	resp := unmarshalResp(t, `{
		"candidates": [{
			"content": {"role":"model", "parts":[{"text":"final"}]},
			"finishReason": "STOP",
			"groundingMetadata": `+groundingSample+`
		}]
	}`)
	out := streamResponseGeminiChat2OpenAI(resp)
	if out.Metadata == nil {
		t.Fatal("expected Metadata on last chunk")
	}
	b, _ := json.Marshal(out)
	if !strings.Contains(string(b), `"metadata"`) {
		t.Fatalf("serialized chunk missing metadata: %s", b)
	}
}

func TestStreamResponseGeminiChat2OpenAI_MiddleChunk(t *testing.T) {
	resp := unmarshalResp(t, `{
		"candidates": [{
			"content": {"role":"model", "parts":[{"text":"partial"}]}
		}]
	}`)
	out := streamResponseGeminiChat2OpenAI(resp)
	if out.Metadata != nil {
		t.Fatalf("expected nil Metadata on middle chunk, got %v", out.Metadata)
	}
	b, _ := json.Marshal(out)
	if strings.Contains(string(b), `"metadata"`) {
		t.Fatalf("middle chunk should not contain metadata: %s", b)
	}
}

func TestBuildMetadata_NullPayload(t *testing.T) {
	resp := unmarshalResp(t, `{
		"candidates": [{
			"content": {"role":"model", "parts":[{"text":"x"}]},
			"groundingMetadata": null
		}]
	}`)
	if md := buildMetadata(resp); md != nil {
		t.Fatalf("null grounding should yield nil, got %v", md)
	}
}
