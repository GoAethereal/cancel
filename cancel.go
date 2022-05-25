package cancel

import (
	"context"
	"sync"
	"time"
)

// Context grants access to the indication of a cancellation.
// It is a sub-set of the context.Context package.
type Context interface {
	// Done indicates the cancel state.
	// A call to Done() after the context was canceled is safe.
	// Typically a function may listen for cancellation like so:
	//
	//	func Do(ctx cancel.Context) {
	//		select {
	//		case <-ctx.Done():
	//			return
	//		default:
	//			// user logic
	//		}
	//	}
	//
	Done() <-chan struct{}
}

// New returns a useable cancellation signal.
func New() *Signal {
	sig := &Signal{}
	sig.init()
	return sig
}

// Signal is a ready to use cancel identifier.
// A termination only occurs once.
type Signal struct {
	done  chan struct{}
	make  sync.Once
	close sync.Once
}

var _ Context = (&Signal{})

// Done indicates the cancellation state of the signal.
// Receiving on the channel identifies the termination.
func (sig *Signal) Done() <-chan struct{} {
	sig.init()
	return sig.done
}

// Cancel manually terminates the signal.
// A call to cancel after cancellation is safe.
func (sig *Signal) Cancel() {
	sig.init()
	sig.close.Do(func() {
		close(sig.done)
	})
}

// init lazily prepares the signal.
func (sig *Signal) init() {
	sig.make.Do(func() {
		sig.done = make(chan struct{})
	})
}

// Timeout sets a new timeout on the signal.
// Other cancellation conditions still apply.
// The first one to reach it`s threshold will cancel the signal.
func (sig *Signal) Timeout(d time.Duration) *Signal {
	go func() {
		select {
		case <-time.After(d):
			sig.Cancel()
		case <-sig.Done():
		}
	}()
	return sig
}

// Deadline sets a deadline on the given signal.
// Other cancellation conditions still apply.
// The first one to reach it`s threshold will cancel the signal.
func (sig *Signal) Deadline(t time.Time) *Signal {
	return sig.Timeout(time.Until(t))
}

// Propagate escalates a cancellation from the parent to the signal.
// Other cancellation conditions still apply.
// The first one to reach it`s threshold will cancel the signal.
func (sig *Signal) Propagate(parent Context) *Signal {
	go func() {
		select {
		case <-parent.Done():
			sig.Cancel()
		case <-sig.Done():
		}
	}()
	return sig
}

// Promote wraps a simplified context in its standard library equivalent.
func Promote(ctx Context) (context.Context, func()) {
	sig, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-sig.Done():
		}
	}()
	return sig, cancel
}
