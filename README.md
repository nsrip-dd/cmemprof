## C memory allocation profiler

This repository contains an experimental C allocation profiler for Go.

It's been lightly tested on Linux (Ubuntu 20.04) and macos (Big Sur).

### Building

On macos, you don't need to do anything special.

On Linux, first install libunwind, e.g.

```
apt install libunwind-dev
```

You'll also need to use the GNU linker on Linux, and will need to enable
the linker wrapper flag for use by CGo, which is required to provide replacements
for the allocation functions since `dlsym` uses `calloc`:

```
export CGO_LDFLAGS_ALLOW="-Wl,--wrap=.*"
```

### Usage

Simply import this package and use the `cmemprof.Profiler` interface to start and stop profiling:

```go
package main

import (
        "os"

        "github.com/nsrip-dd/cmemprof"
)

func main() {
        f, _ := os.Create("cmem.pprof")
        profiler := cmemprof.Profile{SampleRate: 500}
        profiler.Start(f)
        defer profiler.Stop()

        // your code here
}
```

### Limitations

* The current stack unwinding can't cross the C-Go boundary so call stacks stop at the point where they enter Go.
* This library doesn't do symbolization for profiles yet. If you have access to both the binary and a profile, `go tool pprof` can symbolize for you.
* It's not optimized so there is noticeable overhead when the profiler is enabled.