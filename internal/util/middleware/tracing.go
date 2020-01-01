package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Comcast/trickster/internal/config"
	"github.com/Comcast/trickster/internal/util/tracing"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
)

func Trace(originName, originType string, paths map[string]*config.PathConfig) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			fmt.Printf("%+v\n", paths)

			tracerName := "Request"

			pathNoOrigin := strings.Replace(r.URL.Path, fmt.Sprintf("/%s", originName), "", 1)

			cfg, ok := paths[pathNoOrigin]
			if ok {
				tracerName = cfg.HandlerName
			}

			r, span := tracing.PrepareRequest(r, tracerName, originName)
			defer func() {

				then := time.Now()
				span.End(trace.WithEndTime(then))
			}()
			span.AddEventWithTimestamp(
				r.Context(),
				time.Now(),
				"Starting Parent Span",
				key.String("originName", originName),
				key.String("originType", originType),
			)

			next.ServeHTTP(w, r)
		})
	}
}
