package server

import (
	"encoding/json"
	"fmt"
)

// Request is one JSONL/JSON-RPC request from a host adapter.
type Request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is a JSONL result, error, or server-pushed event.
type Response struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Result interface{}     `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
	Method string          `json:"method,omitempty"`
	Params interface{}     `json:"params,omitempty"`
}

// Error is a protocol-level failure.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func rpcError(id json.RawMessage, code, message string) Response {
	return Response{ID: id, Error: &Error{Code: code, Message: message}}
}

func rpcResult(id json.RawMessage, result interface{}) Response {
	return Response{ID: id, Result: result}
}

func rpcEvent(method string, params interface{}) Response {
	return Response{Method: method, Params: params}
}

func decodeParams[T any](raw json.RawMessage) (T, error) {
	var out T
	if len(raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("invalid params: %w", err)
	}
	return out, nil
}
