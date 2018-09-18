// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package measure

//go:generate protoc --proto_path=model/apitrace --gogo_out=model/apitrace model/apitrace/apitrace.proto

import (
	"sync"
	"time"

	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/measure/model/apitrace"
)

// Tracer allows to measure, store and list measured time entries.
type Tracer interface {
	// LogTime puts measured time to the table and resets the time.
	LogTime(entity string, start time.Time)
	// Get all trace entries stored
	Get() *apitrace.Trace
	// Clear removes entries from the log database
	Clear()
}

// NewTracer creates new tracer object
func NewTracer(name string, log logging.Logger) Tracer {
	return &tracer{
		name:   name,
		log:    log,
		index:  1,
		timedb: make([]*entry, 0),
	}
}

// Inner structure handling database and measure results
type tracer struct {
	sync.Mutex

	name string
	log  logging.Logger
	// Entry index, used in database as key and increased after every entry. Never resets since the tracer object is
	// created or the database is cleared
	index uint64
	// Time database, uses index as key and entry as value
	timedb []*entry
}

// Single time entry
type entry struct {
	index      uint64
	name       string
	loggedTime time.Duration
}

func (t *tracer) LogTime(entity string, start time.Time) {
	if t == nil {
		return
	}

	t.Lock()
	defer t.Unlock()

	// Store time
	t.timedb = append(t.timedb, &entry{
		index:      t.index,
		name:       entity,
		loggedTime: time.Since(start),
	})
	t.index++
}

func (t *tracer) Get() *apitrace.Trace {
	t.Lock()
	defer t.Unlock()

	trace := &apitrace.Trace{
		TracedEntries: make([]*apitrace.Trace_Entry, 0),
		AverageTimes:  make([]*apitrace.Trace_Average, 0),
	}

	average := make(map[string][]time.Duration) // message name -> measured times
	for _, entry := range t.timedb {
		// Add to total
		trace.Overall += uint64(entry.loggedTime)
		// Add to trace data
		trace.TracedEntries = append(trace.TracedEntries, &apitrace.Trace_Entry{
			Index:    entry.index,
			MsgName:  entry.name,
			Duration: uint64(entry.loggedTime.Nanoseconds()),
		})
		// Add to map for average data
		average[entry.name] = append(average[entry.name], entry.loggedTime)
	}

	// Prepare list of average times
	for msgName, times := range average {
		var total time.Duration
		for _, timeVal := range times {
			total += timeVal
		}
		averageTime := total.Nanoseconds() / int64(len(times))

		trace.AverageTimes = append(trace.AverageTimes, &apitrace.Trace_Average{
			MsgName:     msgName,
			AverageTime: uint64(averageTime),
		})
	}
	return trace
}

func (t *tracer) Clear() {
	t.timedb = make([]*entry, 0)
}
