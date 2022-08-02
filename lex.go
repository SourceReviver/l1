package main

import (
	"fmt"
	"strings"

	"github.com/eigenhombre/lexutil"
)

// Use with lexutil.go (which should eventually be its own package).

// Lexemes:
const (
	itemNumber lexutil.ItemType = iota
	itemAtom
	itemLeftParen
	itemRightParen
	itemError
)

// Human-readable versions of above:
var typeMap = map[lexutil.ItemType]string{
	itemNumber:     "NUM",
	itemAtom:       "ATOM",
	itemLeftParen:  "LP",
	itemRightParen: "RP",
	itemError:      "ERR",
}

// LexRepr returns a string representation of a known lexeme.
func LexRepr(i lexutil.LexItem) string {
	switch i.Typ {
	case itemNumber:
		return fmt.Sprintf("%s(%s)", typeMap[i.Typ], i.Val)
	case itemAtom:
		return fmt.Sprintf("%s(%s)", typeMap[i.Typ], i.Val)
	case itemLeftParen:
		return "LP"
	case itemRightParen:
		return "RP"
	case itemError:
		return fmt.Sprintf("%s(%s)", typeMap[i.Typ], i.Val)
	default:
		panic("bad item type")
	}
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isSpace(r rune) bool {
	return strings.ContainsRune(" \t\n\r", r)
}

func ignoreComment(l *lexutil.Lexer) {
	for {
		if r := l.Next(); r == '\n' || r == lexutil.EOF {
			return
		}
	}
}

var validAtomChars = ("0123456789abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"+*/-_!=<>?[]{}&$^")

func isAtomChar(r rune) bool {
	return strings.ContainsRune(validAtomChars, r)
}

func lexStart(l *lexutil.Lexer) lexutil.StateFn {
	for {
		switch r := l.Next(); {
		case isSpace(r):
			l.Ignore()
		case r == ';':
			ignoreComment(l)
		case r == lexutil.EOF:
			return nil
		case isDigit(r) || r == '-' || r == '+':
			l.Backup()
			return lexInt
		case r == '(':
			l.Emit(itemLeftParen)
		case r == ')':
			l.Emit(itemRightParen)
		case isAtomChar(r):
			return lexAtom
		default:
			l.Errorf("unexpected character %q in input", itemError, r)
		}
	}
}

func lexAtom(l *lexutil.Lexer) lexutil.StateFn {
	l.AcceptRun(validAtomChars)
	l.Emit(itemAtom)
	return lexStart
}

func lexInt(l *lexutil.Lexer) lexutil.StateFn {
	l.Accept("-+")
	nextRune := l.Peek()
	if isDigit(nextRune) {
		l.AcceptRun("0123456789")
		l.Emit(itemNumber)
		return lexStart
	}
	return lexAtom
}

func lexItems(s string) []lexutil.LexItem {
	l := lexutil.Lex("main", s, lexStart)
	ret := []lexutil.LexItem{}
	for tok := range l.Items {
		ret = append(ret, tok)
	}
	return ret
}

func isBalanced(tokens []lexutil.LexItem) bool {
	level := 0
	for _, token := range tokens {
		switch token.Typ {
		case itemLeftParen:
			level++
		case itemRightParen:
			level--
		}
	}
	return level == 0
}
