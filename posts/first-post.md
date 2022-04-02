---
title: "TIL: Creating a custom bufio.Scanner in Go"
date: 2022-04-02
tags:
  - tag
  - another
  - til
  - golang

layout: layouts/post.njk
---
The Go module [`bufio`](https://pkg.go.dev/bufio) provides a struct and methods `Scanner` that allow for easy iteration over input from an `io.Reader`.

Built in functions of type `type SplitFunc func(data []byte, atEOF bool) (advance int, token []byte, err error)` include those that allow tokenizing on words, lines, runes, and bytes. You can also write your own `SplitFunc` and pass it as an argument to a scanner instance with `scanner.Split(split SplitFunc)` to tokenize on another condition.

For example, in a current project to implement a [STOMP messaging protocol server](https://stomp.github.io/stomp-specification-1.2.html) I need to tokenize input based on null bytes, so I wrote this `SplitFunc`:

```go
func ScanNullTerm(data []byte, atEOF bool) (int, []byte, error) {
	// if we're at EOF, we're done for now
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexByte(data, '\000'); i >= 0 {
		// there is a null-terminated frame
		return i + 1, data[0:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}
```

Which turns `Alpha^@Beta^@Gamma\nDelta\Theta` into the tokens `Alpha`, `Beta`, and `Gamma\nDelta\nTheta`.
