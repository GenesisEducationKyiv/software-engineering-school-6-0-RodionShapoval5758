package respond

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorResponse struct {
	ErrorMessage string `json:"error"`
}

func JSON(w http.ResponseWriter, statusCode int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		log.Printf("Encoding to json has failed: %v", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(data)
}

func Error(w http.ResponseWriter, statusCode int, msg string) {
	JSON(w, statusCode, ErrorResponse{ErrorMessage: msg})
}
