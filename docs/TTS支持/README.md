# Gemini 2.5 TTS 模型支持方案

## 一、目标

在现有 Gemini 渠道（channeltype=24）中支持以下 TTS 模型：

- `gemini-2.5-flash-tts`
- `gemini-2.5-pro-tts`

用户通过标准 OpenAI TTS 接口 `POST /v1/audio/speech` 发送请求，One API 自动将其转换为 Gemini generateContent 格式，返回音频数据。

---

## 二、Google Gemini TTS API 说明

### 2.1 端点

与普通 Gemini 聊天相同，使用 `generateContent` 端点：

```
POST https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-tts:generateContent
```

认证方式不变：`x-goog-api-key` Header 或 URL 参数 `?key=API_KEY`。

### 2.2 请求格式

**单说话人：**

```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        { "text": "Say cheerfully: I'm doing well, thank you!" }
      ]
    }
  ],
  "generationConfig": {
    "responseModalities": ["AUDIO"],
    "speechConfig": {
      "voiceConfig": {
        "prebuiltVoiceConfig": {
          "voiceName": "Kore"
        }
      }
    }
  }
}
```

**多说话人（最多 2 个）：**

```json
{
  "generationConfig": {
    "responseModalities": ["AUDIO"],
    "speechConfig": {
      "multiSpeakerVoiceConfig": {
        "speakerVoiceConfigs": [
          {
            "speaker": "Joe",
            "voiceConfig": {
              "prebuiltVoiceConfig": { "voiceName": "Kore" }
            }
          },
          {
            "speaker": "Jane",
            "voiceConfig": {
              "prebuiltVoiceConfig": { "voiceName": "Puck" }
            }
          }
        ]
      }
    }
  }
}
```

### 2.3 响应格式

```json
{
  "candidates": [
    {
      "content": {
        "parts": [
          {
            "inlineData": {
              "mimeType": "audio/L16;codec=pcm;rate=24000",
              "data": "<base64-encoded-PCM-audio>"
            }
          }
        ]
      }
    }
  ]
}
```

- 音频格式：PCM 24kHz，16-bit，单声道
- 编码：base64
- **注意**：返回的是裸 PCM 数据，不含 WAV 头，需要自行添加 WAV 头才能直接播放

### 2.4 可用语音（30 个）

Achernar, Achird, Algenib, Algieba, Alnilam, Aoede, Autonoe, Callirrhoe,
Charon, Despina, Enceladus, Erinome, Fenrir, Gacrux, Iapetus, Kore,
Laomedeia, Leda, Orus, Puck, Pulcherrima, Rasalgethi, Sadachbia,
Sadaltager, Schedar, Sulafat, Umbriel, Vindemiatrix, Zephyr, Zubenelgenubi

### 2.5 定价

| 模型 | 输入价格 (per 1M tokens) | 输出价格 (per 1M tokens) | 按字符 (per 1M chars) |
|------|--------------------------|--------------------------|----------------------|
| gemini-2.5-flash-tts | $0.50 | $10.00 | $0.12 |
| gemini-2.5-pro-tts | $1.00 | $20.00 | $0.24 |

### 2.6 特性

- 支持 90+ 种语言，自动语言检测
- 支持风格提示（情感、语速、口音控制）
- 单次最大 32,000 tokens
- Flash 首字节延迟约 200ms，Pro 约 450ms

---

## 三、架构分析与核心难点

### 3.1 当前请求流程

```
Client → POST /v1/audio/speech
       → relaymode.GetByPath() → AudioSpeech
       → controller/relay.go → RelayAudioHelper()
       → relay/controller/audio.go → 直接 HTTP 代理到上游（仅 OpenAI/Azure）
```

**核心问题**：`RelayAudioHelper` 不走 Adaptor 模式，直接代理 HTTP 请求。而 Gemini TTS 需要：
1. 将 OpenAI TTS 请求格式转换为 Gemini generateContent 格式
2. 将 Gemini 响应中的 inlineData 音频提取为裸音频流返回

### 3.2 方案选择

| 方案 | 描述 | 优点 | 缺点 |
|------|------|------|------|
| A：重构 audio handler 为 Adaptor 模式 | 像 text handler 一样走 adaptor | 架构统一，可扩展 | 改动大，影响现有 OpenAI/Azure TTS |
| **B：在 audio handler 中添加 Gemini 分支** | 检测 channelType 为 Gemini 时走专用逻辑 | 改动小，不影响现有功能 | 代码有些耦合 |

**推荐方案 B**，改动最小、风险最低。

---

## 四、详细开发方案

### 4.1 需要修改的文件清单

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `relay/adaptor/geminiv2/constants.go` | 修改 | 添加 TTS 模型到 ModelList |
| `relay/adaptor/gemini/constants.go` | 修改 | 添加 TTS 模型到支持列表 |
| `relay/adaptor/gemini/model.go` | 修改 | 添加 SpeechConfig 结构体 |
| `relay/adaptor/gemini/main.go` | 修改 | 添加 TTS 请求转换和响应处理函数 |
| `relay/billing/ratio/model.go` | 修改 | 添加 TTS 模型计费比率 |
| `relay/controller/audio.go` | 修改 | 添加 Gemini 渠道 TTS 处理分支 |

### 4.2 Step 1：添加模型定义

**文件：`relay/adaptor/geminiv2/constants.go`**

在 ModelList 中添加：

```go
// Gemini 2.5 TTS models
"gemini-2.5-flash-tts",
"gemini-2.5-pro-tts",
```

**文件：`relay/adaptor/gemini/constants.go`**

在 ModelList 引用中确保包含 TTS 模型（如果是引用 geminiv2.ModelList 则无需额外改动）。

### 4.3 Step 2：添加 SpeechConfig 结构体

**文件：`relay/adaptor/gemini/model.go`**

新增以下结构体：

```go
// SpeechConfig for TTS
type SpeechConfig struct {
	VoiceConfig           *VoiceConfig           `json:"voiceConfig,omitempty"`
	MultiSpeakerVoiceConfig *MultiSpeakerVoiceConfig `json:"multiSpeakerVoiceConfig,omitempty"`
}

type VoiceConfig struct {
	PrebuiltVoiceConfig *PrebuiltVoiceConfig `json:"prebuiltVoiceConfig,omitempty"`
}

type PrebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

type MultiSpeakerVoiceConfig struct {
	SpeakerVoiceConfigs []SpeakerVoiceConfig `json:"speakerVoiceConfigs"`
}

type SpeakerVoiceConfig struct {
	Speaker     string      `json:"speaker"`
	VoiceConfig VoiceConfig `json:"voiceConfig"`
}
```

修改 `ChatGenerationConfig`，添加 `SpeechConfig` 字段：

```go
type ChatGenerationConfig struct {
	ResponseMimeType    string        `json:"responseMimeType,omitempty"`
	ResponseSchema      any           `json:"responseSchema,omitempty"`
	ResponseModalities  []string      `json:"responseModalities,omitempty"`
	SpeechConfig        *SpeechConfig `json:"speechConfig,omitempty"`  // 新增
	Temperature         *float64      `json:"temperature,omitempty"`
	TopP                *float64      `json:"topP,omitempty"`
	TopK                float64       `json:"topK,omitempty"`
	MaxOutputTokens     int           `json:"maxOutputTokens,omitempty"`
	CandidateCount      int           `json:"candidateCount,omitempty"`
	StopSequences       []string      `json:"stopSequences,omitempty"`
}
```

### 4.4 Step 3：添加 TTS 请求转换函数

**文件：`relay/adaptor/gemini/main.go`**

新增 OpenAI TTS → Gemini generateContent 转换函数：

```go
// ConvertTTSRequest converts OpenAI TTS request to Gemini generateContent format
func ConvertTTSRequest(ttsRequest openai.TextToSpeechRequest) *ChatRequest {
	// Map OpenAI voice to Gemini voice name
	// OpenAI voices: alloy, echo, fable, onyx, nova, shimmer
	// Gemini voices: Kore, Puck, Charon, Aoede, Fenrir, Leda, ...
	voiceName := mapOpenAIVoiceToGemini(ttsRequest.Voice)

	geminiRequest := ChatRequest{
		Contents: []ChatContent{
			{
				Role: "user",
				Parts: []Part{
					{
						Text: ttsRequest.Input,
					},
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

	return &geminiRequest
}

// mapOpenAIVoiceToGemini maps OpenAI voice names to Gemini voice names
// Users can also pass Gemini voice names directly
func mapOpenAIVoiceToGemini(voice string) string {
	voiceMap := map[string]string{
		"alloy":   "Kore",
		"echo":    "Charon",
		"fable":   "Achernar",
		"onyx":    "Fenrir",
		"nova":    "Leda",
		"shimmer": "Aoede",
	}
	if mapped, ok := voiceMap[voice]; ok {
		return mapped
	}
	// Allow direct Gemini voice names (e.g., "Kore", "Puck")
	return voice
}
```

新增 TTS 响应处理函数：

```go
// TTSHandler handles Gemini TTS response, extracts audio and returns raw audio bytes
func TTSHandler(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	defer resp.Body.Close()

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
			StatusCode: resp.StatusCode,
		}, nil
	}

	inlineData := geminiResponse.Candidates[0].Content.Parts[0].InlineData
	audioData, err := base64.StdEncoding.DecodeString(inlineData.Data)
	if err != nil {
		return openai.ErrorWrapper(err, "decode_audio_data_failed", http.StatusInternalServerError), nil
	}

	// Determine output format based on MIME type
	mimeType := inlineData.MimeType
	if strings.Contains(mimeType, "pcm") || strings.Contains(mimeType, "L16") {
		// Convert raw PCM to WAV for compatibility
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

	// Usage: estimate based on input text length
	usage := &model.Usage{
		PromptTokens:     0,
		CompletionTokens: 0,
		TotalTokens:      0,
	}
	return nil, usage
}

// pcmToWav adds WAV header to raw PCM data
func pcmToWav(pcmData []byte, sampleRate, bitsPerSample, numChannels int) []byte {
	dataSize := len(pcmData)
	fileSize := 36 + dataSize // 44-byte header - 8 for RIFF header

	header := make([]byte, 44)
	// RIFF header
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(fileSize))
	copy(header[8:12], "WAVE")
	// fmt sub-chunk
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // sub-chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)  // PCM format
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	blockAlign := numChannels * bitsPerSample / 8
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))
	// data sub-chunk
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))

	return append(header, pcmData...)
}
```

### 4.5 Step 4：修改 audio.go 添加 Gemini 分支

**文件：`relay/controller/audio.go`**

在 `RelayAudioHelper` 函数中，当 `relayMode == relaymode.AudioSpeech` 且 `channelType == channeltype.Gemini` 时，走专用逻辑：

```go
// 在现有代码的 HTTP 请求构建之前（约 line 118 之后），添加 Gemini 分支：

if relayMode == relaymode.AudioSpeech && channelType == channeltype.Gemini {
    // Convert OpenAI TTS request to Gemini format
    geminiRequest := gemini.ConvertTTSRequest(ttsRequest)
    jsonBody, err := json.Marshal(geminiRequest)
    if err != nil {
        return openai.ErrorWrapper(err, "marshal_request_failed", http.StatusInternalServerError)
    }

    // Build Gemini API URL
    apiKey := c.Request.Header.Get("Authorization")
    apiKey = strings.TrimPrefix(apiKey, "Bearer ")
    baseURL := channeltype.ChannelBaseURLs[channelType]
    if c.GetString(ctxkey.BaseURL) != "" {
        baseURL = c.GetString(ctxkey.BaseURL)
    }
    requestURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent", baseURL, audioModel)

    req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonBody))
    if err != nil {
        return openai.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-goog-api-key", apiKey)

    resp, err := client.HTTPClient.Do(req)
    if err != nil {
        return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return RelayErrorHandler(resp)
    }

    errWithStatus, _ := gemini.TTSHandler(c, resp)
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

### 4.6 Step 5：添加计费比率

**文件：`relay/billing/ratio/model.go`**

在 Gemini 模型定价区域添加：

```go
// Gemini 2.5 TTS - 按字符计费
// gemini-2.5-flash-tts: $0.12/1M chars ≈ $0.00012/1K chars
// gemini-2.5-pro-tts:   $0.24/1M chars ≈ $0.00024/1K chars
// 换算为 ratio：1K chars = ratio * $0.002
// flash: $0.00012 / $0.002 = 0.06
// pro:   $0.00024 / $0.002 = 0.12
"gemini-2.5-flash-tts": 0.06,
"gemini-2.5-pro-tts":   0.12,
```

### 4.7 Step 6：版本路由

**文件：`relay/adaptor/gemini/adaptor.go`**

在 `GetRequestURL` 中添加 `gemini-2.5` 的版本检测：

```go
if strings.Contains(meta.ActualModelName, "gemini-3.1") ||
    strings.Contains(meta.ActualModelName, "gemini-3.0") ||
    strings.Contains(meta.ActualModelName, "gemini-3-") ||
    strings.Contains(meta.ActualModelName, "gemini-2.5") ||  // 新增
    strings.Contains(meta.ActualModelName, "gemini-2.0") ||
    strings.Contains(meta.ActualModelName, "gemini-1.5") {
    defaultVersion = "v1beta"
}
```

---

## 五、OpenAI TTS → Gemini TTS 参数映射

| OpenAI 参数 | Gemini 对应 | 说明 |
|-------------|------------|------|
| `model` | URL 中的模型名 | `tts-1` → `gemini-2.5-flash-tts` (通过模型映射) |
| `input` | `contents[0].parts[0].text` | 要合成的文本 |
| `voice` | `speechConfig.voiceConfig.prebuiltVoiceConfig.voiceName` | 语音映射（见 4.4） |
| `speed` | 不支持 | Gemini 通过文本提示控制语速 |
| `response_format` | 固定 PCM→WAV | Gemini TTS 固定返回 PCM |

### 语音映射表

| OpenAI Voice | Gemini Voice | 描述 |
|-------------|-------------|------|
| alloy | Kore | 中性平衡 |
| echo | Charon | 低沉男声 |
| fable | Achernar | 叙述风格 |
| onyx | Fenrir | 深沉有力 |
| nova | Leda | 明亮女声 |
| shimmer | Aoede | 柔和温暖 |

用户也可直接传入 Gemini 原生语音名称（如 `Puck`、`Zephyr` 等）。

---

## 六、测试计划

### 6.1 单元测试

```go
// relay/adaptor/gemini/tts_test.go
func TestConvertTTSRequest(t *testing.T) {
    req := openai.TextToSpeechRequest{
        Model: "gemini-2.5-flash-tts",
        Input: "Hello world",
        Voice: "alloy",
    }
    geminiReq := ConvertTTSRequest(req)
    assert.Equal(t, []string{"AUDIO"}, geminiReq.GenerationConfig.ResponseModalities)
    assert.Equal(t, "Kore", geminiReq.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName)
    assert.Equal(t, "Hello world", geminiReq.Contents[0].Parts[0].Text)
}

func TestMapOpenAIVoiceToGemini(t *testing.T) {
    assert.Equal(t, "Kore", mapOpenAIVoiceToGemini("alloy"))
    assert.Equal(t, "Puck", mapOpenAIVoiceToGemini("Puck")) // passthrough
}

func TestPcmToWav(t *testing.T) {
    pcm := make([]byte, 100)
    wav := pcmToWav(pcm, 24000, 16, 1)
    assert.Equal(t, 144, len(wav)) // 44-byte header + 100 bytes data
    assert.Equal(t, "RIFF", string(wav[0:4]))
    assert.Equal(t, "WAVE", string(wav[8:12]))
}
```

### 6.2 集成测试

```bash
# 使用 OpenAI 兼容格式调用 Gemini TTS
curl -X POST http://localhost:3000/v1/audio/speech \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-flash-tts",
    "input": "Hello, this is a test of Gemini text to speech.",
    "voice": "alloy"
  }' \
  --output test.wav

# 使用 Gemini 原生语音名称
curl -X POST http://localhost:3000/v1/audio/speech \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-pro-tts",
    "input": "你好，这是一段中文语音测试。",
    "voice": "Kore"
  }' \
  --output test_zh.wav
```

### 6.3 验证清单

- [ ] OpenAI TTS 格式请求能正确转换为 Gemini 格式
- [ ] 6 种 OpenAI 语音映射正确
- [ ] Gemini 原生语音名称透传正确
- [ ] PCM → WAV 转换正确，生成的 WAV 文件可播放
- [ ] 计费正确（按字符数计费）
- [ ] 错误处理：输入过长、无效语音名、API 错误
- [ ] 不影响现有 OpenAI/Azure TTS 功能
- [ ] 模型在管理界面的渠道配置中可见

---

## 七、开发优先级与风险

| 优先级 | 任务 | 风险 |
|--------|------|------|
| P0 | 模型列表 + 计费比率 | 低 |
| P0 | SpeechConfig 结构体 | 低 |
| P0 | TTS 请求转换 | 中 - 需确保格式正确 |
| P0 | audio.go Gemini 分支 | 中 - 核心改动 |
| P1 | PCM→WAV 转换 | 低 |
| P1 | 语音映射优化 | 低 |
| P2 | 流式 TTS 支持 | 高 - 需要额外研究 |

### 风险项

1. **PCM 格式兼容性**：Gemini 返回裸 PCM，某些客户端可能期望 MP3/OGG 格式。初版仅支持 WAV 输出。
2. **字符限制**：OpenAI TTS 限制 4096 字符，Gemini 支持 32,000 tokens，需考虑是否放宽限制。
3. **流式 TTS**：Gemini 支持 `streamGenerateContent`，但音频流式输出的处理更复杂，建议二期实现。
