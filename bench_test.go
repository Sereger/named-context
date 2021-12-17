package context

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func Benchmark(b *testing.B) {
	b.Run("usual context", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx1, cnlFN := context.WithCancel(context.Background())
				ctx2, _ := context.WithCancel(ctx1)
				cnlFN()
				<-ctx2.Done()
			}
		})
	})

	b.Run("named context", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx1, cnlFN := WithCancel(context.Background(), "one")
				ctx2, _ := WithCancel(ctx1, "two")
				cnlFN(fmt.Errorf("err"))
				<-ctx2.Done()
			}
		})
	})

	b.Run("patched context", func(b *testing.B) {
		defer PatchContext("github.com/Sereger/").Unpatch()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx1, cnlFN := context.WithCancel(context.Background())
				ctx2, _ := context.WithCancel(ctx1)
				cnlFN()
				<-ctx2.Done()
			}
		})
	})
	b.Run("patched named context", func(b *testing.B) {
		defer PatchContext("github.com/Sereger/").Unpatch()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx1, cnlFN := WithCancel(context.Background(), "test")
				ctx2, _ := context.WithCancel(ctx1)
				cnlFN(fmt.Errorf("my err"))
				<-ctx2.Done()
			}
		})
	})
	b.Run("cancel+timeout context", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx1, cnlFN := context.WithCancel(context.Background())
				ctx2, _ := context.WithTimeout(ctx1, time.Microsecond)
				cnlFN()
				<-ctx2.Done()
			}
		})
	})
	b.Run("patched cancel+timeout context", func(b *testing.B) {
		defer PatchContext("github.com/Sereger/").Unpatch()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx1, cnlFN := WithCancel(context.Background(), "test")
				ctx2, _ := context.WithTimeout(ctx1, time.Microsecond)
				cnlFN(fmt.Errorf("my err"))
				<-ctx2.Done()
			}
		})
	})
	b.Run("cancel+value context", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx1, cnlFN := context.WithCancel(context.Background())
				ctx2 := context.WithValue(ctx1, "xxx", "yyy")
				ctx3, _ := context.WithTimeout(ctx2, time.Microsecond)
				cnlFN()
				<-ctx3.Done()
			}
		})
	})
	b.Run("patched cancel+value context", func(b *testing.B) {
		defer PatchContext("github.com/Sereger/").Unpatch()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx1, cnlFN := context.WithCancel(Context(context.Background(), "test"))
				ctx2 := context.WithValue(ctx1, "xxx", "yyy")
				ctx3, _ := context.WithTimeout(ctx2, time.Microsecond)
				cnlFN()
				<-ctx3.Done()
			}
		})
	})
	b.Run("creating context", func(b *testing.B) {
		ctx1, _ := context.WithCancel(Context(context.Background(), "test"))
		for i := 0; i < b.N; i++ {
			_, _ = context.WithCancel(ctx1)
		}
	})
	b.Run("creating context patched", func(b *testing.B) {
		defer PatchContext("github.com/Sereger/").Unpatch()
		ctx1, _ := context.WithCancel(Context(context.Background(), "test"))
		for i := 0; i < b.N; i++ {
			ctx1, _ = context.WithCancel(ctx1)
		}
	})
}
