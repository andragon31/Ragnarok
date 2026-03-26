package mcp

import (
	"context"
	"encoding/json"
)

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     string          `json:"id,omitempty"`
}

type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
	ID     string      `json:"id,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ToolHandler func(ctx context.Context, req *Request) (*Response, error)
