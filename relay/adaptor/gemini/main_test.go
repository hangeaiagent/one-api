package gemini

import (
	"encoding/json"
	"fmt"
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

func TestConvertRequest_MultipleSystemMessagesMerged(t *testing.T) {
	req := model.GeneralOpenAIRequest{
		Model: "gemini-3.8-flash",
		Messages: []model.Message{
			{Role: "system", Content: "人设"},
			{Role: "system", Content: "注入数据"},
			{Role: "user", Content: "问题"},
			{Role: "system", Content: "必须中文"},
		},
	}
	out := ConvertRequest(req)
	if out.SystemInstruction == nil || len(out.SystemInstruction.Parts) != 3 {
		t.Fatalf("expected 3 merged system parts, got %+v", out.SystemInstruction)
	}
	if out.SystemInstruction.Parts[0].Text != "人设" || out.SystemInstruction.Parts[2].Text != "必须中文" {
		t.Fatalf("unexpected order: %+v", out.SystemInstruction.Parts)
	}
	if len(out.Contents) != 1 || out.Contents[0].Role != "user" {
		t.Fatalf("expected single user content, got %+v", out.Contents)
	}
}

func TestMaxOutputTokensWithThinking(t *testing.T) {
	cases := []struct {
		model string
		in    int
		want  int
	}{
		{"gemini-3.8-flash", 200, 200 + 8192},
		{"gemini-2.5-flash", 1000, 1000 + 8192},
		{"gemini-3.8-flash", 0, 0},
		{"gemini-3.8-flash", 60000, 65536},
		{"gemini-3-pro-image-preview", 4096, 4096},
		{"gemini-2.0-flash", 200, 200},
	}
	for _, c := range cases {
		if got := maxOutputTokensWithThinking(c.model, c.in); got != c.want {
			t.Fatalf("%s/%d: got %d want %d", c.model, c.in, got, c.want)
		}
	}
}

func TestResponseGeminiChat2OpenAI_MaxTokensIsLength(t *testing.T) {
	resp := unmarshalResp(t, `{"candidates":[{"content":{"role":"model","parts":[{"text":"作为"}]},"finishReason":"MAX_TOKENS"}]}`)
	out := responseGeminiChat2OpenAI(resp)
	if out.Choices[0].FinishReason != "length" {
		t.Fatalf("expected length, got %s", out.Choices[0].FinishReason)
	}
}

func TestConvertRequest_ToolResultWithoutToolsBecomesText(t *testing.T) {
	name := "stock_quickview"
	msgs := []model.Message{
		{Role: "system", Content: "投研助手"},
		{Role: "user", Content: "茅台怎么样"},
		{Role: "assistant", ToolCalls: []model.Tool{{Id: "call_1", Type: "function", Function: model.Function{Name: name, Arguments: "{}"}}}},
		{Role: "tool", ToolCallId: "call_1", Content: `{"price":1258}`},
	}
	out := ConvertRequest(model.GeneralOpenAIRequest{Model: "gemini-3.8-flash", Messages: msgs})
	last := out.Contents[len(out.Contents)-1]
	if last.Parts[0].FunctionResponse != nil || !strings.Contains(last.Parts[0].Text, "stock_quickview") {
		t.Fatalf("expected plain text tool result, got %+v", last)
	}

	withTools := model.GeneralOpenAIRequest{Model: "gemini-3.8-flash", Messages: msgs,
		Tools: []model.Tool{{Type: "function", Function: model.Function{Name: name}}}}
	out = ConvertRequest(withTools)
	last = out.Contents[len(out.Contents)-1]
	if last.Parts[0].FunctionResponse == nil {
		t.Fatalf("expected functionResponse when tools declared, got %+v", last)
	}
}

// 多轮工具调用：assistant 的 tool_calls 必须还原成 model 轮 functionCall（带占位签名），
// 否则 Gemini 3 在累计 3 轮后返回空候选。
func TestConvertRequest_EchoFunctionCallTurns(t *testing.T) {
	tools := []model.Tool{{Type: "function", Function: model.Function{Name: "invoke"}}}
	msgs := []model.Message{
		{Role: "system", Content: "投研助手"},
		{Role: "user", Content: "分三步查"},
	}
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("call_%d", i)
		msgs = append(msgs,
			model.Message{Role: "assistant", Content: "", ToolCalls: []model.Tool{{Id: id, Type: "function", Function: model.Function{Name: "invoke", Arguments: `{"tool":"fund_list"}`}}}},
			model.Message{Role: "tool", ToolCallId: id, Content: `{"ok":true}`},
		)
	}
	out := ConvertRequest(model.GeneralOpenAIRequest{Model: "gemini-3.8-flash", Messages: msgs, Tools: tools})
	// user + 3×(model functionCall, user functionResponse)
	if len(out.Contents) != 7 {
		t.Fatalf("expected 7 contents, got %d: %+v", len(out.Contents), out.Contents)
	}
	for i := 1; i < 7; i += 2 {
		m, u := out.Contents[i], out.Contents[i+1]
		if m.Role != "model" || m.Parts[0].FunctionCall == nil || m.Parts[0].ThoughtSignature != skipThoughtSignature {
			t.Fatalf("content %d should be model functionCall with signature, got %+v", i, m)
		}
		args, ok := m.Parts[0].FunctionCall.Arguments.(map[string]any)
		if !ok || args["tool"] != "fund_list" {
			t.Fatalf("arguments not parsed into object: %#v", m.Parts[0].FunctionCall.Arguments)
		}
		if u.Role != "user" || u.Parts[0].FunctionResponse == nil {
			t.Fatalf("content %d should be user functionResponse, got %+v", i+1, u)
		}
	}
	if out.Contents[len(out.Contents)-1].Role != "user" {
		t.Fatalf("request must not end with a model turn")
	}
}

// 并行调用：一个 model 轮多个 functionCall，只有第一个带签名；结果合并进同一个 user 轮。
func TestConvertRequest_ParallelToolCallsGrouped(t *testing.T) {
	tools := []model.Tool{{Type: "function", Function: model.Function{Name: "a"}}, {Type: "function", Function: model.Function{Name: "b"}}}
	msgs := []model.Message{
		{Role: "user", Content: "并行查"},
		{Role: "assistant", ToolCalls: []model.Tool{
			{Id: "c1", Type: "function", Function: model.Function{Name: "a", Arguments: "{}"}},
			{Id: "c2", Type: "function", Function: model.Function{Name: "b", Arguments: ""}},
		}},
		{Role: "tool", ToolCallId: "c1", Content: "1"},
		{Role: "tool", ToolCallId: "c2", Content: "2"},
	}
	out := ConvertRequest(model.GeneralOpenAIRequest{Model: "gemini-3.8-flash", Messages: msgs, Tools: tools})
	if len(out.Contents) != 3 {
		t.Fatalf("expected 3 contents, got %d: %+v", len(out.Contents), out.Contents)
	}
	m := out.Contents[1]
	if len(m.Parts) != 2 || m.Parts[0].ThoughtSignature == "" || m.Parts[1].ThoughtSignature != "" {
		t.Fatalf("parallel calls: only first part carries signature, got %+v", m.Parts)
	}
	u := out.Contents[2]
	if len(u.Parts) != 2 || u.Parts[0].FunctionResponse.Name != "a" || u.Parts[1].FunctionResponse.Name != "b" {
		t.Fatalf("tool results should be grouped in one user turn, got %+v", u.Parts)
	}
}
