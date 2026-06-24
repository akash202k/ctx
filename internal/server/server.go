package server

import (
	"encoding/json"
	"log"
	"net/http"

	"ctxengine/internal/run"
)

// NewServer returns an *http.ServeMux with the /select endpoint registered.
// Call http.ListenAndServe(addr, mux) to start serving.
func NewServer() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/select", handleSelect)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return mux
}

func handleSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var inp run.Input
	if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	output, err := run.Run(inp)
	if err != nil {
		log.Printf("run error: %v", err)
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		log.Printf("encode error: %v", err)
	}
}
