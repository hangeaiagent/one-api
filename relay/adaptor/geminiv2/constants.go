package geminiv2

// https://ai.google.dev/gemini-api/docs/models
// https://ai.google.dev/gemini-api/docs/pricing

var ModelList = []string{
	// Legacy
	"gemini-pro", "gemini-1.0-pro",

	// Gemini 1.5 (kept for backward compatibility)
	"gemini-1.5-flash", "gemini-1.5-flash-8b",
	"gemini-1.5-pro", "gemini-1.5-pro-experimental",

	// Embeddings
	"text-embedding-004",
	"gemini-embedding-2",
	"aqa",

	// Gemini 2.0
	"gemini-2.0-flash", "gemini-2.0-flash-exp",
	"gemini-2.0-flash-lite-preview-02-05",
	"gemini-2.0-flash-thinking-exp-01-21",
	"gemini-2.0-pro-exp-02-05",

	// Gemini 2.5 GA (2025)
	"gemini-2.5-pro",
	"gemini-2.5-flash",
	"gemini-2.5-flash-lite",

	// Gemini 3.0 preview (existing)
	"gemini-3-pro-preview",
	"gemini-3-pro-image-preview",
	"gemini-3-flash-preview",

	// Gemini 3.1 (preview + GA flash-lite)
	"gemini-3.1-pro-preview",
	"gemini-3.1-flash-lite",

	// Gemini 3.5 GA (2026-07)
	"gemini-3.5-flash",
	"gemini-3.5-flash-lite",

	// Gemini 3.6 GA - workhorse main model (2026-07-21)
	"gemini-3.6-flash",

	// TTS models
	"gemini-2.5-flash-preview-tts",
	"gemini-2.5-pro-preview-tts",
	"gemini-3.1-flash-tts",
}
