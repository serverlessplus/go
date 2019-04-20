package scfgo

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/tencentyun/scf-go-lib/functioncontext"
)

const (
	// Host is the host of HTTP server
	Host = "127.0.0.1"
)

// APIGatewayRequestContext represents a request context
type APIGatewayRequestContext struct {
	ServiceID string `json:"serviceId"`
	RequestID string `json:"requestId"`
	Method    string `json:"httpMethod"`
	Path      string `json:"path"`
	SourceIP  string `json:"sourceIp"`
	Stage     string `json:"stage"`
	Identity  struct {
		SecretID *string `json:"secretId"`
	} `json:"identity"`
}

// APIGatewayRequest represents an API gateway request
type APIGatewayRequest struct {
	Headers     map[string]string        `json:"headers"`
	Method      string                   `json:"httpMethod"`
	Path        string                   `json:"path"`
	QueryString map[string]interface{}   `json:"queryString"`
	Body        string                   `json:"body"`
	Context     APIGatewayRequestContext `json:"requestContext"`

	// the following fields are ignored
	// HeaderParameters      interface{} `json:"headerParameters"`
	// PathParameters        interface{} `json:"pathParameters"`
	// QueryStringParameters interface{} `json:"queryStringParameters"`
}

// APIGatewayResponse represents a API gateway response
type APIGatewayResponse struct {
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
}

// Handler represents a request handler
type Handler struct {
	client          *http.Client
	host            string
	port            int
	binaryMIMETypes map[string]struct{}
}

// NewHandler creates a new handler
func NewHandler(port int) *Handler {
	return &Handler{
		client: http.DefaultClient,
		host:   Host,
		port:   port,
	}
}

// WithClient allows user to specify a custom `http.Client`
func (h *Handler) WithClient(c *http.Client) *Handler {
	h.client = c
	return h
}

// WithBinaryMIMETypes allows user to specify MIME types that should be base64 encoded
func (h *Handler) WithBinaryMIMETypes(types map[string]struct{}) *Handler {
	h.binaryMIMETypes = types
	return h
}

// Handle processes the incoming request
func (h *Handler) Handle(ctx context.Context, r *APIGatewayRequest) (*APIGatewayResponse, error) {
	req := r.toHTTPRequest(ctx, h.port)
	resp, err := h.client.Do(req)
	if err != nil {
		fmt.Printf("send http request failed: %v\n", err)
		return &APIGatewayResponse{StatusCode: 500}, err
	}
	defer resp.Body.Close()
	return h.toAPIGatewayResponse(resp)
}

func toQueryString(m map[string]interface{}) string {
	var values url.Values
	for name, value := range m {
		switch value.(type) {
		case string:
			rawValue, _ := value.(string)
			values.Add(name, rawValue)
		case []string:
			rawValues, _ := value.([]string)
			for _, value := range rawValues {
				values.Add(name, value)
			}
		default:
			// should not reach here
			fmt.Printf("headerName=%s, headerValue=%v\n", name, value)
		}
	}
	return values.Encode()
}

func (r *APIGatewayRequest) toHTTPRequest(ctx context.Context, port int) *http.Request {
	req := http.Request{}
	req.Method = r.Method
	req.URL = &url.URL{
		Scheme:   "http",
		Host:     fmt.Sprintf("%s:%d", Host, port),
		RawPath:  r.Path,
		RawQuery: toQueryString(r.QueryString),
	}
	for name, value := range r.Headers {
		req.Header.Add(name, value)
	}
	// set request context to header
	funcCtx, ok := functioncontext.FromContext(ctx)
	if ok {
		req.Header.Add("x-scf-requestid", funcCtx.RequestID)
	}
	req.Header.Add("x-apigateway-serviceid", r.Context.ServiceID)
	req.Header.Add("x-apigateway-requestid", r.Context.RequestID)
	req.Header.Add("x-apigateway-method", r.Context.Method)
	req.Header.Add("x-apigateway-path", r.Context.Path)
	req.Header.Add("x-apigateway-sourceip", r.Context.SourceIP)
	req.Header.Add("x-forwarded-for", r.Context.SourceIP)
	req.Header.Add("x-apigateway-stage", r.Context.Stage)
	if r.Context.Identity.SecretID != nil {
		req.Header.Add("x-apigateway-secretid", *r.Context.Identity.SecretID)
	}
	req.Body = ioutil.NopCloser(bytes.NewBufferString(r.Body))
	return &req
}

func (h *Handler) toAPIGatewayResponse(r *http.Response) (*APIGatewayResponse, error) {
	resp := APIGatewayResponse{
		StatusCode: r.StatusCode,
		Headers:    make(map[string]string),
	}
	var contentType string
	for name, values := range r.Header {
		resp.Headers[name] = values[0]
		if strings.ToLower(name) == "content-type" {
			contentType = strings.Split(values[0], ";")[0]
		}
	}
	_, resp.IsBase64Encoded = h.binaryMIMETypes[contentType]
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return &APIGatewayResponse{StatusCode: 500}, err
	}
	if resp.IsBase64Encoded {
		resp.Body = base64.StdEncoding.EncodeToString(body)
	} else {
		resp.Body = string(body)
	}
	return &resp, nil
}
