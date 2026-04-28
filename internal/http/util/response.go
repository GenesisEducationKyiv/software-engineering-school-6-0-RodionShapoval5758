package util

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorResponse struct {
	ErrorMessage string `json:"error"`
}

func WriteJSONResponse(w http.ResponseWriter, statusCode int, t any) {

	data, err := json.Marshal(t)
	if err != nil {
		log.Printf("Encoding to json has failed: %v", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(data)
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, err string) {
	WriteJSONResponse(w, statusCode, ErrorResponse{ErrorMessage: err})
}
