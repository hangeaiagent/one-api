package gemini

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/songquanpeng/one-api/relay/model"
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

func TestConvertRequest_SystemUserDoesNotEndWithModelTurn(t *testing.T) {
	req := model.GeneralOpenAIRequest{
		Model: "gemini-3.8-flash",
		Messages: []model.Message{
			{Role: "system", Content: "只回复 OK"},
			{Role: "user", Content: "ping"},
		},
	}
	out := ConvertRequest(req)
	if out.SystemInstruction == nil {
		t.Fatal("expected systemInstruction to be set")
	}
	if len(out.Contents) != 1 || out.Contents[0].Role != "user" {
		t.Fatalf("expected single user content, got %+v", out.Contents)
	}
}

func TestConvertRequest_SystemFallbackAddsDummyModelTurn(t *testing.T) {
	req := model.GeneralOpenAIRequest{
		Model: "gemini-unknown-model",
		Messages: []model.Message{
			{Role: "system", Content: "只回复 OK"},
			{Role: "user", Content: "ping"},
		},
	}
	out := ConvertRequest(req)
	roles := make([]string, 0, len(out.Contents))
	for _, c := range out.Contents {
		roles = append(roles, c.Role)
	}
	if strings.Join(roles, ",") != "user,model,user" {
		t.Fatalf("unexpected roles: %v", roles)
	}
}

func TestBuildThinkingConfig(t *testing.T) {
	cases := []struct {
		model, effort, level string
		budget               int
		isNil                bool
	}{
		{"gemini-3.8-flash", "high", "high", 0, false},
		{"gemini-3.8-flash", "minimal", "low", 0, false},
		{"gemini-3.1-flash-lite", "minimal", "minimal", 0, false},
		{"gemini-2.5-flash", "none", "", 0, false},
		{"gemini-2.5-flash", "medium", "", 8192, false},
		{"gemini-3.8-flash", "bogus", "", 0, true},
		{"gpt-4o", "high", "", 0, true},
	}
	for _, c := range cases {
		got := buildThinkingConfig(c.model, c.effort)
		if c.isNil {
			if got != nil {
				t.Fatalf("%s/%s: expected nil, got %+v", c.model, c.effort, got)
			}
			continue
		}
		if got == nil || got.ThinkingLevel != c.level {
			t.Fatalf("%s/%s: unexpected %+v", c.model, c.effort, got)
		}
		if got.ThinkingLevel == "" && (got.ThinkingBudget == nil || *got.ThinkingBudget != c.budget) {
			t.Fatalf("%s/%s: unexpected budget %+v", c.model, c.effort, got.ThinkingBudget)
		}
	}
}
