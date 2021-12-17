### Description
to be write...

### API:

```go
import (
    namedContext "github.com/Sereger/named-context"
)
...
func (*OrderRepos) Orders(ctx context.Context) {
    // or NamedWithDeadline, NamedWithCancel, NamedWithValue, Context
    ctx := namedContext.NamedWithTimeout(ctx, time.Second, "order_storage.getOrders")
    ...
}
```

### Metrics:
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
Patch default context for best performance. Because, default context create goroutine for sync with extend context.
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
named context [named-context.Test.func11] deadline exceeded after [10ms]
```