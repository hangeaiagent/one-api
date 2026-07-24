package gemini

import (
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

	// TTS
	"gemini-2.5-flash-preview-tts": {TTS: true},
	"gemini-2.5-pro-preview-tts":   {TTS: true},
	"gemini-3.1-flash-tts":         {TTS: true},
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
