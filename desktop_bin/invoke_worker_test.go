//go:build windows || (linux && !android)

package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestServeInvokeSequentialAndConcurrent(t *testing.T) {
	if runtime.GOOS == "linux" {
		if orig, err := os.ReadFile("/proc/self/comm"); err == nil {
			origName := strings.TrimRight(string(orig), "\x00\r\n")
			t.Cleanup(func() {
				setWorkerProcessName(origName)
			})
		}
	}

	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveInvoke(inReader, outWriter)
		_ = outWriter.Close()
	}()

	scanner := bufio.NewReader(outReader)
	readyLine, err := scanner.ReadString('\n')
	if err != nil {
		t.Fatalf("read ready line: %v", err)
	}
	var ready invokeWorkerReady
	if err := json.Unmarshal([]byte(readyLine), &ready); err != nil || !ready.Ready {
		t.Fatalf("unexpected ready payload: %q (err=%v)", readyLine, err)
	}

	reqs := []invokeWorkerRequest{
		{ID: 1, Request: `{"apiVersion":3,"method":"xrayVersion","payload":{}}`},
		{ID: 2, Request: `{"apiVersion":3,"method":"getFreePorts","payload":{"count":1}}`},
		{ID: 3, Request: `{"apiVersion":99,"method":"xrayVersion","payload":{}}`},
	}
	for _, r := range reqs {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		if _, err := inWriter.Write(append(b, '\n')); err != nil {
			t.Fatalf("write request %d: %v", r.ID, err)
		}
	}
	if _, err := inWriter.Write([]byte("{not-json}\n")); err != nil {
		t.Fatalf("write malformed line: %v", err)
	}

	seen := make(map[int64]map[string]any)
	for i := 0; i < len(reqs)+1; i++ {
		line, err := scanner.ReadString('\n')
		if err != nil {
			t.Fatalf("read response %d: %v", i, err)
		}
		var env invokeWorkerResponse
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			t.Fatalf("unmarshal response %q: %v", line, err)
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(env.Response), &body); err != nil {
			t.Fatalf("unmarshal invoke body %q: %v", env.Response, err)
		}
		seen[env.ID] = body
	}

	if ok, _ := seen[1]["success"].(bool); !ok {
		t.Fatalf("expected xrayVersion success, got %v", seen[1])
	}
	if ok, _ := seen[2]["success"].(bool); !ok {
		t.Fatalf("expected getFreePorts success, got %v", seen[2])
	}
	if ok, _ := seen[3]["success"].(bool); ok {
		t.Fatalf("expected unsupported apiVersion failure, got %v", seen[3])
	}
	if ok, _ := seen[0]["success"].(bool); ok {
		t.Fatalf("expected malformed envelope failure, got %v", seen[0])
	}

	_ = inWriter.Close()
	if err := <-errCh; err != nil {
		t.Fatalf("serveInvoke returned error on EOF: %v", err)
	}
}
