// Copyright (c) the go-ruby-stringio/stringio authors
//
// SPDX-License-Identifier: BSD-3-Clause

package stringio

// Pos returns the current read/write cursor position (MRI's StringIO#pos / #tell).
func (s *StringIO) Pos() int { return s.pos }

// Tell is an alias for Pos (IO#tell).
func (s *StringIO) Tell() int { return s.pos }

// SetPos sets the cursor to n (MRI's StringIO#pos=). A negative n returns
// ErrInvalidSeek (Ruby: Errno::EINVAL). The position may be set past the end of
// the buffer, where a subsequent write extends and NUL-pads it.
func (s *StringIO) SetPos(n int) error {
	if n < 0 {
		return ErrInvalidSeek
	}
	s.pos = n
	return nil
}

// Seek moves the cursor by off relative to whence (SeekSet / SeekCur / SeekEnd)
// and returns 0, matching StringIO#seek. A resulting negative position returns
// ErrInvalidSeek (Ruby: Errno::EINVAL); an unknown whence returns ErrInvalidSeek
// as well (MRI raises Errno::EINVAL for an invalid whence).
func (s *StringIO) Seek(off, whence int) (int, error) {
	var target int
	switch whence {
	case SeekSet:
		target = off
	case SeekCur:
		target = s.pos + off
	case SeekEnd:
		target = len(s.buf) + off
	default:
		return 0, ErrInvalidSeek
	}
	if target < 0 {
		return 0, ErrInvalidSeek
	}
	s.pos = target
	return 0, nil
}

// Rewind resets the cursor to the start and the line number to 0, returning 0
// (StringIO#rewind).
func (s *StringIO) Rewind() int {
	s.pos = 0
	s.lineno = 0
	return 0
}

// String returns the entire buffer contents as a string (StringIO#string),
// independent of the cursor.
func (s *StringIO) String() string { return string(s.buf) }

// SetString replaces the entire buffer with str and resets the cursor and line
// number to 0 (StringIO#string=).
func (s *StringIO) SetString(str string) {
	s.buf = []byte(str)
	s.pos = 0
	s.lineno = 0
}

// Truncate shrinks or grows the buffer to exactly n bytes — NUL-padding when
// growing — and returns 0, matching StringIO#truncate. The cursor is left
// unchanged. A negative n returns ErrInvalidSeek (Ruby: Errno::EINVAL).
func (s *StringIO) Truncate(n int) (int, error) {
	if n < 0 {
		return 0, ErrInvalidSeek
	}
	if n < len(s.buf) {
		s.buf = s.buf[:n]
	} else if n > len(s.buf) {
		s.buf = append(s.buf, make([]byte, n-len(s.buf))...)
	}
	return 0, nil
}

// Size returns the buffer length in bytes (StringIO#size / #length).
func (s *StringIO) Size() int { return len(s.buf) }

// Length is an alias for Size (StringIO#length).
func (s *StringIO) Length() int { return len(s.buf) }

// Eof reports whether the cursor is at or past the end of the buffer
// (StringIO#eof? / #eof). It returns ErrNotReadable on a write-only stream, as
// MRI raises when querying EOF on a non-readable StringIO.
func (s *StringIO) Eof() (bool, error) {
	if err := s.checkReadable(); err != nil {
		return false, err
	}
	return s.pos >= len(s.buf), nil
}

// Close marks the stream closed; subsequent reads and writes return ErrClosed
// (StringIO#close).
func (s *StringIO) Close() { s.closed = true }

// Closed reports whether the stream has been closed (StringIO#closed?).
func (s *StringIO) Closed() bool { return s.closed }

// Flush is a no-op that returns the receiver (StringIO#flush — an in-memory buffer
// has nothing to flush).
func (s *StringIO) Flush() *StringIO { return s }

// Lineno returns the current line number — the count of lines read via Gets and
// friends since construction or the last Rewind (StringIO#lineno).
func (s *StringIO) Lineno() int { return s.lineno }

// SetLineno sets the line number (StringIO#lineno=). It does not move the cursor.
func (s *StringIO) SetLineno(n int) { s.lineno = n }
