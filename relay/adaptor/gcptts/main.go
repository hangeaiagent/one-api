package gcptts

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/model"
)

// ConvertTTSRequest converts an OpenAI TTS request to Google Cloud TTS format
func ConvertTTSRequest(ttsRequest openai.TextToSpeechRequest, modelName string) *SynthesizeRequest {
	audioEncoding := DefaultAudioEncoding
	if enc, ok := AudioEncodingMap[ttsRequest.ResponseFormat]; ok {
		audioEncoding = enc
	}

	voiceName := resolveVoiceName(ttsRequest.Voice, modelName, DefaultLanguageCode)

	req := &SynthesizeRequest{
		Input: SynthesisInput{
			Text: ttsRequest.Input,
		},
		Voice: VoiceSelection{
			LanguageCode: DefaultLanguageCode,
			Name:         voiceName,
		},
		AudioConfig: AudioConfig{
			AudioEncoding: audioEncoding,
		},
	}

	if ttsRequest.Speed > 0 {
		req.AudioConfig.SpeakingRate = ttsRequest.Speed
	}

	return req
}

// resolveVoiceName builds the full Google Cloud TTS voice name.
// If the voice already contains "Chirp", it's treated as a full name and returned as-is.
// Otherwise, it maps OpenAI voice names and builds the full name like "en-US-Chirp3-HD-Kore".
func resolveVoiceName(voice string, modelName string, languageCode string) string {
	if strings.Contains(voice, "Chirp") {
		return voice
	}

	chirpVoice := voice
	lowerVoice := strings.ToLower(voice)
	// Use model-specific voice map
	if modelName == "chirp-2" || modelName == "chirp_telephony" {
		if mapped, ok := OpenAIVoiceMapChirp2[lowerVoice]; ok {
			chirpVoice = mapped
		}
	} else {
		if mapped, ok := OpenAIVoiceMapChirp3[lowerVoice]; ok {
			chirpVoice = mapped
		}
	}

	prefix := "Chirp3-HD"
	if p, ok := ModelVoicePrefix[modelName]; ok {
		prefix = p
	}

	return fmt.Sprintf("%s-%s-%s", languageCode, prefix, chirpVoice)
}

// TTSHandler handles Google Cloud TTS response: decodes base64 audioContent and writes to client
func TTSHandler(c *gin.Context, resp *http.Response, audioEncoding string) (*model.ErrorWithStatusCode, *model.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(responseBody, &errResp) == nil && errResp.Error.Message != "" {
			return &model.ErrorWithStatusCode{
				Error: model.Error{
					Message: errResp.Error.Message,
					Type:    errResp.Error.Status,
					Code:    errResp.Error.Code,
				},
				StatusCode: resp.StatusCode,
			}, nil
		}
		return openai.ErrorWrapper(
			fmt.Errorf("upstream error: %s", string(responseBody)),
			"upstream_error", resp.StatusCode,
		), nil
	}

	var ttsResponse SynthesizeResponse
	err = json.Unmarshal(responseBody, &ttsResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError), nil
	}

	if ttsResponse.AudioContent == "" {
		return &model.ErrorWithStatusCode{
			Error: model.Error{
				Message: "No audio content in response",
				Type:    "server_error",
				Code:    500,
			},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	audioData, err := base64.StdEncoding.DecodeString(ttsResponse.AudioContent)
	if err != nil {
		return openai.ErrorWrapper(err, "decode_audio_failed", http.StatusInternalServerError), nil
	}

	contentType := "audio/mpeg"
	if ct, ok := AudioContentTypeMap[audioEncoding]; ok {
		contentType = ct
	}

	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(audioData)))
	c.Writer.WriteHeader(http.StatusOK)
	_, err = c.Writer.Write(audioData)
	if err != nil {
		return openai.ErrorWrapper(err, "write_response_failed", http.StatusInternalServerError), nil
	}

	return nil, &model.Usage{}
}
