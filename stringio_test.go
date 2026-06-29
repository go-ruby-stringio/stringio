// Copyright (c) the go-ruby-stringio/stringio authors
//
// SPDX-License-Identifier: BSD-3-Clause

package stringio

import (
	"errors"
	"reflect"
	"testing"
)

// must builds a StringIO with New, failing the test on an unexpected error.
func must(t *testing.T, s, mode string) *StringIO {
	t.Helper()
	io, err := New(s, mode)
	if err != nil {
		t.Fatalf("New(%q, %q): %v", s, mode, err)
	}
	return io
}

func TestNewModes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		mode                       string
		readable, writable, append bool
	}{
		{"r", true, false, false},
		{"r+", true, true, false},
		{"w", false, true, false},
		{"w+", true, true, false},
		{"a", false, true, true},
		{"a+", true, true, true},
		{"", true, true, false}, // default r+
	}
	for _, c := range cases {
		io := must(t, "seed", c.mode)
		if io.readable != c.readable || io.writable != c.writable || io.append != c.append {
			t.Errorf("mode %q: got r=%v w=%v a=%v", c.mode, io.readable, io.writable, io.append)
		}
	}
	// w / w+ truncate the seed.
	if got := must(t, "seed", "w").String(); got != "" {
		t.Errorf("w mode should truncate, got %q", got)
	}
	if got := must(t, "seed", "w+").String(); got != "" {
		t.Errorf("w+ mode should truncate, got %q", got)
	}
	// Invalid mode.
	if _, err := New("x", "z+bad"); err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestNewString(t *testing.T) {
	t.Parallel()
	io := NewString("hi")
	if !io.readable || !io.writable {
		t.Error("NewString should be read-write")
	}
	if io.String() != "hi" {
		t.Errorf("got %q", io.String())
	}
}

func TestRead(t *testing.T) {
	t.Parallel()
	io := NewString("hello")
	// read(3), read all, read(1) at EOF.
	if d, ok, err := io.Read(3); d != "hel" || !ok || err != nil {
		t.Fatalf("Read(3) = %q,%v,%v", d, ok, err)
	}
	if d, err := io.ReadAll(); d != "lo" || err != nil {
		t.Fatalf("ReadAll = %q,%v", d, err)
	}
	if d, ok, err := io.Read(1); d != "" || ok || err != nil {
		t.Fatalf("Read(1) at EOF = %q,%v,%v (want nil)", d, ok, err)
	}
	// read(0) returns "" without consuming.
	io2 := NewString("ab")
	if d, ok, err := io2.Read(0); d != "" || !ok || err != nil || io2.Pos() != 0 {
		t.Fatalf("Read(0) = %q,%v,%v pos=%d", d, ok, err, io2.Pos())
	}
	// negative length.
	if _, _, err := io2.Read(-1); !errors.Is(err, ErrNegativeLength) {
		t.Fatalf("Read(-1) err = %v", err)
	}
	// read past requested length clamps.
	io3 := NewString("ab")
	if d, ok, _ := io3.Read(10); d != "ab" || !ok {
		t.Fatalf("Read(10) = %q,%v", d, ok)
	}
	// ReadAll with pos past end.
	io4 := NewString("ab")
	_ = io4.SetPos(10)
	if d, err := io4.ReadAll(); d != "" || err != nil {
		t.Fatalf("ReadAll past end = %q,%v", d, err)
	}
}

func TestReadModeChecks(t *testing.T) {
	t.Parallel()
	w := must(t, "", "w")
	if _, _, err := w.Read(1); !errors.Is(err, ErrNotReadable) {
		t.Errorf("Read on w = %v", err)
	}
	if _, err := w.ReadAll(); !errors.Is(err, ErrNotReadable) {
		t.Errorf("ReadAll on w = %v", err)
	}
	if _, _, err := w.Gets("\n"); !errors.Is(err, ErrNotReadable) {
		t.Errorf("Gets on w = %v", err)
	}
	if _, _, err := w.Getc(); !errors.Is(err, ErrNotReadable) {
		t.Errorf("Getc on w = %v", err)
	}
	if _, _, err := w.Getbyte(); !errors.Is(err, ErrNotReadable) {
		t.Errorf("Getbyte on w = %v", err)
	}
	if _, err := w.ReadLines("\n"); !errors.Is(err, ErrNotReadable) {
		t.Errorf("ReadLines on w = %v", err)
	}
	if err := w.Each("\n", func(string) error { return nil }); !errors.Is(err, ErrNotReadable) {
		t.Errorf("Each on w = %v", err)
	}
	if err := w.EachChar(func(string) error { return nil }); !errors.Is(err, ErrNotReadable) {
		t.Errorf("EachChar on w = %v", err)
	}
	if err := w.EachByte(func(byte) error { return nil }); !errors.Is(err, ErrNotReadable) {
		t.Errorf("EachByte on w = %v", err)
	}
	if _, err := w.Eof(); !errors.Is(err, ErrNotReadable) {
		t.Errorf("Eof on w = %v", err)
	}
	if err := w.Ungetc("x"); !errors.Is(err, ErrNotReadable) {
		t.Errorf("Ungetc on w = %v", err)
	}
	if err := w.Ungetbyte('x'); !errors.Is(err, ErrNotReadable) {
		t.Errorf("Ungetbyte on w = %v", err)
	}
}

func TestWriteModeChecks(t *testing.T) {
	t.Parallel()
	r := must(t, "ab", "r")
	if _, err := r.Write("x"); !errors.Is(err, ErrNotWritable) {
		t.Errorf("Write on r = %v", err)
	}
	if err := r.Puts("x"); !errors.Is(err, ErrNotWritable) {
		t.Errorf("Puts on r = %v", err)
	}
	if err := r.Print("x"); !errors.Is(err, ErrNotWritable) {
		t.Errorf("Print on r = %v", err)
	}
	if err := r.Printf("%d", 1); !errors.Is(err, ErrNotWritable) {
		t.Errorf("Printf on r = %v", err)
	}
	if _, err := r.Putc('x'); !errors.Is(err, ErrNotWritable) {
		t.Errorf("Putc on r = %v", err)
	}
	if _, err := r.PutString("x"); !errors.Is(err, ErrNotWritable) {
		t.Errorf("PutString on r = %v", err)
	}
}

func TestClosed(t *testing.T) {
	t.Parallel()
	io := NewString("ab")
	if io.Closed() {
		t.Error("new stream should be open")
	}
	io.Close()
	if !io.Closed() {
		t.Error("should be closed")
	}
	if _, _, err := io.Read(1); !errors.Is(err, ErrClosed) {
		t.Errorf("Read closed = %v", err)
	}
	if _, err := io.Write("x"); !errors.Is(err, ErrClosed) {
		t.Errorf("Write closed = %v", err)
	}
}

func TestGets(t *testing.T) {
	t.Parallel()
	io := NewString("a\nb\nc")
	want := []struct {
		line string
		ok   bool
	}{{"a\n", true}, {"b\n", true}, {"c", true}, {"", false}}
	for i, w := range want {
		line, ok, err := io.Gets("\n")
		if line != w.line || ok != w.ok || err != nil {
			t.Errorf("Gets #%d = %q,%v,%v want %q,%v", i, line, ok, err, w.line, w.ok)
		}
	}
	// custom separator.
	io2 := NewString("a;b")
	if l, ok, _ := io2.Gets(";"); l != "a;" || !ok {
		t.Errorf("Gets(;) = %q,%v", l, ok)
	}
	if l, ok, _ := io2.Gets(";"); l != "b" || !ok {
		t.Errorf("Gets(;) = %q,%v", l, ok)
	}
	// separator absent → rest.
	io3 := NewString("xyz")
	if l, ok, _ := io3.Gets("\n"); l != "xyz" || !ok {
		t.Errorf("Gets no-sep = %q,%v", l, ok)
	}
}

func TestGetsLimit(t *testing.T) {
	t.Parallel()
	io := NewString("hello world")
	// limit only (sep "\n" not present within limit).
	if l, ok, _ := io.GetsLimit("\n", 3); l != "hel" || !ok {
		t.Errorf("GetsLimit(\\n,3) = %q,%v", l, ok)
	}
	// limit 0 returns "".
	if l, ok, _ := io.GetsLimit("\n", 0); l != "" || !ok {
		t.Errorf("GetsLimit limit=0 = %q,%v", l, ok)
	}
	// negative limit = unlimited.
	io2 := NewString("ab\ncd")
	if l, ok, _ := io2.GetsLimit("\n", -1); l != "ab\n" || !ok {
		t.Errorf("GetsLimit neg = %q,%v", l, ok)
	}
	// limit truncates before separator found.
	io3 := NewString("abcdef")
	if l, _, _ := io3.GetsLimit("e", 3); l != "abc" {
		t.Errorf("GetsLimit sep+limit = %q", l)
	}
	// limit must not split a multibyte rune: "é" is 2 bytes; limit 1 backs up to 0.
	io4 := NewString("é")
	if l, _, _ := io4.GetsLimit("\n", 1); l != "" {
		t.Errorf("GetsLimit rune-split = %q (want empty)", l)
	}
	// limit at EOF.
	io5 := NewString("")
	if _, ok, _ := io5.GetsLimit("\n", 5); ok {
		t.Error("GetsLimit at EOF should be !ok")
	}
}

func TestGetsParagraph(t *testing.T) {
	t.Parallel()
	io := NewString("a\n\n\nb\nc\n\nd")
	if l, _, _ := io.Gets(""); l != "a\n\n\n" {
		t.Errorf("para 1 = %q", l)
	}
	if l, _, _ := io.Gets(""); l != "b\nc\n\n" {
		t.Errorf("para 2 = %q", l)
	}
	if l, _, _ := io.Gets(""); l != "d" {
		t.Errorf("para 3 = %q", l)
	}
	if _, ok, _ := io.Gets(""); ok {
		t.Error("para EOF should be !ok")
	}
	// Paragraph mode where only leading newlines remain → EOF.
	io2 := NewString("\n\n\n")
	if _, ok, _ := io2.Gets(""); ok {
		t.Error("all-newline para should be !ok")
	}
	// Paragraph mode with a byte limit.
	io3 := NewString("hello\n\nworld")
	if l, _, _ := io3.GetsLimit("", 3); l != "hel" {
		t.Errorf("para limit = %q", l)
	}
}

func TestReadLine(t *testing.T) {
	t.Parallel()
	io := NewString("x\ny")
	if l, err := io.ReadLine("\n"); l != "x\n" || err != nil {
		t.Errorf("ReadLine = %q,%v", l, err)
	}
	if l, err := io.ReadLine("\n"); l != "y" || err != nil {
		t.Errorf("ReadLine = %q,%v", l, err)
	}
	if _, err := io.ReadLine("\n"); !errors.Is(err, ErrEOF) {
		t.Errorf("ReadLine EOF = %v", err)
	}
	// mode check propagates.
	w := must(t, "", "w")
	if _, err := w.ReadLine("\n"); !errors.Is(err, ErrNotReadable) {
		t.Errorf("ReadLine on w = %v", err)
	}
}

func TestReadLinesAndEach(t *testing.T) {
	t.Parallel()
	io := NewString("p\nq\n")
	lines, err := io.ReadLines("\n")
	if err != nil || !reflect.DeepEqual(lines, []string{"p\n", "q\n"}) {
		t.Errorf("ReadLines = %v,%v", lines, err)
	}
	io2 := NewString("p\nq\n")
	var got []string
	if err := io2.Each("\n", func(l string) error { got = append(got, l); return nil }); err != nil {
		t.Fatalf("Each: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"p\n", "q\n"}) {
		t.Errorf("Each = %v", got)
	}
	// Each early stop.
	io3 := NewString("a\nb\nc\n")
	stop := errors.New("stop")
	got = nil
	if err := io3.Each("\n", func(l string) error {
		got = append(got, l)
		if len(got) == 1 {
			return stop
		}
		return nil
	}); !errors.Is(err, stop) {
		t.Errorf("Each stop err = %v", err)
	}
	if len(got) != 1 {
		t.Errorf("Each should have stopped at 1, got %v", got)
	}
}

func TestGetcAndChars(t *testing.T) {
	t.Parallel()
	io := NewString("héllo")
	if c, ok, _ := io.Getc(); c != "h" || !ok {
		t.Errorf("Getc = %q,%v", c, ok)
	}
	if c, ok, _ := io.Getc(); c != "é" || !ok {
		t.Errorf("Getc = %q,%v", c, ok)
	}
	// Getc at EOF.
	empty := NewString("")
	if c, ok, _ := empty.Getc(); c != "" || ok {
		t.Errorf("Getc empty = %q,%v", c, ok)
	}
	// malformed byte → single byte.
	bad := NewString("\xff")
	if c, ok, _ := bad.Getc(); c != "\xff" || !ok {
		t.Errorf("Getc malformed = %q,%v", c, ok)
	}
	// EachChar.
	io2 := NewString("ab")
	var got []string
	if err := io2.EachChar(func(c string) error { got = append(got, c); return nil }); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("EachChar = %v", got)
	}
	// EachChar early stop.
	io3 := NewString("abc")
	stop := errors.New("stop")
	got = nil
	if err := io3.EachChar(func(c string) error {
		got = append(got, c)
		return stop
	}); !errors.Is(err, stop) {
		t.Errorf("EachChar stop = %v", err)
	}
	if len(got) != 1 {
		t.Errorf("EachChar should stop at 1, got %v", got)
	}
}

func TestReadChar(t *testing.T) {
	t.Parallel()
	io := NewString("x")
	if c, err := io.ReadChar(); c != "x" || err != nil {
		t.Errorf("ReadChar = %q,%v", c, err)
	}
	if _, err := io.ReadChar(); !errors.Is(err, ErrEOF) {
		t.Errorf("ReadChar EOF = %v", err)
	}
	w := must(t, "", "w")
	if _, err := w.ReadChar(); !errors.Is(err, ErrNotReadable) {
		t.Errorf("ReadChar on w = %v", err)
	}
}

func TestBytes(t *testing.T) {
	t.Parallel()
	io := NewString("AB")
	if b, ok, _ := io.Getbyte(); b != 'A' || !ok {
		t.Errorf("Getbyte = %d,%v", b, ok)
	}
	if b, ok, _ := io.Getbyte(); b != 'B' || !ok {
		t.Errorf("Getbyte = %d,%v", b, ok)
	}
	if _, ok, _ := io.Getbyte(); ok {
		t.Error("Getbyte at EOF should be !ok")
	}
	// EachByte.
	io2 := NewString("AB")
	var got []byte
	if err := io2.EachByte(func(b byte) error { got = append(got, b); return nil }); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []byte{'A', 'B'}) {
		t.Errorf("EachByte = %v", got)
	}
	// EachByte early stop.
	io3 := NewString("ABC")
	stop := errors.New("stop")
	got = nil
	if err := io3.EachByte(func(b byte) error { got = append(got, b); return stop }); !errors.Is(err, stop) {
		t.Errorf("EachByte stop = %v", err)
	}
	if len(got) != 1 {
		t.Errorf("EachByte should stop at 1, got %v", got)
	}
	// ReadByte.
	io4 := NewString("Z")
	if b, err := io4.ReadByte(); b != 'Z' || err != nil {
		t.Errorf("ReadByte = %d,%v", b, err)
	}
	if _, err := io4.ReadByte(); !errors.Is(err, ErrEOF) {
		t.Errorf("ReadByte EOF = %v", err)
	}
	w := must(t, "", "w")
	if _, err := w.ReadByte(); !errors.Is(err, ErrNotReadable) {
		t.Errorf("ReadByte on w = %v", err)
	}
}

func TestUnget(t *testing.T) {
	t.Parallel()
	// ungetc mid-buffer overwrites.
	io := NewString("hello")
	c, _, _ := io.Getc()
	if err := io.Ungetc(c); err != nil {
		t.Fatal(err)
	}
	if d, _ := io.ReadAll(); d != "hello" {
		t.Errorf("after ungetc = %q", d)
	}
	// ungetc at pos 0 prepends.
	io2 := NewString("abc")
	if err := io2.Ungetc("X"); err != nil {
		t.Fatal(err)
	}
	if io2.String() != "Xabc" || io2.Pos() != 0 {
		t.Errorf("ungetc prepend = %q pos=%d", io2.String(), io2.Pos())
	}
	// ungetc multibyte mid-buffer.
	io3 := NewString("abc")
	io3.Getc()
	if err := io3.Ungetc("é"); err != nil {
		t.Fatal(err)
	}
	if io3.String() != "ébc" {
		t.Errorf("ungetc multibyte = %q", io3.String())
	}
	// ungetbyte at pos 0.
	io4 := NewString("abc")
	if err := io4.Ungetbyte('x'); err != nil {
		t.Fatal(err)
	}
	if io4.String() != "xabc" {
		t.Errorf("ungetbyte = %q", io4.String())
	}
	// empty unget is a no-op.
	io5 := NewString("ab")
	if err := io5.Ungetc(""); err != nil {
		t.Fatal(err)
	}
	if io5.String() != "ab" || io5.Pos() != 0 {
		t.Errorf("empty unget changed state: %q pos=%d", io5.String(), io5.Pos())
	}
	// ungetc onto an empty buffer prepends and leaves the cursor at 0.
	io6 := NewString("")
	if err := io6.Ungetc("x"); err != nil {
		t.Fatal(err)
	}
	if io6.String() != "x" || io6.Pos() != 0 {
		t.Errorf("ungetc empty = %q pos=%d", io6.String(), io6.Pos())
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()
	io := NewString("")
	if n, _ := io.Write("ab"); n != 2 {
		t.Errorf("Write n = %d", n)
	}
	if _, err := io.Write("c"); err != nil {
		t.Fatal(err)
	}
	if io.String() != "abc" {
		t.Errorf("String = %q", io.String())
	}
	// overwrite at cursor.
	io2 := NewString("hello")
	io2.SetPos(1)
	io2.Write("X")
	if io2.String() != "hXllo" {
		t.Errorf("overwrite = %q", io2.String())
	}
	// write multibyte byte count.
	io3 := NewString("")
	if n, _ := io3.Write("héllo"); n != 6 {
		t.Errorf("Write multibyte n = %d (want 6)", n)
	}
}

func TestWriteSeekPastEnd(t *testing.T) {
	t.Parallel()
	io := NewString("")
	io.Write("abc")
	io.Seek(6, SeekSet)
	io.Write("z")
	if io.String() != "abc\x00\x00\x00z" {
		t.Errorf("nul-pad = %q", io.String())
	}
}

func TestAppendMode(t *testing.T) {
	t.Parallel()
	io := must(t, "ab", "a")
	if io.Pos() != 0 {
		t.Errorf("append initial pos = %d (want 0)", io.Pos())
	}
	io.Write("cd")
	if io.String() != "abcd" || io.Pos() != 4 {
		t.Errorf("append = %q pos=%d", io.String(), io.Pos())
	}
	// append ignores SetPos for write location.
	io2 := must(t, "abcd", "a")
	io2.SetPos(1)
	io2.Write("Z")
	if io2.String() != "abcdZ" {
		t.Errorf("append ignores pos = %q", io2.String())
	}
}

func TestPutsPrintPrintf(t *testing.T) {
	t.Parallel()
	io := NewString("")
	io.Puts("x", "y")
	if io.String() != "x\ny\n" {
		t.Errorf("Puts = %q", io.String())
	}
	// Puts with no args writes a newline.
	io2 := NewString("")
	io2.Puts()
	if io2.String() != "\n" {
		t.Errorf("Puts() = %q", io2.String())
	}
	// Puts does not double a trailing newline.
	io3 := NewString("")
	io3.Puts("a\n")
	if io3.String() != "a\n" {
		t.Errorf("Puts trailing nl = %q", io3.String())
	}
	// Print.
	io4 := NewString("")
	io4.Print("a", "b")
	if io4.String() != "ab" {
		t.Errorf("Print = %q", io4.String())
	}
	// Printf.
	io5 := NewString("")
	io5.Printf("%03d", 7)
	if io5.String() != "007" {
		t.Errorf("Printf = %q", io5.String())
	}
}

func TestPutc(t *testing.T) {
	t.Parallel()
	io := NewString("")
	if b, _ := io.Putc('A'); b != 'A' {
		t.Errorf("Putc ret = %d", b)
	}
	if s, _ := io.PutString("Xyz"); s != "Xyz" {
		t.Errorf("PutString ret = %q", s)
	}
	if io.String() != "AX" {
		t.Errorf("putc string = %q", io.String())
	}
	// PutString with empty writes nothing.
	io2 := NewString("")
	io2.PutString("")
	if io2.String() != "" {
		t.Errorf("PutString empty = %q", io2.String())
	}
}

func TestPosSeek(t *testing.T) {
	t.Parallel()
	io := NewString("hello")
	if io.Pos() != 0 || io.Tell() != 0 {
		t.Error("initial pos/tell != 0")
	}
	if err := io.SetPos(2); err != nil {
		t.Fatal(err)
	}
	if d, _ := io.ReadAll(); d != "llo" {
		t.Errorf("after SetPos = %q", d)
	}
	// negative SetPos.
	if err := io.SetPos(-1); !errors.Is(err, ErrInvalidSeek) {
		t.Errorf("SetPos(-1) = %v", err)
	}
	// seek whences.
	io2 := NewString("hello")
	if r, _ := io2.Seek(1, SeekSet); r != 0 || io2.Pos() != 1 {
		t.Errorf("SeekSet pos=%d", io2.Pos())
	}
	io2.Seek(2, SeekCur)
	if io2.Pos() != 3 {
		t.Errorf("SeekCur pos=%d", io2.Pos())
	}
	io2.Seek(-1, SeekEnd)
	if io2.Pos() != 4 {
		t.Errorf("SeekEnd pos=%d", io2.Pos())
	}
	// negative result.
	if _, err := io2.Seek(-100, SeekSet); !errors.Is(err, ErrInvalidSeek) {
		t.Errorf("Seek neg = %v", err)
	}
	// invalid whence.
	if _, err := io2.Seek(0, 9); !errors.Is(err, ErrInvalidSeek) {
		t.Errorf("Seek bad whence = %v", err)
	}
}

func TestRewind(t *testing.T) {
	t.Parallel()
	io := NewString("a\nb\n")
	io.Gets("\n")
	if io.Lineno() != 1 {
		t.Errorf("lineno = %d", io.Lineno())
	}
	if r := io.Rewind(); r != 0 {
		t.Errorf("Rewind ret = %d", r)
	}
	if io.Pos() != 0 || io.Lineno() != 0 {
		t.Errorf("after Rewind pos=%d lineno=%d", io.Pos(), io.Lineno())
	}
}

func TestStringAndSet(t *testing.T) {
	t.Parallel()
	io := NewString("old")
	io.Read(1)
	io.SetString("new")
	if io.String() != "new" || io.Pos() != 0 {
		t.Errorf("SetString = %q pos=%d", io.String(), io.Pos())
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	io := NewString("abcdef")
	io.SetPos(4)
	if n, err := io.Truncate(2); n != 0 || err != nil {
		t.Errorf("Truncate = %d,%v", n, err)
	}
	if io.String() != "ab" || io.Pos() != 4 {
		t.Errorf("after truncate = %q pos=%d", io.String(), io.Pos())
	}
	// grow with nul-pad.
	io2 := NewString("ab")
	io2.Truncate(4)
	if io2.String() != "ab\x00\x00" {
		t.Errorf("truncate grow = %q", io2.String())
	}
	// truncate to same length.
	io3 := NewString("ab")
	io3.Truncate(2)
	if io3.String() != "ab" {
		t.Errorf("truncate same = %q", io3.String())
	}
	// negative.
	if _, err := io2.Truncate(-1); !errors.Is(err, ErrInvalidSeek) {
		t.Errorf("Truncate(-1) = %v", err)
	}
}

func TestSizeLengthEof(t *testing.T) {
	t.Parallel()
	io := NewString("hi")
	if io.Size() != 2 || io.Length() != 2 {
		t.Errorf("size/length = %d/%d", io.Size(), io.Length())
	}
	if e, _ := io.Eof(); e {
		t.Error("not at EOF yet")
	}
	io.ReadAll()
	if e, _ := io.Eof(); !e {
		t.Error("should be at EOF")
	}
}

func TestCloseRead(t *testing.T) {
	t.Parallel()
	io := NewString("ab")
	io.Close()
	// all the read paths return ErrClosed via checkReadable.
	if _, err := io.ReadAll(); !errors.Is(err, ErrClosed) {
		t.Errorf("ReadAll closed = %v", err)
	}
	if _, err := io.Eof(); !errors.Is(err, ErrClosed) {
		t.Errorf("Eof closed = %v", err)
	}
}

func TestFlush(t *testing.T) {
	t.Parallel()
	io := NewString("x")
	if io.Flush() != io {
		t.Error("Flush should return receiver")
	}
}

func TestLineno(t *testing.T) {
	t.Parallel()
	io := NewString("a\nb\nc\n")
	io.Gets("\n")
	if io.Lineno() != 1 {
		t.Errorf("lineno = %d", io.Lineno())
	}
	io.Gets("\n")
	if io.Lineno() != 2 {
		t.Errorf("lineno = %d", io.Lineno())
	}
	io.SetLineno(10)
	if io.Lineno() != 10 {
		t.Errorf("SetLineno = %d", io.Lineno())
	}
}
