# l1

<img src="/l1.jpg" width="400">

![build](https://github.com/eigenhombre/l1/actions/workflows/build.yml/badge.svg)

First attempt at a simple Lisp in Go.

WIP toy Lisp for fun projects.  Currently lexing and parsing works, eval is next:

        11:57:20 l1 39.3F     ≡ * ☐ ~ (master) >  ./l1
        > (QUOTE (LAMBDA (X) (+ X X)))

        LEXEMES_____________
        LP('(')
        ATOM('QUOTE')
        LP('(')
        ATOM('LAMBDA')
        LP('(')
        ATOM('X')
        RP(')')
        LP('(')
        ATOM('+')
        ATOM('X')
        ATOM('X')
        RP(')')
        RP(')')
        RP(')')

        PARSED ITEMS________
        [(QUOTE (LAMBDA (X) (+ X X)))]
        > ^C
        11:57:31 l1 39.3F     ≡ * ☐ ~ (master) >

Take a look at the tests and the rest of the code for a sense of what's there so far.

# L1 features (planned)

- [Lisp 1](https://en.wikipedia.org/wiki/Common_Lisp#The_function_namespace);
- Sufficient power to [implement itself](http://www.paulgraham.com/rootsoflisp.html);
- Implement math as bignums from the get-go;
- Curses-based terminal control for text games, command line utilities, ...;

# Goals

- Improve my Go skills;
- Build a small, fast-loading Lisp that I can extend how I like;
- Learn more about [Lisp as a model for computation](http://www.paulgraham.com/rootsoflisp.html).

# Usage

Check out this repo and `cd` to it. Then,

- `go test` and maybe 
- `go build` followed by
- `go install`; then
- `l1`
