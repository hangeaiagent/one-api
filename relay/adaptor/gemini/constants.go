package gemini

import (
	"github.com/songquanpeng/one-api/relay/adaptor/geminiv2"
)

// https://ai.google.dev/models/gemini

var ModelList = geminiv2.ModelList

// ModelsSupportSystemInstruction is the list of models that support system instruction.
//
// https://cloud.google.com/vertex-ai/generative-ai/docs/learn/prompts/system-instructions
var ModelsSupportSystemInstruction = []string{
	// "gemini-1.0-pro-002",
	// "gemini-1.5-flash", "gemini-1.5-flash-001", "gemini-1.5-flash-002",
	// "gemini-1.5-flash-8b",
	// "gemini-1.5-pro", "gemini-1.5-pro-001", "gemini-1.5-pro-002",
	// "gemini-1.5-pro-experimental",
	"gemini-2.0-flash", "gemini-2.0-flash-exp",
	"gemini-2.0-flash-thinking-exp-01-21",
	// Gemini 3.0 models support system instruction
	"gemini-3-pro-preview",
	"gemini-3-pro-image-preview",
	"gemini-3-flash-preview",
	// Gemini 3.1 models support system instruction
	"gemini-3.1-pro-preview",
}

// IsModelSupportSystemInstruction check if the model support system instruction.
//
// Because the main version of Go is 1.20, slice.Contains cannot be used
func IsModelSupportSystemInstruction(model string) bool {
	for _, m := range ModelsSupportSystemInstruction {
		if m == model {
			return true
		}
	}

	return false
}

// ModelsWithImageGeneration is the list of models that support image generation
// via responseModalities: ["TEXT", "IMAGE"] in generationConfig.
var ModelsWithImageGeneration = []string{
	"gemini-2.0-flash-exp",
	"gemini-3-pro-image-preview",
}

// IsModelSupportImageGeneration check if the model supports image generation output.
func IsModelSupportImageGeneration(model string) bool {
	for _, m := range ModelsWithImageGeneration {
		if m == model {
			return true
		}
	}
	return false
}
