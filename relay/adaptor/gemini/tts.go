package gemini

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/model"
)

// openAIVoiceToGemini maps OpenAI voice names to Gemini prebuilt voice names
var openAIVoiceToGemini = map[string]string{
	"alloy":   "Kore",
	"echo":    "Charon",
	"fable":   "Achernar",
	"onyx":    "Fenrir",
	"nova":    "Leda",
	"shimmer": "Aoede",
}

// mapOpenAIVoiceToGemini maps OpenAI voice name to Gemini voice name.
// If not found, returns the input as-is (allows direct Gemini voice names like "Puck").
func mapOpenAIVoiceToGemini(voice string) string {
	if mapped, ok := openAIVoiceToGemini[strings.ToLower(voice)]; ok {
		return mapped
	}
	return voice
}

// ConvertTTSRequest converts an OpenAI TTS request to Gemini generateContent format
func ConvertTTSRequest(ttsRequest openai.TextToSpeechRequest) *ChatRequest {
	voiceName := mapOpenAIVoiceToGemini(ttsRequest.Voice)

	return &ChatRequest{
		Contents: []ChatContent{
			{
				Role: "user",
				Parts: []Part{
					{Text: ttsRequest.Input},
				},
			},
		},
		GenerationConfig: ChatGenerationConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: &SpeechConfig{
				VoiceConfig: &VoiceConfig{
					PrebuiltVoiceConfig: &PrebuiltVoiceConfig{
						VoiceName: voiceName,
					},
				},
			},
		},
	}
}

// TTSHandler handles Gemini TTS response: extracts audio from inlineData and writes WAV to client
func TTSHandler(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var geminiResponse ChatResponse
		if json.Unmarshal(responseBody, &geminiResponse) == nil {
			return &model.ErrorWithStatusCode{
				Error: model.Error{
					Message: fmt.Sprintf("upstream error: %s", string(responseBody)),
					Type:    "upstream_error",
					Code:    resp.StatusCode,
				},
				StatusCode: resp.StatusCode,
			}, nil
		}
		return openai.ErrorWrapper(fmt.Errorf("upstream error: %s", string(responseBody)), "upstream_error", resp.StatusCode), nil
	}

	var geminiResponse ChatResponse
	err = json.Unmarshal(responseBody, &geminiResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}

	if len(geminiResponse.Candidates) == 0 ||
		len(geminiResponse.Candidates[0].Content.Parts) == 0 ||
		geminiResponse.Candidates[0].Content.Parts[0].InlineData == nil {
		return &model.ErrorWithStatusCode{
			Error: model.Error{
				Message: "No audio data in response",
				Type:    "server_error",
				Code:    500,
			},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	inlineData := geminiResponse.Candidates[0].Content.Parts[0].InlineData
	audioData, err := base64.StdEncoding.DecodeString(inlineData.Data)
	if err != nil {
		return openai.ErrorWrapper(err, "decode_audio_data_failed", http.StatusInternalServerError), nil
	}

	// Gemini returns raw PCM (24kHz, 16-bit, mono) — convert to WAV for compatibility
	mimeType := inlineData.MimeType
	if strings.Contains(mimeType, "pcm") || strings.Contains(mimeType, "L16") {
		audioData = pcmToWav(audioData, 24000, 16, 1)
		mimeType = "audio/wav"
	}

	c.Writer.Header().Set("Content-Type", mimeType)
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(audioData)))
	c.Writer.WriteHeader(http.StatusOK)
	_, err = c.Writer.Write(audioData)
	if err != nil {
		return openai.ErrorWrapper(err, "write_audio_response_failed", http.StatusInternalServerError), nil
	}

	return nil, &model.Usage{}
}

// pcmToWav adds a 44-byte WAV header to raw PCM data
func pcmToWav(pcmData []byte, sampleRate, bitsPerSample, numChannels int) []byte {
	dataSize := len(pcmData)
	fileSize := 36 + dataSize

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(fileSize))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1) // PCM format
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	blockAlign := numChannels * bitsPerSample / 8
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))

	return append(header, pcmData...)
}
