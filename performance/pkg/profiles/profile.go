// Copyright (c) The EfficientGo Authors.
// Licensed under the Apache License 2.0.

package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	"github.com/efficientgo/tools/core/pkg/errcapture"
	"github.com/felixge/fgprof"
	"github.com/pkg/errors"
)

// StopHeapRecording stops heap recording, which will mean that
// allocation after this function will be not recorded in any heap profile
// until we resume it.
// NOTE: Given the runtime implementation this setting is global.
func StopHeapRecording() {
	runtime.MemProfileRate = 0
}

// StartHeapRecording is setting memory profile rate to default value of
// profiler sample every 512 KB allocated.
// If `everyAlloc` is true it sets profile rate to sample for every byte allocated.
// NOTE: Given the runtime implementation this setting is global.
func StartHeapRecording(everyAlloc bool) {
	if everyAlloc {
		runtime.MemProfileRate = 1
		return
	}
	runtime.MemProfileRate = 512 * 1024
}

// Heap creates a heap profile in mem.pprof file in given directory.
// Directory might be non-existent, heap will create it if needed.
// NOTE: Given the runtime implementation this setting is global.
func Heap(dir string) (err error) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(dir, "mem.pprof"))
	if err != nil {
		return err
	}
	defer errcapture.Do(&err, f.Close, "close")
	return pprof.WriteHeapProfile(f)
}

type CPUType string

const (
	CPUTypeBuiltIn CPUType = "built-in"
	// CPUTypeFGProf represents enhanced https://github.com/felixge/fgprof CPU profiling.
	CPUTypeFGProf CPUType = "fgprof"
)

// StartCPU starts CPU profiling. If no error is returned, it returns close function that stops and flushes
// profile to cpu.pprof or cpu.fgprof.pprof file in a given directory.
// Directory might be non-existent, heap will create it if needed.
// NOTE: Given the runtime implementation this setting is global.
func StartCPU(dir string, typ CPUType) (closeFn func() error, err error) {
	fileName := "cpu.pprof"
	switch typ {
	case CPUTypeBuiltIn:
	case CPUTypeFGProf:
		fileName = "cpu.fgprof.pprof"
	default:
		return nil, errors.Errorf("unknown CPU profile type %v", typ)
	}

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}

	f, err := os.Create(filepath.Join(dir, fileName))
	if err != nil {
		return nil, err
	}

	switch typ {
	case CPUTypeBuiltIn:
		if err = pprof.StartCPUProfile(f); err != nil {
			errcapture.Do(&err, f.Close, fmt.Sprintf("close %v", filepath.Join(dir, fileName)))
			return nil, err
		}
		closeFn = func() (ferr error) {
			pprof.StopCPUProfile()
			return errors.Wrapf(f.Close(), "close %v", filepath.Join(dir, fileName))
		}
	case CPUTypeFGProf:
		closeFGProfFn := fgprof.Start(f, fgprof.FormatPprof)
		closeFn = func() (ferr error) {
			defer errcapture.Do(&ferr, f.Close, fmt.Sprintf("close %v", filepath.Join(dir, fileName)))
			return closeFGProfFn()
		}
	}
	return closeFn, nil
}

// StartTrace starts tracingIf no error is returned, it returns close function that stops and flushes
// profile to trace.out file in a given directory.
// Directory might be non-existent, heap will create it if needed.
// NOTE: Given the runtime implementation this setting is global.
func StartTrace(dir string) (closeFn func() error, err error) {
	fileName := "trace.out"

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}

	f, err := os.Create(filepath.Join(dir, fileName))
	if err != nil {
		return nil, err
	}

	if err := trace.Start(f); err != nil {
		return nil, err
	}
	return func() error {
		trace.Stop()
		return errors.Wrapf(f.Close(), "close %v", filepath.Join(dir, fileName))
	}, nil
}
