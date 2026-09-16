package gemini

import (
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/relay/adaptor/geminiv2"
)

// https://ai.google.dev/gemini-api/docs/models

var ModelList = geminiv2.ModelList

// ModelCapability describes optional features a Gemini model supports.
// Extend the struct (and modelCapabilities map below) when Google exposes
// additional modalities.
type ModelCapability struct {
	SystemInstruction bool
	ImageGeneration   bool
	TTS               bool
}

// modelCapabilities is the single source of truth for Gemini model features.
// Reference: https://cloud.google.com/vertex-ai/generative-ai/docs/learn/prompts/system-instructions
var modelCapabilities = map[string]ModelCapability{
	// Gemini 2.0
	"gemini-2.0-flash":                    {SystemInstruction: true},
	"gemini-2.0-flash-exp":                {SystemInstruction: true, ImageGeneration: true},
	"gemini-2.0-flash-thinking-exp-01-21": {SystemInstruction: true},

	// Gemini 2.5 GA
	"gemini-2.5-pro":        {SystemInstruction: true},
	"gemini-2.5-flash":      {SystemInstruction: true},
	"gemini-2.5-flash-lite": {SystemInstruction: true},

	// Gemini 3.0 preview
	"gemini-3-pro-preview":       {SystemInstruction: true},
	"gemini-3-pro-image-preview": {SystemInstruction: true, ImageGeneration: true},
	"gemini-3-flash-preview":     {SystemInstruction: true},

	// Gemini 3.1
	"gemini-3.1-pro-preview": {SystemInstruction: true},
	"gemini-3.1-flash-lite":  {SystemInstruction: true},

	// Gemini 3.5 GA (2026-07)
	"gemini-3.5-flash":      {SystemInstruction: true},
	"gemini-3.5-flash-lite": {SystemInstruction: true},

	// Gemini 3.6 GA - workhorse
	"gemini-3.6-flash": {SystemInstruction: true},

	// Gemini 3.7 GA (2026-08-13)
	"gemini-3.7-flash": {SystemInstruction: true},

	// Gemini 3.8 GA - latest workhorse
	"gemini-3.8-flash": {SystemInstruction: true},

	// 生图模型 GA（Nano Banana 系列）
	"gemini-2.5-flash-image":      {SystemInstruction: true, ImageGeneration: true},
	"gemini-3-pro-image":          {SystemInstruction: true, ImageGeneration: true},
	"gemini-3.1-flash-image":      {SystemInstruction: true, ImageGeneration: true},
	"gemini-3.1-flash-lite-image": {SystemInstruction: true, ImageGeneration: true},

	// TTS
	"gemini-2.5-flash-preview-tts": {TTS: true},
	"gemini-2.5-pro-preview-tts":   {TTS: true},
	"gemini-3.1-flash-tts":         {TTS: true},
	"gemini-3.1-flash-tts-preview": {TTS: true},
}

// IsModelSupportSystemInstruction reports whether the model accepts
// systemInstruction. Kept as a top-level function because Go 1.20 lacks
// slices.Contains and the code base cannot bump the minimum yet.
func IsModelSupportSystemInstruction(model string) bool {
	return modelCapabilities[model].SystemInstruction
}

// IsModelSupportImageGeneration reports whether the model can return image
// parts via responseModalities: ["TEXT", "IMAGE"].
func IsModelSupportImageGeneration(model string) bool {
	return modelCapabilities[model].ImageGeneration
}

// IsModelSupportTTS reports whether the model produces speech via
// responseModalities: ["AUDIO"].
func IsModelSupportTTS(model string) bool {
	return modelCapabilities[model].TTS
}

// thinkingBudgetByEffort 仅用于 gemini-2.5（按 token 预算控制思考）
var thinkingBudgetByEffort = map[string]int{
	"none":    0,
	"minimal": 1024,
	"low":     1024,
	"medium":  8192,
	"high":    24576,
}

// buildThinkingConfig 把 OpenAI 的 reasoning_effort 翻译成 Gemini thinkingConfig。
// gemini-3.x 只接受 thinkingLevel；其中 flash-lite 支持 minimal，
// 其余（如 3.8-flash）传 minimal 会报错，统一降为 low。无法识别的取值返回 nil（不下发）。
func buildThinkingConfig(model string, effort string) *ThinkingConfig {
	effort = strings.ToLower(strings.TrimSpace(effort))
	if strings.HasPrefix(model, "gemini-2.5") {
		budget, ok := thinkingBudgetByEffort[effort]
		if !ok {
			return nil
		}
		return &ThinkingConfig{ThinkingBudget: &budget}
	}
	if !strings.HasPrefix(model, "gemini-3") {
		return nil
	}
	switch effort {
	case "low", "medium", "high":
		return &ThinkingConfig{ThinkingLevel: effort}
	case "minimal", "none":
		if strings.Contains(model, "flash-lite") {
			return &ThinkingConfig{ThinkingLevel: "minimal"}
		}
		return &ThinkingConfig{ThinkingLevel: "low"}
	}
	return nil
}

// geminiMaxOutputTokens 是 Gemini 3.x / 2.5 文本模型的输出上限
const geminiMaxOutputTokens = 65536

// maxOutputTokensWithThinking 给会思考的模型在客户端 max_tokens 之上追加思考余量。
// Gemini 的 maxOutputTokens 同时包含思考 token 和正文 token；gemini-3.7/3.8 默认思考较深，
// 客户端按 OpenAI 语义传 max_tokens=200 时思考就会耗尽额度，正文只剩几个字。
// 余量由 GEMINI_THINKING_HEADROOM 配置，设为 0 可关闭。
func maxOutputTokensWithThinking(model string, maxTokens int) int {
	if maxTokens <= 0 || config.GeminiThinkingHeadroom <= 0 {
		return maxTokens
	}
	if IsModelSupportImageGeneration(model) || IsModelSupportTTS(model) {
		return maxTokens
	}
	if !strings.HasPrefix(model, "gemini-2.5") && !strings.HasPrefix(model, "gemini-3") && !strings.HasPrefix(model, "gemini-4") {
		return maxTokens
	}
	total := maxTokens + config.GeminiThinkingHeadroom
	if total > geminiMaxOutputTokens {
		total = geminiMaxOutputTokens
	}
	return total
}
