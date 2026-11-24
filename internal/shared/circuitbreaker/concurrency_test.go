package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSingleProbeAndStaleResults(t *testing.T) {
	for _, lateResult := range []error{nil, dependencyFailure} {
		t.Run(map[bool]string{true: "late failure", false: "late success"}[lateResult != nil], func(t *testing.T) {
			b, now := testBreaker(t, 1)
			started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			go func() {
				done <- b.Do(context.Background(), func() error {
					close(started)
					<-release
					return lateResult
				}, func(err error) bool { return errors.Is(err, dependencyFailure) })
			}()
			<-started
			if err := call(b, dependencyFailure); !errors.Is(err, dependencyFailure) {
				t.Fatal(err)
			}
			*now = now.Add(time.Second)
			probeStarted, probeRelease, probeDone := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			go func() {
				probeDone <- b.Do(context.Background(), func() error {
					close(probeStarted)
					<-probeRelease
					return nil
				}, func(err error) bool { return errors.Is(err, dependencyFailure) })
			}()
			<-probeStarted
			executed := false
			if err := b.Do(context.Background(), func() error { executed = true; return nil }, func(error) bool { return false }); !errors.Is(err, ErrOpen) || executed {
				t.Fatalf("concurrent probe = %v, executed = %v", err, executed)
			}
			close(probeRelease)
			if err := <-probeDone; err != nil {
				t.Fatal(err)
			}
			close(release)
			if err := <-done; !errors.Is(err, lateResult) {
				t.Fatalf("late result = %v", err)
			}
			if err := call(b, nil); err != nil {
				t.Fatalf("stale completion changed recovered state: %v", err)
			}
		})
	}
}

func TestPanickingProbeIsReleased(t *testing.T) {
	b, now := testBreaker(t, 1)
	if err := call(b, dependencyFailure); !errors.Is(err, dependencyFailure) {
		t.Fatal(err)
	}
	*now = now.Add(time.Second)
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("panic was not propagated")
			}
		}()
		_ = b.Do(context.Background(), func() error { panic("probe") }, func(error) bool { return true })
	}()
	if err := call(b, nil); err != nil {
		t.Fatalf("probe not released: %v", err)
	}
}
