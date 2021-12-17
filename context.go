package context

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"

	"bou.ke/monkey"
)

const (
	namedErrFormat              = "named context [%s]"
	namedParentErrFormat        = "named context [%s] parent err"
	namedDeadlineExceededFormat = "named context [%s] deadline exceeded after [%dms]"
)

var (
	closedchan   = make(chan struct{})
	cancelCtxKey int
)

func init() {
	close(closedchan)
}

type (
	canceler interface {
		fmt.Stringer
		cancel(removeFromParent bool, err error)
		Done() <-chan struct{}
	}
	CancelFunc   func(err error)
	namedContext struct {
		context.Context

		mu       sync.Mutex
		done     chan struct{}
		children map[canceler]struct{}
		err      error
		name     string
		key, val interface{}
		start    time.Time
	}
)

// nolint: golint
func Context(parent context.Context, name string) *namedContext {
	return &namedContext{Context: parent, name: name, start: time.Now()}
}

// nolint: golint
func WithValue(parent context.Context, name string, key, val interface{}) *namedContext {
	if key == nil {
		panic("nil key")
	}
	if !reflect.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}

	ctx := Context(parent, name)
	ctx.key, ctx.val = key, val
	return ctx
}

// nolint: golint
func WithCancel(parent context.Context, name string) (ctx *namedContext, cancel CancelFunc) {
	c := Context(parent, name)
	propagateCancel(parent, c)
	return c, func(err error) {
		if c.err == nil {
			incCancels(c.name)
		}
		c.cancel(true, errors.Wrapf(err, namedErrFormat, name))
	}
}

func (c *namedContext) Value(key interface{}) interface{} {
	if key == &cancelCtxKey {
		return c
	}
	if key == c.key {
		return c.val
	}

	return c.Context.Value(key)
}

func (c *namedContext) Done() <-chan struct{} {
	c.mu.Lock()
	if c.done == nil {
		c.done = make(chan struct{})
	}
	d := c.done
	c.mu.Unlock()
	return d
}

func (c *namedContext) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}
func (c *namedContext) String() string {
	return c.name
}

type timerCtx struct {
	namedContext
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}

func WithDeadline(parent context.Context, d time.Time, name string) (context.Context, CancelFunc) {
	if cur, ok := parent.Deadline(); ok && cur.Before(d) {
		// The current deadline is already sooner than the new one.
		return WithCancel(parent, name)
	}

	c := &timerCtx{
		namedContext: *Context(parent, name),
		deadline:     d,
	}
	propagateCancel(parent, c)
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(true, fmt.Errorf(namedDeadlineExceededFormat, name, 0))
		return c, func(err error) { c.cancel(true, errors.Wrapf(err, namedErrFormat, name)) }
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.timer = time.AfterFunc(dur, func() {
			if c.err == nil {
				incTimeouts(c.name)
			}
			c.cancel(true, fmt.Errorf(namedDeadlineExceededFormat, name, time.Since(c.start).Milliseconds()))
		})
	}
	return c, func(err error) { c.cancel(true, errors.Wrapf(err, namedErrFormat, name)) }
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}

func WithTimeout(parent context.Context, timeout time.Duration, name string) (context.Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout), name)
}

func propagateCancel(parent context.Context, child canceler) {
	done := parent.Done()
	if done == nil {
		return // parent is never canceled
	}

	select {
	case <-done:
		// parent is already canceled
		child.cancel(false, parent.Err())
		return
	default:
	}

	if p, ok := parentNamedCtx(parent); ok {
		p.mu.Lock()
		if p.err != nil {
			// parent has already been canceled
			child.cancel(false, p.err)
		} else {
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
		incGorutines()
		go func() {
			select {
			case <-parent.Done():
				child.cancel(false, errors.Wrapf(parent.Err(), namedParentErrFormat, child.String()))
			case <-child.Done():
			}
			decGorutines()
		}()
	}
}
func parentNamedCtx(parent context.Context) (*namedContext, bool) {
	done := parent.Done()
	if done == closedchan || done == nil {
		return nil, false
	}
	p, ok := parent.Value(&cancelCtxKey).(*namedContext)
	if !ok {
		return nil, false
	}
	p.mu.Lock()
	ok = p.done == done
	p.mu.Unlock()
	if !ok {
		return nil, false
	}
	return p, true
}

func (c *namedContext) cancel(removeFromParent bool, err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
	if c.done == nil {
		c.done = closedchan
	} else {
		close(c.done)
	}
	for child := range c.children {
		// NOTE: acquiring the child's lock while holding parent's lock.
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock()

	if removeFromParent {
		removeChild(c.Context, c)
	}
}
func removeChild(parent context.Context, child canceler) {
	p, ok := parentNamedCtx(parent)
	if !ok {
		return
	}
	p.mu.Lock()
	if p.children != nil {
		delete(p.children, child)
	}
	p.mu.Unlock()
}

var patchCtxName = []byte("PatchContext")

func ctxName(mainPkg string) string {
	name := "undefined"
	trace := make([]byte, 2048)
	runtime.Stack(trace, false)
	start := bytes.Index(trace, patchCtxName) + 12 // 12 - len(patchCtxName)
	start += bytes.IndexRune(trace[start:], '\n') + 1
	start += bytes.IndexRune(trace[start:], '\n') + 1
	if shift := bytes.Index(trace[start:], []byte("vendor")); shift > 0 {
		start += shift
	}
	start += bytes.Index(trace[start:], []byte(mainPkg))
	start += len(mainPkg)
	trace = trace[start:]
	end := bytes.IndexRune(trace, '\n')
	if end > 0 {
		trace = trace[:end]
	}
	end = bytes.LastIndexByte(trace, '(')
	if end > 0 {
		name = string(trace[:end])
	}

	return name
}

type unpatcher struct{}

func (u unpatcher) Unpatch() {
	monkey.UnpatchAll()
}

func PatchContext(mainPkg string) interface{ Unpatch() } {
	monkey.Patch(context.WithCancel, func(parent context.Context) (ctx context.Context, cancel context.CancelFunc) {
		ctx, cnlFN := WithCancel(parent, ctxName(mainPkg))
		return ctx, func() {
			cnlFN(context.Canceled)
		}
	})
	monkey.Patch(context.WithDeadline, func(parent context.Context, d time.Time) (context.Context, context.CancelFunc) {
		ctx, cnlFN := WithDeadline(parent, d, ctxName(mainPkg))
		return ctx, func() {
			cnlFN(context.Canceled)
		}
	})
	monkey.Patch(context.WithTimeout, func(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
		ctx, cnlFN := WithTimeout(parent, timeout, ctxName(mainPkg))
		return ctx, func() {
			cnlFN(context.Canceled)
		}
	})
	monkey.Patch(context.WithValue, func(parent context.Context, key, val interface{}) context.Context {
		ctx := WithValue(parent, ctxName(mainPkg), key, val)
		return ctx
	})

	return unpatcher{}
}
