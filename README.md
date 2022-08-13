# l1

<img src="/l1.jpg" width="400">

![build](https://github.com/eigenhombre/l1/actions/workflows/build.yml/badge.svg)

`l1` is a small interpreted [Lisp
1](https://en.wikipedia.org/wiki/Common_Lisp#The_function_namespace)
written in Go.  Emphasizing simplicity of data types (atoms,
arbitrarily large integers, and lists) and start-up speed, it aims to
be a playground for future language experiments.

`l1` eschews strings in favor of using atoms and lists in the style of
[some classic AI programs](https://github.com/norvig/paip-lisp).  It
features tail-call optimization and a few unique functions for
converting atoms and numbers to lists, and vice-versa.

# Usage / Examples

You should have Go installed and configured.  At some later point, pre-built
artifacts for various architectures may be available here.

## Installing

    go install github.com/eigenhombre/l1@latest

Specific versions are tagged and available as well.  See [the tags
page](https://github.com/eigenhombre/l1/tags) for available versions
and then, for example,

    go install github.com/eigenhombre/l1@v0.0.8

## Usage

To execute a file:

    $ l1 <file.l1>

Example (using a file in this project):

    $ cat examples/fact.l1
    ;; Return the factorial of `n`:
    (defn fact (n)
      (cond ((= 0 n) 1)
            (t (* n (fact (- n 1)))))))

    (print (fact 100))
    $ time l1 examples/fact.l1
    933262154439441526816992388562667004907159682643816214685929638
    952175999932299156089414639761565182862536979208272237582511852
    10916864000000000000000000000000

    real	0m0.008s
    user	0m0.004s
    sys	0m0.004s

See the `examples` directory for more sample `l1` files.

## Example REPL Session

<!-- The following examples are autogenerated, do not change by hand! -->
<!-- BEGIN EXAMPLES -->

    $ l1
    > (quote foo)
    foo
    > 'foo
    foo
    > '123
    123
    > (quote (the (ten (laws (of (greenspun))))))
    (the (ten (laws (of (greenspun)))))
    > ((lambda (x . xs) (list x xs)) 1 2 3 4)
    (1 (2 3 4))
    > (help)
    Builtins and Special Forms:
          Name  Arity    Description
             *    0+     Multiply 0 or more numbers
             +    0+     Add 0 or more numbers
             -    1+     Subtract 0 or more numbers from the first argument
             /    2+     Divide the first argument by the rest
             <    1+     Return t if the arguments are in strictly increasing order, () otherwise
            <=    1+     Return t if the arguments are in increasing (or qual) order, () otherwise
             =    1+     Return t if the arguments are equal, () otherwise
             >    1+     Return t if the arguments are in strictly decreasing order, () otherwise
            >=    1+     Return t if the arguments are in decreasing (or equal) order, () otherwise
         apply    2      Apply a function to a list of arguments
          atom    1      Return true if the argument is an atom, false otherwise
          bang    1      Return a new atom with exclamation point added
    capitalize    1      Return a new atom with the first character capitalized
           car    1      Return the first element of a list
           cdr    1      Return a list with the first element removed
         comma    1      Return a new atom with a comma at the end
          cond    0+     SPECIAL FORM: Conditional branching
          cons    2      Add an element to the front of a (possibly empty) list
           def    2      SPECIAL FORM: Set a value
          defn    2+     SPECIAL FORM: Create and name a function
      downcase    1      Return a new atom with all characters in lower case
        errors    1+     SPECIAL FORM: Error checking (for tests)
          fuse    1      Fuse a list of numbers or atoms into a single atom
          help    0      Print this message
            is    1      Assert that the argument is truthy (not ())
        lambda    1+     SPECIAL FORM: Create a function
           len    1      Return the length of a list
           let    1+     SPECIAL FORM: Create a local scope
          list    0+     Return a list of the given arguments
          neg?    1      Return true if the (numeric) argument is negative, else ()
           not    1      Return t if the argument is nil, () otherwise
        period    1      Return a new atom with a period added to the end
          pos?    1      Return true if the (numeric) argument is positive, else ()
         print    0+     Print the arguments
        printl    1      Print a list argument, without parentheses
         quote    1      SPECIAL FORM: Quote an expression
     randalpha    1      Return a list of random (English/Latin) alphabetic characters
    randchoice    1      Return a random element from a list
     randigits    1      Return a list of random digits of the given length
           rem    2      Return remainder when second arg divides first
         split    0      Split an atom or number into a list of single-digit numbers or single-character atoms
          test    0+     Establish a testing block (return last expression)
        upcase    1      Return the uppercase version of the given atom
       version    0      Return the version of the interpreter
         zero?    1      Return t if the argument is zero, () otherwise
    > ^D
    $
<!-- END EXAMPLES -->

Many of the [unit tests](https://github.com/eigenhombre/l1/blob/master/tests.l1) are written in `l1` itself.  Here are a few examples:

```
(test '(split and fuse)
  (is (= '(1) (split 1)))
  (is (= '(-1) (split -1)))
  (is (= '(-3 2 1) (split -321)))
  (is (= '(a) (split (quote a))))
  (is (= '(g r e e n s p u n) (split 'greenspun)))
  (is (= '(8 3 8 1 0 2 0 5 0) (split (* 12345 67890))))
  (is (= 15 (len (split (* 99999 99999 99999)))))
  (errors '(expects a single argument)
    (split))
  (errors '(expects a single argument)
    (split 1 1))
  (errors '(expects an atom or a number)
    (split '(a b c)))

  (is (= '() (fuse ())))
  (is (= 'a (fuse (quote (a)))))
  (is (= 'aa (fuse (quote (aa)))))
  (is (= 'ab (fuse (quote (a b)))))
  (is (= 1 (fuse (quote (1)))))
  (is (= 12 (fuse (quote (1 2)))))
  (is (= 125 (+ 2 (fuse (quote (1 2 3))))))
  (is (= 1295807125987 (fuse (split 1295807125987))))
  (errors '(expects a single argument)
    (fuse)))

(test '(factorial)
  (defn fact (n)
    (cond ((zero? n) 1)
          (t (* n (fact (- n 1))))))
  (is (= 30414093201713378043612608166064768844377641568960512000000000000
         (fact 50)))
  (is (= 2568 (len (split (fact 1000))))))
```

# Local Development

Check out this repo and `cd` to it. Then,

    go test
    go build
    go install

Extra testing and build infrastructure for this project relies on
GitHub Actions, Docker, and Make.  Please look at the `Dockerfile` and
`Makefile` for more information.

New releases are made using `make release`.  You must commit all
outstanding changes first.

# Emacs Integration

If you are using Emacs, you can set it up to work with `l1` as an "inferior
lisp" process as described in [the Emacs manual](https://www.gnu.org/software/emacs/manual/html_node/emacs/External-Lisp.html).
I currently derive a new major mode from the base `lisp-mode` and bind a few
keys for convenience as follows:

    (define-derived-mode l1-mode
      lisp-mode "L1 Mode"
      "Major mode for L1 Lisp code"
      (setq inferior-lisp-program (executable-find "l1")
      (paredit-mode 1)
      (define-key l1-mode-map (kbd "s-i") 'lisp-eval-last-sexp)
      (define-key l1-mode-map (kbd "s-I") 'lisp-eval-form-and-next)
      (define-key l1-mode-map (kbd "C-o j") 'run-lisp))

    (add-to-list 'auto-mode-alist '("\\.l1" . l1-mode))

If `l1` has been installed on your path, `M-x run-lisp` or using the appropriate
keybinding should be enough to start a REPL within Emacs and start sending
expressions to it.

# Goals

- Learn more about Lisp as a model for computation by building a Lisp with sufficient power to [implement itself](http://www.paulgraham.com/rootsoflisp.html);
- Improve my Go skills;
- Build a small, fast-loading Lisp that I can extend how I like;
- Possibly implement Curses-based terminal control for text games, command line utilities, ...;

# Non-goals

- Backwards compatibility with any existing, popular Lisp.
- Stability (for now) -- everything is subject to change.

# Resources / Further Reading

- [Structure and Interpretation of Computer Programs](https://mitpress.mit.edu/sites/default/files/sicp/index.html).  Classic MIT
  text, presents several Lisp evaluation models, written in Scheme.
- [Crafting Interpreters](https://craftinginterpreters.com/) book / website.  Stunning, thorough,
  approachable and beautiful book on building a language in Java and
  in C.
- Donovan & Kernighan, [The Go Programming Language](https://www.amazon.com/Programming-Language-Addison-Wesley-Professional-Computing/dp/0134190440). Great Go reference.
- Rob Pike, [Lexical Scanning in Go](https://www.youtube.com/watch?v=HxaD_trXwRE) (YouTube).  I took the code described in this talk and spun it out into [its own package](https://github.com/eigenhombre/lexutil/) for reuse in `l1`.
- A [more detailed blog post](http://johnj.com/posts/l1/) on `l1`.
- A [blog post on adding Tail Call Optimization](http://johnj.com/posts/tco/) to `l1`.

# License

Copyright © 2022, John Jacobsen. MIT License.

# Disclaimer

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
