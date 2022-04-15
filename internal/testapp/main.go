package main

import (
	"math/rand"
	"os"
	"sync"

	"github.com/nsrip-dd/cmemprof"
	_ "github.com/nsrip-dd/cmemprof/internal/cgosymbolizer"
	"github.com/pkg/profile"
)

/*
#cgo CFLAGS: -g -O0
#include <stdlib.h>

int *side_effect;

int baz(int x) {
	int *y = malloc(sizeof(int));
	side_effect = y;
	*y = ++x;
	x = *y;
	free(y);
	return x;
}

int bonk(int x) { return baz(x); }
int cronk(int x) { return bonk(x); }

int chonk(int x) {
	int *y = calloc(sizeof(int), 1);
	side_effect = y;
	*y = ++x;
	x = *y;
	free(y);
	return x;
}
*/
import "C"

func main() {
	defer profile.Start(profile.CPUProfile, profile.ProfilePath("cpu.pprof")).Stop()
	f, err := os.Create("test.pprof")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	profiler := &cmemprof.Profile{
		SamplingRate: 1,
	}
	profiler.Start(f)
	var x int
	for i := 0; i < 100000; i++ {
		switch uint(rand.Int()) % 4 {
		case 0:
			x = int(C.baz(C.int(x)))
		case 1:
			x = int(C.bonk(C.int(x)))
		case 2:
			x = int(C.cronk(C.int(x)))
		case 3:
			x = int(C.chonk(C.int(x)))
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 250000; j++ {
				C.baz(42)
			}
		}()
	}
	wg.Wait()
	err = profiler.Stop()
	if err != nil {
		panic(err)
	}
}
