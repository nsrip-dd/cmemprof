// cmemprof profiles C memory allocations (malloc, calloc, and realloc)
//
// Importing this package in a program will replace malloc, calloc, and realloc
// with wrappers which will sample allocations and record them to a profile.
//
// To use this package:
//
//	f, _ := os.Create("cmem.pprof")
//	profiler := cmemprof.Profiler{SampleRate: 500}
//	profiler.Start(f)
//	defer profiler.Stop()
package cmemprof

/*
#cgo CFLAGS: -g -fno-omit-frame-pointer
#cgo linux LDFLAGS: -pthread -Wl,--wrap=calloc -Wl,--wrap=malloc -Wl,--wrap=realloc
#cgo darwin LDFLAGS: -ldl -pthread
#include <stdint.h> // for uintptr_t

#include "profiler.h"
*/
import "C"

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

const DefaultSamplingRate = 1024 * 1024 // 1 MB

// Profile provides access to a C memory allocation profiler based on
// instrumenting malloc, calloc, and realloc.
type Profile struct {
	done    chan error
	state   int64
	mu      sync.Mutex
	samples map[uintptr][]*sample

	// SamplingRate is the value, in bytes, such that an average of one
	// sample will be recorded for every SamplingRate bytes allocated.  An
	// allocation of N bytes will be recorded with probability min(1, N /
	// SamplingRate).
	SamplingRate int
}

func stackHash(p []C.uintptr_t) uintptr {
	var h uintptr
	// hash copied from runtime/mprof.go
	for _, pc := range p {
		h += uintptr(pc)
		h += h << 10
		h ^= h >> 6
	}
	// finalize
	h += h << 3
	h ^= h >> 11
	return h
}

type sample struct {
	stack []C.uintptr_t
	count int
	size  uint
}

//func cmpStacks(p []C.uintptr_t, q []uintptr) bool {
func cmpStacks(p, q []C.uintptr_t) bool {
	if len(p) != len(q) {
		return false
	}
	for i := range p {
		if q[i] != p[i] {
			return false
		}
	}
	return true
}

func (c *Profile) insert(p []C.uintptr_t, size uint) {
	h := stackHash(p)
	bucket := c.samples[h]
	rate := c.SamplingRate
	if rate == 0 {
		rate = DefaultSamplingRate
	}
	for _, sample := range bucket {
		if cmpStacks(p, sample.stack) {
			// Adjust recorded samples according to their likelihood
			// of being observed so that our profile more accurately
			// represents the true amount of allocation
			if size < uint(rate) {
				// The allocation was sample with probability p
				// = size / rate.  So we assume there were
				// actually (1 / p) similar allocations for a
				// total size of (1 / p) * size = rate
				sample.count += int(float64(rate) / float64(size))
				sample.size += uint(rate)
			} else {
				sample.count += 1
				sample.size += size
			}
			return
		}
	}
	// need to copy the slice in case it's re-used
	dup := append([]C.uintptr_t{}, p...)
	c.samples[h] = append(c.samples[h], &sample{stack: dup, count: 1, size: size})
}

// Start begins profiling C memory allocations. The pprof-encoded profile will be
// writen to w when profiling is stopped.
func (c *Profile) Start(w io.Writer) {
	if !atomic.CompareAndSwapInt64(&c.state, 0, 1) {
		return
	}
	if c.done == nil {
		c.done = make(chan error)
	}
	if c.samples == nil {
		c.samples = make(map[uintptr][]*sample)
	}
	go c.profile(w)
}

func (c *Profile) profile(w io.Writer) {
	rate := c.SamplingRate
	if rate == 0 {
		rate = DefaultSamplingRate
	}
	C.cgo_heap_profiler_set_sampling_rate(C.int(rate))
	C.cgo_heap_profiler_start()
	var s C.size_t
	stack := make([]C.uintptr_t, 64)
	for atomic.LoadInt64(&c.state) != 0 {
		n := C.cgo_heap_profiler_get_sample(&stack[0], 64, &s)
		if n == 0 {
			break
		}
		c.mu.Lock()
		c.insert(stack[:n], uint(s))
		c.mu.Unlock()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	p := buildProfile(c.samples)
	err := p.CheckValid()
	if err != nil {
		err = fmt.Errorf("bad profile: %s", err)
		c.done <- err
		return
	}
	err = p.Write(w)
	if err != nil {
		err = fmt.Errorf("writing profile: %s", err)
	}
	c.done <- err
}

// Stop cancels memory profiling and waits for the profile to be written to the
// io.Writer passed to Start. Returns any error from writing the profile.
func (c *Profile) Stop() error {
	if !atomic.CompareAndSwapInt64(&c.state, 1, 0) {
		return fmt.Errorf("profiling isn't started")
	}
	C.cgo_heap_profiler_stop()
	return <-c.done
}
