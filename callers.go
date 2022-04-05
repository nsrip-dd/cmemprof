package cmemprof

import (
	"runtime"
	"unsafe"
)

import "C"

//export goCallers
func goCallers(pcs *uintptr, max int) int {
	pc := unsafe.Slice(pcs, max)
	return runtime.Callers(0, pc)
}
