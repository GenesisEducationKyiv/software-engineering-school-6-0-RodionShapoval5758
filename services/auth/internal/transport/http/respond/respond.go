package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type ErrorResponse struct {
	ErrorMessage string `json:"error"`
}

func JSON(w http.ResponseWriter, statusCode int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		slog.Error("encoding response to json failed", "error", err)
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
