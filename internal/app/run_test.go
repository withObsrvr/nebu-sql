package app

import (
	"context"
	"errors"
	"testing"
)

func TestIsCanceled(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	liveCtx := context.Background()

	cases := []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{name: "context canceled", ctx: canceledCtx, err: context.Canceled, want: true},
		{name: "wrapped context canceled", ctx: liveCtx, err: errors.New("context canceled"), want: true},
		{name: "signal killed", ctx: liveCtx, err: errors.New("processor exited with error: signal: killed"), want: true},
		{name: "other", ctx: liveCtx, err: errors.New("boom"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isCanceled(tc.ctx, tc.err)
			if got != tc.want {
				t.Fatalf("isCanceled() = %v, want %v", got, tc.want)
			}
		})
	}
}
