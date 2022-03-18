#include <dlfcn.h>
#include <stddef.h>
#include <stdint.h>

#include <pthread.h>

#include "profiler_internal.h"

void *(*real_malloc)(size_t);
void *(*real_calloc)(size_t, size_t);
void *(*real_realloc)(void *, size_t);

pthread_once_t alloc_funcs_init_once;
static void alloc_funcs_init(void) {
	void *f = NULL;
	f = dlsym(RTLD_NEXT, "malloc");
	if (f != NULL) {
		real_malloc = f;
	}
	f = dlsym(RTLD_NEXT, "calloc");
	if (f != NULL) {
		real_calloc = f;
	}
	f = dlsym(RTLD_NEXT, "realloc");
	if (f != NULL) {
		real_realloc = f;
	}
}


void *malloc(size_t size) {
	pthread_once(&alloc_funcs_init_once, alloc_funcs_init);
	profile_allocation(size);
	return real_malloc(size);
}

void *calloc(size_t nmemb, size_t size) {
	pthread_once(&alloc_funcs_init_once, alloc_funcs_init);
	// If the allocation size would overflow, don't bother profiling, and
	// let the real calloc implementation (possibly) fail.
	if ((size > 0) && (nmemb > (SIZE_MAX/size))) {
		return real_calloc(nmemb, size);
	}
	profile_allocation(size * nmemb);
	return real_calloc(nmemb, size);
}

void *realloc(void *p, size_t size) {
	pthread_once(&alloc_funcs_init_once, alloc_funcs_init);
	profile_allocation(size);
	return real_realloc(p, size);
}
