package editor

import (
	"errors"
	"strings"
)

// shellSyntax holds the characters that make an editor setting a shell
// command rather than a list of words.
const shellSyntax = "$`|&;<>()*?[#~\n"

// needsShell reports whether s uses shell syntax that splitWords cannot
// express, such as a variable, a pipe or a leading VAR=value assignment.
func needsShell(s string) bool {
	if strings.ContainsAny(s, shellSyntax) {
		return true
	}
	fields := strings.Fields(s)
	return len(fields) > 0 && strings.Contains(fields[0], "=")
}

// splitWords splits s into words. With posix it follows POSIX shell quoting:
// single quotes are literal, double quotes group and let a backslash escape
// only $, `, ", \ and newline, and a backslash outside quotes escapes the next
// character. Without posix, as on Windows, only double quotes group and a
// backslash is an ordinary character, so "C:\Program Files\x.exe" works.
func splitWords(s string, posix bool) ([]string, error) {
	var words []string
	var word strings.Builder
	inWord := false
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch quote {
		case '\'':
			if c == '\'' {
				quote = 0
			} else {
				word.WriteByte(c)
			}
			continue
		case '"':
			switch {
			case c == '"':
				quote = 0
			case c == '\\' && posix && i+1 < len(s) && strings.IndexByte("$`\"\\\n", s[i+1]) >= 0:
				i++
				if s[i] != '\n' {
					word.WriteByte(s[i])
				}
			default:
				word.WriteByte(c)
			}
			continue
		}
		switch {
		case c == ' ' || c == '\t' || c == '\n':
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		case c == '"' || (c == '\'' && posix):
			quote = c
			inWord = true
		case c == '\\' && posix && i+1 < len(s):
			i++
			if s[i] != '\n' {
				word.WriteByte(s[i])
				inWord = true
			}
		default:
			word.WriteByte(c)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote")
	}
	if inWord {
		words = append(words, word.String())
	}
	return words, nil
}
