package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	mbtrace "moonbridge/internal/service/trace"
)

type AnthropicConfig struct {
	UpstreamBaseURL string
	APIKey          string
	Version         string
	Client          *http.Client
	Tracer          *mbtrace.Tracer
	TraceErrors     io.Writer
	// IsEnabled is called on each request; if nil or returns false, returns 404.
	IsEnabled func() bool
	// ModelMap maps incoming model names to upstream model names.
	ModelMap map[string]string
}

type AnthropicServer struct {
	upstreamBaseURL string
	apiKey          string
	version         string
	client          *http.Client
	tracer          *mbtrace.Tracer
	traceErrors     io.Writer
	isEnabled       func() bool
	modelMap        map[string]string
}

func NewAnthropic(cfg AnthropicConfig) (*AnthropicServer, error) {
	upstreamBaseURL, err := normalizeUpstreamBaseURL(cfg.UpstreamBaseURL, "https://api.anthropic.com/v1")
	if err != nil {
		return nil, err
	}
	client := cfg.Client
	if client == nil {
		client = http.DefaultClient
	}
	return &AnthropicServer{
		upstreamBaseURL: upstreamBaseURL,
		apiKey:          cfg.APIKey,
		version:         cfg.Version,
		client:          client,
		tracer:          cfg.Tracer,
		traceErrors:     cfg.TraceErrors,
		isEnabled:       cfg.IsEnabled,
		modelMap:        cfg.ModelMap,
	}, nil
}

func (server *AnthropicServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if server.isEnabled != nil && !server.isEnabled() {
		http.Error(writer, "Anthropic 代理未启用", http.StatusNotFound)
		return
	}
	server.serveProxy(writer, request)
}

func (server *AnthropicServer) serveProxy(writer http.ResponseWriter, request *http.Request) {
	log := slog.Default().With("path", request.URL.Path, "method", request.Method)
	log.Debug("代理请求已收到")
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		log.Error("读取请求体失败", "error", err)
		http.Error(writer, "读取请求体失败", http.StatusBadRequest)
		return
	}

	// Model mapping: if the body has a "model" field that matches the map, replace it.
	if len(server.modelMap) > 0 {
		var bodyMap map[string]any
		if err := json.Unmarshal(requestBody, &bodyMap); err == nil {
			if model, ok := bodyMap["model"].(string); ok {
				if mapped, ok := server.modelMap[model]; ok {
					log.Info("模型映射", "from", model, "to", mapped)
					bodyMap["model"] = mapped
					if newBody, err := json.Marshal(bodyMap); err == nil {
						requestBody = newBody
					}
				}
			}
		}
	}

	targetURL := server.upstreamBaseURL + "/messages"
	upstreamRequest, err := newUpstreamRequest(request, targetURL, requestBody, server.overrideAuth)
	if err != nil {
		http.Error(writer, "创建上游请求失败", http.StatusBadGateway)
		return
	}

	record := mbtrace.Record{
		HTTPRequest: mbtrace.NewHTTPRequest(request),
		ProxyRequest: ProxyRequest{
			Method:  request.Method,
			URL:     request.URL.RequestURI(),
			Headers: request.Header.Clone(),
			Body:    mbtrace.RawJSONOrString(requestBody),
		},
		UpstreamRequest: ProxyRequest{
			Method:  upstreamRequest.Method,
			URL:     targetURL,
			Headers: upstreamRequest.Header.Clone(),
			Body:    mbtrace.RawJSONOrString(requestBody),
		},
	}

	upstreamResponse, err := server.client.Do(upstreamRequest)
	if err != nil {
		log.Error("上游请求失败", "error", err)
		record.Error = map[string]string{"stage": "upstream_request", "message": err.Error()}
		writeTrace(server.tracer, server.traceErrors, record)
		http.Error(writer, err.Error(), http.StatusBadGateway)
		return
	}
	defer upstreamResponse.Body.Close()

	copyHeaders(writer.Header(), upstreamResponse.Header)
	writer.WriteHeader(upstreamResponse.StatusCode)

	var responseBody bytes.Buffer
	copyErr := copyStreaming(writer, upstreamResponse.Body, &responseBody)
	record.UpstreamResponse = ProxyResponse{
		StatusCode: upstreamResponse.StatusCode,
		Headers:    upstreamResponse.Header.Clone(),
		Body:       mbtrace.RawJSONOrString(responseBody.Bytes()),
	}
	if copyErr != nil {
		log.Error("复制上游响应失败", "error", copyErr)
		record.Error = map[string]string{"stage": "copy_upstream_response", "message": copyErr.Error()}
	}
	log.Info("代理响应", "status", upstreamResponse.StatusCode, "bytes", responseBody.Len())
	writeTrace(server.tracer, server.traceErrors, record)
}

func (server *AnthropicServer) overrideAuth(headers http.Header) {
	headers.Del("Authorization")
	if server.apiKey != "" {
		headers.Set("X-Api-Key", server.apiKey)
	}
	if server.version != "" {
		headers.Set("Anthropic-Version", server.version)
	}
}
