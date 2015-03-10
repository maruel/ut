// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package ut (for UtilTest) contains testing utilities to shorten unit tests.
package ut

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const sep = string(os.PathSeparator)

var blacklistedItems = []string{
	// TODO(maruel): Not very efficient.
	filepath.Join("runtime", "asm_386.s"),
	filepath.Join("runtime", "asm_amd64.s"),
	filepath.Join("runtime", "asm_arm.s"),
	filepath.Join("runtime", "proc.c"),
	filepath.Join("testing", "testing.go"),
	filepath.Join("utiltest", "utiltest.go"),
}

// truncatePath only keep the base filename and its immediate containing directory.
func truncatePath(file string) string {
	if index := strings.LastIndex(file, sep); index >= 0 {
		if index2 := strings.LastIndex(file[:index], sep); index2 >= 0 {
			// Keep the first directory to help figure out which file it is.
			return file[index2+1:]
		}
		return file[index+1:]
	}
	return file
}

// Decorate adds a prefix 'file:line: ' to a string, containing the 3 callers
// in the stack.
//
// It is mostly meant to be used internally.
//
// It is inspired by testing's decorate().
func Decorate(s string) string {
	type item struct {
		file string
		line int
	}
	items := make([]item, 4)
	for i := len(items); i > 0; i-- {
		_, file, line, ok := runtime.Caller(i) // decorate + log + public function.
		if ok {
			items[i-1].file = truncatePath(file)
			items[i-1].line = line
		} else {
			items[i-1].file = ""
		}
	}
	blacklisted := false
	for i := range items {
		for _, b := range blacklistedItems {
			if items[i].file == b {
				items[i].file = ""
				blacklisted = true
				break
			}
		}
	}
	if !blacklisted {
		items[0].file = ""
	}
	for _, i := range items {
		if i.file != "" {
			s = fmt.Sprintf("%s:%d: %s", i.file, i.line, s)
		}
	}
	return s
}

// AssertEqual verifies that two objects are equals and fails the test case
// otherwise.
//
// Equality is determined via reflect.DeepEqual().
func AssertEqual(t testing.TB, expected, actual interface{}) {
	AssertEqualf(t, expected, actual, "assertEqual() failure.\nExpected: %#v\nActual:   %#v", expected, actual)
}

// AssertEqualIndex verifies that two objects are equals and fails the test case
// otherwise.
//
// It is meant to be used in loops where a list of intrant->expected is
// processed so the assert failure message contains the index of the failing
// expectation.
//
// Equality is determined via reflect.DeepEqual().
func AssertEqualIndex(t testing.TB, index int, expected, actual interface{}) {
	AssertEqualf(t, expected, actual, "assertEqual() failure.\nIndex: %d\nExpected: %#v\nActual:   %#v", index, expected, actual)
}

// AssertEqualf verifies that two objects are equals and fails the test case
// otherwise.
//
// This functions enables specifying an arbitrary string on failure.
//
// Equality is determined via reflect.DeepEqual().
func AssertEqualf(t testing.TB, expected, actual interface{}, format string, items ...interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf(Decorate(format), items...)
	}
}

// testingWriter is used by NewWriter().
type testingWriter struct {
	t testing.TB
	b bytes.Buffer
}

func (t testingWriter) Write(p []byte) (int, error) {
	n, err := t.b.Write(p)
	if err != nil || n != len(p) {
		return n, err
	}
	// Manually scan for lines.
	for {
		b := t.b.Bytes()
		i := bytes.Index(b, []byte("\n"))
		if i == -1 {
			break
		}
		t.t.Log(string(b[:i]))
		t.b.Next(i + 1)
	}
	return n, err
}

func (t testingWriter) Close() error {
	remaining := t.b.Bytes()
	if len(remaining) != 0 {
		t.t.Log(string(remaining))
	}
	return nil
}

// NewWriter adapts a testing.TB into a io.WriteCloser that can be used
// with to log.SetOutput().
//
// Don't forget to defer foo.Close().
func NewWriter(t testing.TB) io.WriteCloser {
	return &testingWriter{t: t}
}
