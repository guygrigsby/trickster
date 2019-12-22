package middleware

import (
	"net/http"
	"time"

	"github.com/Comcast/trickster/internal/util/tracing"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
)

func Trace(tracerName string) mux.MiddlewareFunc {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			r, span := tracing.PrepareRequest(r, tracerName, tracing.MiddlewareSpanName)
			defer func() {

				then := time.Now()
				span.End(trace.WithEndTime(then))
			}()
			span.AddEvent(r.Context(), "Middleware Event", key.String(tracing.RequestIDKey, "TODO"))

			next.ServeHTTP(w, r)
		})
	}
}
