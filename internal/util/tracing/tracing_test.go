package tracing

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/api/distributedcontext"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/exporter/trace/stdout"
	"go.opentelemetry.io/otel/plugin/httptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func init() {
	// Create stdout exporter to be able to retrieve
	// the collected spans.
	exporter, err := stdout.NewExporter(stdout.Options{PrettyPrint: true})
	if err != nil {
		log.Fatal(err)
	}

	// For the demonstration, use sdktrace.AlwaysSample sampler to sample all traces.
	// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
}

func setup(routes map[string]http.HandlerFunc) *mux.Router {

	router := mux.Router{}
	router.Use(func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r, span := PrepareRequest(r, "tracername", "middleware-span-name")
			defer span.End()
			span.AddEvent(r.Context(), "", key.String("internal-id", "trickster"))

			fmt.Println("MIDDLEWARE START")
			next.ServeHTTP(w, r)
			fmt.Println("MIDDLEWARE END")

		})
	})

	for route, handler := range routes {
		router.HandleFunc(route, handler)

	}

	return &router
}

func TestTrace(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"/test": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := SpanFromContext(r.Context(), "test-span-name")
			defer span.End()
			span.AddEvent(ctx, "", key.String("server", "add-green-chili"))
			fmt.Println("REQUEST")

		}),
	}
	router := setup(routes)
	go func() {
		if err := http.ListenAndServe(":8080", router); err != nil {
			panic(err)
		}
	}()

	client := http.DefaultClient
	ctx := distributedcontext.NewContext(context.Background(),
		key.String("username", "guy"),
		key.String("burritotype", "carnitas"),
	)

	req, _ := http.NewRequest("GET", "http://localhost:8080/test", nil)

	// For full stack use:
	//ctx, req = httptrace.W3C(ctx, req)
	httptrace.Inject(ctx, req)

	fmt.Printf("Sending request...\n")
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	_ = res.Body.Close()

}
