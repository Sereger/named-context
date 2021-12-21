### Description
Context is a chain and when the one of this chain is canceled then all subcontexts are canceled as well.
At the tail of this chain we receive an error like "context canceled", but we don't know which context of entire chain was canceled. 
Use this package to find out which of context was canceled.

### API:

```go
import (
    namedContext "github.com/Sereger/named-context"
)
...
func (*OrderRepos) Orders(ctx context.Context) {
    // or WithDeadline, WithCancel, WithValue, Context
    ctx := namedContext.WithTimeout(ctx, time.Second, "order_storage.getOrders")
    ...
}
```

### Metrics:
Package write the next metrics:

`timeouts` - vector by names with counting of timeouts

`cancels` - vector by names with context cancellation counter

`gorutines` - gorutines creation counter (for sync default and named contexts)

`gorutines_current` - gauge of current gorutines



if you use Prometheus for metrics, can use the next examples:
```go
import (
    namedContext "github.com/Sereger/named-context"
    "github.com/prometheus/client_golang/prometheus"
)

func main() {
    m := namedContext.NewPrometheusMetrics("my_app")
    namedContext.InitMetrics(m)

    prometheus.NewRegistry().MustRegister(m.Collectors()...)
}
```

If you have own metrics system, you can use your own collector by implementing the interface of metrics collector:
```go
type metrics interface {
    IncGorutinesAll()
    IncGorutinesCurrent()
    DecGorutinesCurrent()
    IncTimeouts(label string)
    IncCancels(label string)
}
```

### PathContext
Patch default context for best performance. Because, default context [create goroutine](https://github.com/golang/go/blob/master/src/context/context.go#L277) for sync with extend context.
```go
import (
    namedContext "github.com/Sereger/named-context"
)
...

defer namedContext.PatchContext(mainPkg string).Unpatch()
...
```

It's change next function calls:
```
context.WithCancel -> WithCancel
context.WithDeadline -> WithDeadline
context.WithTimeout -> WithTimeout
context.WithValue -> WithValue
```

`mainPkg` - set main package of your project for auto generation name.
For example `github.com/Sereger/` and the err after canceling context will be next:
```text
named context [named-context.Test] deadline exceeded after [10ms]
```