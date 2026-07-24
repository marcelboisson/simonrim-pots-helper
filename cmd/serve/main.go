package main

import (
	"log/slog"
	"net/http"
	"os"
)

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		h.ServeHTTP(w, r)
	})
}

func fatal(msg string, args ...any) {
	slog.Error("fatal: "+msg, args...)
	os.Exit(1)
}
func main() {
	if len(os.Args) != 2 {
		fatal("missing dir")
	}

	dir := os.Args[1]
	listen := "0.0.0.0:3000"

	fs := http.FileServer(http.Dir(dir))
	http.Handle("/", noCache(fs))

	slog.Info("Listening", "listen", listen)
	err := http.ListenAndServe(listen, nil)
	if err != nil {
		fatal("listen", "err", err)
	}
}
