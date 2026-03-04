//go:build ignore

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		var result any
		switch req.Method {
		case "plugin.init":
			result = map[string]any{}
		case "plugin.start":
			result = map[string]any{}
		case "plugin.stop":
			result = map[string]any{}
		case "plugin.handles":
			result = map[string]bool{"handles": true}
		case "plugin.upstreamFor":
			result = map[string]string{"upstream": "localhost:3000"}
		case "plugin.serviceStatus":
			result = map[string]int{"status": 1}
		case "plugin.startService":
			result = map[string]any{}
		case "plugin.stopService":
			result = map[string]any{}
		default:
			resp, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   map[string]any{"code": -32601, "message": "method not found"},
			})
			fmt.Fprintln(os.Stdout, string(resp))
			continue
		}

		resp, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  result,
		})
		fmt.Fprintln(os.Stdout, string(resp))
	}
}
