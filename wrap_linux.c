#include <stdatomic.h>
#include <stdint.h>

#include "profiler_internal.h"

/* The GNU linker supports a --wrap flag that lets wrap allocation calls
 * without the problem of using dlsym in functions called by dlsym */

void *__real_malloc(size_t size);
void *__wrap_malloc(size_t size) {
	profile_allocation(size);
	return __real_malloc(size);
}

void *__real_calloc(size_t nmemb, size_t size);
void *__wrap_calloc(size_t nmemb, size_t size) {
	// If the allocation size would overflow, don't bother profiling, and
	// let the real calloc implementation (possibly) fail.
	if ((size > 0) && (nmemb > (SIZE_MAX/size))) {
		return __real_calloc(nmemb, size);
	}
	profile_allocation(size * nmemb);
	return __real_calloc(nmemb, size);
}

void *__real_realloc(void *p, size_t size);
void *__wrap_realloc(void *p, size_t size) {
	profile_allocation(size);
	return __real_realloc(p, size);
}

extern __thread atomic_int in_cgo_start;

__attribute__((weak)) extern void __real_x_cgo_thread_start(void *p);
void __wrap_x_cgo_thread_start(void *p) {
        atomic_store(&in_cgo_start, 1);
        __real_x_cgo_thread_start(p);
        atomic_store(&in_cgo_start, 0);
}