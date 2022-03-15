package main

import (
	"strings"
	"testing"
)

func TestEval(t *testing.T) {
	type evalCase struct {
		in  string
		out string
		err string
	}
	S := func(in, out, err string) evalCase {
		return evalCase{in, out, err}
	}
	Cases := func(cases ...evalCase) []evalCase {
		return cases
	}
	OK := "" // No error reported
	var tests = []struct {
		evalCases []evalCase
	}{
		{Cases(S("t", "t", OK))},
		{Cases(S("()", "()", OK))},
		{Cases(S("(cons t ())", "(t)", OK))},
		{Cases(S("(cons (quote hello) (quote (world)))", "(hello world)", OK))},
		{Cases(S("(quote foo)", "foo", OK))},
		{Cases(S("(quote (the (ten (laws (of (greenspun))))))", "(the (ten (laws (of (greenspun)))))", OK))},
		{Cases(S("(cdr (quote (is not common lisp)))", "(not common lisp)", OK))},
		{Cases(S("(car (quote (is not common lisp)))", "is", OK))},
		{Cases(S("(cond (() 1) (2 3))", "3", OK))},
		{Cases(S("1", "1", OK))},
		{Cases(S("-5", "-5", OK))},
		{Cases(S("(* 12349807213490872130987 12349807213490872130987)", "152517738210391179737088822267441718485594169", OK))},
		{Cases(S("(+)", "0", OK))},
		{Cases(S("(+ 1 1 2 3)", "7", OK))},
		{Cases(S("(+ 1 1)", "2", OK))},
		{Cases(S("(eq (quote foo) (quote foo))", "t", OK))},
		{Cases(S("(eq (quote foo) (quote bar))", "()", OK))},
		{Cases(S("(eq (quote foo) (quote (foo bar)))", "()", OK))},
		{Cases(S("(atom (quote foo))", "t", OK))},
		{Cases(S("(atom (quote (foo bar)))", "()", OK))},
		{Cases(S("(+ 1)", "1", OK))},
		{Cases(S("(+ -1)", "-1", OK))},
		{Cases(S("(+ 0)", "0", OK))},
		{Cases(S("(+ 1 2 3 4 5 6 7 8 9 10)", "55", OK))},
		{Cases(S("(+ 999999999999999 1)", "1000000000000000", OK))},
		{Cases(S("(+ 1 999999999999999)", "1000000000000000", OK))},
		{Cases(S("(+ (+ 1))", "1", OK))},
		{Cases(S("(+ (+ 1 2 3) 4 5 6)", "21", OK))},
		{Cases(S("(- 1)", "-1", OK))},
		{Cases(S("(- 1 1)", "0", OK))},
		{Cases(S("31489071430987532109487513094875031984750983147", "31489071430987532109487513094875031984750983147", OK))},
		{Cases(S("(- 12349807213490872130987 12349807213490872130987)", "0", OK))},
		{Cases(S("(- (+ 1 2 3) 4 5 6)", "-9", OK))},
		{Cases(S("(*)", "1", OK))},
		{Cases(S("(* 1 1)", "1", OK))},
		{Cases(S("(* 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20)", "2432902008176640000", OK))},
		{Cases(S("(/ 1 1)", "1", OK))},
		// Future test cases w/ errors:
		// (/ 1 0)
		// (/)
		// (-)
		// (cons)
		{Cases(S("(/ 4 2)", "2", OK))},
		{Cases(S("(/ 1 2)", "0", OK))},
		{Cases(S("(* 1 1 1 (*) (*) (*))", "1", OK))},
		{Cases(S("(+ 1 1 1 (+) (+) (+))", "3", OK))},
		{Cases(S("(quote 3)", "3", OK))},
		{Cases(S("(quote (1 2 3))", "(1 2 3)", OK))},
		{Cases(S("(quote ())", "()", OK))},
		{Cases(S("(quote (((1 2 3))))", "(((1 2 3)))", OK))},
		{Cases(S("(car (quote (1 2 3)))", "1", OK))},
		{Cases(S("(cdr (quote (1 2 3)))", "(2 3)", OK))},
		{Cases(S("(cons 1 (quote (2 3 4)))", "(1 2 3 4)", OK))},
		{Cases(S("(cons (cons 1 (cons 2 ())) (quote (3 4)))", "((1 2) 3 4)", OK))},
		{Cases(S("(quote ((1)))", "((1))", OK))},
		{Cases(S("(quote (()))", "(())", OK))},
		{Cases(S("(quote (() ()))", "(() ())", OK))},
		{Cases(S("(cons () ())", "(())", OK))},
		{Cases(S("(eq (quote (foo bar)) (quote (foo bar)))", "()", OK))},
		{Cases(S("(cond)", "()", OK))},
		{Cases(S("(cond (() 3))", "()", OK))},
		{Cases(S("(cond (3 3))", "3", OK))},
		{Cases(S("(cond (() 3) (() 4))", "()", OK))},
		{Cases(S("(cond (t 3) (t 4))", "3", OK))},
		// Whitespace cases
		{Cases(S(" t ", "t", OK))},
		{Cases(S("t\n", "t", OK))},
		{Cases(S("\n\nt\n", "t", OK))},
		// Multiple statements
		{Cases(
			S("1", "1", OK),
			S("2", "2", OK),
			S("3", "3", OK))},
		// Global scope
		{Cases(
			S("(def a 3)", "3", OK),
			S("a", "3", OK))},
		{Cases(
			S("(def a 3)", "3", OK),
			S("(def a 4)", "4", OK),
			S("a", "4", OK))},
		{Cases(
			S("(def a 6)", "6", OK),
			S("(def b 7)", "7", OK),
			S("(+ a b)", "13", OK))},
		{Cases(
			S("(def l (quote (1 2 3)))", "(1 2 3)", OK),
			S("l", "(1 2 3)", OK))},
		{Cases(
			S("(def a (+ 1 1))", "2", OK),
			S("(def b (+ a a))", "4", OK),
			S("b", "4", OK))},
	}
	for _, test := range tests {
		e := env{}
		for _, testCase := range test.evalCases {
			got, err := lexAndParse(testCase.in)
			if len(got) != 1 {
				t.Errorf("%s: got %d results, want 1", testCase.in, len(got))
				continue
			}
			if err != nil && !strings.Contains(err.Error(), testCase.err) {
				t.Errorf("%s: got error %q, want %q", testCase.in, err, testCase.err)
				continue
			}
			result := got[0].Eval(&e).String()
			if result != testCase.out {
				t.Errorf("%s: got %q, want %q", testCase.in, result, testCase.out)
				continue
			}
			t.Logf("%s -> %q", testCase.in, result)
		}
	}
}
