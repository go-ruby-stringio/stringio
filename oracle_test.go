// Copyright (c) the go-ruby-stringio/stringio authors
//
// SPDX-License-Identifier: BSD-3-Clause

package stringio

import (
	"os/exec"
	"strings"
	"testing"
)

// rubyBin locates a usable `ruby` once. The oracle tests skip themselves when it
// is absent (the qemu cross-arch lanes and the Windows lane), so the deterministic
// suite alone drives the 100% gate there.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI oracle")
	}
	return path
}

// rubyEval runs a StringIO script under MRI and returns its stdout. The script
// binmodes $stdout/$stdin so Windows text-mode never rewrites the bytes (the
// go-ruby-erb lesson), and gates itself on Ruby >= 4.0 — the StringIO semantics
// pinned here are MRI 4.0.5's — printing "SKIP" on an older ruby so the caller
// can ignore it rather than diff against stale behaviour.
func rubyEval(t *testing.T, bin, script string) string {
	t.Helper()
	preamble := "$stdout.binmode\n$stdin.binmode\nrequire 'stringio'\n" +
		"if RUBY_VERSION < '4.0'\n  print 'SKIP'\n  exit\nend\n"
	cmd := exec.Command(bin, "-e", preamble+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return string(out)
}

// TestOracleAgainstMRI runs a corpus of StringIO programs in MRI and reproduces
// each here, asserting the two agree byte-for-byte. Each case is a Ruby snippet
// (printing its result) paired with the equivalent Go computation.
func TestOracleAgainstMRI(t *testing.T) {
	bin := rubyBin(t)

	cases := []struct {
		name string
		ruby string
		got  func() string
	}{
		{
			name: "read_chunks",
			ruby: `s = StringIO.new("hello"); print [s.read(3), s.read, s.read(1)].inspect`,
			got: func() string {
				s := NewString("hello")
				d1, _, _ := s.Read(3)
				d2, _ := s.ReadAll()
				d3, ok, _ := s.Read(1)
				return inspectArr(quote(d1), quote(d2), nilOr(ok, quote(d3)))
			},
		},
		{
			name: "gets_lines",
			ruby: `s = StringIO.new("a\nb\nc"); print [s.gets, s.gets, s.gets, s.gets].inspect`,
			got: func() string {
				s := NewString("a\nb\nc")
				var parts []string
				for i := 0; i < 4; i++ {
					l, ok, _ := s.Gets("\n")
					parts = append(parts, nilOr(ok, quote(l)))
				}
				return inspectArr(parts...)
			},
		},
		{
			name: "gets_sep",
			ruby: `s = StringIO.new("a;b"); print [s.gets(";"), s.gets(";")].inspect`,
			got: func() string {
				s := NewString("a;b")
				l1, _, _ := s.Gets(";")
				l2, _, _ := s.Gets(";")
				return inspectArr(quote(l1), quote(l2))
			},
		},
		{
			name: "gets_paragraph",
			ruby: `s = StringIO.new("a\n\n\nb\nc\n\nd"); print [s.gets(""), s.gets(""), s.gets("")].inspect`,
			got: func() string {
				s := NewString("a\n\n\nb\nc\n\nd")
				var parts []string
				for i := 0; i < 3; i++ {
					l, _, _ := s.Gets("")
					parts = append(parts, quote(l))
				}
				return inspectArr(parts...)
			},
		},
		{
			name: "gets_limit",
			ruby: `s = StringIO.new("hello world"); print [s.gets(3), s.gets("o", 4)].inspect`,
			got: func() string {
				s := NewString("hello world")
				l1, _, _ := s.GetsLimit("\n", 3)
				l2, _, _ := s.GetsLimit("o", 4)
				return inspectArr(quote(l1), quote(l2))
			},
		},
		{
			name: "getc_utf8",
			ruby: `s = StringIO.new("héllo"); print [s.getc, s.getc].inspect`,
			got: func() string {
				s := NewString("héllo")
				c1, _, _ := s.Getc()
				c2, _, _ := s.Getc()
				return inspectArr(quote(c1), quote(c2))
			},
		},
		{
			name: "write_seek_nulpad",
			ruby: `s = StringIO.new; s.write("abc"); s.seek(6); s.write("z"); print s.string.inspect`,
			got: func() string {
				s := NewString("")
				s.Write("abc")
				s.Seek(6, SeekSet)
				s.Write("z")
				return quote(s.String())
			},
		},
		{
			name: "append_mode",
			ruby: `s = StringIO.new("ab", "a"); s.pos = 1; s.write("Z"); print [s.string, s.pos].inspect`,
			got: func() string {
				s := must(t, "ab", "a")
				s.SetPos(1)
				s.Write("Z")
				return inspectArr(quote(s.String()), itoa(s.Pos()))
			},
		},
		{
			name: "puts_print_printf",
			ruby: `s = StringIO.new; s.puts("x","y"); s.print("p"); s.printf("%03d", 7); print s.string.inspect`,
			got: func() string {
				s := NewString("")
				s.Puts("x", "y")
				s.Print("p")
				s.Printf("%03d", 7)
				return quote(s.String())
			},
		},
		{
			name: "seek_whences",
			ruby: `s = StringIO.new("hello"); s.seek(1); a=[s.pos]; s.seek(2,1); a<<s.pos; s.seek(-1,2); a<<s.pos; print a.inspect`,
			got: func() string {
				s := NewString("hello")
				s.Seek(1, SeekSet)
				a := []string{itoa(s.Pos())}
				s.Seek(2, SeekCur)
				a = append(a, itoa(s.Pos()))
				s.Seek(-1, SeekEnd)
				a = append(a, itoa(s.Pos()))
				return inspectArr(a...)
			},
		},
		{
			name: "truncate",
			ruby: `s = StringIO.new("abcdef"); s.pos = 4; s.truncate(2); print [s.string, s.pos].inspect`,
			got: func() string {
				s := NewString("abcdef")
				s.SetPos(4)
				s.Truncate(2)
				return inspectArr(quote(s.String()), itoa(s.Pos()))
			},
		},
		{
			name: "lineno",
			ruby: `s = StringIO.new("a\nb\nc\n"); s.gets; a=[s.lineno]; s.gets; a<<s.lineno; print a.inspect`,
			got: func() string {
				s := NewString("a\nb\nc\n")
				s.Gets("\n")
				a := []string{itoa(s.Lineno())}
				s.Gets("\n")
				a = append(a, itoa(s.Lineno()))
				return inspectArr(a...)
			},
		},
		{
			name: "ungetc",
			ruby: `s = StringIO.new("abc"); c = s.getc; s.ungetc("X"); print [s.string, s.read].inspect`,
			got: func() string {
				s := NewString("abc")
				s.Getc()
				s.Ungetc("X")
				d, _ := s.ReadAll()
				return inspectArr(quote(s.String()), quote(d))
			},
		},
		{
			name: "each_byte",
			ruby: `s = StringIO.new("AB"); r=[]; s.each_byte { |b| r << b }; print r.inspect`,
			got: func() string {
				s := NewString("AB")
				var r []string
				s.EachByte(func(b byte) error { r = append(r, itoa(int(b))); return nil })
				return inspectArr(r...)
			},
		},
		{
			name: "size_eof",
			ruby: `s = StringIO.new("hi"); print [s.size, s.eof?, s.read, s.eof?].inspect`,
			got: func() string {
				s := NewString("hi")
				e1, _ := s.Eof()
				d, _ := s.ReadAll()
				e2, _ := s.Eof()
				return inspectArr(itoa(s.Size()), boolStr(e1), quote(d), boolStr(e2))
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := rubyEval(t, bin, c.ruby)
			if want == "SKIP" {
				t.Skipf("ruby %s too old for the StringIO 4.0 semantics", rubyVersion(t, bin))
			}
			if got := c.got(); got != want {
				t.Errorf("MRI=%q go=%q\nscript:\n%s", want, got, c.ruby)
			}
		})
	}
}

// rubyVersion returns the RUBY_VERSION of bin, for a clearer skip message.
func rubyVersion(t *testing.T, bin string) string {
	t.Helper()
	out, err := exec.Command(bin, "-e", "print RUBY_VERSION").Output()
	if err != nil {
		return "unknown"
	}
	return string(out)
}

// The helpers below render Go values in Ruby's `p` / Array#inspect form so the
// oracle can compare a Go-built string against MRI's printed output directly.

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\t':
			b.WriteString("\\t")
		case '\r':
			b.WriteString("\\r")
		case 0:
			b.WriteString("\\u0000") // MRI inspect renders a NUL byte as \u0000
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func inspectArr(elems ...string) string { return "[" + strings.Join(elems, ", ") + "]" }

func nilOr(ok bool, s string) string {
	if !ok {
		return "nil"
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	if neg {
		return "-" + string(d)
	}
	return string(d)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
