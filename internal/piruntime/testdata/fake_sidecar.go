// Fake sidecar for piruntime client tests. Build/run:
//
//	go run ./internal/piruntime/testdata/fake_sidecar.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	sc := bufio.NewScanner(os.Stdin)
	// Allow long JSON lines in tests.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	for sc.Scan() {
		line := sc.Bytes()
		var peek map[string]any
		if err := json.Unmarshal(line, &peek); err != nil {
			writeJSON(map[string]any{"ok": false, "error_code": "bad_json", "error_message": err.Error()})
			continue
		}
		if typ, _ := peek["type"].(string); typ == "handshake" {
			writeJSON(map[string]any{
				"type":             "handshake_ok",
				"protocol_version": 1,
				"ok":               true,
			})
			continue
		}
		tool, _ := peek["tool_name"].(string)
		cwd, _ := peek["cwd"].(string)
		var args struct {
			Path string `json:"path"`
		}
		if raw, ok := peek["args"]; ok {
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &args)
		}
		writeJSON(map[string]any{
			"ok":      true,
			"content": fmt.Sprintf("echo:%s:%s:%s", tool, args.Path, cwd),
		})
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeJSON(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(string(b))
}
