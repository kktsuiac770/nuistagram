package tracing

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type Span struct {
	name       string
	startTime  time.Time
	endTime    time.Time
	attributes map[string]interface{}
	events     []Event
	status     SpanStatus
	mu         sync.Mutex
}

type SpanStatus struct {
	Code        string
	Description string
}

type Event struct {
	Name       string
	Time       time.Time
	Attributes map[string]interface{}
}

type Tracer struct {
	serviceName string
	exporter    Exporter
}

type Exporter interface {
	ExportSpans(spans []*Span) error
}

type stdoutExporter struct {
	w io.Writer
}

func (e *stdoutExporter) ExportSpans(spans []*Span) error {
	for _, s := range spans {
		duration := s.endTime.Sub(s.startTime)
		fmt.Fprintf(e.w, "---\nSPAN: %s\nDuration: %v\nStatus: %s\n", s.name, duration, s.status.Code)
		if len(s.attributes) > 0 {
			fmt.Fprintf(e.w, "Attributes:\n")
			for k, v := range s.attributes {
				fmt.Fprintf(e.w, "  %s: %v\n", k, v)
			}
		}
		if len(s.events) > 0 {
			fmt.Fprintf(e.w, "Events:\n")
			for _, ev := range s.events {
				fmt.Fprintf(e.w, "  %s @ %v\n", ev.Name, ev.Time.Format(time.RFC3339Nano))
			}
		}
	}
	return nil
}

type ctxKeySpan struct{}

var defaultTracer *Tracer

func Init(serviceName string, exporter Exporter) {
	if exporter == nil {
		exporter = &stdoutExporter{w: os.Stdout}
	}
	defaultTracer = &Tracer{
		serviceName: serviceName,
		exporter:    exporter,
	}
}

func Start(ctx context.Context, name string, attrs ...interface{}) (context.Context, *Span) {
	span := &Span{
		name:       name,
		startTime:  time.Now(),
		attributes: make(map[string]interface{}),
	}
	for i := 0; i < len(attrs)-1; i += 2 {
		if key, ok := attrs[i].(string); ok {
			span.attributes[key] = attrs[i+1]
		}
	}
	return context.WithValue(ctx, ctxKeySpan{}, span), span
}

func (s *Span) End() {
	s.mu.Lock()
	s.endTime = time.Now()
	s.mu.Unlock()

	if defaultTracer != nil && defaultTracer.exporter != nil {
		defaultTracer.exporter.ExportSpans([]*Span{s})
	}
}

func (s *Span) SetAttribute(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attributes[key] = value
}

func (s *Span) AddEvent(name string, attrs ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event := Event{Name: name, Time: time.Now(), Attributes: make(map[string]interface{})}
	for i := 0; i < len(attrs)-1; i += 2 {
		if key, ok := attrs[i].(string); ok {
			event.Attributes[key] = attrs[i+1]
		}
	}
	s.events = append(s.events, event)
}

func (s *Span) SetStatus(code, desc string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = SpanStatus{Code: code, Description: desc}
}

func (s *Span) RecordError(err error) {
	s.AddEvent("exception", "type", "error", "message", err.Error())
	s.SetStatus("ERROR", err.Error())
}

func SpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(ctxKeySpan{}).(*Span); ok {
		return span
	}
	return nil
}

func LoggerAttrs(span *Span) []interface{} {
	if span == nil {
		return nil
	}
	attrs := []interface{}{slog.String("span_name", span.name)}
	for k, v := range span.attributes {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}
