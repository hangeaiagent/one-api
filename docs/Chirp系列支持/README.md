# Chirp 系列 TTS 模型支持方案

## 一、目标

新增 Google Cloud Text-to-Speech 渠道类型，支持以下 Chirp 系列模型：

- `chirp-3`（Chirp 3 HD）
- `chirp-2`（Chirp 2）
- `chirp_telephony`（电话场景优化）

用户通过标准 OpenAI TTS 接口 `POST /v1/audio/speech` 发送请求，One API 自动将其转换为 Google Cloud TTS API 格式。

---

## 二、为什么需要全新渠道类型

Chirp 系列与 Gemini TTS **完全不同**：

| 对比项 | Gemini TTS | Chirp 系列 |
|--------|-----------|-----------|
| API 服务 | Gemini API (AI Studio) | Google Cloud Text-to-Speech |
| 端点 | `generativelanguage.googleapis.com` | `texttospeech.googleapis.com` |
| 认证 | API Key | API Key / Service Account / OAuth |
| 请求格式 | generateContent + speechConfig | text:synthesize |
| 响应格式 | candidates[].content.parts[].inlineData | audioContent (base64) |
| 计费单位 | tokens / chars | chars |

**结论**：无法复用 Gemini 渠道（channeltype=24），必须新建独立渠道类型。

---

## 三、Google Cloud Text-to-Speech API 说明

### 3.1 端点

```
POST https://texttospeech.googleapis.com/v1/text:synthesize
```

也支持区域端点：
- `https://us-texttospeech.googleapis.com/v1/text:synthesize`
- `https://eu-texttospeech.googleapis.com/v1/text:synthesize`
- `https://asia-southeast1-texttospeech.googleapis.com/v1/text:synthesize`

### 3.2 认证方式

Google Cloud TTS 支持多种认证方式：

**方式 1：API Key（推荐，与 One API 架构最兼容）**
```
POST https://texttospeech.googleapis.com/v1/text:synthesize?key=API_KEY
```

**方式 2：Bearer Token（Service Account）**
```
Authorization: Bearer ya29.xxx
```

**在 One API 中的实现**：渠道配置中的 "密钥" 字段填入 API Key，通过 URL 参数传递。
如需支持 Service Account，可在渠道配置的 "其他" 字段中配置 JSON 凭证。

### 3.3 请求格式

```json
{
  "input": {
    "text": "Hello, this is a test."
  },
  "voice": {
    "languageCode": "en-US",
    "name": "en-US-Chirp3-HD-Charon"
  },
  "audioConfig": {
    "audioEncoding": "MP3",
    "sampleRateHertz": 24000,
    "speakingRate": 1.0,
    "pitch": 0.0,
    "volumeGainDb": 0.0
  }
}
```

也支持 SSML 输入：
```json
{
  "input": {
    "ssml": "<speak>Hello <break time='300ms'/> world</speak>"
  },
  "voice": {
    "languageCode": "en-US",
    "name": "en-US-Chirp3-HD-Charon"
  },
  "audioConfig": {
    "audioEncoding": "MP3"
  }
}
```

### 3.4 响应格式

```json
{
  "audioContent": "//NExAASGoHwABhGud..."
}
```

- `audioContent`：base64 编码的音频数据
- 音频包含完整的文件头（MP3 头 / WAV 头等），可直接播放

### 3.5 Chirp 模型详情

#### Chirp 3 HD (`chirp-3`)

- 最新一代高质量语音合成
- 8 个预置语音（4 男 4 女）
- 支持 31+ 种语言
- 语音名称格式：`{lang}-{region}-Chirp3-HD-{VoiceName}`
- 可用语音：Achernar, Achird, Algenib, Algieba, Alnilam, Aoede, Autonoe, Callirrhoe, Charon, Despina, Enceladus, Erinome, Fenrir, Gacrux, Iapetus, Kore, Laomedeia, Leda, Orus, Puck, Sulafat, Umbriel, Zephyr

#### Chirp 2 (`chirp-2`)

- 上一代语音合成模型
- 语音名称格式：`{lang}-{region}-Chirp-HD-{VoiceName}` 或 `{lang}-{region}-Chirp-{VoiceName}`
- 支持多种语言

#### chirp_telephony

- 电话场景优化版本
- 8kHz 采样率，适合电话通信
- 低延迟，适合实时语音交互
- 语音名称格式类似 Chirp 系列

### 3.6 支持的音频格式

| 格式 | audioEncoding | 说明 |
|------|--------------|------|
| MP3 | `MP3` | 通用压缩格式，推荐 |
| OGG Opus | `OGG_OPUS` | 高质量压缩 |
| WAV (Linear16) | `LINEAR16` | 无损 PCM |
| A-law | `ALAW` | 电话系统 |
| mu-law | `MULAW` | 电话系统 |

### 3.7 定价

| 模型 | 价格 (per 1M chars) | 免费额度 |
|------|--------------------| ---------|
| Chirp 3 HD | $30.00 | 1M chars/月 |
| Chirp 2 | $16.00 | 1M chars/月 |
| chirp_telephony | $16.00 | 1M chars/月 |
| Standard voices | $4.00 | 4M chars/月 |
| WaveNet voices | $16.00 | 1M chars/月 |

### 3.8 语音列表 API

```
GET https://texttospeech.googleapis.com/v1/voices?key=API_KEY
GET https://texttospeech.googleapis.com/v1/voices?languageCode=en-US&key=API_KEY
```

---

## 四、渠道类型设计

### 4.1 新增渠道常量

渠道编号：**52**（当前 `Dummy` 的位置，`Dummy` 后移至 53）

```
GoogleCloudTTS = 52  // Google Cloud Text-to-Speech (Chirp)
Dummy          = 53  // sentinel
```

### 4.2 API 类型

新增 API 类型：

```
GoogleCloudTTS = 19  // 当前 Dummy 位置
Dummy          = 20  // sentinel
```

### 4.3 基础 URL

```
"https://texttospeech.googleapis.com"  // index 52
```

---

## 五、详细开发方案

### 5.1 需要新增/修改的文件清单

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `relay/channeltype/define.go` | 修改 | 新增 GoogleCloudTTS 常量 |
| `relay/channeltype/url.go` | 修改 | 添加 base URL |
| `relay/channeltype/helper.go` | 修改 | 添加 API 类型映射 |
| `relay/apitype/define.go` | 修改 | 新增 GoogleCloudTTS API 类型 |
| `relay/adaptor.go` | 修改 | 注册新 adaptor |
| `relay/adaptor/gcptts/` | **新建目录** | Google Cloud TTS adaptor |
| `relay/adaptor/gcptts/adaptor.go` | 新建 | Adaptor 接口实现 |
| `relay/adaptor/gcptts/model.go` | 新建 | 请求/响应数据结构 |
| `relay/adaptor/gcptts/constants.go` | 新建 | 模型列表和语音映射 |
| `relay/adaptor/gcptts/main.go` | 新建 | 请求转换和响应处理 |
| `relay/billing/ratio/model.go` | 修改 | 添加 Chirp 模型计费比率 |
| `relay/controller/audio.go` | 修改 | 添加 GoogleCloudTTS 分支 |
| `web/*/src/constants/channel.constants.js` | 修改 | 前端渠道选项 |

### 5.2 Step 1：注册渠道类型和 API 类型

**文件：`relay/channeltype/define.go`**

```go
const (
    // ... 现有常量 ...
    GeminiOpenAICompatible            // 51
    GoogleCloudTTS                    // 52 - 新增
    Dummy                             // 53 (sentinel)
)
```

**文件：`relay/channeltype/url.go`**

在 `ChannelBaseURLs` 末尾（`Dummy` 之前）添加：

```go
"https://texttospeech.googleapis.com", // 52 - GoogleCloudTTS
```

**文件：`relay/apitype/define.go`**

```go
const (
    // ... 现有常量 ...
    Replicate              // 18
    GoogleCloudTTS         // 19 - 新增
    Dummy                  // 20 (sentinel)
)
```

**文件：`relay/channeltype/helper.go`**

在 `ToAPIType()` switch 中添加：

```go
case GoogleCloudTTS:
    apiType = apitype.GoogleCloudTTS
```

### 5.3 Step 2：创建 Adaptor 包

#### 5.3.1 数据结构

**文件：`relay/adaptor/gcptts/model.go`**

```go
package gcptts

// SynthesizeRequest is the Google Cloud TTS API request
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
	AudioEncoding   string   `json:"audioEncoding"`
	SampleRateHertz int      `json:"sampleRateHertz,omitempty"`
	SpeakingRate    float64  `json:"speakingRate,omitempty"`
	Pitch           float64  `json:"pitch,omitempty"`
	VolumeGainDb    float64  `json:"volumeGainDb,omitempty"`
	EffectsProfileId []string `json:"effectsProfileId,omitempty"`
}

// SynthesizeResponse is the Google Cloud TTS API response
type SynthesizeResponse struct {
	AudioContent string `json:"audioContent"` // base64 encoded
}

// ErrorResponse for API errors
type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}
```

#### 5.3.2 常量和模型列表

**文件：`relay/adaptor/gcptts/constants.go`**

```go
package gcptts

// ModelList defines the supported Chirp models
var ModelList = []string{
	"chirp-3",
	"chirp-2",
	"chirp_telephony",
}

// DefaultAudioEncoding is the default output format
const DefaultAudioEncoding = "MP3"

// VoiceMap maps OpenAI voice names to Chirp 3 HD voice names
// When using Chirp models, the full voice name includes language prefix
// e.g., "en-US-Chirp3-HD-Charon"
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
	"aac":  "MP3",     // AAC not supported, fallback to MP3
	"flac": "LINEAR16", // FLAC not supported, fallback to LINEAR16
	"wav":  "LINEAR16",
	"pcm":  "LINEAR16",
	"":     "MP3",     // default
}

// AudioContentTypeMap maps audioEncoding to HTTP Content-Type
var AudioContentTypeMap = map[string]string{
	"MP3":       "audio/mpeg",
	"OGG_OPUS":  "audio/ogg",
	"LINEAR16":  "audio/wav",
	"ALAW":      "audio/alaw",
	"MULAW":     "audio/basic",
}

// DefaultLanguageCode used when no language hint is provided
const DefaultLanguageCode = "en-US"
```

#### 5.3.3 请求转换和响应处理

**文件：`relay/adaptor/gcptts/main.go`**

```go
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

// ConvertTTSRequest converts OpenAI TTS request to Google Cloud TTS format
func ConvertTTSRequest(ttsRequest openai.TextToSpeechRequest, modelName string) *SynthesizeRequest {
	// Determine audio encoding from response_format
	audioEncoding := DefaultAudioEncoding
	if enc, ok := AudioEncodingMap[ttsRequest.ResponseFormat]; ok {
		audioEncoding = enc
	}

	// Map voice name
	voiceName := resolveVoiceName(ttsRequest.Voice, modelName, DefaultLanguageCode)

	// Build request
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

	// Map speed to speakingRate (OpenAI: 0.25-4.0, Google: 0.25-4.0 - same range)
	if ttsRequest.Speed > 0 {
		req.AudioConfig.SpeakingRate = ttsRequest.Speed
	}

	return req
}

// resolveVoiceName builds the full Google Cloud TTS voice name
// Input: OpenAI voice name (e.g., "alloy") or Chirp voice name (e.g., "Charon")
// Output: Full voice name (e.g., "en-US-Chirp3-HD-Charon")
func resolveVoiceName(voice string, modelName string, languageCode string) string {
	// If it already looks like a full voice name (contains "-Chirp"), use as-is
	if strings.Contains(voice, "Chirp") {
		return voice
	}

	// Map OpenAI voice name to Chirp voice name
	chirpVoice := voice
	if mapped, ok := OpenAIVoiceMap[strings.ToLower(voice)]; ok {
		chirpVoice = mapped
	}

	// Get model-specific prefix
	prefix := "Chirp3-HD"
	if p, ok := ModelVoicePrefix[modelName]; ok {
		prefix = p
	}

	// Build full voice name: {lang}-{region}-{prefix}-{voice}
	return fmt.Sprintf("%s-%s-%s", languageCode, prefix, chirpVoice)
}

// TTSHandler handles Google Cloud TTS response
func TTSHandler(c *gin.Context, resp *http.Response, audioEncoding string) (*model.ErrorWithStatusCode, *model.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	defer resp.Body.Close()

	// Check for API error
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

	// Parse response
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

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(ttsResponse.AudioContent)
	if err != nil {
		return openai.ErrorWrapper(err, "decode_audio_failed", http.StatusInternalServerError), nil
	}

	// Determine Content-Type
	contentType := "audio/mpeg" // default
	if ct, ok := AudioContentTypeMap[audioEncoding]; ok {
		contentType = ct
	}

	// Write audio response
	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(audioData)))
	c.Writer.WriteHeader(http.StatusOK)
	_, err = c.Writer.Write(audioData)
	if err != nil {
		return openai.ErrorWrapper(err, "write_response_failed", http.StatusInternalServerError), nil
	}

	usage := &model.Usage{
		PromptTokens:     0,
		CompletionTokens: 0,
		TotalTokens:      0,
	}
	return nil, usage
}
```

#### 5.3.4 Adaptor 实现

**文件：`relay/adaptor/gcptts/adaptor.go`**

```go
package gcptts

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	channelhelper "github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

type Adaptor struct {
}

func (a *Adaptor) Init(meta *meta.Meta) {
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	// Google Cloud TTS has a single endpoint
	return fmt.Sprintf("%s/v1/text:synthesize", meta.BaseURL), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	channelhelper.SetupCommonRequestHeader(c, req, meta)
	// Use API Key via query parameter
	q := req.URL.Query()
	q.Set("key", meta.APIKey)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")
	return nil
}

func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	// TTS requests are handled separately in audio.go, not through this path
	return nil, errors.New("use ConvertTTSRequest for TTS")
}

func (a *Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	return nil, errors.New("image generation not supported for Google Cloud TTS")
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	return channelhelper.DoRequestHelper(a, c, meta, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode) {
	// TTS responses are handled separately in audio.go
	return nil, nil
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return "Google Cloud TTS"
}
```

### 5.4 Step 3：注册 Adaptor

**文件：`relay/adaptor.go`**

添加 import：

```go
import (
    // ... 现有 imports ...
    "github.com/songquanpeng/one-api/relay/adaptor/gcptts"
)
```

在 `GetAdaptor()` switch 中添加：

```go
case apitype.GoogleCloudTTS:
    return &gcptts.Adaptor{}
```

### 5.5 Step 4：修改 audio.go 添加 Google Cloud TTS 分支

**文件：`relay/controller/audio.go`**

在 `RelayAudioHelper` 函数中添加 GoogleCloudTTS 渠道分支：

```go
if relayMode == relaymode.AudioSpeech && channelType == channeltype.GoogleCloudTTS {
    // Convert OpenAI TTS request to Google Cloud TTS format
    gcpRequest := gcptts.ConvertTTSRequest(ttsRequest, audioModel)
    jsonBody, err := json.Marshal(gcpRequest)
    if err != nil {
        return openai.ErrorWrapper(err, "marshal_request_failed", http.StatusInternalServerError)
    }

    // Build Google Cloud TTS API URL
    apiKey := c.Request.Header.Get("Authorization")
    apiKey = strings.TrimPrefix(apiKey, "Bearer ")
    baseURL := channeltype.ChannelBaseURLs[channelType]
    if c.GetString(ctxkey.BaseURL) != "" {
        baseURL = c.GetString(ctxkey.BaseURL)
    }
    requestURL := fmt.Sprintf("%s/v1/text:synthesize?key=%s", baseURL, apiKey)

    req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonBody))
    if err != nil {
        return openai.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := client.HTTPClient.Do(req)
    if err != nil {
        return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
    }
    defer resp.Body.Close()

    // Determine audio encoding for Content-Type mapping
    audioEncoding := gcptts.DefaultAudioEncoding
    if enc, ok := gcptts.AudioEncodingMap[ttsRequest.ResponseFormat]; ok {
        audioEncoding = enc
    }

    errWithStatus, _ := gcptts.TTSHandler(c, resp, audioEncoding)
    if errWithStatus != nil {
        return errWithStatus
    }

    succeed = true
    quotaDelta := quota - preConsumedQuota
    defer func(ctx context.Context) {
        go billing.PostConsumeQuota(ctx, tokenId, quotaDelta, quota, userId, channelId, modelRatio, groupRatio, audioModel, tokenName)
    }(c.Request.Context())
    return nil
}
```

### 5.6 Step 5：添加计费比率

**文件：`relay/billing/ratio/model.go`**

```go
// Google Cloud TTS - Chirp models
// chirp-3 (Chirp 3 HD): $30/1M chars = $0.030/1K chars
//   ratio = $0.030 / $0.002 = 15
// chirp-2: $16/1M chars = $0.016/1K chars
//   ratio = $0.016 / $0.002 = 8
// chirp_telephony: $16/1M chars = $0.016/1K chars
//   ratio = $0.016 / $0.002 = 8
"chirp-3":         15,
"chirp-2":         8,
"chirp_telephony": 8,
```

### 5.7 Step 6：前端渠道选项

**文件：`web/default/src/constants/channel.constants.js`**（及其他主题）

在渠道类型列表中添加：

```javascript
{ key: 52, text: 'Google Cloud TTS (Chirp)', value: 52 },
```

---

## 六、OpenAI TTS → Google Cloud TTS 参数映射

### 6.1 请求参数映射

| OpenAI 参数 | Google Cloud TTS 对应 | 说明 |
|-------------|----------------------|------|
| `model` | 决定语音名称前缀 | `chirp-3` → `Chirp3-HD-*` |
| `input` | `input.text` | 直接映射 |
| `voice` | `voice.name` | 映射到完整语音名（见下方） |
| `speed` | `audioConfig.speakingRate` | 范围相同 0.25-4.0 |
| `response_format` | `audioConfig.audioEncoding` | mp3→MP3, opus→OGG_OPUS 等 |

### 6.2 语音映射流程

用户传入 → 判断类型 → 构建完整语音名

```
"alloy"                     → OpenAI 语音 → 映射为 "Kore"    → "en-US-Chirp3-HD-Kore"
"Charon"                    → Chirp 语音   → 直接使用        → "en-US-Chirp3-HD-Charon"
"en-US-Chirp3-HD-Charon"   → 完整语音名   → 直接使用        → "en-US-Chirp3-HD-Charon"
```

### 6.3 语言处理

Google Cloud TTS 的语音名称包含语言信息。考虑两种策略：

**策略 A（简单，推荐初版）**：默认 `en-US`，用户可通过传完整语音名指定语言。

**策略 B（进阶）**：在渠道配置的 "其他" JSON 字段中添加 `language_code` 配置：
```json
{
  "language_code": "zh-CN"
}
```

---

## 七、完整文件结构

```
relay/adaptor/gcptts/
├── adaptor.go       # Adaptor 接口实现
├── constants.go     # 模型列表、语音映射、编码映射
├── main.go          # 请求转换、响应处理
└── model.go         # 数据结构定义
```

---

## 八、高级功能（二期）

### 8.1 语言自动检测

通过渠道配置或请求扩展字段支持语言指定：

```json
// 渠道配置 "其他" 字段
{
  "language_code": "ja-JP",
  "audio_effects_profile": ["headphone-class-device"]
}
```

### 8.2 SSML 支持

检测输入文本是否以 `<speak>` 开头，自动使用 SSML 输入：

```go
if strings.HasPrefix(strings.TrimSpace(input), "<speak>") {
    req.Input = SynthesisInput{SSML: input}
} else {
    req.Input = SynthesisInput{Text: input}
}
```

### 8.3 自定义语音（Chirp 3 Instant Custom Voice）

Chirp 3 支持自定义语音克隆，需要额外的 API 调用来创建语音，属于高级功能。

### 8.4 流式合成

Google Cloud TTS 也支持流式合成（`streamingSynthesize`），适合长文本场景：

```
POST https://texttospeech.googleapis.com/v1/text:streamingSynthesize
```

返回分块音频数据，需要 SSE 或 gRPC 处理。

### 8.5 语音列表 API

提供管理接口查询可用语音：

```go
// GET /api/gcptts/voices?language=en-US
func ListVoices(c *gin.Context) {
    // Proxy to https://texttospeech.googleapis.com/v1/voices
}
```

---

## 九、测试计划

### 9.1 单元测试

```go
// relay/adaptor/gcptts/main_test.go

func TestConvertTTSRequest(t *testing.T) {
    req := openai.TextToSpeechRequest{
        Model: "chirp-3",
        Input: "Hello world",
        Voice: "alloy",
        Speed: 1.2,
        ResponseFormat: "mp3",
    }
    gcpReq := ConvertTTSRequest(req, "chirp-3")
    assert.Equal(t, "Hello world", gcpReq.Input.Text)
    assert.Equal(t, "en-US-Chirp3-HD-Kore", gcpReq.Voice.Name)
    assert.Equal(t, "en-US", gcpReq.Voice.LanguageCode)
    assert.Equal(t, "MP3", gcpReq.AudioConfig.AudioEncoding)
    assert.Equal(t, 1.2, gcpReq.AudioConfig.SpeakingRate)
}

func TestResolveVoiceName(t *testing.T) {
    tests := []struct{
        voice    string
        model    string
        expected string
    }{
        {"alloy", "chirp-3", "en-US-Chirp3-HD-Kore"},
        {"Charon", "chirp-3", "en-US-Chirp3-HD-Charon"},
        {"echo", "chirp-2", "en-US-Chirp-HD-Charon"},
        {"en-US-Chirp3-HD-Charon", "chirp-3", "en-US-Chirp3-HD-Charon"}, // passthrough
    }
    for _, tt := range tests {
        result := resolveVoiceName(tt.voice, tt.model, "en-US")
        assert.Equal(t, tt.expected, result)
    }
}

func TestAudioEncodingMap(t *testing.T) {
    assert.Equal(t, "MP3", AudioEncodingMap["mp3"])
    assert.Equal(t, "OGG_OPUS", AudioEncodingMap["opus"])
    assert.Equal(t, "LINEAR16", AudioEncodingMap["wav"])
    assert.Equal(t, "MP3", AudioEncodingMap[""])
}
```

### 9.2 集成测试

```bash
# 基本 TTS 请求
curl -X POST http://localhost:3000/v1/audio/speech \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "chirp-3",
    "input": "Hello, this is Google Cloud Text-to-Speech with Chirp 3.",
    "voice": "alloy"
  }' \
  --output test_chirp3.mp3

# 指定完整语音名
curl -X POST http://localhost:3000/v1/audio/speech \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "chirp-3",
    "input": "Hello from Charon voice.",
    "voice": "en-US-Chirp3-HD-Charon",
    "response_format": "opus"
  }' \
  --output test_charon.ogg

# Chirp 2 模型
curl -X POST http://localhost:3000/v1/audio/speech \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "chirp-2",
    "input": "Testing Chirp 2 model.",
    "voice": "echo",
    "speed": 1.5
  }' \
  --output test_chirp2.mp3

# 中文测试
curl -X POST http://localhost:3000/v1/audio/speech \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "chirp-3",
    "input": "你好，这是一段中文语音测试。",
    "voice": "zh-CN-Chirp3-HD-Kore"
  }' \
  --output test_chinese.mp3
```

### 9.3 验证清单

- [ ] 新渠道类型 `GoogleCloudTTS (52)` 在管理界面可选
- [ ] API Key 认证正确（通过 URL 参数传递）
- [ ] OpenAI 语音名称映射正确（alloy→Kore 等）
- [ ] Chirp 原生语音名称透传正确
- [ ] 完整语音名（含语言前缀）透传正确
- [ ] MP3/OGG_OPUS/LINEAR16 输出格式正确
- [ ] speed 参数正确映射到 speakingRate
- [ ] 计费正确（按字符数计费）
- [ ] 错误处理：API 错误、无效语音、超限
- [ ] 不影响现有 OpenAI/Azure/Gemini TTS 功能
- [ ] chirp-3、chirp-2、chirp_telephony 三个模型均可用

---

## 十、渠道配置示例

### 管理界面配置

| 字段 | 值 |
|------|-----|
| 类型 | Google Cloud TTS (Chirp) |
| 名称 | 自定义（如 "Google TTS"） |
| 密钥 | Google Cloud API Key |
| 代理 | 留空（使用默认 `texttospeech.googleapis.com`）或填区域端点 |
| 模型 | chirp-3, chirp-2, chirp_telephony |

### 高级配置（其他字段）

```json
{
  "language_code": "en-US"
}
```

---

## 十一、开发排期建议

| 阶段 | 任务 | 预计工作量 |
|------|------|-----------|
| Phase 1 | 渠道类型注册 + API 类型 + URL 映射 | 小 |
| Phase 1 | Adaptor 包创建（结构体 + 常量） | 小 |
| Phase 1 | 请求转换 + 响应处理 | 中 |
| Phase 1 | audio.go 分支逻辑 | 中 |
| Phase 1 | 计费比率 | 小 |
| Phase 1 | 前端渠道选项 | 小 |
| Phase 2 | 语言自动检测 / 渠道配置 | 中 |
| Phase 2 | SSML 支持 | 小 |
| Phase 3 | 流式合成 | 大 |
| Phase 3 | 语音列表管理 API | 中 |
| Phase 3 | 自定义语音支持 | 大 |
