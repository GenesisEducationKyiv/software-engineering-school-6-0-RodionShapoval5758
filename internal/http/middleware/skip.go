package middleware

import "net/http"

func SkipRoutes(mw func(http.Handler) http.Handler, paths ...string) func(http.Handler) http.Handler {
	skip := make(map[string]bool, len(paths))
	for _, p := range paths {
		skip[p] = true
	}
	return func(next http.Handler) http.Handler {
		withMw := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			withMw.ServeHTTP(w, r)
		})
	}
}
