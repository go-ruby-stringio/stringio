// Copyright (c) the go-ruby-stringio/stringio authors
//
// SPDX-License-Identifier: BSD-3-Clause

package stringio

import (
	"bytes"
	"unicode/utf8"
)

// Read reads up to n bytes from the current position and advances the cursor. A
// negative n returns ErrNegativeLength (Ruby: ArgumentError). At end of input a
// length read yields ("", ErrEOF-less) — specifically (nil-equivalent) per MRI it
// returns nil; here that surfaces as ("", false) where the bool reports whether
// any data was produced:
//
//	data, ok, err := io.Read(3)
//
//	ok == false  → MRI returned nil (read past EOF with a length given)
//	ok == true   → data holds the (possibly shorter) bytes read
//
// A zero length always returns ("", true) without consuming input (MRI returns "").
func (s *StringIO) Read(n int) (data string, ok bool, err error) {
	if err := s.checkReadable(); err != nil {
		return "", false, err
	}
	if n < 0 {
		return "", false, ErrNegativeLength
	}
	if n == 0 {
		return "", true, nil
	}
	if s.pos >= len(s.buf) {
		return "", false, nil // length read at EOF → nil in MRI
	}
	end := s.pos + n
	if end > len(s.buf) {
		end = len(s.buf)
	}
	data = string(s.buf[s.pos:end])
	s.pos = end
	return data, true, nil
}

// ReadAll reads from the current position to the end of the buffer (MRI's
// StringIO#read with no length), returning "" at or past EOF and advancing the
// cursor to the end.
func (s *StringIO) ReadAll() (string, error) {
	if err := s.checkReadable(); err != nil {
		return "", err
	}
	start := s.pos
	if start > len(s.buf) {
		start = len(s.buf) // SetPos may have moved past the end
	}
	data := string(s.buf[start:])
	s.pos = len(s.buf)
	return data, nil
}

// Gets reads one line — bytes up to and including the next occurrence of sep,
// or the rest of the buffer if sep is not found — advancing the cursor and
// bumping the line number. It returns ("", false, nil) at end of input (MRI: nil).
//
// sep follows MRI's $/ conventions:
//
//	"\n"  (the default) splits on newline
//	""    paragraph mode: a record runs to a run of two or more newlines, whose
//	      leading blank lines are then skipped
//	other any string separator
//
// Use GetsLimit for the (separator, limit) and (limit) forms.
func (s *StringIO) Gets(sep string) (line string, ok bool, err error) {
	return s.GetsLimit(sep, -1)
}

// GetsLimit is Gets with a byte limit: at most limit bytes are returned even if
// the separator has not yet been reached (MRI's gets(sep, limit) / gets(limit)).
// A negative limit means unlimited. A separator of "" selects paragraph mode.
func (s *StringIO) GetsLimit(sep string, limit int) (line string, ok bool, err error) {
	if err := s.checkReadable(); err != nil {
		return "", false, err
	}
	line, ok = s.gets(sep, limit)
	return line, ok, nil
}

// gets is the unchecked core of GetsLimit: it assumes the stream is readable
// (the caller has already verified) and returns (line, found). The loop-based
// readers (ReadLines / Each) call it so they have no unreachable error branch.
func (s *StringIO) gets(sep string, limit int) (line string, ok bool) {
	if s.pos >= len(s.buf) {
		return "", false
	}
	if limit == 0 {
		return "", true
	}

	if sep == "" {
		// Paragraph mode: skip leading newlines, then read to a blank line (a run
		// of two or more newlines), including the first trailing newline run.
		for s.pos < len(s.buf) && s.buf[s.pos] == '\n' {
			s.pos++
		}
		if s.pos >= len(s.buf) {
			return "", false
		}
		rest := s.buf[s.pos:]
		end := len(rest)
		// Index directly on the []byte (no string(rest) copy): bytes.Index scans
		// only up to the first "\n\n", so the paragraph split stays linear.
		if i := bytes.Index(rest, []byte("\n\n")); i >= 0 {
			// Include the run of newlines that terminates the paragraph.
			j := i + 1
			for j < len(rest) && rest[j] == '\n' {
				j++
			}
			end = j
		}
		if limit >= 0 && limit < end {
			end = s.clampToRune(s.pos, limit) - s.pos
		}
		line = string(rest[:end])
		s.pos += end
		s.lineno++
		return line, true
	}

	rest := s.buf[s.pos:]
	end := len(rest)
	// Scan for the separator directly on the []byte slice starting at the cursor.
	// This never copies the remaining buffer to a string (the old string(rest)
	// allocated and rescanned a shrinking copy per line — O(n²) over the line
	// count) and never rescans already-consumed bytes: bytes.Index/IndexByte reads
	// only as far as the next separator, so the whole line walk is linear. For the
	// common single-byte separator ("\n", the $/ default) IndexByte is the tight
	// memchr-equivalent path.
	var i int
	if len(sep) == 1 {
		i = bytes.IndexByte(rest, sep[0])
	} else {
		i = bytes.Index(rest, []byte(sep))
	}
	if i >= 0 {
		end = i + len(sep)
	}
	if limit >= 0 && limit < end {
		end = s.clampToRune(s.pos, limit) - s.pos
	}
	line = string(rest[:end])
	s.pos += end
	s.lineno++
	return line, true
}

// clampToRune returns the largest absolute buffer index >= start and <= start+limit
// that does not split a multi-byte UTF-8 character, so a limited gets never returns
// a partial rune (MRI's behaviour). The caller guarantees start+limit < len(buf).
func (s *StringIO) clampToRune(start, limit int) int {
	end := start + limit
	// Back up while the byte at end is a UTF-8 continuation byte, so the rune that
	// spans it is excluded rather than truncated.
	for end > start && !utf8.RuneStart(s.buf[end]) {
		end--
	}
	return end
}

// ReadLine is Gets that raises at EOF: it returns ErrEOF (Ruby: EOFError) instead
// of a nil line when there is no more input.
func (s *StringIO) ReadLine(sep string) (string, error) {
	line, ok, err := s.Gets(sep)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrEOF
	}
	return line, nil
}

// ReadLines reads every remaining line (per Gets / sep) into a slice, advancing
// the cursor to the end.
func (s *StringIO) ReadLines(sep string) ([]string, error) {
	if err := s.checkReadable(); err != nil {
		return nil, err
	}
	var lines []string
	for {
		line, ok := s.gets(sep, -1)
		if !ok {
			break
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// Each calls fn with each remaining line (per Gets / sep). It stops early — and
// returns the error — if fn returns one.
func (s *StringIO) Each(sep string, fn func(line string) error) error {
	if err := s.checkReadable(); err != nil {
		return err
	}
	for {
		line, ok := s.gets(sep, -1)
		if !ok {
			return nil
		}
		if err := fn(line); err != nil {
			return err
		}
	}
}

// Getc reads one UTF-8 character from the current position, advancing the cursor
// past it. It returns ("", false, nil) at end of input (MRI: nil). A malformed
// byte is returned as a single-byte string, like MRI.
func (s *StringIO) Getc() (ch string, ok bool, err error) {
	if err := s.checkReadable(); err != nil {
		return "", false, err
	}
	ch, ok = s.getc()
	return ch, ok, nil
}

// getc is the unchecked core of Getc (the caller has verified readability).
func (s *StringIO) getc() (ch string, ok bool) {
	if s.pos >= len(s.buf) {
		return "", false
	}
	r, size := utf8.DecodeRune(s.buf[s.pos:])
	if r == utf8.RuneError && size <= 1 {
		ch = string(s.buf[s.pos : s.pos+1])
		s.pos++
		return ch, true
	}
	ch = string(s.buf[s.pos : s.pos+size])
	s.pos += size
	return ch, true
}

// ReadChar is Getc that raises at EOF (Ruby: EOFError).
func (s *StringIO) ReadChar() (string, error) {
	ch, ok, err := s.Getc()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrEOF
	}
	return ch, nil
}

// Getbyte reads one byte from the current position, advancing the cursor. It
// returns (0, false, nil) at end of input (MRI: nil).
func (s *StringIO) Getbyte() (b byte, ok bool, err error) {
	if err := s.checkReadable(); err != nil {
		return 0, false, err
	}
	b, ok = s.getbyte()
	return b, ok, nil
}

// getbyte is the unchecked core of Getbyte (the caller has verified readability).
func (s *StringIO) getbyte() (b byte, ok bool) {
	if s.pos >= len(s.buf) {
		return 0, false
	}
	b = s.buf[s.pos]
	s.pos++
	return b, true
}

// ReadByte is Getbyte that raises at EOF (Ruby: EOFError).
func (s *StringIO) ReadByte() (byte, error) {
	b, ok, err := s.Getbyte()
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrEOF
	}
	return b, nil
}

// EachChar calls fn with each remaining UTF-8 character. It stops early — and
// returns the error — if fn returns one.
func (s *StringIO) EachChar(fn func(ch string) error) error {
	if err := s.checkReadable(); err != nil {
		return err
	}
	for {
		ch, ok := s.getc()
		if !ok {
			return nil
		}
		if err := fn(ch); err != nil {
			return err
		}
	}
}

// EachByte calls fn with each remaining byte. It stops early — and returns the
// error — if fn returns one.
func (s *StringIO) EachByte(fn func(b byte) error) error {
	if err := s.checkReadable(); err != nil {
		return err
	}
	for {
		b, ok := s.getbyte()
		if !ok {
			return nil
		}
		if err := fn(b); err != nil {
			return err
		}
	}
}

// Ungetc pushes a character back so the next read returns it. The cursor moves
// back by the byte length of ch and ch's bytes overwrite the buffer there (MRI
// rewrites the buffer at the new position). At position 0 the bytes are prepended.
func (s *StringIO) Ungetc(ch string) error {
	if err := s.checkReadable(); err != nil {
		return err
	}
	return s.ungetBytes([]byte(ch))
}

// Ungetbyte pushes a single byte back so the next read returns it (MRI's
// StringIO#ungetbyte).
func (s *StringIO) Ungetbyte(b byte) error {
	if err := s.checkReadable(); err != nil {
		return err
	}
	return s.ungetBytes([]byte{b})
}

// ungetBytes is the shared backing for Ungetc / Ungetbyte: it moves the cursor
// back by len(p) (prepending when that would go before the start) and writes p at
// the new position.
func (s *StringIO) ungetBytes(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	if s.pos < len(p) {
		// Prepend the missing prefix, then the cursor sits at 0. After this the
		// buffer is at least len(p) long, so the overwrite below stays in bounds.
		grow := len(p) - s.pos
		s.buf = append(make([]byte, grow), s.buf...)
		s.pos = 0
	} else {
		// The cursor moves back by len(p); pos+len(p) is its old value, in bounds.
		s.pos -= len(p)
	}
	copy(s.buf[s.pos:], p)
	return nil
}
