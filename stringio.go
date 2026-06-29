// Copyright (c) the go-ruby-stringio/stringio authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package stringio is a pure-Go (no cgo), interpreter-independent reimplementation
// of Ruby's StringIO — an in-memory IO whose backing store is a String buffer.
//
// It reproduces MRI 4.0.5's StringIO semantics exactly: a read/write cursor over a
// byte buffer, mode-gated access (r/w/a and the "+" read-write variants), gets-style
// line splitting (separator, limit, and paragraph mode), seek-past-end NUL padding,
// byte-oriented reads vs. character-oriented getc/each_char, append-mode writes that
// always land at the end, and the EOFError / IOError / ArgumentError / Errno::EINVAL
// raises MRI emits. It carries no dependency on any Ruby runtime: it is the StringIO
// backend for go-embedded-ruby, but is a standalone, reusable module — a sibling of
// go-ruby-yaml, go-ruby-regexp, and go-ruby-erb.
//
// The API is idiomatic Go (methods return values and an error rather than raising),
// and a host maps the typed errors below — ErrNotReadable, ErrNotWritable,
// ErrClosed, ErrEOF, ErrNegativeLength, ErrInvalidSeek — onto Ruby's IOError /
// EOFError / ArgumentError / Errno::EINVAL when binding StringIO into an interpreter.
package stringio

import (
	"errors"
	"fmt"
)

// Errors returned by the methods, modelling the exceptions MRI's StringIO raises.
// A host binding StringIO into a Ruby runtime maps each onto its Ruby counterpart.
var (
	// ErrClosed is returned by an operation on a closed stream (Ruby: IOError,
	// "closed stream").
	ErrClosed = errors.New("closed stream")
	// ErrNotReadable is returned by a read on a write-only stream (Ruby: IOError,
	// "not opened for reading").
	ErrNotReadable = errors.New("not opened for reading")
	// ErrNotWritable is returned by a write on a read-only stream (Ruby: IOError,
	// "not opened for writing").
	ErrNotWritable = errors.New("not opened for writing")
	// ErrEOF is returned by readline/readchar/readbyte at end of input (Ruby:
	// EOFError, "end of file reached").
	ErrEOF = errors.New("end of file reached")
	// ErrNegativeLength is returned by Read with a negative length (Ruby:
	// ArgumentError, "negative length -N given").
	ErrNegativeLength = errors.New("negative length given")
	// ErrInvalidSeek is returned by SetPos / Seek with a negative resulting
	// position (Ruby: Errno::EINVAL, "Invalid argument").
	ErrInvalidSeek = errors.New("Invalid argument")
)

// Seek whence constants, matching IO::SEEK_SET / SEEK_CUR / SEEK_END (and the raw
// integer values MRI accepts).
const (
	SeekSet = 0 // from the start of the buffer
	SeekCur = 1 // from the current position
	SeekEnd = 2 // from the end of the buffer
)

// StringIO is an in-memory IO over a byte buffer with a read/write cursor.
//
// The mode (set at construction) gates access exactly as MRI's: "r" read-only,
// "w" write-only (initial content truncated), "a" append (writes always land at
// the end), and the "+" variants ("r+", "w+", "a+") allow both. The zero value is
// not usable; construct one with New or NewString.
type StringIO struct {
	buf    []byte
	pos    int
	lineno int

	readable bool
	writable bool
	append   bool

	closed bool
}

// New returns a StringIO over the bytes of s, opened in the given mode. The mode
// follows MRI's fopen-style strings:
//
//	"r"  read-only            "r+"  read-write, cursor at start
//	"w"  write-only, truncated "w+"  read-write, truncated
//	"a"  append (write at end) "a+"  read + append
//
// An empty mode defaults to "r+" (MRI's default when StringIO.new is given no
// mode), which is read-write over the supplied string. An unrecognised mode
// returns an error (Ruby: ArgumentError, "invalid access mode").
func New(s string, mode string) (*StringIO, error) {
	io := &StringIO{buf: []byte(s)}
	if mode == "" {
		// StringIO.new(str) with no mode is read-write ("r+") when a string is
		// given, but write-protected only if the string is frozen — which this
		// value model does not track, so it is read-write, matching the common case.
		mode = "r+"
	}
	switch mode {
	case "r":
		io.readable = true
	case "r+":
		io.readable, io.writable = true, true
	case "w":
		io.writable = true
		io.buf = io.buf[:0]
	case "w+":
		io.readable, io.writable = true, true
		io.buf = io.buf[:0]
	case "a":
		io.writable, io.append = true, true
	case "a+":
		io.readable, io.writable, io.append = true, true, true
	default:
		return nil, fmt.Errorf("invalid access mode %s", mode)
	}
	return io, nil
}

// NewString returns a read-write StringIO over s — the StringIO.new(s) default.
func NewString(s string) *StringIO {
	io, _ := New(s, "r+")
	return io
}

func (s *StringIO) checkReadable() error {
	if s.closed {
		return ErrClosed
	}
	if !s.readable {
		return ErrNotReadable
	}
	return nil
}

func (s *StringIO) checkWritable() error {
	if s.closed {
		return ErrClosed
	}
	if !s.writable {
		return ErrNotWritable
	}
	return nil
}
