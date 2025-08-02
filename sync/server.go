package sync

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Syncer) Start(listenAddress string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sync", s.SyncHandler)
	s.Logger.Info("serving traffic", "address", listenAddress)
	return http.ListenAndServe(listenAddress, mux)
}

func (s *Syncer) SyncHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.Sync(); err != nil {
		s.Logger.Error("sync error", "error", err)
		writeJSON(w, http.StatusInternalServerError, fmt.Sprintf("sync error: %v", err))
	} else {
		writeJSON(w, http.StatusOK, "success")
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]any{
		"data": data,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode JSON response", http.StatusInternalServerError)
	}
}
