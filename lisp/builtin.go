package lisp

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func InitGlobals() Env {
	globals := mkEnv(nil)
	globals.Set("SPACE", Atom{" "})
	globals.Set("NEWLINE", Atom{"\n"})
	globals.Set("TAB", Atom{"\t"})
	globals.Set("BANG", Atom{"!"})
	globals.Set("QMARK", Atom{"?"})
	globals.Set("PERIOD", Atom{"."})
	globals.Set("COMMA", Atom{","})
	globals.Set("COLON", Atom{":"})
	globals.Set("HASH", Atom{"#"})
	globals.Set("ATSIGN", Atom{"@"})
	globals.Set("CHECK", Atom{"✓"})
	return globals
}

// ReadLine reads a line from stdin "robustly".
func ReadLine() (string, error) {
	bio := bufio.NewReader(os.Stdin)
	// FIXME: don't discard hasMoreInLine
	line, _, err := bio.ReadLine()
	switch err {
	case nil:
		return string(line), nil
	default:
		return "", err
	}
}

// Builtin represents a function with a native (Go) implementation.
type Builtin struct {
	Name string
	Fn   func([]Sexpr, *Env) (Sexpr, error)
	// Fn must take at least this many arguments:
	FixedArity int
	// If true, fn can take more arguments:
	NAry      bool
	Docstring string
	ArgString string
	Examples  *ConsCell
}

func (b Builtin) String() string {
	return fmt.Sprintf("<builtin: %s>", b.Name)
}

// Eval for builtin returns itself.
func (b Builtin) Eval(e *Env) (Sexpr, error) {
	return b, nil
}

// Equal returns true if the receiver and the arg are both builtins and have the
// same name.
func (b Builtin) Equal(o Sexpr) bool {
	if o, ok := o.(Builtin); ok {
		return b.Name == o.Name
	}
	return false
}

func compareMultipleNums(cmp func(a, b Number) bool, args []Sexpr) (Sexpr, error) {
	if len(args) < 1 {
		return nil, baseError("missing argument")
	}
	first, ok := args[0].(Number)
	if !ok {
		return nil, baseError(fmt.Sprintf("'%s' is not a number", args[0]))
	}
	last := first
	for i := 1; i < len(args); i++ {
		num, ok := args[i].(Number)
		if !ok {
			return nil, baseError(fmt.Sprintf("'%s' is not a number", args[i]))
		}
		if !cmp(num, last) {
			return Nil, nil
		}
		last = num
	}
	return True, nil
}

func applyFn(args []Sexpr, env *Env) (Sexpr, error) {
	if len(args) < 2 {
		return nil, baseError("apply: not enough arguments")
	}
	l := len(args)
	var fnArgs []Sexpr
	// Support (apply f a b l) where l is a list and a, b are scalars:
	singleArgs := args[1 : l-1]
	c, ok := args[l-1].(*ConsCell)
	if !ok {
		return nil, baseError(fmt.Sprintf("'%s' is not a list", args[l-1]))
	}
	asCons, err := consToExprs(c)
	if err != nil {
		return nil, extendError("apply", err)
	}
	fnArgs = append(singleArgs, asCons...)

	// Note: what follows is very similar to the function evaluation
	// logic in eval(), but TCO (goto) there makes it hard to DRY out with
	// respect to what follows.

	evalCar := args[0]
	// User-defined functions:
	lambda, ok := evalCar.(*lambdaFn)
	if ok {
		newEnv := mkEnv(lambda.env)
		err := setLambdaArgsInEnv(&newEnv, lambda, fnArgs)
		if err != nil {
			return nil, extendError("apply", err)
		}
		var ret Sexpr = Nil
		bodyExpr := lambda.body
		for {
			if bodyExpr == Nil {
				return ret, nil
			}
			ret, err = eval(bodyExpr.car, &newEnv)
			if err != nil {
				return nil, extendError("apply", err)
			}
			bodyExpr = bodyExpr.cdr.(*ConsCell)
		}
	}
	// Built-in functions:
	builtin, ok := evalCar.(*Builtin)
	if !ok {
		return nil, baseError(fmt.Sprintf("%s is not a function", evalCar))
	}
	biResult, err := builtin.Fn(fnArgs, env)
	if err != nil {
		return nil, extendError("apply", err)
	}
	return biResult, nil
}

func LoadFile(e *Env, filename string) error {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	err = LexParseEval(string(bytes), e)
	if err != nil {
		return err
	}
	return nil
}

// moving `builtins` into `init` avoids initialization loop for doHelp:
var builtins map[string]*Builtin

func init() {
	A := func(s string) Atom {
		return Atom{s}
	}
	N := func(n int) Number {
		return Num(n)
	}
	L := func(args ...Sexpr) Sexpr {
		return mkListAsConsWithCdr(args, Nil)
	}
	E := func(args ...Sexpr) *ConsCell {
		return mkListAsConsWithCdr(args, Nil).(*ConsCell)
	}
	QL := func(args ...Sexpr) *ConsCell {
		return L(A("quote"), L(args...)).(*ConsCell)
	}
	QA := func(s string) *ConsCell {
		return L(A("quote"), A(s)).(*ConsCell)
	}
	builtins = map[string]*Builtin{
		"+": {
			Name:       "+",
			Docstring:  "Add 0 or more numbers",
			FixedArity: 0,
			NAry:       true,
			ArgString:  "(() . xs)",
			Examples: E(
				L(A("+"), N(1), N(2), N(3)),
				L(A("+")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) == 0 {
					return Num(0), nil
				}
				sum := Num(0)
				for _, arg := range args {
					n, ok := arg.(Number)
					if !ok {
						return nil, baseError(fmt.Sprintf("expected number, got '%s'", arg))
					}
					sum = sum.Add(n)
				}
				return sum, nil
			},
		},
		"-": {
			Name:       "-",
			Docstring:  "Subtract 0 or more numbers from the first argument",
			FixedArity: 1,
			NAry:       true,
			ArgString:  "(x . xs)",
			Examples: E(
				L(A("-"), N(1), N(1)),
				L(A("-"), N(5), N(2), N(1)),
				L(A("-"), N(99)),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) == 0 {
					return nil, baseError("missing argument")
				}
				sum, ok := args[0].(Number)
				if !ok {
					return nil, baseError(fmt.Sprintf("expected number, got '%s'", args[0]))
				}
				if len(args) == 1 {
					return args[0].(Number).Neg(), nil
				}
				for _, arg := range args[1:] {
					n, ok := arg.(Number)
					if !ok {
						return nil, baseError(fmt.Sprintf("expected number, got '%s'", arg))
					}
					sum = sum.Sub(n)
				}
				return sum, nil
			},
		},
		"*": {
			Name:       "*",
			Docstring:  "Multiply 0 or more numbers",
			FixedArity: 0,
			NAry:       true,
			ArgString:  "(() . xs)",
			Examples: E(
				L(A("*"), N(1), N(2), N(3)),
				L(A("*")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) == 0 {
					return Num(1), nil
				}
				prod := Num(1)
				for _, arg := range args {
					n, ok := arg.(Number)
					if !ok {
						return nil, baseError(fmt.Sprintf("expected number, got '%s'", arg))
					}
					prod = prod.Mul(n)
				}
				return prod, nil
			},
		},
		"/": {
			Name:       "/",
			Docstring:  "Divide the first argument by the rest",
			FixedArity: 2,
			NAry:       true,
			ArgString:  "(numerator denominator1 . more)",
			Examples: E(
				L(A("/"), N(1), N(2)),
				L(A("/"), N(12), N(2), N(3)),
				L(A("/"), N(1), N(0)),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) < 1 {
					return nil, baseError("missing argument")
				}
				quot, ok := args[0].(Number)
				if !ok {
					return nil, baseError(fmt.Sprintf("expected number, got '%s'", args[0]))
				}
				for _, arg := range args[1:] {
					if arg.Equal(Num(0)) {
						return nil, baseError("division by zero")
					}
					n, ok := arg.(Number)
					if !ok {
						return nil, baseError(fmt.Sprintf("expected number, got '%s'", arg))
					}
					quot = quot.Div(n)
				}
				return quot, nil
			},
		},
		"=": {
			Name:       "=",
			Docstring:  "Return t if the arguments are equal, () otherwise",
			FixedArity: 1,
			NAry:       true,
			ArgString:  "(x . xs)",
			Examples: E(
				L(A("="), N(1), N(1)),
				L(A("="), N(1), N(2)),
				L(A("apply"), A("="), L(A("repeat"), N(10), A("t"))),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) < 1 {
					return nil, baseError("missing argument")
				}
				for _, arg := range args[1:] {
					if !args[0].Equal(arg) {
						return Nil, nil
					}
				}
				return True, nil
			},
		},
		"rem": {
			Name:       "rem",
			Docstring:  "Return remainder when second arg divides first",
			FixedArity: 2,
			NAry:       false,
			ArgString:  "(x y)",
			Examples: E(
				L(A("rem"), N(5), N(2)),
				L(A("rem"), N(4), N(2)),
				L(A("rem"), N(1), N(0)),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 2 {
					return nil, baseError("rem requires two arguments")
				}
				n1, ok := args[0].(Number)
				if !ok {
					return nil, baseError(fmt.Sprintf("expected number, got '%s'", args[0]))
				}
				n2, ok := args[1].(Number)
				if !ok {
					return nil, baseError(fmt.Sprintf("expected number, got '%s'", args[1]))
				}
				if n2.Equal(Num(0)) {
					return nil, baseError("division by zero")
				}
				return n1.Rem(n2), nil
			},
		},
		"<": {
			Name:       "<",
			Docstring:  "Return t if the arguments are in strictly increasing order, () otherwise",
			FixedArity: 1,
			NAry:       true,
			ArgString:  "(x . xs)",
			Examples: E(
				L(A("<"), N(1), N(2)),
				L(A("<"), N(1), N(1)),
				L(A("<"), N(1)),
				L(A("apply"), A("<"), L(A("range"), N(100))),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				return compareMultipleNums(func(a, b Number) bool {
					return b.Less(a)
				}, args)
			},
		},
		"<=": {
			Name:       "<=",
			Docstring:  "Return t if the arguments are in increasing (or qual) order, () otherwise",
			FixedArity: 1,
			NAry:       true,
			ArgString:  "(x . xs)",
			Examples: E(
				L(A("<="), N(1), N(2)),
				L(A("<="), N(1), N(1)),
				L(A("<="), N(1)),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				return compareMultipleNums(func(a, b Number) bool {
					return b.LessEqual(a)
				}, args)
			},
		},
		">": {
			Name:       ">",
			Docstring:  "Return t if the arguments are in strictly decreasing order, () otherwise",
			FixedArity: 1,
			NAry:       true,
			ArgString:  "(x . xs)",
			Examples: E(
				L(A(">"), N(1), N(2)),
				L(A(">"), N(1), N(1)),
				L(A(">"), N(1)),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				return compareMultipleNums(func(a, b Number) bool {
					return b.Greater(a)
				}, args)
			},
		},
		">=": {
			Name:       ">=",
			Docstring:  "Return t if the arguments are in decreasing (or equal) order, () otherwise",
			FixedArity: 1,
			NAry:       true,
			ArgString:  "(x . xs)",
			Examples: E(
				L(A(">="), N(1), N(2)),
				L(A(">="), N(1), N(1)),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				return compareMultipleNums(func(a, b Number) bool {
					return b.GreaterEqual(a)
				}, args)
			},
		},
		"apply": {
			Name:       "apply",
			Docstring:  "Apply a function to a list of arguments",
			FixedArity: 2,
			NAry:       false,
			ArgString:  "(f args)",
			Examples: E(
				L(A("apply"), A("+"), L(A("repeat"), N(10), N(1))),
				L(A("apply"), A("*"), L(A("cdr"), L(A("range"), N(10)))),
			),
			Fn: applyFn,
		},
		"atom?": {
			Name:       "atom?",
			Docstring:  "Return t if the argument is an atom, () otherwise",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("atom?"), N(1)),
				L(A("atom?"), QA("one")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("atom? expects a single argument")
				}
				if _, ok := args[0].(Atom); ok {
					return True, nil
				}
				return Nil, nil
			},
		},
		"body": {
			Name:       "body",
			Docstring:  "Return the body of a lambda function",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(f)",
			Examples: E(
				L(A("body"), L(A("lambda"), L(A("x")), L(A("+"), A("x"), N(1)))),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("missing argument")
				}
				l, ok := args[0].(*lambdaFn)
				if !ok {
					return nil, baseErrorf("expected lambda function, got '%s'", args[0])
				}
				return l.body, nil
			},
		},
		"car": {
			Name:       "car",
			Docstring:  "Return the first element of a list",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("car"), QL(A("one"), A("two"))),
				L(A("car"), L()),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("missing argument")
				}
				carCons, ok := args[0].(*ConsCell)
				if !ok {
					return nil, baseErrorf("'%s' is not a list", args[0])
				}
				if carCons == Nil {
					return Nil, nil
				}
				return carCons.car, nil
			},
		},
		"cdr": {
			Name:       "cdr",
			Docstring:  "Return a list with the first element removed",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("cdr"), QL(A("one"), A("two"))),
				L(A("cdr"), L()),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("missing argument")
				}
				cdrCons, ok := args[0].(*ConsCell)
				if !ok {
					return nil, baseErrorf("'%s' is not a list", args[0])
				}
				if cdrCons == Nil {
					return Nil, nil
				}
				return cdrCons.cdr, nil
			},
		},
		"cons": {
			Name:       "cons",
			Docstring:  "Add an element to the front of a (possibly empty) list",
			FixedArity: 2,
			NAry:       false,
			ArgString:  "(x xs)",
			Examples: E(
				L(A("cons"), N(1), QL(A("one"), A("two"))),
				L(A("cons"), N(1), L()),
				L(A("cons"), N(1), N(2)),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 2 {
					return nil, baseError("missing argument")
				}
				return Cons(args[0], args[1]), nil
			},
		},
		"doc": {
			Name:       "doc",
			Docstring:  "Return the doclist for a function",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("doc"), L(A("lambda"), L(A("x")),
					L(A("doc"), L(A("does"), A("stuff")),
						L(A("and"), A("other"), A("stuff"))),
					L(A("+"), A("x"), N(1)))),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("missing argument")
				}
				switch t := args[0].(type) {
				case *lambdaFn:
					return t.doc, nil
				case *Builtin:
					return stringsToList(strings.Split(builtins[t.Name].Docstring, " ")...), nil
				default:
					return nil, baseErrorf("'%s' is not a function", args[0])
				}
			},
		},
		"downcase": {
			Name:       "downcase",
			Docstring:  "Return a new atom with all characters in lower case",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("downcase"), QA("Hello")),
				L(A("downcase"), QA("HELLO")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("downcase requires one argument")
				}
				a, ok := args[0].(Atom)
				if !ok {
					return nil, baseErrorf("expected atom, got '%s'", args[0])
				}
				return Atom{strings.ToLower(a.s)}, nil
			},
		},
		"eval": {
			Name:       "eval",
			Docstring:  "Evaluate an expression",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("eval"), QL(A("one"), A("two"))),
				L(A("eval"), QL(A("+"), N(1), N(2))),
			),
			Fn: func(args []Sexpr, e *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("missing argument")
				}
				return eval(args[0], e)
			},
		},
		"exit": {
			Name:       "exit",
			Docstring:  "Exit the program",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				os.Exit(0)
				return nil, nil
			},
		},
		"forms": {
			Name:       "forms",
			Docstring:  "Return available operators, as a list",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Fn: func(args []Sexpr, e *Env) (Sexpr, error) {
				return mkListAsConsWithCdr(formsAsSexprList(e), Nil), nil
			},
		},
		"fuse": {
			Name:       "fuse",
			Docstring:  "Fuse a list of numbers or atoms into a single atom",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("fuse"), QL(A("A"), A("B"), A("C"))),
				L(A("fuse"), L(A("reverse"), L(A("range"), N(10)))),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("fuse expects a single argument")
				}
				if args[0] == Nil {
					return Nil, nil
				}
				switch s := args[0].(type) {
				case *ConsCell:
					cons := s
					var str string
					for cons != nil {
						this := cons.car.String()
						str += this
						if cons.cdr == nil {
							break
						}
						cons = cons.cdr.(*ConsCell)
					}
					// if first rune is a digit, return a Number
					firstRune, _ := utf8.DecodeRuneInString(str)
					if unicode.IsDigit(firstRune) {
						return Num(str), nil
					}
					return Atom{str}, nil
				default:
					return nil, baseError("fuse expects a list")
				}
			},
		},
		"gensym": {
			Name:       "gensym",
			Docstring:  "Return a new symbol",
			FixedArity: 0,
			NAry:       true,
			ArgString:  "(() . x)",
			Fn: func(args []Sexpr, e *Env) (Sexpr, error) {
				if len(args) != 0 && len(args) != 1 {
					return nil, baseError("gensym expects 0 or 1 arguments")
				}
				if len(args) == 0 {
					return Atom{gensym("")}, nil
				}
				prefix, ok := args[0].(Atom)
				if !ok {
					return nil, baseError("gensym expects an atom as its first argument")
				}
				return Atom{gensym("-" + prefix.s)}, nil
			},
		},
		"help": {
			Name:       "help",
			Docstring:  "Print this message",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Fn: func(args []Sexpr, e *Env) (Sexpr, error) {
				fmt.Println(ShortDocStr(e))
				return Nil, nil
			},
		},
		"isqrt": {
			Name:       "isqrt",
			Docstring:  "Integer square root",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("isqrt"), N(4)),
				L(A("isqrt"), N(5)),
				// Breaks on several platforms!
				// L(A("isqrt"), N(9139571243709)),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("isqrt expects a single argument")
				}
				n, ok := args[0].(Number)
				if !ok {
					return nil, baseError("isqrt expects a number")
				}
				sqrt := n.bi.Sqrt(&n.bi)
				return Num(sqrt.String()), nil
			},
		},
		"len": {
			Name:       "len",
			Docstring:  "Return the length of a list",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("len"), L(A("range"), N(10))),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("len expects a single argument")
				}
				list, ok := args[0].(*ConsCell)
				if !ok {
					return nil, baseErrorf("'%s' is not a list", args[0])
				}
				count := 0
				for list != nil {
					count++
					list = list.cdr.(*ConsCell)
				}
				return Num(count), nil
			},
		},
		"list": {
			Name:       "list",
			Docstring:  "Return a list of the given arguments",
			FixedArity: 0,
			NAry:       true,
			ArgString:  "(() . xs)",
			Examples: E(
				L(A("list"), N(1), N(2), N(3)),
				L(A("list")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				return mkListAsConsWithCdr(args, Nil), nil
			},
		},
		"list?": {
			Name:       "list?",
			Docstring:  "Return t if the argument is a list, () otherwise",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("list?"), L(A("range"), N(10))),
				L(A("list?"), N(1)),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("list? expects a single argument")
				}
				if _, ok := args[0].(*ConsCell); ok {
					return True, nil
				}
				return Nil, nil
			},
		},
		"load": {
			Name:       "load",
			Docstring:  "Load and execute a file",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(filename)",
			Fn: func(args []Sexpr, e *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("load expects a single argument")
				}
				filename, ok := args[0].(Atom)
				if !ok {
					return nil, baseError("load expects a filename")
				}
				err := LoadFile(e, filename.s)
				if err != nil {
					return nil, extendError("load file", err)
				}
				return Nil, nil
			},
		},
		"macroexpand-1": {
			Name:       "macroexpand-1",
			Docstring:  "Expand a macro",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("macroexpand-1"), QL(A("+"), A("x"), N(1))),
				L(A("macroexpand-1"), QL(A("if"), L(), N(1), N(2))),
			),
			Fn: func(args []Sexpr, e *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("macroexpand-1 expects a single argument")
				}
				return macroexpand1(args[0], e)
			},
		},
		"not": {
			Name:       "not",
			Docstring:  "Return t if the argument is nil, () otherwise",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("not"), L()),
				L(A("not"), A("t")),
				L(A("not"), L(A("range"), N(10))),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("not expects a single argument")
				}
				if args[0] == Nil {
					return True, nil
				}
				return Nil, nil
			},
		},
		"number?": {
			Name:       "number?",
			Docstring:  "Return true if the argument is a number, else ()",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("number?"), N(1)),
				L(A("number?"), A("t")),
				L(A("number?"), A("+")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("number? expects a single argument")
				}
				_, ok := args[0].(Number)
				if ok {
					return True, nil
				}
				return Nil, nil
			},
		},
		"print": {
			Name:       "print",
			Docstring:  "Print the arguments",
			FixedArity: 0,
			NAry:       true,
			ArgString:  "(() . xs)",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				strArgs := []string{}
				for _, arg := range args {
					strArgs = append(strArgs, arg.String())
				}
				fmt.Print(strings.Join(strArgs, " "))
				return Nil, nil
			},
		},
		"println": {
			Name:       "println",
			Docstring:  "Print the arguments and a newline",
			FixedArity: 0,
			NAry:       true,
			ArgString:  "(() . xs)",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				strArgs := []string{}
				for _, arg := range args {
					strArgs = append(strArgs, arg.String())
				}
				fmt.Println(strings.Join(strArgs, " "))
				return Nil, nil
			},
		},
		"printl": {
			Name:       "printl",
			Docstring:  "Print a list argument, without parentheses",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("missing argument")
				}
				list, ok := args[0].(*ConsCell)
				if !ok {
					return nil, baseErrorf("expected list, got '%s'", args[0])
				}
				fmt.Println(unwrapList(list))
				return Nil, nil
			},
		},
		"randint": {
			Name:       "randint",
			Docstring:  "Return a random integer between 0 and the argument minus 1",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("randint expects a single argument")
				}
				num, ok := args[0].(Number)
				if !ok {
					return nil, baseErrorf("'%s' is not a number", args[0])
				}
				if num.Equal(N(0)) {
					return nil, baseError("randint expects a non-zero argument")
				}
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				return Num(r.Intn(int(num.bi.Uint64()))), nil
			},
		},
		"readlist": {
			Name:       "readlist",
			Docstring:  "Read a list from stdin",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				line, err := ReadLine()
				if err != nil {
					return nil, extendError("reading readlist input", err)
				}
				parsed, err := lexAndParse(strings.Split(line, "\n"))
				if err != nil {
					return nil, extendError("parsing readlist input", err)
				}
				return mkListAsConsWithCdr(parsed, Nil), nil
			},
		},
		"screen-start": {
			Name:       "screen-start",
			Docstring:  "Start screen for text UIs",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				err := termStart()
				if err != nil {
					return nil, extendError("starting screen", err)
				}
				return Nil, nil
			},
		},
		"screen-end": {
			Name:       "screen-end",
			Docstring:  "Stop screen for text UIs, return to console mode",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				err := termEnd()
				if err != nil {
					return nil, extendError("stopping screen", err)
				}
				return Nil, nil
			},
		},
		"screen-size": {
			Name:       "screen-size",
			Docstring:  "Return the screen size (width, height)",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				width, height, err := termSize()
				if err != nil {
					return nil, extendError("screen-size termSize", err)
				}
				return Cons(Num(width), Cons(Num(height), Nil)), nil
			},
		},
		"screen-clear": {
			Name:       "screen-clear",
			Docstring:  "Clear the screen",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				err := termClear()
				if err != nil {
					return nil, extendError("screen-clear termClear", err)
				}
				return Nil, nil
			},
		},
		"screen-get-key": {
			Name:       "screen-get-key",
			Docstring:  "Return a keystroke as an atom",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 0 {
					return nil, baseError("getkey expects no arguments")
				}
				key, err := termGetKey()
				if err != nil {
					return nil, extendError("screen-get-key termGetKey", err)
				}
				return Atom{key}, nil
			},
		},
		"screen-write": {
			Name:       "screen-write",
			Docstring:  "Write a string to the screen",
			FixedArity: 3,
			NAry:       false,
			ArgString:  "(x y list)",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 3 {
					return nil, baseError("screen-write expects 3 arguments")
				}
				x, ok := args[0].(Number)
				if !ok {
					return nil, baseErrorf("'%s' is not a number", args[0])
				}
				y, ok := args[1].(Number)
				if !ok {
					return nil, baseErrorf("'%s' is not a number", args[1])
				}
				s, ok := args[2].(*ConsCell)
				if !ok {
					return nil, baseErrorf("'%s' is not a list", args[2])
				}
				err := termDrawText(int(x.bi.Uint64()), int(y.bi.Uint64()), unwrapList(s))
				if err != nil {
					return nil, extendError("screen-write termDrawText", err)
				}
				return Nil, nil
			},
		},
		"shell": {
			Name:       "shell",
			Docstring:  "Run a shell subprocess, and return stdout, stderr, and exit code",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(cmd)",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("shell expects a single argument")
				}
				return doShell(args[0])
			},
		},
		"shuffle": {
			Name:       "shuffle",
			Docstring:  "Return a (quickly!) shuffled list",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(xs)",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("shuffle expects a single argument")
				}
				l, ok := args[0].(*ConsCell)
				if !ok {
					return nil, baseErrorf("'%s' is not a list", args[0])
				}
				exprs, err := consToExprs(l)
				if err != nil {
					return nil, extendError("shuffle consToExprs", err)
				}
				rand.Shuffle(len(exprs), func(i, j int) {
					exprs[i], exprs[j] = exprs[j], exprs[i]
				})
				return mkListAsConsWithCdr(exprs, Nil), nil
			},
		},
		"sleep": {
			Name:       "sleep",
			Docstring:  "Sleep for the given number of milliseconds",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(ms)",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("sleep expects a single argument")
				}
				num, ok := args[0].(Number)
				if !ok {
					return nil, baseErrorf("'%s' is not a number", args[0])
				}
				time.Sleep(time.Duration(num.bi.Uint64()) * time.Millisecond)
				return Nil, nil
			},
		},
		"sort": {
			Name:       "sort",
			Docstring:  "Sort a list",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(xs)",
			Examples: E(
				L(A("sort"), QL(N(3), N(2), N(1))),
				L(A("sort"), QL()),
				L(A("sort"), QL(A("c"), A("b"), A("a"))),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("sort expects a single argument")
				}
				l, ok := args[0].(*ConsCell)
				if !ok {
					return nil, baseErrorf("'%s' is not a list", args[0])
				}
				exprs, err := consToExprs(l)
				if err != nil {
					return nil, extendError("sort consToExprs", err)
				}
				if len(exprs) == 0 {
					return Nil, nil
				}
				for i := 1; i < len(exprs); i++ {
					if reflect.TypeOf(exprs[i]) != reflect.TypeOf(exprs[0]) {
						return nil, baseErrorf("%s is not same type as %s", exprs[i], exprs[0])
					}
				}
				switch exprs[0].(type) {
				case Number:
					sort.Slice(exprs, func(i, j int) bool {
						return exprs[i].(Number).Less(exprs[j].(Number))
					})
				case Atom:
					sort.Slice(exprs, func(i, j int) bool {
						return exprs[i].(Atom).s < exprs[j].(Atom).s
					})
				default:
					return nil, baseErrorf("'%s' is not a sortable type", exprs[0])
				}
				return mkListAsConsWithCdr(exprs, Nil), nil
			},
		},
		"sort-by": {
			Name:       "sort-by",
			Docstring:  "Sort a list by a function",
			FixedArity: 2,
			NAry:       false,
			ArgString:  "(f xs)",
			Examples: E(
				L(A("sort-by"), A("first"), QL(L(N(3)), L(N(2)), L(N(1)))),
				L(A("sort-by"), A("first"), QL()),
				L(A("sort-by"), A("second"), QL(L(A("quux"), N(333)), L(A("zip"), N(222)), L(A("afar"), N(111)))),
			),
			Fn: func(args []Sexpr, e *Env) (Sexpr, error) {
				if len(args) != 2 {
					return nil, baseError("sort-by expects two arguments")
				}
				l, ok := args[1].(*ConsCell)
				if !ok {
					return nil, baseErrorf("'%s' is not a list", args[1])
				}
				exprs, err := consToExprs(l)
				if err != nil {
					return nil, extendError("sort consToExprs", err)
				}
				if len(exprs) == 0 {
					return Nil, nil
				}
				var sortHadErr error = nil
				sort.Slice(exprs, func(i, j int) bool {
					apply1, err := applyFn([]Sexpr{args[0], list(exprs[i])}, e)
					if err != nil {
						sortHadErr = err
						return false
					}
					apply2, err := applyFn([]Sexpr{args[0], list(exprs[j])}, e)
					if err != nil {
						sortHadErr = err
						return false
					}
					if reflect.TypeOf(apply1) != reflect.TypeOf(apply2) {
						sortHadErr = baseErrorf("apply result %s is not same type as %s",
							apply1, apply2)
						return false
					}
					switch apply1.(type) {
					case Number:
						return apply1.(Number).Less(apply2.(Number))
					case Atom:
						return apply1.(Atom).s < apply2.(Atom).s
					default:
						sortHadErr = baseErrorf("'%s' is not a sortable type", apply1)
					}
					return false
				})
				return mkListAsConsWithCdr(exprs, Nil), sortHadErr
			},
		},
		"source": {
			Name:       "source",
			Docstring:  "Show source for a function",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(f)",
			Examples: E(
				L(A("source"), A("map")),
				L(A("source"), A("+")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("source expects a single argument")
				}
				switch t := args[0].(type) {
				case *Builtin:
					return nil, baseErrorf("cannot get source of builtin function %s", t)
				case *lambdaFn:
					var argsCdr Sexpr = Nil
					if t.restArg != "" {
						argsCdr = Atom{t.restArg}
					}
					argExprs := stringsToExprs(t.args...)
					return Cons(Atom{"lambda"}, Cons(mkListAsConsWithCdr(argExprs, argsCdr), t.body)), nil
				default:
					return nil, baseErrorf("'%s' is not a function", args[0])
				}
			},
		},
		"split": {
			Name:       "split",
			Docstring:  "Split an atom or number into a list of single-digit numbers or single-character atoms",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("split"), N(123)),
				L(A("split"), QA("abc")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("split expects a single argument")
				}
				switch s := args[0].(type) {
				case Atom:
					return listOfChars(s.String()), nil
				case Number:
					return listOfNums(s.String())
				default:
					return nil, baseError("split expects an atom or a number")
				}
			},
		},
		"test": {
			Name:       "test",
			Docstring:  "Establish a testing block (return last expression)",
			FixedArity: 0,
			NAry:       true,
			ArgString:  "(() . exprs)",
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) == 0 {
					return Nil, nil
				}
				fmt.Printf("TEST %s ", args[0].String())
				for range args[1:] {
					fmt.Print(".")
				}
				fmt.Println("✓")
				return args[len(args)-1], nil
			},
		},
		"upcase": {
			Name:       "upcase",
			Docstring:  "Return the uppercase version of the given atom",
			FixedArity: 1,
			NAry:       false,
			ArgString:  "(x)",
			Examples: E(
				L(A("upcase"), QA("abc")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				if len(args) != 1 {
					return nil, baseError("upcase expects a single argument")
				}
				a, ok := args[0].(Atom)
				if !ok {
					return nil, baseError("upcase expects an atom")
				}
				return Atom{strings.ToUpper(a.s)}, nil
			},
		},
		"version": {
			Name:       "version",
			Docstring:  "Return the version of the interpreter",
			FixedArity: 0,
			NAry:       false,
			ArgString:  "()",
			Examples: E(
				L(A("version")),
			),
			Fn: func(args []Sexpr, _ *Env) (Sexpr, error) {
				versionSexprs := semverAsExprs(Version)
				return mkListAsConsWithCdr(versionSexprs, Nil), nil
			},
		},
	}
}

// listOfChars returns a list of single-character atoms from another, presumably
// longer atom; used by `split`
func listOfChars(s string) *ConsCell {
	if len(s) == 0 {
		return nil
	}
	r, size := utf8.DecodeRuneInString(s)
	return Cons(Atom{string(r)}, listOfChars(s[size:]))
}

// listOfNums returns a list of single-digit numbers from another, presumably
// longer number; used by `split`; if the input represents a negative number,
// the first digit is negative:
func listOfNums(s string) (*ConsCell, error) {
	if len(s) == 0 {
		return nil, nil
	}
	if s[0] == '-' {
		if len(s) < 2 {
			return nil, baseError("unexpected end of input")
		}
		lon, err := listOfNums(s[2:])
		if err != nil {
			return nil, err
		}
		return Cons(Num(s[0:2]), lon), nil
	}
	lon, err := listOfNums(s[1:])
	if err != nil {
		return nil, err
	}
	return Cons(Num(s[0:1]), lon), nil
}

func semverAsExprs(semver string) []Sexpr {
	reg := regexp.MustCompile(`(?:^v)?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-(dirty))?`)
	matches := reg.FindStringSubmatch(semver)
	if len(matches) == 0 {
		return nil
	}
	list := []Sexpr{}
	for _, m := range matches[1:] {
		if len(m) == 0 {
			continue
		}
		if m == "dirty" {
			list = append(list, Atom{"dirty"})
		} else {
			list = append(list, Num(m))
		}
	}
	return list
}