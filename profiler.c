#include <stdatomic.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include <pthread.h>

#include "profiler.h"

#define NSAMPLES 1024
#define MAX_STACK_SIZE 64

#define PROFILER_STOPPED 0
#define PROFILER_STARTED 1

// sampling_rate is the portion of allocations to sample.
atomic_int sampling_rate;

// sample is a single allocation stack trace
struct sample {
	// count is the number of allocations in the stack
	size_t count;
	// stack is the sequence of program counters in the call stack
	void *stack[MAX_STACK_SIZE];
	// size is the number of bytes requested by the call to malloc/calloc/realloc
	size_t size;
	// ready indticates that there is a stack to read, set to 0 after the
	// stack is read
	int ready;
};

// sample_buffer is a fixed-size circular buffer of allocation
// stack traces
struct sample_buffer {
	// state indicates whether the profiler is started or stopped
	int state;
	// mu guards all the fields in this struct
	pthread_mutex_t mu;
	// cond signals that a new sample has been recorded
	pthread_cond_t cond;
	// samples is a fixed size buffer of all read samples
	struct sample samples[NSAMPLES];

	// writer and reader are cursors that track where the next sample
	// should be inserted or read
	size_t writer;
	size_t reader;
};

struct sample_buffer global_buffer;
static void sample_buffer_insert(struct sample_buffer *b, void **stack, int n, size_t size);

// calls tracks how many times malloc has been called
atomic_size_t calls;

extern int goCallers(uintptr_t *pcs, int max);

static inline int go_backtrace(void **stack, int max) {
	uintptr_t *stack_head = (void *) &stack[0];
	return goCallers(stack_head, max);
}

extern __thread atomic_int in_cgo_start;

__thread uint64_t rng_state;

static uint64_t rng_state_advance(uint64_t seed) {
	while (seed == 0) {
		uint64_t lo = rand();
		uint64_t hi = rand();
		seed = (hi << 32) | lo;
	}
	uint64_t x = seed;
	x ^= x << 13;
	x ^= x >> 17;
	x ^= x << 5;
	return x;
}

static int should_sample(size_t rate, size_t size) {
	if (rate == 1) {
		return 1;
	}
	if (size > rate) {
		return 1;
	}
	rng_state = rng_state_advance(rng_state);
	uint64_t check = rng_state % rate;
	return check <= size;
}

void profile_allocation(size_t size) {
	// When starting a thread in CGo mode, malloc is called. Long story short,
	// calling back into Go in that situtation crashes the program. So don't
	// profile in that case.
	if (atomic_load(&in_cgo_start) == 1) {
		return;
	}
	// TODO: more sophisticated sampling?
	size_t rate = atomic_load_explicit(&sampling_rate, memory_order_relaxed);
	if (rate == 0) {
		return;
	}
	if (should_sample(rate, size) != 0) {
		void *stack[64];
		int n = go_backtrace(stack, 64);
		// TODO: read the backtrace directly into the buffer, eliminate
		// one extra copy?
		// TODO: skip this function in the stack trace?
		sample_buffer_insert(&global_buffer, stack, n, size);
	}
}

static void sample_buffer_insert(struct sample_buffer *b, void **stack, int n, size_t size) {
	pthread_mutex_lock(&b->mu);
	if (b->state == PROFILER_STOPPED) {
		pthread_mutex_unlock(&b->mu);
		return;
	}
	if (n > MAX_STACK_SIZE) {
		n = MAX_STACK_SIZE;
	}
	memcpy(b->samples[b->writer].stack, stack, n*sizeof(void *));
	b->samples[b->writer].count = n;
	b->samples[b->writer].size = size;
	b->samples[b->writer].ready = 1;
	b->writer = (b->writer + 1) % NSAMPLES;
	pthread_cond_signal(&b->cond);
	pthread_mutex_unlock(&b->mu);
}

void cgo_heap_profiler_start() {
	struct sample_buffer *b = &global_buffer;
	pthread_mutex_lock(&b->mu);
	b->state = PROFILER_STARTED;
	pthread_mutex_unlock(&b->mu);
}

void cgo_heap_profiler_stop() {
	struct sample_buffer *b = &global_buffer;
	pthread_mutex_lock(&b->mu);
	b->state = PROFILER_STOPPED;
	pthread_cond_signal(&b->cond);
	atomic_store(&sampling_rate, 0);
	pthread_mutex_unlock(&b->mu);
}

int cgo_heap_profiler_set_sampling_rate(int hz) {
	int rate = atomic_load(&sampling_rate);
	if (hz > 0) {
		atomic_store(&sampling_rate, hz);
	}
	return rate;
}

int cgo_heap_profiler_get_sample(uintptr_t *stack, int max, size_t *size) {
	struct sample_buffer *b = &global_buffer;
	pthread_mutex_lock(&b->mu);
	while (b->samples[b->reader].ready != 1) {
		if (b->state == PROFILER_STOPPED) {
			pthread_mutex_unlock(&b->mu);
			return 0;
		}
		pthread_cond_wait(&b->cond, &b->mu);
	}
	int n = b->samples[b->reader].count;
	if (n > max) {
		n = max;
	}
	memcpy(stack, b->samples[b->reader].stack, n*sizeof(void *));
	*size = b->samples[b->reader].size;
	b->samples[b->reader].ready = 0;
	b->reader = (b->reader + 1) % NSAMPLES;
	pthread_mutex_unlock(&b->mu);
	return n;
}
