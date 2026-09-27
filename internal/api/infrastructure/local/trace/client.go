package trace

import "context"

type Tracer struct{}

func NewNoopTracer() *Tracer { return &Tracer{} }

func NewTracer(string) (*Tracer, error) { return &Tracer{}, nil }

func (t *Tracer) Shutdown(context.Context) {}
