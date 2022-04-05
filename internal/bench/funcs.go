package bench

/*
#cgo CFLAGS: -g -O0
#include <stdlib.h>

int *side_effect;

void do_malloc(int size) {
	int *y = malloc(sizeof(int) * size);
	side_effect = y;
	free(y);
}

void do_calloc(int size) {
	int *y = calloc(sizeof(int), size);
	side_effect = y;
	free(y);
}
*/
import "C"

func doMalloc(size int) {
	C.do_malloc(C.int(size))
}

func doCalloc(size int) {
	C.do_calloc(C.int(size))
}
