package cmemprof

import (
	"os"

	"github.com/google/pprof/profile"
)

func buildProfile(samples map[uintptr][]*sample) *profile.Profile {
	p := &profile.Profile{}
	m := &profile.Mapping{
		ID:   1,
		File: os.Args[0], // XXX: Is there a better way to get the executable?
	}
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
	}
	locations := make(map[uint64]*profile.Location)
	for _, bucket := range samples {
		if len(bucket) == 0 {
			continue
		}
		for _, s := range bucket {
			psample := &profile.Sample{
				Value: []int64{int64(s.count), int64(s.size)},
			}
			// TODO: sample location, including address and if possilbe, line
			for _, pc := range s.stack {
				addr := uint64(uintptr(pc))
				loc, ok := locations[addr]
				if !ok {
					loc = &profile.Location{
						ID:      uint64(len(locations)) + 1,
						Mapping: m,
						Address: addr,
					}
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
