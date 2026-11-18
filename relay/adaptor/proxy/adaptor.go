package proxy

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/songquanpeng/one-api/relay/adaptor"
	channelhelper "github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

var _ adaptor.Adaptor = new(Adaptor)

const channelName = "proxy"

type Adaptor struct{}

func (a *Adaptor) Init(meta *meta.Meta) {
}

func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	return nil, errors.New("notimplement")
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode) {
	for k, v := range resp.Header {
		for _, vv := range v {
			c.Writer.Header().Set(k, vv)
		}
	}

	c.Writer.WriteHeader(resp.StatusCode)
	if _, gerr := io.Copy(c.Writer, resp.Body); gerr != nil {
		return nil, &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusInternalServerError,
			Error: relaymodel.Error{
				Message: gerr.Error(),
			},
		}
	}

	return nil, nil
}

func (a *Adaptor) GetModelList() (models []string) {
	return nil
}

func (a *Adaptor) GetChannelName() string {
	return channelName
}

// GetRequestURL remove static prefix, and return the real request url to the upstream service
func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	baseURL := strings.TrimSuffix(meta.BaseURL, "/")
	if strings.HasPrefix(meta.RequestURLPath, "/v1/oneapi/proxy/") {
		prefix := fmt.Sprintf("/v1/oneapi/proxy/%d", meta.ChannelId)
		return baseURL + strings.TrimPrefix(meta.RequestURLPath, prefix), nil
	}
	if strings.HasPrefix(meta.RequestURLPath, "/v1beta/") || strings.HasPrefix(meta.RequestURLPath, "/upload/v1beta/") {
		return baseURL + meta.RequestURLPath, nil
	}
	prefix := fmt.Sprintf("/v1/oneapi/proxy/%d", meta.ChannelId)
	return baseURL + strings.TrimPrefix(meta.RequestURLPath, prefix), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			req.Header.Set(k, v[0])
		}
	}

	// remove unnecessary headers
	req.Header.Del("Host")
	req.Header.Del("Content-Length")
	req.Header.Del("Accept-Encoding")
	req.Header.Del("Connection")
	req.Header.Del("Authorization")
	req.Header.Del("x-goog-api-key")
	req.Header.Del("X-Goog-Api-Key")

	// set authentication header according to channel type / upstream URL
	if meta.ChannelType == channeltype.Gemini || strings.Contains(meta.BaseURL, "googleapis.com") || (req.URL != nil && strings.Contains(req.URL.Host, "googleapis.com")) {
		req.Header.Set("x-goog-api-key", meta.APIKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+meta.APIKey)
	}

	return nil
}

func (a *Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	return nil, errors.Errorf("not implement")
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	return channelhelper.DoRequestHelper(a, c, meta, requestBody)
}
