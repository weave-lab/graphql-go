package trace

import (
	"github.com/weave-lab/graphql-go/errors"
)

type TraceValidationFinishFunc = TraceQueryFinishFunc

type ValidationTracer interface {
	TraceValidation() TraceValidationFinishFunc
}

type NoopValidationTracer struct{}

func (NoopValidationTracer) TraceValidation() TraceValidationFinishFunc {
	return func(errs []*errors.QueryError) {}
}
