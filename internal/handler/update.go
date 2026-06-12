package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

var (
	Version   = "v0.1.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

type UpdateRequest struct {
	BinaryURL string `json:"binary_url"`
	CheckOnly bool   `json:"check_only"`
}

type UpdateResponse struct {
	Current  string `json:"current"`
	Target   string `json:"target,omitempty"`
	Updated  bool   `json:"updated"`
	Error    string `json:"error,omitempty"`
	Message  string `json:"message,omitempty"`
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(VersionInfo{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	})
}

func SelfUpdate(binaryDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, `{"error":"failed to read body"}`, http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req UpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if req.BinaryURL == "" {
			http.Error(w, `{"error":"binary_url is required"}`, http.StatusBadRequest)
			return
		}

		resp := UpdateResponse{Current: Version}

		if req.CheckOnly {
			resp.Message = "check only mode, no update performed"
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		currentPath, err := os.Executable()
		if err != nil {
			resp.Error = fmt.Sprintf("cannot find current binary: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
			return
		}

		tmpPath := currentPath + ".new"
		backupPath := currentPath + ".bak"

		if err := downloadBinary(req.BinaryURL, tmpPath); err != nil {
			resp.Error = fmt.Sprintf("download failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
			return
		}

		if err := os.Chmod(tmpPath, 0755); err != nil {
			os.Remove(tmpPath)
			resp.Error = fmt.Sprintf("chmod failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
			return
		}

		if err := os.Rename(currentPath, backupPath); err != nil {
			os.Remove(tmpPath)
			resp.Error = fmt.Sprintf("backup failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
			return
		}

		if err := os.Rename(tmpPath, currentPath); err != nil {
			os.Rename(backupPath, currentPath)
			resp.Error = fmt.Sprintf("swap failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
			return
		}

		resp.Updated = true
		resp.Message = "binary updated, restart required to take effect"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		go func() {
			time.Sleep(2 * time.Second)
			cmd := exec.Command("systemctl", "restart", "stackobrite-agent")
			cmd.Run()
		}()
	}
}

func downloadBinary(url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
