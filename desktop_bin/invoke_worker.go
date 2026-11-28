//go:build windows || (linux && !android)

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"

	libXray "github.com/xtls/libxray"
)

type invokeWorkerRequest struct {
	ID      int64  `json:"id"`
	Request string `json:"request"`
}

type invokeWorkerResponse struct {
	ID       int64  `json:"id"`
	Response string `json:"response"`
}

type invokeWorkerReady struct {
	Ready bool `json:"ready"`
}

func serveInvoke(stdin io.Reader, stdout io.Writer) error {
	setWorkerProcessName("OneXrayHelper")

	writer := bufio.NewWriterSize(stdout, 64*1024)
	var writeMu sync.Mutex
	writeJSON := func(value any) error {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		if _, err := writer.Write(encoded); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
		return writer.Flush()
	}

	if err := writeJSON(invokeWorkerReady{Ready: true}); err != nil {
		return err
	}

	reader := bufio.NewReaderSize(stdin, 64*1024)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			var req invokeWorkerRequest
			if unmarshalErr := json.Unmarshal([]byte(trimmed), &req); unmarshalErr != nil {
				_ = writeJSON(invokeWorkerResponse{
					ID:       req.ID,
					Response: `{"success":false,"data":null,"error":"invalid invoke worker request"}`,
				})
			} else {
				go func(r invokeWorkerRequest) {
					resp := libXray.Invoke(r.Request)
					_ = writeJSON(invokeWorkerResponse{
						ID:       r.ID,
						Response: resp,
					})
				}(req)
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
	}
}
