package ridgenative

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type invoke struct {
	id      string
	payload []byte
	headers http.Header
}

func callBytesHandlerFunc(ctx context.Context, payload []byte, h handlerFunc) (response []byte, err error) {
	defer func() {
		if v := recover(); v != nil {
			err = lambdaPanicResponse(v)
		}
	}()

	var req *request
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}
	resp, err := h(ctx, req)
	if err != nil {
		return nil, err
	}
	return json.Marshal(resp)
}

func callHandlerFuncSteaming(ctx context.Context, payload []byte, h handlerFuncSteaming) (response io.ReadCloser, contentType string, err error) {
	defer func() {
		if v := recover(); v != nil {
			err = lambdaPanicResponse(v)
		}
	}()

	var req *request
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, "", err
	}

	r, w := io.Pipe()
	contentType, err = h(ctx, req, w)
	if err != nil {
		return nil, "", err
	}
	return r, contentType, nil
}
