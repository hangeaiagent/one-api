package gcptts

// SynthesizeRequest is the Google Cloud Text-to-Speech API request
type SynthesizeRequest struct {
	Input       SynthesisInput `json:"input"`
	Voice       VoiceSelection `json:"voice"`
	AudioConfig AudioConfig    `json:"audioConfig"`
}

type SynthesisInput struct {
	Text string `json:"text,omitempty"`
	SSML string `json:"ssml,omitempty"`
}

type VoiceSelection struct {
	LanguageCode string `json:"languageCode"`
	Name         string `json:"name,omitempty"`
	SsmlGender   string `json:"ssmlGender,omitempty"`
}

type AudioConfig struct {
	AudioEncoding   string  `json:"audioEncoding"`
	SampleRateHertz int     `json:"sampleRateHertz,omitempty"`
	SpeakingRate    float64 `json:"speakingRate,omitempty"`
	Pitch           float64 `json:"pitch,omitempty"`
	VolumeGainDb    float64 `json:"volumeGainDb,omitempty"`
}

// SynthesizeResponse is the Google Cloud Text-to-Speech API response
type SynthesizeResponse struct {
	AudioContent string `json:"audioContent"` // base64 encoded audio
}

// ErrorResponse for Google Cloud API errors
type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}
