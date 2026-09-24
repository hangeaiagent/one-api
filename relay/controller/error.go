package controller

import (
	"encoding/json"
	"fmt"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
	"io"
	"net/http"
	"strconv"
)

type GeneralErrorResponse struct {
	Error    model.Error `json:"error"`
	Message  string      `json:"message"`
	Msg      string      `json:"msg"`
	Err      string      `json:"err"`
	ErrorMsg string      `json:"error_msg"`
	Header   struct {
		Message string `json:"message"`
	} `json:"header"`
	Response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"response"`
}

func (e GeneralErrorResponse) ToMessage() string {
	if e.Error.Message != "" {
		return e.Error.Message
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != "" {
		return e.Err
	}
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	if e.Header.Message != "" {
		return e.Header.Message
	}
	if e.Response.Error.Message != "" {
		return e.Response.Error.Message
	}
	return ""
}

func RelayErrorHandler(resp *http.Response) (ErrorWithStatusCode *model.ErrorWithStatusCode) {
	if resp == nil {
		return &model.ErrorWithStatusCode{
			StatusCode: 500,
			Error: model.Error{
				Message: "resp is nil",
				Type:    "upstream_error",
				Code:    "bad_response",
			},
		}
	}
	ErrorWithStatusCode = &model.ErrorWithStatusCode{
		StatusCode: resp.StatusCode,
		Error: model.Error{
			Message: "",
			Type:    "upstream_error",
			Code:    "bad_response_status_code",
			Param:   strconv.Itoa(resp.StatusCode),
		},
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if config.DebugEnabled {
		logger.SysLog(fmt.Sprintf("error happened, status code: %d, response: \n%s", resp.StatusCode, string(responseBody)))
	}
	err = resp.Body.Close()
	if err != nil {
		return
	}
	var errResponse GeneralErrorResponse
	err = json.Unmarshal(responseBody, &errResponse)
	if err != nil {
		// Google 的 OpenAI 兼容端点（51 号渠道）把错误包在数组里：[{"error":{...}}]。
		// 按对象解析会失败，之前这里直接返回，错误信息就成了空串，排查时看不到上游原因。
		googleErr, ok := parseGoogleArrayError(responseBody)
		if !ok {
			return
		}
		ErrorWithStatusCode.Error = googleErr
		return
	}
	if errResponse.Error.Message != "" {
		// OpenAI format error, so we override the default one
		ErrorWithStatusCode.Error = errResponse.Error
	} else {
		ErrorWithStatusCode.Error.Message = errResponse.ToMessage()
	}
	if ErrorWithStatusCode.Error.Message == "" {
		ErrorWithStatusCode.Error.Message = fmt.Sprintf("bad response status code %d", resp.StatusCode)
	}
	return
}

// googleArrayError 是 Google OpenAI 兼容端点的错误体：code 是数字，status 是 UNAVAILABLE 这类字符串。
type googleArrayError []struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// parseGoogleArrayError 解析 [{"error":{...}}] 形式的错误体，取第一条。
// code 用 status 字符串（如 UNAVAILABLE），与 OpenAI 格式里 code 是字符串的约定一致。
func parseGoogleArrayError(body []byte) (model.Error, bool) {
	var arr googleArrayError
	if err := json.Unmarshal(body, &arr); err != nil || len(arr) == 0 || arr[0].Error.Message == "" {
		return model.Error{}, false
	}
	e := arr[0].Error
	code := any(e.Status)
	if e.Status == "" {
		code = strconv.Itoa(e.Code)
	}
	return model.Error{
		Message: e.Message,
		Type:    "upstream_error",
		Param:   strconv.Itoa(e.Code),
		Code:    code,
	}, true
}
