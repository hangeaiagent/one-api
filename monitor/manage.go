package monitor

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
)

// channel429Cache 存储渠道 429 错误的时间戳
var channel429Cache = make(map[int]time.Time)
var channel429Mutex sync.RWMutex

func ShouldDisableChannel(err *model.Error, statusCode int) bool {
	if !config.AutomaticDisableChannelEnabled {
		return false
	}
	if err == nil {
		return false
	}
	if statusCode == http.StatusUnauthorized {
		return true
	}
	switch err.Type {
	case "insufficient_quota", "authentication_error", "permission_error", "forbidden":
		return true
	}
	if err.Code == "invalid_api_key" || err.Code == "account_deactivated" {
		return true
	}

	lowerMessage := strings.ToLower(err.Message)
	if strings.Contains(lowerMessage, "your access was terminated") ||
		strings.Contains(lowerMessage, "violation of our policies") ||
		strings.Contains(lowerMessage, "your credit balance is too low") ||
		strings.Contains(lowerMessage, "organization has been disabled") ||
		strings.Contains(lowerMessage, "credit") ||
		strings.Contains(lowerMessage, "balance") ||
		strings.Contains(lowerMessage, "permission denied") ||
		strings.Contains(lowerMessage, "organization has been restricted") || // groq
		strings.Contains(lowerMessage, "api key not valid") || // gemini
		strings.Contains(lowerMessage, "api key expired") || // gemini
		strings.Contains(lowerMessage, "已欠费") {
		return true
	}
	return false
}

// ShouldTemporarilyDisableChannelFor429 检查是否应该因为 429 错误临时禁用渠道
func ShouldTemporarilyDisableChannelFor429(statusCode int) bool {
	if !config.Channel429AutoDisable {
		return false
	}
	return statusCode == http.StatusTooManyRequests
}

// RecordChannel429 记录渠道 429 错误
func RecordChannel429(channelId int) {
	if !config.Channel429AutoDisable {
		return
	}
	channel429Mutex.Lock()
	defer channel429Mutex.Unlock()
	channel429Cache[channelId] = time.Now()
	logger.SysLogf("channel #%d returned 429, recorded in cache", channelId)
}

// IsChannel429Blocked 检查渠道是否因为 429 错误被临时阻止
func IsChannel429Blocked(channelId int) bool {
	if !config.Channel429AutoDisable {
		return false
	}
	channel429Mutex.RLock()
	defer channel429Mutex.RUnlock()
	blockedTime, exists := channel429Cache[channelId]
	if !exists {
		return false
	}
	// 检查是否还在冷却期内
	elapsed := time.Since(blockedTime)
	return elapsed < time.Duration(config.Channel429DisableDuration)*time.Second
}

// CleanExpired429Records 清理过期的 429 记录
func CleanExpired429Records() {
	channel429Mutex.Lock()
	defer channel429Mutex.Unlock()
	now := time.Now()
	expirationDuration := time.Duration(config.Channel429DisableDuration) * time.Second
	for channelId, blockedTime := range channel429Cache {
		if now.Sub(blockedTime) > expirationDuration {
			delete(channel429Cache, channelId)
		}
	}
}

// init429Cleaner 初始化 429 记录清理协程
func init429Cleaner() {
	if config.Channel429AutoDisable {
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				CleanExpired429Records()
			}
		}()
	}
}

func ShouldEnableChannel(err error, openAIErr *model.Error) bool {
	if !config.AutomaticEnableChannelEnabled {
		return false
	}
	if err != nil {
		return false
	}
	if openAIErr != nil {
		return false
	}
	return true
}
