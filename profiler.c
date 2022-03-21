#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdatomic.h>
#include <stddef.h>
#include <string.h>

#include <pthread.h>
#define UNW_LOCAL_ONLY
#include <libunwind.h>

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

// This is a really basic implementation of unw_backtrace. libunwind actually
// has faster arch-specific implementations of this function, but not every
// libunwind has the unw_backtrace function.
static inline int get_backtrace(void **stack, int max) {
	unw_cursor_t cursor;
	unw_context_t uc;
	unw_getcontext(&uc);
	unw_init_local(&cursor, &uc);
	int n = 0;
	while ((unw_step(&cursor) > 0) && (n < max)) {
		unw_word_t ip;
		unw_get_reg(&cursor, UNW_REG_IP, &ip);
		stack[n] = (void *)ip;
		n++;
	}
	return n;
}

void profile_allocation(size_t size) {
	// TODO: more sophisticated sampling?
	size_t rate = atomic_load_explicit(&sampling_rate, memory_order_relaxed);
	if (rate == 0) {
		return;
	}
	size_t old = atomic_fetch_add_explicit(&calls, 1, memory_order_relaxed);
	if (old % rate == 0) {
		void *stack[64];
		//int n = unw_backtrace(stack, 64);
		int n = get_backtrace(stack, 64);
		// TODO: read the backtrace directly into the buffer, eliminate
		// one extra copy?
		// TODO: skip this function in the stack trace?
		// TODO: Cross the cgo boundary?
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

int cgo_heap_profiler_get_sample(void **stack, int max, size_t *size) {
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
