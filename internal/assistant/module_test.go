package assistant

import (
	"context"
	"testing"
)

func TestNewInboundHookNilSafety(t *testing.T) {
	hook := NewInboundHook(nil)
	hook.OnInboundMessage(context.Background(), "conv-1", "msg-1")

	var nilHook *InboundEnqueuer
	nilHook.OnInboundMessage(context.Background(), "conv-1", "msg-1")
}

func TestNewModuleNilDB(t *testing.T) {
	mod := NewModule(nil, Config{Workers: 1}, nil, nil, nil, nil)
	if mod == nil {
		t.Fatal("expected non-nil module")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mod.StartWorker(ctx)

	var nilMod *Module
	nilMod.StartWorker(ctx)
}
