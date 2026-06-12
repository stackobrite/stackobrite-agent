package handler

import (
	"encoding/json"
	"net/http"
	"os"
)

type KubeconfigResponse struct {
	Kubeconfig string `json:"kubeconfig"`
	Path       string `json:"path"`
	Error      string `json:"error,omitempty"`
}

func Kubeconfig(kubeconfigPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		data, err := os.ReadFile(kubeconfigPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(KubeconfigResponse{
				Error: "kubeconfig not found: " + err.Error(),
				Path:  kubeconfigPath,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(KubeconfigResponse{
			Kubeconfig: string(data),
			Path:       kubeconfigPath,
		})
	}
}
