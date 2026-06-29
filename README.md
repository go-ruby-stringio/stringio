<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-stringio/brand/main/social/go-ruby-stringio-stringio.png" alt="go-ruby-stringio/stringio" width="720"></p>

# stringio — go-ruby-stringio

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-stringio.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's
[StringIO](https://docs.ruby-lang.org/en/master/StringIO.html)** — an in-memory IO
whose backing store is a String buffer. It reproduces MRI 4.0.5's StringIO
semantics exactly: a read/write cursor over a byte buffer, mode-gated access, gets
line splitting, seek-past-end NUL padding, and the EOFError / IOError / ArgumentError
raises MRI emits — **without any Ruby runtime**.

It is the StringIO backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime — a sibling
of [go-ruby-yaml](https://github.com/go-ruby-yaml/yaml) (the Psych port),
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) (the Onigmo engine), and
[go-ruby-erb](https://github.com/go-ruby-erb/erb) (the ERB compiler).

> **What it is — and isn't.** An in-memory IO over a String buffer is pure compute:
> the cursor arithmetic, the mode gating, the line/character/byte iteration, and the
> NUL-padding extension on a seek-past-end write are all deterministic and need **no
> interpreter**, so they live here as pure Go. Binding the type into a live Ruby
> object model — wiring `$stdout = StringIO.new` or routing `Kernel#puts` through it
> — is the host's job; this library hands back an idiomatic Go `StringIO` whose
> typed errors the host maps onto Ruby's exception classes.

## Features

Faithful port of StringIO's behaviour, validated against the `ruby` binary on every
supported platform:

- **Modes** — `r` / `w` / `a` and the `r+` / `w+` / `a+` read-write variants, with
  `w`/`w+` truncating the seed string and append mode writing at the end regardless
  of the cursor.
- **Reading** — byte-oriented `read(n)` / `read` (returning `nil` for a length read
  at EOF), `gets` with a separator, a byte limit, and **paragraph mode** (`gets("")`),
  `readline` / `readlines` / `each_line`, character-oriented `getc` / `each_char` /
  `readchar`, and byte-oriented `getbyte` / `each_byte` / `readbyte`.
- **Writing** — `write` / `<<`, `puts` / `print` / `printf`, `putc`, and seek-past-end
  writes that extend and NUL-pad the buffer like MRI.
- **Positioning** — `pos` / `pos=` / `tell`, `seek` (SEEK_SET / CUR / END), `rewind`,
  with a negative position raising `Errno::EINVAL`.
- **Content** — `string` / `string=`, `truncate` (shrink or grow with NUL-pad), `size`
  / `length`, `eof?`, `close` / `closed?`, `flush`, `lineno` / `lineno=`, and `ungetc`
  / `ungetbyte` (prepending at the start of the buffer).
- **Exact MRI raises** — read on a write-only stream / write on a read-only stream /
  any op on a closed stream raise `IOError`; `readline` / `readchar` / `readbyte` at
  EOF raise `EOFError`; a negative `read` length raises `ArgumentError`.

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x) and the three host OSes (Linux, macOS, Windows).

## Install

```sh
go get github.com/go-ruby-stringio/stringio
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-stringio/stringio"
)

func main() {
	// Write into an in-memory buffer.
	w := stringio.NewString("")
	w.Puts("hello", "world")
	fmt.Printf("%q\n", w.String()) // "hello\nworld\n"

	// Read it back line by line.
	r := stringio.NewString(w.String())
	for {
		line, ok, _ := r.Gets("\n")
		if !ok {
			break
		}
		fmt.Printf("%q\n", line) // "hello\n" then "world\n"
	}

	// Seek-past-end writes NUL-pad, exactly like MRI.
	g := stringio.NewString("")
	g.Write("abc")
	g.Seek(6, stringio.SeekSet)
	g.Write("z")
	fmt.Printf("%q\n", g.String()) // "abc\x00\x00\x00z"
}
```

## API

A `StringIO` is constructed with `New(s, mode)` or `NewString(s)` (the read-write
default). Every method maps to its Ruby counterpart; methods that Ruby implements as
queries return a value, and the ones that can raise return a typed error a host maps
onto Ruby's exception classes.

```go
func New(s, mode string) (*StringIO, error) // StringIO.new(s, mode)
func NewString(s string) *StringIO          // StringIO.new(s) — read-write

// Reading.
func (s *StringIO) Read(n int) (data string, ok bool, err error) // ok=false ⇒ MRI nil
func (s *StringIO) ReadAll() (string, error)
func (s *StringIO) Gets(sep string) (line string, ok bool, err error)
func (s *StringIO) GetsLimit(sep string, limit int) (string, bool, error)
func (s *StringIO) ReadLine(sep string) (string, error)   // EOFError at end
func (s *StringIO) ReadLines(sep string) ([]string, error)
func (s *StringIO) Each(sep string, fn func(line string) error) error
func (s *StringIO) Getc() (ch string, ok bool, err error)
func (s *StringIO) ReadChar() (string, error)             // EOFError at end
func (s *StringIO) Getbyte() (b byte, ok bool, err error)
func (s *StringIO) ReadByte() (byte, error)               // EOFError at end
func (s *StringIO) EachChar(fn func(ch string) error) error
func (s *StringIO) EachByte(fn func(b byte) error) error
func (s *StringIO) Ungetc(ch string) error
func (s *StringIO) Ungetbyte(b byte) error

// Writing.
func (s *StringIO) Write(str string) (int, error)
func (s *StringIO) Puts(args ...string) error
func (s *StringIO) Print(args ...string) error
func (s *StringIO) Printf(format string, args ...any) error
func (s *StringIO) Putc(c byte) (byte, error)
func (s *StringIO) PutString(str string) (string, error)  // IO#putc with a String

// Positioning, content, state.
func (s *StringIO) Pos() int
func (s *StringIO) Tell() int
func (s *StringIO) SetPos(n int) error
func (s *StringIO) Seek(off, whence int) (int, error)     // SeekSet / SeekCur / SeekEnd
func (s *StringIO) Rewind() int
func (s *StringIO) String() string
func (s *StringIO) SetString(str string)
func (s *StringIO) Truncate(n int) (int, error)
func (s *StringIO) Size() int
func (s *StringIO) Length() int
func (s *StringIO) Eof() (bool, error)
func (s *StringIO) Close()
func (s *StringIO) Closed() bool
func (s *StringIO) Flush() *StringIO
func (s *StringIO) Lineno() int
func (s *StringIO) SetLineno(n int)
```

### Errors

The typed errors below model the exceptions MRI's StringIO raises; a host maps each
onto its Ruby counterpart when binding the type into an interpreter.

| Go error            | Ruby exception                          |
| ------------------- | --------------------------------------- |
| `ErrClosed`         | `IOError`, "closed stream"              |
| `ErrNotReadable`    | `IOError`, "not opened for reading"     |
| `ErrNotWritable`    | `IOError`, "not opened for writing"     |
| `ErrEOF`            | `EOFError`, "end of file reached"       |
| `ErrNegativeLength` | `ArgumentError`, "negative length"      |
| `ErrInvalidSeek`    | `Errno::EINVAL`, "Invalid argument"     |

## Tests & coverage

The suite pairs deterministic, ruby-free tests (which alone hold coverage at 100%,
so the qemu cross-arch and Windows lanes pass the gate) with a **differential MRI
oracle**: a corpus of StringIO programs is run by the system `ruby` and reproduced
here, asserting the two agree byte-for-byte. The oracle scripts `$stdout.binmode`
(and binmode stdin) so Windows text-mode never rewrites the bytes, gate themselves
on `RUBY_VERSION >= "4.0"`, and skip where `ruby` is absent.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-stringio/stringio authors.
