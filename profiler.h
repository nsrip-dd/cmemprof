#ifndef PROFILER_H
#define PROFILER_H

#include <stddef.h>
#include <stdint.h>

void cgo_heap_profiler_start();
void cgo_heap_profiler_stop();

// cgo_heap_profiler_set_sampling_rate configures profiling to capture 1/hz of
// allocations, and returns the previous rate. If hz <= 0, then the rate is
// unchanged and the current rate is returned.
int cgo_heap_profiler_set_sampling_rate(int hz);

// cgo_heap_profiler_get_sample copies the most recently-read stack trace into
// stack, up to max entries, and stores the size of the allocation in the
// provided pointer. Blocks until a sample is available or profiling is
// canceled.  Returns the size of the call stack, where size 0 means profiling
// was canceled.
int cgo_heap_profiler_get_sample(uintptr_t *stack, int max, size_t *size);

#endif
