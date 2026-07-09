# frozen_string_literal: true

require "stringio"

# StringIO is an in-memory IO whose backing store is a String buffer.

# Write into a fresh buffer, then read the accumulated string back.
out = StringIO.new
out.puts "hello"
out.print "world"
out.write "!"
puts out.string.inspect            # => "hello\nworld!"

# Read line by line over a seeded buffer; the cursor advances as you go.
input = StringIO.new("line1\nline2\nline3")
puts input.gets.inspect            # => "line1\n"
puts input.read(2).inspect         # => "li"
input.rewind
puts input.readlines.inspect       # => ["line1\n", "line2\n", "line3"]

# Random access: seek, read a slice, and query the cursor position.
data = StringIO.new("abcdef")
data.seek(2)
puts data.read(3).inspect          # => "cde"
puts data.pos.inspect              # => 5
puts data.eof?.inspect             # => false

# The append operator chains and returns the StringIO itself.
buf = StringIO.new
buf << "chain" << "ed"
puts buf.string.inspect            # => "chained"
