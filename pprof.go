package cmemprof

import (
	"os"
	"runtime"

	"github.com/google/pprof/profile"
)

func buildProfile(samples map[uintptr][]*sample) *profile.Profile {
	p := &profile.Profile{}
	m := &profile.Mapping{
		ID:   1,
		File: os.Args[0], // XXX: Is there a better way to get the executable?
	}
	p.PeriodType = &profile.ValueType{Type: "space", Unit: "bytes"}
	p.Period = 1
	p.Mapping = []*profile.Mapping{m}
	p.SampleType = []*profile.ValueType{
		{
			Type: "alloc_objects",
			Unit: "count",
		},
		{
			Type: "alloc_space",
			Unit: "bytes",
		},
		// This profiler doesn't actually do heap profiling yet, but in
		// order to view Go allocation profiles and C allocation
		// profiles at the same time, the sample types need to be the
		// same
		{
			Type: "inuse_objects",
			Unit: "count",
		},
		{
			Type: "inuse_space",
			Unit: "bytes",
		},
	}
	locations := make(map[uint64]*profile.Location)
	var funcid uint64
	for _, bucket := range samples {
		if len(bucket) == 0 {
			continue
		}
		for _, s := range bucket {
			psample := &profile.Sample{
				Value: []int64{int64(s.count), int64(s.size), 0, 0},
			}
			// TODO: remove runtime.goexit from call stacks. This
			// function is added to the top of every Go call stack
			// and marks the point where a goroutine exits. The rest
			// of the Go profiles have this frame removed from the
			// call stack since it's not *really* part of the call
			// stack. Removing it allows the C and Go allocations to
			// show up side-by-side in a combined profile.
			for _, pc := range s.stack {
				addr := uint64(pc)
				loc, ok := locations[addr]
				if !ok {
					frames := runtime.CallersFrames([]uintptr{uintptr(pc)})
					frame, _ := frames.Next()
					loc = &profile.Location{
						ID:      uint64(len(locations)) + 1,
						Mapping: m,
						Address: uint64(frame.PC),
					}
					funcid++
					function := &profile.Function{
						ID:       funcid,
						Filename: frame.File,
						Name:     frame.Function,
					}
					p.Function = append(p.Function, function)
					loc.Line = append(loc.Line, profile.Line{
						Function: function,
						Line:     int64(frame.Line),
					})
					locations[addr] = loc
					p.Location = append(p.Location, loc)
				}
				psample.Location = append(psample.Location, loc)
			}
			p.Sample = append(p.Sample, psample)
		}
	}
	return p
}
