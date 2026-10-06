package gemini

// Gemini 的 functionDeclarations[].parameters 与 generationConfig.responseSchema
// 只接受 OpenAPI 3.0 Schema 的一个子集。Google 遇到不认识的关键字不是忽略，而是整包返回
// 400：
//
//	Invalid JSON payload received. Unknown name "$schema" at
//	'tools[0].function_declarations[0].parameters': Cannot find field.
//
// 而 OpenAI 兼容客户端（Claude Code、各家 SDK）常按 JSON Schema 2019-09 / 2020-12 下发
// 工具参数，带 $schema、additionalProperties、allOf、exclusiveMinimum、type 数组等写法，
// 原样直通就会被整包打回（对外表现为 400 All target providers failed）。
//
// 这里递归清洗成 Gemini 认的子集：
//   - 白名单之外的关键字一律剔除（additionalProperties、$schema、patternProperties…）
//   - allOf 合进本层（本层显式写的键优先，allOf 只补缺）
//   - oneOf 改写成 anyOf（Gemini 只有 anyOf）
//   - exclusiveMinimum / exclusiveMaximum 降级为 minimum / maximum
//   - type 数组（如 ["string","null"]）改写为单个 type + nullable
//   - $ref 从顶层 $defs / definitions 里解析展开
//
// 对已经是合法 Gemini schema 的输入是幂等的（跑两遍结果一致）。
//
// 参考 https://ai.google.dev/api/caching#Schema

import "github.com/songquanpeng/one-api/relay/model"

// maxSchemaRefDepth 防 $ref 互相引用转不出来。
const maxSchemaRefDepth = 8

// geminiSchemaKeywords 是 Gemini Schema 认的关键字白名单。
// 注意：properties 里的「属性名」不参与过滤，那层单独处理。
var geminiSchemaKeywords = map[string]bool{
	"type": true, "format": true, "title": true, "description": true, "nullable": true,
	"default": true, "items": true, "minItems": true, "maxItems": true, "enum": true,
	"properties": true, "required": true, "minProperties": true, "maxProperties": true,
	"minimum": true, "maximum": true, "minLength": true, "maxLength": true, "pattern": true,
	"example": true, "anyOf": true, "propertyOrdering": true,
}

// droppedSchemaKeywords 是明确要丢掉的 JSON Schema 关键字。放在白名单之外也能被
// 过滤掉，列出来只为让意图显式、并给一处集中的说明。
var droppedSchemaKeywords = map[string]bool{
	"$schema": true, "$id": true, "$comment": true, "$anchor": true, "$defs": true,
	"definitions": true, "additionalProperties": true, "patternProperties": true,
	"propertyNames": true, "unevaluatedProperties": true, "unevaluatedItems": true,
	"dependentRequired": true, "dependentSchemas": true, "if": true, "then": true,
	"else": true, "not": true, "const": true, "examples": true, "contentMediaType": true,
	"contentEncoding": true, "readOnly": true, "writeOnly": true, "deprecated": true,
	"$dynamicRef": true, "$dynamicAnchor": true,
}

// sanitizeGeminiSchema 清洗一个参数 / 响应 schema。入参出参都用 any，
// 便于挂在 model.Function.Parameters（any）与 JSONSchema.Schema（map）两处。
func sanitizeGeminiSchema(schema any) any {
	if schema == nil {
		return nil
	}
	return sanitizeSchemaNode(schema, collectSchemaDefs(schema), 0)
}

// sanitizeGeminiFunctions 清洗旧版 /v1/completions 风格的 functions 字段
// （model.GeneralOpenAIRequest.Functions 是 any，实际可能是 []model.Function 或 []any）。
func sanitizeGeminiFunctions(functions any) any {
	switch v := functions.(type) {
	case []model.Function:
		out := make([]model.Function, 0, len(v))
		for _, fn := range v {
			fn.Parameters = sanitizeGeminiSchema(fn.Parameters)
			out = append(out, fn)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			fn, ok := item.(map[string]any)
			if !ok {
				out = append(out, item)
				continue
			}
			if params, ok := fn["parameters"]; ok {
				cloned := make(map[string]any, len(fn))
				for k, val := range fn {
					cloned[k] = val
				}
				cloned["parameters"] = sanitizeGeminiSchema(params)
				out = append(out, cloned)
				continue
			}
			out = append(out, fn)
		}
		return out
	default:
		return functions
	}
}

// collectSchemaDefs 收拢顶层 $defs / definitions，供 $ref 解析。
func collectSchemaDefs(schema any) map[string]any {
	root, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	defs := map[string]any{}
	for _, key := range []string{"$defs", "definitions"} {
		sub, ok := root[key].(map[string]any)
		if !ok {
			continue
		}
		for name, def := range sub {
			defs["#/$defs/"+name] = def
			defs["#/definitions/"+name] = def
		}
	}
	if len(defs) == 0 {
		return nil
	}
	return defs
}

func sanitizeSchemaNode(node any, defs map[string]any, depth int) any {
	switch v := node.(type) {
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, sanitizeSchemaNode(item, defs, depth))
		}
		return out
	case map[string]any:
		return sanitizeSchemaMap(v, defs, depth)
	default:
		return node
	}
}

func sanitizeSchemaMap(in map[string]any, defs map[string]any, depth int) map[string]any {
	if depth > maxSchemaRefDepth {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}

	// $ref 先展开成实际 schema，同层其余关键字照旧生效（JSON Schema 语义如此）。
	if ref, ok := in["$ref"].(string); ok && defs != nil {
		if target, found := defs[ref]; found {
			targetMap, ok := target.(map[string]any)
			if ok {
				merged := make(map[string]any, len(targetMap)+len(in))
				for k, v := range targetMap {
					merged[k] = v
				}
				for k, v := range in {
					if k != "$ref" {
						merged[k] = v
					}
				}
				return sanitizeSchemaMap(merged, defs, depth+1)
			}
		}
	}

	out := map[string]any{}
	for k, v := range in {
		switch {
		case droppedSchemaKeywords[k], k == "$ref", k == "allOf", k == "oneOf",
			k == "exclusiveMinimum", k == "exclusiveMaximum":
			// 下面单独处理
		case k == "properties":
			props, ok := v.(map[string]any)
			if !ok {
				continue
			}
			// 注意：这一层的键是「属性名」，不是关键字，只递归值、不过滤键名。
			sanitized := make(map[string]any, len(props))
			for name, sub := range props {
				sanitized[name] = sanitizeSchemaNode(sub, defs, depth+1)
			}
			out["properties"] = sanitized
		case k == "items" || k == "anyOf":
			out[k] = sanitizeSchemaNode(v, defs, depth+1)
		case k == "type":
			if arr, ok := v.([]any); ok {
				// JSON Schema 允许 type 是数组；Gemini 只认单个 type + nullable。
				nullable := false
				chosen := ""
				for _, t := range arr {
					s, ok := t.(string)
					if !ok {
						continue
					}
					if s == "null" {
						nullable = true
						continue
					}
					if chosen == "" {
						chosen = s
					}
				}
				if chosen != "" {
					out["type"] = chosen
				}
				if nullable {
					out["nullable"] = true
				}
				continue
			}
			out["type"] = v
		case geminiSchemaKeywords[k]:
			out[k] = v
		}
	}

	// oneOf → anyOf（Gemini 只有 anyOf）
	if list, ok := in["oneOf"].([]any); ok {
		if _, exists := out["anyOf"]; !exists {
			out["anyOf"] = sanitizeSchemaNode(list, defs, depth+1)
		}
	}

	// 互斥边界降级为闭区间（Gemini 没有 exclusive*）
	if v, ok := in["exclusiveMinimum"]; ok {
		if _, exists := out["minimum"]; !exists {
			out["minimum"] = v
		}
	}
	if v, ok := in["exclusiveMaximum"]; ok {
		if _, exists := out["maximum"]; !exists {
			out["maximum"] = v
		}
	}

	// allOf 最后合：本层显式写的键优先，allOf 只补缺。
	if list, ok := in["allOf"].([]any); ok {
		for _, item := range list {
			if sub, ok := sanitizeSchemaNode(item, defs, depth+1).(map[string]any); ok {
				mergeSchemaInto(out, sub)
			}
		}
	}

	// 有 properties 却没 type 的，补成 object：Gemini 对无 type 的对象结构容忍度低。
	if _, hasType := out["type"]; !hasType {
		if _, hasProps := out["properties"]; hasProps {
			out["type"] = "object"
		}
	}

	return out
}

// mergeSchemaInto 把 src 并进 dst：properties 取并集（dst 已有的同名属性优先），
// required 取并集，其余标量键只在 dst 缺失时补。
func mergeSchemaInto(dst, src map[string]any) {
	for k, v := range src {
		switch k {
		case "properties":
			srcProps, ok := v.(map[string]any)
			if !ok {
				continue
			}
			dstProps, _ := dst["properties"].(map[string]any)
			if dstProps == nil {
				dstProps = map[string]any{}
				dst["properties"] = dstProps
			}
			for name, sub := range srcProps {
				if _, exists := dstProps[name]; !exists {
					dstProps[name] = sub
				}
			}
		case "required":
			srcReq, ok := toStringSlice(v)
			if !ok {
				continue
			}
			dstReq, _ := toStringSlice(dst["required"])
			seen := make(map[string]bool, len(dstReq)+len(srcReq))
			merged := make([]any, 0, len(dstReq)+len(srcReq))
			for _, s := range dstReq {
				if !seen[s] {
					seen[s] = true
					merged = append(merged, s)
				}
			}
			for _, s := range srcReq {
				if !seen[s] {
					seen[s] = true
					merged = append(merged, s)
				}
			}
			dst["required"] = merged
		default:
			if _, exists := dst[k]; !exists {
				dst[k] = v
			}
		}
	}
}

func toStringSlice(v any) ([]string, bool) {
	arr, ok := v.([]any)
	if !ok {
		if strs, ok := v.([]string); ok {
			return strs, true
		}
		return nil, false
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}
