/**
* Copyright 2018 Comcast Cable Communications Management, LLC
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
* http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package tracing

import (
	"context"
	"net/http"
	"sync"

	"github.com/Comcast/trickster/internal/config"
	"github.com/Comcast/trickster/internal/util/log"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/distributedcontext"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
)

const (
	MiddlewareSpanName = "trickster-middleware-span"
	RequestIDKey       = "trickster-internal-id"
	ServiceName        = "trickster"
)

const (
	// Trace implementation enum
	StdoutTracerImplementation TracerImplementation = iota

	// New Implemetations go here

	JaegerTracer
)

type TracerImplementation int

var (
	tracerImplemetationStrings = []string{
		"stdout",
		"jaeger",
	}
	TracerImplementations = map[string]TracerImplementation{

		tracerImplemetationStrings[StdoutTracerImplementation]: StdoutTracerImplementation,
		tracerImplemetationStrings[JaegerTracer]:               JaegerTracer,
	}

	once sync.Once
)

// Init initializes tracing
func Init(cfg *config.TracingConfig) func() {
	log.Debug(
		"Trace Init",
		log.Pairs{
			"Implementation": cfg.Implementation,
			"Collector":      cfg.CollectorEndpoint,
			"Type":           TracerImplementations[cfg.Implementation],
		},
	)
	var flusher func()
	f := func() {
		fl, err := SetTracer(
			TracerImplementations[cfg.Implementation],
			cfg.CollectorEndpoint,
		)
		if err != nil {
			log.Error(
				"Cannot initialize tracing",
				log.Pairs{
					"Error":     err,
					"Tracer":    cfg.Implementation,
					"Collector": cfg.CollectorEndpoint,
				},
			)
		}
		flusher = fl

	}
	once.Do(f)
	return flusher
}

func (t TracerImplementation) String() string {
	if t < StdoutTracerImplementation || t > JaegerTracer {
		return "unknown-tracer"
	}
	return tracerImplemetationStrings[t]
}

func SetTracer(t TracerImplementation, collectorURL string) (func(), error) {

	switch t {
	case StdoutTracerImplementation:

		return setStdOutTracer()
	case JaegerTracer:

		return setJaegerTracer(collectorURL)
	default:

		return setStdOutTracer()
	}

}
func NewSpan(ctx context.Context, tracerName string, spanName string) (context.Context, trace.Span) {
	tr := global.TraceProvider().Tracer(tracerName)

	attrs := ctx.Value(attrKey).([]core.KeyValue)
	spanCtx := ctx.Value(spanCtxKey).(core.SpanContext)

	ctx, span := tr.Start(
		ctx,
		spanName,
		trace.WithAttributes(attrs...),
		trace.ChildOf(spanCtx),
	)
	if span == nil {
		// Just in case
		span = trace.NoopSpan{}
	}
	return ctx, span

}

type currentSpanKeyType struct{}

var (
	currentSpanKey = &currentSpanKeyType{}
)

func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return context.WithValue(ctx, currentSpanKey, span)
}

func SpanFromContext(ctx context.Context) trace.Span {
	if span, has := ctx.Value(currentSpanKey).(trace.Span); has {
		return span
	}
	return trace.NoopSpan{}
}
func PrepareRequest(r *http.Request, tracerName string, spanName string) (*http.Request, trace.Span) {

	attrs, entries, spanCtx := httptrace.Extract(r.Context(), r)

	ctx := distributedcontext.WithMap(
		r.Context(),
		distributedcontext.NewMap(
			distributedcontext.MapUpdate{
				MultiKV: entries,
			},
		),
	)

	ctx = context.WithValue(ctx, attrKey, attrs)
	ctx = context.WithValue(ctx, spanCtxKey, spanCtx)

	tr := global.TraceProvider().Tracer(tracerName)

	ctx, span := tr.Start(
		ctx,
		spanName,
		trace.WithAttributes(attrs...),
		trace.ChildOf(spanCtx),
	)

	return r.WithContext(ctx), span
}

type ctxSpanType struct{}
type ctxAttrType struct{}

var (
	attrKey    = ctxAttrType{}
	spanCtxKey = &ctxSpanType{}
)
