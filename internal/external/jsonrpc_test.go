package external

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestRPCClientCall(t *testing.T) {
	// Simulate a plugin subprocess: reads request from stdin, writes response to stdout
	reqBuf := &bytes.Buffer{}
	respJSON := `{"jsonrpc":"2.0","id":1,"result":{"handles":true}}` + "\n"
	respReader := strings.NewReader(respJSON)

	client := newRPCClient(respReader, reqBuf)

	var result struct {
		Handles bool `json:"handles"`
	}
	err := client.Call("plugin.handles", map[string]any{"site": "test"}, &result)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if !result.Handles {
		t.Fatal("expected handles=true")
	}

	// Verify the request was written correctly
	var req rpcRequest
	if err := json.NewDecoder(bytes.NewReader(reqBuf.Bytes())).Decode(&req); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if req.Method != "plugin.handles" {
		t.Errorf("method = %q, want plugin.handles", req.Method)
	}
}

func TestRPCClientCallError(t *testing.T) {
	respJSON := `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"plugin error"}}` + "\n"
	client := newRPCClient(strings.NewReader(respJSON), io.Discard)

	var result struct{}
	err := client.Call("plugin.init", nil, &result)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "plugin error") {
		t.Errorf("error = %q, want contains 'plugin error'", err.Error())
	}
}

func TestRPCClientCallNilResult(t *testing.T) {
	respJSON := `{"jsonrpc":"2.0","id":1,"result":{}}` + "\n"
	client := newRPCClient(strings.NewReader(respJSON), io.Discard)

	err := client.Call("plugin.start", nil, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
}
