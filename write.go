// Copyright (c) the go-ruby-stringio/stringio authors
//
// SPDX-License-Identifier: BSD-3-Clause

package stringio

import (
	"fmt"
	"strings"
)

// writeAt writes p, returning the byte count. In append mode every write lands at
// the end of the buffer regardless of the cursor (MRI semantics); otherwise it
// overwrites at the cursor, extending and NUL-padding past the end as needed. The
// cursor advances to just after the written bytes.
func (s *StringIO) writeAt(p []byte) int {
	if s.append {
		s.pos = len(s.buf)
	}
	if end := s.pos + len(p); end > len(s.buf) {
		s.buf = append(s.buf, make([]byte, end-len(s.buf))...)
	}
	copy(s.buf[s.pos:], p)
	s.pos += len(p)
	return len(p)
}

// Write writes s to the stream at the current position (or the end, in append
// mode), returning the number of bytes written. It returns ErrNotWritable on a
// read-only stream and ErrClosed on a closed one.
func (s *StringIO) Write(str string) (int, error) {
	if err := s.checkWritable(); err != nil {
		return 0, err
	}
	return s.writeAt([]byte(str)), nil
}

// Puts writes each argument followed by a newline (unless it already ends in one),
// matching IO#puts. A call with no arguments writes a single newline. It does not
// flatten nested slices — a host flattens Ruby arrays before calling, as MRI's
// puts does at the Ruby level.
func (s *StringIO) Puts(args ...string) error {
	if err := s.checkWritable(); err != nil {
		return err
	}
	if len(args) == 0 {
		s.writeAt([]byte("\n"))
		return nil
	}
	for _, a := range args {
		if strings.HasSuffix(a, "\n") {
			s.writeAt([]byte(a))
		} else {
			s.writeAt([]byte(a + "\n"))
		}
	}
	return nil
}

// Print writes each argument verbatim, with no separators or trailing newline,
// matching IO#print (the host having already stringified each value).
func (s *StringIO) Print(args ...string) error {
	if err := s.checkWritable(); err != nil {
		return err
	}
	for _, a := range args {
		s.writeAt([]byte(a))
	}
	return nil
}

// Printf writes format expanded with args using Go's fmt verbs (the host having
// translated Ruby's format directives, or passing through where they coincide).
func (s *StringIO) Printf(format string, args ...any) error {
	if err := s.checkWritable(); err != nil {
		return err
	}
	s.writeAt([]byte(fmt.Sprintf(format, args...)))
	return nil
}

// Putc writes a single byte (the low 8 bits of c) and returns it, matching
// IO#putc with an Integer argument.
func (s *StringIO) Putc(c byte) (byte, error) {
	if err := s.checkWritable(); err != nil {
		return 0, err
	}
	s.writeAt([]byte{c})
	return c, nil
}

// PutString writes the first byte of str (IO#putc with a String argument); an
// empty string writes nothing. It returns str unchanged, as MRI's putc does.
func (s *StringIO) PutString(str string) (string, error) {
	if err := s.checkWritable(); err != nil {
		return "", err
	}
	if len(str) > 0 {
		s.writeAt([]byte{str[0]})
	}
	return str, nil
}
