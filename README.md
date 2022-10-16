## C memory allocation profiler

This repository contains an experimental C allocation profiler for Go.

**WARNING**: This is very much a work in progress. It still needs plenty of work
to be production-ready. Use at your own peril!

It's been lightly tested on Linux (Ubuntu 20.04) and macos (Big Sur).

### Building

On macos, you don't need to do anything special.

On Linux, you'll need a linker that supports the `--wrap` argument (such as the GNU
linker, or mold) which is required to provide replacements for the allocation
functions since `dlsym` uses `calloc`.

For Go versions prior to Go 1.15, you will also need to explicitly allow this
argument for use by CGo by setting the following environment variable:

```
export CGO_LDFLAGS_ALLOW="-Wl,--wrap=.*"
```

### Usage

Import this package and use the `cmemprof.Profiler` interface to start and stop
profiling:

```go
package main

import (
        "os"

        "github.com/nsrip-dd/cmemprof"
)

func main() {
        profiler := cmemprof.Profile{SampleRate: 500}
        profiler.Start()
        defer func() {
		f, _ := os.Create("cmem.pprof")
		defer f.Close()
		p, err := profiler.Stop()
		if err != nil {
			return
		}
		p.Write(f)
	}

        // your code here
}
```

You will also need to import a cgo symbolizer such as
`github.com/ianlancetaylor/cgosymbolizer` somewhere in your program to see the C
portion of call stacks in the profiles.

### Limitations

* It's not optimized so there is noticeable overhead when the profiler is enabled.
* This library has a few workarounds to avoid crashes due to unusual interactions
  between the Go runtime, cgo-generated code, and intercepting malloc. These
  workarounds depend on unstable implementation details. As such, this code may
  break your program for future Go releases. Use with caution.
