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

	"github.com/kr/pretty"
)

const sep = string(os.PathSeparator)

var blacklistedItems = map[string]bool{
	filepath.Join("runtime", "asm_386.s"):   true,
	filepath.Join("runtime", "asm_amd64.s"): true,
	filepath.Join("runtime", "asm_arm.s"):   true,
	filepath.Join("runtime", "proc.c"):      true,
	filepath.Join("testing", "testing.go"):  true,
	filepath.Join("ut", "utiltest.go"):      true,
	"utiltest.go":                           true,
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

func isBlacklisted(file string) bool {
	_, ok := blacklistedItems[file]
	return ok
}

// Decorate adds a prefix 'file:line: ' to a string, containing the 3 recent
// callers in the stack.
//
// It skips internal functions. It is mostly meant to be used internally.
//
// It is inspired by testing's decorate().
func Decorate(s string) string {
	type item struct {
		file string
		line int
	}
	items := []item{}
	for i := 1; i < 8 && len(items) < 3; i++ {
		_, file, line, ok := runtime.Caller(i) // decorate + log + public function.
		if ok {
			file = truncatePath(file)
			if !isBlacklisted(file) {
				items = append(items, item{file, line})
			}
		}
	}
	for _, i := range items {
		s = fmt.Sprintf("%s:%d: %s", strings.Replace(i.file, "%", "%%", -1), i.line, s)
	}
	return s
}

// AssertEqual verifies that two objects are equals and fails the test case
// otherwise.
//
// Equality is determined via reflect.DeepEqual().
func AssertEqual(t testing.TB, expected, actual interface{}) {
	AssertEqualf(t, expected, actual, "AssertEqual() failure.\nExpected: %# v\nActual:   %# v", pretty.Formatter(expected), pretty.Formatter(actual))
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
	AssertEqualf(t, expected, actual, "AssertEqual() failure.\nIndex: %d\nExpected: %# v\nActual:   %# v", index, pretty.Formatter(expected), pretty.Formatter(actual))
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
