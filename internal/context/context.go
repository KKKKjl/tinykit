package context

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/KKKKjl/tinykit/internal/response"
)

type IHttpContext interface {
}

type HttpContext struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	IsAbort        bool
	Method         string
	Err            error
	MetaData       map[string]interface{}
}

func New(w http.ResponseWriter, r *http.Request) HttpContext {
	return HttpContext{
		ResponseWriter: w,
		Request:        r,
		Method:         r.Method,
		MetaData:       make(map[string]interface{}),
		IsAbort:        false,
		Err:            nil,
	}
}

func (c *HttpContext) SetValue(key interface{}, value interface{}) {
	ctx := context.WithValue(c.Request.Context(), key, value)
	c.Request.WithContext(ctx)
}

func (c *HttpContext) GetValue(key interface{}) interface{} {
	return c.Request.Context().Value(key)
}

func (c *HttpContext) SetMetaData(key string, value interface{}) {
	if _, ok := c.MetaData[key]; !ok {
		c.MetaData[key] = value
	}
}

func (c *HttpContext) GetMetaData(key string) (interface{}, error) {
	if value, ok := c.MetaData[key]; ok {
		return value, nil
	}

	return nil, fmt.Errorf("%s not found in metadata", key)
}

func (c *HttpContext) ToJSON(obj interface{}) {
	c.SetResponseHeader("Content-Type", "application/json; charset=utf-8")

	if val, ok := obj.([]byte); ok {
		c.ResponseWriter.Write(val)
		return
	}

	encoder := json.NewEncoder(c.ResponseWriter)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *HttpContext) ToString(format string, values ...interface{}) {
	c.SetResponseHeader("Content-Type", "text/plain")
	c.ResponseWriter.Write([]byte(fmt.Sprintf(format, values...)))
}

// should be called before writing data to response writer or it will rewrite the status code
func (c *HttpContext) WriteStatusCode(statusCode int) {
	c.ResponseWriter.WriteHeader(statusCode)
}

func (c *HttpContext) Abort() {
	c.IsAbort = true
}

func (c *HttpContext) AbortWithStatus(code int) {
	c.WriteStatusCode(code)
	c.Abort()
}

func (c *HttpContext) AbortWithMsg(msg string) {
	c.AbortWithStatus(http.StatusInternalServerError)
	c.ToJSON(&response.ResponseModel{
		Code:    http.StatusInternalServerError,
		Message: msg,
	})
}

func (c *HttpContext) Error(err error) {
	c.Err = err
}

func (c *HttpContext) SetResponseHeader(key, value string) {
	c.ResponseWriter.Header().Set(key, value)
}

func (c *HttpContext) SetResponseHeaders(headers map[string]string) {
	for key, value := range headers {
		c.SetResponseHeader(key, value)
	}
}
