package gcptts

// ModelList defines the supported Chirp TTS models
var ModelList = []string{
	"chirp-3",
	"chirp-2",
	"chirp_telephony",
}

// DefaultAudioEncoding is the default output format
const DefaultAudioEncoding = "MP3"

// DefaultLanguageCode used when no language hint is provided
const DefaultLanguageCode = "en-US"

// OpenAIVoiceMap maps OpenAI voice names to Chirp voice names
var OpenAIVoiceMap = map[string]string{
	"alloy":   "Kore",
	"echo":    "Charon",
	"fable":   "Achernar",
	"onyx":    "Fenrir",
	"nova":    "Leda",
	"shimmer": "Aoede",
}

// ModelVoicePrefix maps model names to voice name prefix patterns
var ModelVoicePrefix = map[string]string{
	"chirp-3":         "Chirp3-HD",
	"chirp-2":         "Chirp-HD",
	"chirp_telephony": "Chirp-HD",
}

// AudioEncodingMap maps OpenAI response_format to Google Cloud TTS audioEncoding
var AudioEncodingMap = map[string]string{
	"mp3":  "MP3",
	"opus": "OGG_OPUS",
	"aac":  "MP3",      // AAC not supported, fallback to MP3
	"flac": "LINEAR16", // FLAC not supported, fallback to LINEAR16
	"wav":  "LINEAR16",
	"pcm":  "LINEAR16",
	"":     "MP3", // default
}

// AudioContentTypeMap maps audioEncoding to HTTP Content-Type
var AudioContentTypeMap = map[string]string{
	"MP3":      "audio/mpeg",
	"OGG_OPUS": "audio/ogg",
	"LINEAR16": "audio/wav",
	"ALAW":     "audio/alaw",
	"MULAW":    "audio/basic",
}
