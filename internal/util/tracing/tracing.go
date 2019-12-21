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
	"fmt"
	"net/http"

	"github.com/Comcast/trickster/internal/runtime"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/distributedcontext"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
)

// Name returns the tracer name for this application
func Name() string {
	return fmt.Sprintf("%s/%s", runtime.ApplicationName, runtime.ApplicationVersion)

}

func SpanFromContext(ctx context.Context, spanName string) (context.Context, trace.Span) {
	tracerName := ctx.Value(tracerCtxKey).(string)
	tr := global.TraceProvider().Tracer(tracerName)

	attrs := ctx.Value(attrKey).([]core.KeyValue)
	spanCtx := ctx.Value(spanCtxKey).(core.SpanContext)

	ctx, span := tr.Start(
		ctx,
		spanName,
		trace.WithAttributes(attrs...),
		trace.ChildOf(spanCtx),
	)
	return ctx, span

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
	ctx = context.WithValue(ctx, tracerCtxKey, tracerName)

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
type tracerCtxType struct{}

var (
	attrKey      = ctxAttrType{}
	spanCtxKey   = &ctxSpanType{}
	tracerCtxKey = &tracerCtxType{}
)
