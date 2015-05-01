// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package ut (for UtilTest) contains testing utilities to shorten unit tests.
package ut

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/kr/pretty"
)

var newLine = []byte{'\n'}

var blacklistedItems = map[string]bool{
	filepath.Join("runtime", "asm_386.s"):   true,
	filepath.Join("runtime", "asm_amd64.s"): true,
	filepath.Join("runtime", "asm_arm.s"):   true,
	filepath.Join("runtime", "proc.c"):      true,
	filepath.Join("testing", "testing.go"):  true,
	filepath.Join("ut", "utiltest.go"):      true,
	"utiltest.go":                           true,
}

// truncatePath only keep the base filename and its immediate containing
// directory.
func truncatePath(file string) string {
	return filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file))
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

// AssertEqual verifies that two objects are equals and calls FailNow() to
// immediately cancel the test case.
//
// It must be called from the main goroutine. Other goroutines must call
// ExpectEqual* flavors.
//
// Equality is determined via reflect.DeepEqual().
func AssertEqual(t testing.TB, expected, actual interface{}) {
	AssertEqualf(t, expected, actual, "AssertEqual() failure.\nExpected: %# v\nActual:   %# v", pretty.Formatter(expected), pretty.Formatter(actual))
}

// AssertEqualIndex verifies that two objects are equals and calls FailNow() to
// immediately cancel the test case.
//
// It must be called from the main goroutine. Other goroutines must call
// ExpectEqual* flavors.
//
// It is meant to be used in loops where a list of intrant->expected is
// processed so the assert failure message contains the index of the failing
// expectation.
//
// Equality is determined via reflect.DeepEqual().
func AssertEqualIndex(t testing.TB, index int, expected, actual interface{}) {
	AssertEqualf(t, expected, actual, "AssertEqualIndex() failure.\nIndex: %d\nExpected: %# v\nActual:   %# v", index, pretty.Formatter(expected), pretty.Formatter(actual))
}

// AssertEqualf verifies that two objects are equals and calls FailNow() to
// immediately cancel the test case.
//
// It must be called from the main goroutine. Other goroutines must call
// ExpectEqual* flavors.
//
// This functions enables specifying an arbitrary string on failure.
//
// Equality is determined via reflect.DeepEqual().
func AssertEqualf(t testing.TB, expected, actual interface{}, format string, items ...interface{}) {
	// TODO(maruel): Warning, this will be added in next commit, then will be
	// enforced.
	/*
		file := ""
		for i := 1; ; i++ {
			if _, file2, _, ok := runtime.Caller(i); ok {
				file = file2
			} else {
				break
			}
		}
		if file != "testing.go" {
			t.Logf(Decorate("ut.AssertEqual*() function MUST be called from within main test goroutine, use ut.ExpectEqual*() instead."))
		}
	*/
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf(Decorate(format), items...)
	}
}

// ExpectEqual verifies that two objects are equals and calls Fail() to mark
// the test case as failed but let it continue.
//
// It is fine to call this function from another goroutine than the main test
// case goroutine.
//
// Equality is determined via reflect.DeepEqual().
func ExpectEqual(t testing.TB, expected, actual interface{}) {
	ExpectEqualf(t, expected, actual, "ExpectEqual() failure.\nExpected: %# v\nActual:   %# v", pretty.Formatter(expected), pretty.Formatter(actual))
}

// ExpectEqualIndex verifies that two objects are equals and calls Fail() to
// mark the test case as failed but let it continue.
//
// It is fine to call this function from another goroutine than the main test
// case goroutine.
//
// It is meant to be used in loops where a list of intrant->expected is
// processed so the assert failure message contains the index of the failing
// expectation.
//
// Equality is determined via reflect.DeepEqual().
func ExpectEqualIndex(t testing.TB, index int, expected, actual interface{}) {
	ExpectEqualf(t, expected, actual, "ExpectEqualIndex() failure.\nIndex: %d\nExpected: %# v\nActual:   %# v", index, pretty.Formatter(expected), pretty.Formatter(actual))
}

// ExpectEqualf verifies that two objects are equals and calls Fail() to mark
// the test case as failed but let it continue.
//
// It is fine to call this function from another goroutine than the main test
// case goroutine.
//
// This functions enables specifying an arbitrary string on failure.
//
// Equality is determined via reflect.DeepEqual().
func ExpectEqualf(t testing.TB, expected, actual interface{}, format string, items ...interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		// Errorf() is thread-safe, t.Fatalf() is not.
		t.Errorf(Decorate(format), items...)
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
		i := bytes.Index(b, newLine)
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
