package handler

import (
	"encoding/json"
	"io"
	"net/http"

	sbexec "github.com/stackobrite/stackobrite-agent/internal/exec"
)

type ShellRequest struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

func Shell(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req ShellRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		http.Error(w, `{"error":"command is required"}`, http.StatusBadRequest)
		return
	}

	timeout := 300
	if req.Timeout > 0 && req.Timeout <= 300 {
		timeout = req.Timeout
	}

	resp := sbexec.Execute(sbexec.ExecRequest{
		Binary:  "bash",
		Args:    []string{"-c", req.Command},
		Timeout: timeout,
		WorkDir: "/",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
