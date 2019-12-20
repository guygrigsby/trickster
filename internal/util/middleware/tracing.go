package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel/api/global"
)

func TraceHandler(path string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		global.TraceProvider().Tracer("ex.com/basic")

	})
}
