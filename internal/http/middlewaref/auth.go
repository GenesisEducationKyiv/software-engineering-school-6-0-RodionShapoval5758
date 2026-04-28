package middlewaref

import "net/http"

func AuthAPIKEY(apiKey string) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			authField := "Bearer " + apiKey
			if request.Header.Get("Authorization") != authField {
				http.Error(writer, "Not authorized", http.StatusUnauthorized)
				return
			}
			handler.ServeHTTP(writer, request)
		})
	}
}
