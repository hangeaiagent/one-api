package gemini

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/songquanpeng/one-api/relay/model"
)

// claudeCodeAgentToolSchema 取自 Claude Code 真实发出的 Agent 工具 input_schema
// （2026-10-06 抓包）。它的每个工具都带 $schema 与 additionalProperties，
// 直通 Gemini 会被整包打回 400，是这次修复的起因。
const claudeCodeAgentToolSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "description": {"description": "A short (3-5 word) description of the task", "type": "string"},
    "prompt": {"description": "The task for the agent to perform", "type": "string"},
    "subagent_type": {"description": "The type of specialized agent", "type": "string"},
    "model": {"description": "Optional model override", "type": "string", "enum": ["sonnet", "opus", "haiku", "fable"]},
    "run_in_background": {"description": "Agents run in the background by default", "type": "boolean"}
  },
  "required": ["description", "prompt"]
}`

func mustUnmarshal(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

// collectKeys 递归收集所有对象键名（properties 里的属性名也算，用于断言没被误删）。
func collectKeys(node any, out map[string]int) {
	switch v := node.(type) {
	case map[string]any:
		for k, val := range v {
			out[k]++
			collectKeys(val, out)
		}
	case []any:
		for _, item := range v {
			collectKeys(item, out)
		}
	}
}

// TestSanitizeGeminiSchema_DropsJSONSchemaOnlyKeywords 复现线上那次 400：
// Claude Code 的工具 schema 直通 Gemini 会带 $schema / additionalProperties。
func TestSanitizeGeminiSchema_DropsJSONSchemaOnlyKeywords(t *testing.T) {
	in := mustUnmarshal(t, claudeCodeAgentToolSchema)
	out, ok := sanitizeGeminiSchema(in).(map[string]any)
	if !ok {
		t.Fatalf("expected object schema, got %T", sanitizeGeminiSchema(in))
	}

	keys := map[string]int{}
	collectKeys(out, keys)
	for _, bad := range []string{"$schema", "additionalProperties"} {
		if keys[bad] > 0 {
			t.Errorf("%s 应被剔除，实际仍在 schema 里", bad)
		}
	}

	// 属性名不能因为过滤而丢
	for _, keep := range []string{"description", "prompt", "subagent_type", "model", "run_in_background"} {
		if keys[keep] == 0 {
			t.Errorf("属性 %s 被误删", keep)
		}
	}
	if out["type"] != "object" {
		t.Errorf("type 应为 object，实际 %v", out["type"])
	}
	req, ok := out["required"].([]any)
	if !ok || len(req) != 2 {
		t.Errorf("required 应为 2 项，实际 %v", out["required"])
	}
}

// TestSanitizeGeminiSchema_RecursesIntoProperties 属性一层里的非法关键字也要清掉。
func TestSanitizeGeminiSchema_RecursesIntoProperties(t *testing.T) {
	in := mustUnmarshal(t, `{
		"type": "object",
		"properties": {
			"findings": {
				"type": "array",
				"additionalProperties": false,
				"items": {
					"type": "object",
					"additionalProperties": false,
					"properties": {"file": {"type": "string", "$schema": "x"}},
					"required": ["file"]
				}
			}
		}
	}`)
	b, _ := json.Marshal(sanitizeGeminiSchema(in))
	if strings.Contains(string(b), "additionalProperties") || strings.Contains(string(b), "$schema") {
		t.Fatalf("嵌套层未清洗干净: %s", b)
	}
	if !strings.Contains(string(b), `"file"`) {
		t.Fatalf("嵌套属性名被误删: %s", b)
	}
}

// TestSanitizeGeminiSchema_AllOfMerged 并成一层，本层显式键优先。
func TestSanitizeGeminiSchema_AllOfMerged(t *testing.T) {
	in := mustUnmarshal(t, `{
		"type": "object",
		"allOf": [
			{"properties": {"a": {"type": "string"}}, "required": ["a"]},
			{"properties": {"b": {"type": "number"}}, "required": ["b"], "description": "被本层盖掉"}
		],
		"properties": {"c": {"type": "boolean"}},
		"description": "本层的描述"
	}`)
	out, _ := sanitizeGeminiSchema(in).(map[string]any)
	if _, still := out["allOf"]; still {
		t.Fatal("allOf 应已合并，不该残留")
	}
	props, _ := out["properties"].(map[string]any)
	for _, name := range []string{"a", "b", "c"} {
		if _, ok := props[name]; !ok {
			t.Errorf("allOf 合并后缺少属性 %s", name)
		}
	}
	if out["description"] != "本层的描述" {
		t.Errorf("本层 description 应优先，实际 %v", out["description"])
	}
	req, _ := toStringSlice(out["required"])
	if len(req) != 2 {
		t.Errorf("required 应取并集 2 项，实际 %v", out["required"])
	}
}

// TestSanitizeGeminiSchema_OneOfBecomesAnyOf Gemini 只有 anyOf。
func TestSanitizeGeminiSchema_OneOfBecomesAnyOf(t *testing.T) {
	in := mustUnmarshal(t, `{"oneOf": [{"type": "string"}, {"type": "number"}]}`)
	out, _ := sanitizeGeminiSchema(in).(map[string]any)
	if _, still := out["oneOf"]; still {
		t.Fatal("oneOf 应改写为 anyOf")
	}
	arr, ok := out["anyOf"].([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("anyOf 应有 2 项，实际 %v", out["anyOf"])
	}
}

// TestSanitizeGeminiSchema_ExclusiveBoundsDownshifted Gemini 没有 exclusive*。
func TestSanitizeGeminiSchema_ExclusiveBoundsDownshifted(t *testing.T) {
	in := mustUnmarshal(t, `{"type": "integer", "exclusiveMinimum": 0, "exclusiveMaximum": 100}`)
	out, _ := sanitizeGeminiSchema(in).(map[string]any)
	if _, still := out["exclusiveMinimum"]; still {
		t.Fatal("exclusiveMinimum 应被降级")
	}
	if out["minimum"] != float64(0) || out["maximum"] != float64(100) {
		t.Fatalf("边界未正确降级: %v", out)
	}
}

// TestSanitizeGeminiSchema_ExplicitBoundWins 已有 minimum 时不被 exclusiveMinimum 覆盖。
func TestSanitizeGeminiSchema_ExplicitBoundWins(t *testing.T) {
	in := mustUnmarshal(t, `{"type": "number", "minimum": 1, "exclusiveMinimum": 5}`)
	out, _ := sanitizeGeminiSchema(in).(map[string]any)
	if out["minimum"] != float64(1) {
		t.Fatalf("显式 minimum 应优先，实际 %v", out["minimum"])
	}
}

// TestSanitizeGeminiSchema_TypeArrayExpanded type 数组要摊成单 type + nullable。
func TestSanitizeGeminiSchema_TypeArrayExpanded(t *testing.T) {
	in := mustUnmarshal(t, `{"type": ["string", "null"], "description": "可空"}`)
	out, _ := sanitizeGeminiSchema(in).(map[string]any)
	if out["type"] != "string" {
		t.Errorf("type 应为 string，实际 %v", out["type"])
	}
	if out["nullable"] != true {
		t.Errorf("nullable 应为 true，实际 %v", out["nullable"])
	}
}

// TestSanitizeGeminiSchema_RefResolved $defs 里的 $ref 要展开。
func TestSanitizeGeminiSchema_RefResolved(t *testing.T) {
	in := mustUnmarshal(t, `{
		"$defs": {"Money": {"type": "object", "additionalProperties": false, "properties": {"amount": {"type": "number"}}}},
		"type": "object",
		"properties": {"price": {"$ref": "#/$defs/Money"}}
	}`)
	out, _ := sanitizeGeminiSchema(in).(map[string]any)
	b, _ := json.Marshal(out)
	if strings.Contains(string(b), "$ref") || strings.Contains(string(b), "additionalProperties") {
		t.Fatalf("$ref 未展开干净: %s", b)
	}
	if !strings.Contains(string(b), `"amount"`) {
		t.Fatalf("展开后丢了字段: %s", b)
	}
}

// TestSanitizeGeminiSchema_Idempotent 幂等：跑两遍结果一致。
func TestSanitizeGeminiSchema_Idempotent(t *testing.T) {
	once := sanitizeGeminiSchema(mustUnmarshal(t, claudeCodeAgentToolSchema))
	twice := sanitizeGeminiSchema(once)
	b1, _ := json.Marshal(once)
	b2, _ := json.Marshal(twice)
	if string(b1) != string(b2) {
		t.Fatalf("不幂等:\n一次 %s\n二次 %s", b1, b2)
	}
}

// TestSanitizeGeminiSchema_NilAndScalar 空值/异常输入不panic。
func TestSanitizeGeminiSchema_NilAndScalar(t *testing.T) {
	if got := sanitizeGeminiSchema(nil); got != nil {
		t.Fatalf("nil 应原样返回 nil，实际 %v", got)
	}
	if got := sanitizeGeminiSchema("oops"); got != "oops" {
		t.Fatalf("非对象应原样返回，实际 %v", got)
	}
	out, ok := sanitizeGeminiSchema(map[string]any{"additionalProperties": true}).(map[string]any)
	if !ok || len(out) != 0 {
		t.Fatalf("空对象 schema 应清成空 map，实际 %v", out)
	}
}

// TestConvertRequest_ToolSchemaGeminiCompatible 端到端：走一遍真实的
// ConvertRequest，确认发往 Google 的请求体里不含任何 Gemini 不认的关键字。
func TestConvertRequest_ToolSchemaGeminiCompatible(t *testing.T) {
	var agentSchema map[string]any
	if err := json.Unmarshal([]byte(claudeCodeAgentToolSchema), &agentSchema); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	req := model.GeneralOpenAIRequest{
		Model: "gemini-3.8-flash",
		Messages: []model.Message{
			{Role: "user", Content: "只回：好"},
		},
		Tools: []model.Tool{
			{
				Type: "function",
				Function: model.Function{
					Name:        "Agent",
					Description: "Launch a new agent to handle complex, multi-step tasks.",
					Parameters:  agentSchema,
				},
			},
		},
	}

	converted := ConvertRequest(req)
	body, err := json.Marshal(converted)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, bad := range []string{"$schema", "additionalProperties", "allOf", "oneOf", "exclusiveMinimum"} {
		if strings.Contains(string(body), `"`+bad+`"`) {
			t.Errorf("请求体里仍含 Gemini 不认的关键字 %s\n%s", bad, body)
		}
	}
	if !strings.Contains(string(body), `"run_in_background"`) {
		t.Errorf("工具属性被误删\n%s", body)
	}
	if !strings.Contains(string(body), `"function_declarations"`) {
		t.Errorf("function_declarations 未生成\n%s", body)
	}
}

// TestConvertRequest_ResponseSchemaSanitized 结构化输出的 responseSchema 同样要清洗。
func TestConvertRequest_ResponseSchemaSanitized(t *testing.T) {
	req := model.GeneralOpenAIRequest{
		Model:    "gemini-3.8-flash",
		Messages: []model.Message{{Role: "user", Content: "hi"}},
		ResponseFormat: &model.ResponseFormat{
			Type: "json_object",
			JsonSchema: &model.JSONSchema{
				Name: "out",
				Schema: map[string]any{
					"$schema":              "https://json-schema.org/draft/2020-12/schema",
					"type":                 "object",
					"additionalProperties": false,
					"properties":           map[string]any{"name": map[string]any{"type": "string"}},
				},
			},
		},
	}
	body, err := json.Marshal(ConvertRequest(req))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(body), "additionalProperties") || strings.Contains(string(body), `"$schema"`) {
		t.Fatalf("responseSchema 未清洗: %s", body)
	}
}
