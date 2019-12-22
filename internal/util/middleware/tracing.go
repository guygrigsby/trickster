package middleware

import (
	"net/http"
	"time"

	"github.com/Comcast/trickster/internal/util/tracing"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
)

func Trace(originName, originType string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			r, span := tracing.PrepareRequest(r, r.URL.Path)
			defer func() {

				then := time.Now()
				span.End(trace.WithEndTime(then))
			}()
			span.AddEventWithTimestamp(
				r.Context(),
				time.Now(),
				r.URL.Path,
				key.String("originName", originName),
				key.String("originType", originType),
				key.String("path", r.URL.Path),
			)

			next.ServeHTTP(w, r)
		})
	}
}
