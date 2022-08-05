package request

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/KKKKjl/tinykit/internal/marshaler"
)

type (
	HttpClient struct {
		parser marshaler.Marshaler
	}

	httpResponseModel struct {
	}
)

func NewHttpClient() *HttpClient {
	return &HttpClient{
		parser: new(marshaler.JsonMarshaler),
	}
}

func (h *HttpClient) Call(url string, data map[string]interface{}) (*httpResponseModel, error) {
	bytesData, err := h.parser.Marshal(data)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, bytes.NewReader(bytesData))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var model httpResponseModel
	if err := h.parser.UnMarshal(buf, &model); err != nil {
		return nil, err
	}

	return &model, nil
}
