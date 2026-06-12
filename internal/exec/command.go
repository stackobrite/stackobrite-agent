package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

type ExecRequest struct {
	Binary  string   `json:"binary"`
	Args    []string `json:"args"`
	Timeout int      `json:"timeout"`
	WorkDir string   `json:"work_dir"`
}

type ExecResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
}

func Execute(req ExecRequest) ExecResponse {
	if !IsAllowed(req.Binary) {
		return ExecResponse{
			ExitCode: 1,
			Error:    fmt.Sprintf("command not allowed: %s", req.Binary),
		}
	}

	timeout := 300
	if req.Timeout > 0 && req.Timeout <= 300 {
		timeout = req.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	binPath, err := findBinary(req.Binary)
	if err != nil {
		return ExecResponse{
			ExitCode: 1,
			Error:    fmt.Sprintf("binary not found: %s", req.Binary),
		}
	}

	args := make([]string, len(req.Args))
	copy(args, req.Args)

	cmd := exec.CommandContext(ctx, binPath, args...)

	if req.WorkDir != "" {
		cmd.Dir = req.WorkDir
	} else {
		cmd.Dir = "/"
	}

	cmd.Env = append(cmd.Env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	cmd.Env = append(cmd.Env, "HOME=/root")
	cmd.Env = append(cmd.Env, "KUBECONFIG=/etc/kubernetes/admin.conf")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	resp := ExecResponse{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: duration.String(),
	}

	if ctx.Err() == context.DeadlineExceeded {
		resp.Error = "command timed out"
	}

	return resp
}

func findBinary(name string) (string, error) {
	paths := []string{
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
		"/snap/bin",
		"/usr/local/go/bin",
		"/home/ubuntu/.local/bin",
	}

	for _, p := range paths {
		full := filepath.Join(p, name)
		if _, err := exec.LookPath(full); err == nil {
			return full, nil
		}
	}

	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("binary %s not found in PATH", name)
}

func MarshalResponse(resp ExecResponse) ([]byte, error) {
	return json.Marshal(resp)
}
