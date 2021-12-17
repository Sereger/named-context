package context

import (
	"context"
	"fmt"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// nolint
func Test(t *testing.T) {
	type testcase struct {
		name      string
		wantErr   string
		fn        func() context.Context
		gorutites float64
		timeouts  map[string]float64
		kv        map[interface{}]interface{}
	}
	tests := []testcase{
		{
			name: "Обычный контекст",
			fn: func() context.Context {
				ctx, cnlFN := context.WithCancel(context.Background())
				defer cnlFN()

				return ctx
			},
			wantErr: "context canceled",
		},
		{
			name: "Обычный контекст, цепочка вызовов",
			fn: func() context.Context {
				ctx1, cnlFN := context.WithCancel(context.Background())
				ctx2 := context.WithValue(ctx1, "xxx", "yyy")
				ctx3, _ := context.WithCancel(ctx2)
				defer cnlFN()

				return ctx3
			},
			wantErr: "context canceled",
		},
		{
			name: "Именованный контекст, простое использование",
			fn: func() context.Context {
				ctx, cnlFN := WithCancel(context.Background(), "test 1")
				defer cnlFN(fmt.Errorf("my error"))

				return ctx
			},
			wantErr: "named context [test 1]: my error",
		},
		{
			name: "Именованный контекст, цепочка вызовов, ожидаем увидеть какой конкретно контекст зафейлился",
			fn: func() context.Context {
				ctx1, cnlFN := WithCancel(context.Background(), "el 1")
				ctx2 := context.WithValue(ctx1, "xxx", "yyy")
				ctx3, _ := WithCancel(ctx2, "el 2")
				defer cnlFN(fmt.Errorf("my error"))

				return ctx3
			},
			wantErr:   "named context [el 1]: my error",
			gorutites: 0,
		},
		{
			name: "Комбинированный вариант контекст, цепочка вызовов, ожидаем, что родительский контекст зафейлилися",
			fn: func() context.Context {
				ctx1, cnlFN := context.WithCancel(context.Background())
				ctx2 := context.WithValue(ctx1, "xxx", "yyy")
				ctx3, _ := WithCancel(ctx2, "named")
				defer cnlFN()

				return ctx3
			},
			wantErr:   "named context [named] parent err: context canceled",
			gorutites: 1,
		},
		{
			name: "Комбинированный вариант контекст, цепочка вызовов, context.WithCancel должен отмениться при отмене namedContext",
			fn: func() context.Context {
				ctx1, cnlFN := context.WithCancel(context.Background())
				ctx2, _ := WithCancel(ctx1, "named")
				ctx3, _ := context.WithCancel(ctx2)

				defer cnlFN()

				return ctx3
			},
			wantErr:   "named context [named] parent err: context canceled",
			gorutites: 1,
		},
		{
			name: "Патчим context.WithCancel, чтобы всегда использовать именованные контексты",
			fn: func() context.Context {
				defer PatchContext("github.com/Sereger/").Unpatch()

				ctx1, _ := WithCancel(context.Background(), "start context")
				g, gCtx := errgroup.WithContext(ctx1)
				g.Go(func() error {
					return fmt.Errorf("group err")
				})
				ctx2, _ := WithCancel(gCtx, "named")
				ctx3, _ := context.WithCancel(ctx2)
				g.Wait()

				return ctx3
			},
			wantErr:   "named context [named-context.Test.func7]: context canceled",
			gorutites: 0,
		},
		{
			name: "Именованный контекст с таймаутом",
			fn: func() context.Context {
				ctx1, _ := WithTimeout(context.Background(), time.Second, "el 1")
				ctx2 := context.WithValue(ctx1, "xxx", "yyy")
				ctx3, _ := WithCancel(ctx2, "el 2")

				return ctx3
			},
			wantErr:   "named context [el 1] deadline exceeded after [100",
			gorutites: 0,
		},
		{
			name: "Патчим context.WithTimeout, чтобы всегда использовать именованные контексты",
			fn: func() context.Context {
				defer PatchContext("github.com/Sereger/").Unpatch()
				ctx1, _ := WithCancel(context.Background(), "start context")
				ctx2, _ := context.WithTimeout(ctx1, time.Second)
				ctx3, _ := context.WithCancel(ctx2)

				return ctx3
			},
			wantErr:   "named context [named-context.Test.func9] deadline exceeded",
			gorutites: 0,
		},
		{
			name: "Проверка именования",
			fn: func() context.Context {
				defer PatchContext("github.com/Sereger/").Unpatch()
				ctx1, cnlFn := context.WithCancel(context.Background())
				cnlFn()

				return ctx1
			},
			wantErr:   "named context [named-context.Test.func10]: context canceled",
			gorutites: 0,
		},
		{
			name: "Отмена контекста по таймауту, проверка метрик",
			fn: func() context.Context {
				defer PatchContext("github.com/Sereger/").Unpatch()
				ctx1, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)

				return ctx1
			},
			wantErr:  "named context [named-context.Test.func11] deadline exceeded after [1",
			timeouts: map[string]float64{"named-context.Test.func11": 1},
		},
		{
			name: "Отмена контекста не по таймауту, проверка метрик",
			fn: func() context.Context {
				defer PatchContext("github.com/Sereger/").Unpatch()
				ctx1, cnlFn := context.WithCancel(context.Background())
				ctx2, _ := context.WithTimeout(ctx1, 10*time.Millisecond)
				cnlFn()

				return ctx2
			},
			wantErr:   "named context [named-context.Test.func12]: context canceled",
			gorutites: 0,
			timeouts:  map[string]float64{"named-context.Test.func12": 0},
		},
		{
			name: "Проверка получение значения из контекста",
			fn: func() context.Context {
				defer PatchContext("github.com/Sereger/").Unpatch()
				ctx1 := context.WithValue(context.Background(), "xxx", "yyy")
				ctx2, _ := context.WithTimeout(ctx1, 1*time.Millisecond)
				return ctx2
			},
			wantErr: "named context [named-context.Test.func13] deadline exceeded after",
			kv:      map[interface{}]interface{}{"xxx": "yyy"},
		},
		{
			name: "Контекст создается внутри функции структуры",
			fn: func() context.Context {
				defer PatchContext("github.com/Sereger/").Unpatch()
				testStruct := &myTest{}
				ctx, cnlFn := testStruct.Handle(context.Background())
				cnlFn()

				return ctx
			},
			wantErr: "named context [named-context.(*myTest).Handle]: context canceled",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := NewPrometheusMetrics("test")
			InitMetrics(m)
			ctx := test.fn()
			<-ctx.Done()
			dtoM := &dto.Metric{}
			m.gorutinesAll.Write(dtoM)
			require.Equal(t, test.gorutites, *dtoM.Counter.Value)
			if len(test.timeouts) > 0 {
				for label, val := range test.timeouts {
					c, _ := m.timeouts.GetMetricWith(map[string]string{"context": label})
					dtoM := &dto.Metric{}
					c.Write(dtoM)
					require.Equal(t, val, *dtoM.Counter.Value)
				}
			}
			if len(test.kv) > 0 {
				for key, val := range test.kv {
					v := ctx.Value(key)
					require.Equal(t, val, v)
				}
			}

			err := ctx.Err()
			if test.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

type myTest struct{}

func (mt *myTest) Handle(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
