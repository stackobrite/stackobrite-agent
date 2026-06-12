package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	osexec "os/exec"
	"time"

	sbexec "github.com/stackobrite/stackobrite-agent/internal/exec"
)

type StreamEvent struct {
	Type     string `json:"type"`
	Data     string `json:"data,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
}

func Stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	binary := r.URL.Query().Get("binary")
	argsStr := r.URL.Query().Get("args")

	if binary == "" {
		http.Error(w, `{"error":"binary is required"}`, http.StatusBadRequest)
		return
	}

	if !sbexec.IsAllowed(binary) {
		http.Error(w, fmt.Sprintf(`{"error":"command not allowed: %s"}`, binary), http.StatusForbidden)
		return
	}

	var args []string
	if argsStr != "" {
		json.Unmarshal([]byte(argsStr), &args)
	}

	timeout := 300
	if t := r.URL.Query().Get("timeout"); t != "" {
		fmt.Sscanf(t, "%d", &timeout)
		if timeout > 300 {
			timeout = 300
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	binPath, err := osexec.LookPath(binary)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"binary not found: %s"}`, binary), http.StatusNotFound)
		return
	}

	cmd := osexec.CommandContext(ctx, binPath, args...)
	cmd.Dir = "/"

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		sendSSE(w, StreamEvent{Type: "error", Error: err.Error()})
		return
	}

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				sendSSE(w, StreamEvent{Type: "stdout", Data: string(buf[:n])})
				flusher.Flush()
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				sendSSE(w, StreamEvent{Type: "stderr", Data: string(buf[:n])})
				flusher.Flush()
			}
			if err != nil {
				break
			}
		}
	}()

	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*osexec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	sendSSE(w, StreamEvent{Type: "done", ExitCode: &exitCode})
	flusher.Flush()
}

func sendSSE(w io.Writer, event StreamEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
}
