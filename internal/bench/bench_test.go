package bench

import (
	"io"
	"testing"

	"github.com/nsrip-dd/cmemprof"
)

func BenchmarkMemoryProfile(b *testing.B) {
	b.Run("no profiling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doMalloc(1)
		}
	})

	b.Run("with profiling", func(b *testing.B) {
		profiler := cmemprof.Profile{}
		profiler.Start(io.Discard)
		defer profiler.Stop()

		for i := 0; i < b.N; i++ {
			doMalloc(1)
		}
	})

	b.Run("big no profiling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doMalloc(1000000)
		}
	})

	b.Run("big with profiling", func(b *testing.B) {
		profiler := cmemprof.Profile{}
		profiler.Start(io.Discard)
		defer profiler.Stop()

		for i := 0; i < b.N; i++ {
			doMalloc(1000000)
		}
	})

	b.Run("calloc no profiling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doCalloc(1)
		}
	})

	b.Run("calloc with profiling", func(b *testing.B) {
		profiler := cmemprof.Profile{}
		profiler.Start(io.Discard)
		defer profiler.Stop()

		for i := 0; i < b.N; i++ {
			doCalloc(1)
		}
	})

	b.Run("calloc big no profiling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doCalloc(1000000)
		}
	})

	b.Run("calloc big with profiling", func(b *testing.B) {
		profiler := cmemprof.Profile{}
		profiler.Start(io.Discard)
		defer profiler.Stop()

		for i := 0; i < b.N; i++ {
			doCalloc(1000000)
		}
	})
}
