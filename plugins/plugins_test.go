package plugins

import (
	"context"
	"errors"
	"testing"
)

type recordingPlugin struct {
	name   string
	called *[]string
	err    error
}

func (p recordingPlugin) Name() string { return p.name }
func (p recordingPlugin) Apply(_ context.Context, _ *Context) error {
	*p.called = append(*p.called, p.name)
	return p.err
}

func TestRunInOrder(t *testing.T) {
	var calls []string
	plugins := []Plugin{
		recordingPlugin{name: "a", called: &calls},
		recordingPlugin{name: "b", called: &calls},
		recordingPlugin{name: "c", called: &calls},
	}
	if err := Run(context.Background(), plugins, &Context{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(calls) != 3 || calls[0] != "a" || calls[2] != "c" {
		t.Fatalf("call order: %v", calls)
	}
}

func TestRunShortCircuitsOnError(t *testing.T) {
	var calls []string
	boom := errors.New("boom")
	plugins := []Plugin{
		recordingPlugin{name: "a", called: &calls},
		recordingPlugin{name: "b", called: &calls, err: boom},
		recordingPlugin{name: "c", called: &calls},
	}
	err := Run(context.Background(), plugins, &Context{})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %v", calls)
	}
}
