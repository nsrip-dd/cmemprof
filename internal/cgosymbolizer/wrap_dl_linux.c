#define _GNU_SOURCE
#include <dlfcn.h>
#include <stddef.h>

struct dl_phdr_info;
typedef int (*dl_iterate_phdr_cbtype)(struct dl_phdr_info *, size_t, void *);
typedef int (*dl_iterate_phdr_functype)(dl_iterate_phdr_cbtype, void*);
typedef int (*_Unwind_Backtrace_functype)(void *trace, void * trace_argument);

static dl_iterate_phdr_functype real_dl_iterate_phdr;
static _Unwind_Backtrace_functype real__Unwind_Backtrace;

static __attribute__((constructor)) void init(void) {
	real_dl_iterate_phdr = dlsym(RTLD_NEXT, "dl_iterate_phdr");
	real__Unwind_Backtrace = dlsym(RTLD_NEXT, "_Unwind_Backtrace");
}

__thread int in_dl_iterate_phdr_call;

int dl_iterate_phdr(dl_iterate_phdr_cbtype callback, void *data) {
	in_dl_iterate_phdr_call++;
	int r = real_dl_iterate_phdr(callback, data);
	in_dl_iterate_phdr_call--;
	return r;
}

int _Unwind_Backtrace(void *trace, void *data) {
	if (in_dl_iterate_phdr_call) {
		return 4; // unwind reason code _URC_NORMAL_STOP
	}
	return real__Unwind_Backtrace(trace, data);
}
