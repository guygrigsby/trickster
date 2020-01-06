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

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
)

const (
	// Trace implementation enum
	StdoutTracerImplementation TracerImplementation = iota

	// TODO New Implemetations go here

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
)

func GlobalTracer(ctx context.Context) trace.Tracer {
	tracerName, ok := ctx.Value(tracerNameKey).(string)
	if !ok {
		return trace.NoopTracer{}

	}

	return global.TraceProvider().Tracer(tracerName)

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
