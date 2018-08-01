package util

import (
	durpb "github.com/golang/protobuf/ptypes/duration"
	"time"
	"github.com/golang/protobuf/ptypes"
)

// DurationsFromProtos turns a list of Duration protobuf objects into a list of time.Duration objects
func DurationsFromProtos(pbs []*durpb.Duration) ([]time.Duration, error) {
	ts := make([]time.Duration, len(pbs))
	for i, p := range pbs {
		var t time.Duration
		if p != nil {
			var err error
			t, err = ptypes.Duration(p)
			if err != nil {
				return nil, Error(InvalidArgument, err.Error())
			}
		}
		ts[i] = t
	}
	return ts, nil
}

// DurationFromProto turns a Duration protobuf object into a time.Duration object
func DurationFromProto(pb *durpb.Duration) (time.Duration, error) {
	ts, err := DurationsFromProtos([]*durpb.Duration{pb})
	if err != nil {
		return 0, err
	}
	return ts[0], nil
}

// ProtosToDurations turns a list of time.Duration objects into a list of Duration protobuf objects
func ProtosToDurations(durations []time.Duration) []*durpb.Duration {
	pbs := make([]*durpb.Duration, len(durations))
	for i, t := range durations {
		pbs[i] = ptypes.DurationProto(t)
	}
	return pbs
}

// ProtoToDuration turns a time.Duration object into a Duration protobuf object
func ProtoToDuration(duration time.Duration) *durpb.Duration {
	pbs := ProtosToDurations([]time.Duration{duration})
	return pbs[0]
}