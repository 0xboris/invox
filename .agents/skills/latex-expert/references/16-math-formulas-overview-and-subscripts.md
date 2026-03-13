# 16 Math formulas: Overview and subscripts

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 16.1 Subscripts & superscripts

## 16 Math formulas
Produce mathematical text by putting LaTeX into math mode or display math mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). This example shows both.
```
The wave equation for \( u \) is
\begin{displaymath}
  \frac{\partial^2u}{\partial t^2} = c^2\nabla^2u
\end{displaymath}
where \( \nabla^2 \) is the spatial Laplacian and \( c \) is constant.

```

Math mode is for inline mathematics. In the above example it is invoked by the starting `\(` and finished by the matching ending `\)`. Display math mode is for displayed equations and here is invoked by the `displaymath` environment. Note that any mathematical text whatever, including mathematical text consisting of just one character, is handled in math mode.
When in math mode or display math mode, LaTeX handles many aspects of your input text differently than in other text modes. For example,
```
contrast x+y with \( x+y \)

```

in math mode the letters are in italics and the spacing around the plus sign is different.
There are three ways to make inline formulas, to put LaTeX in math mode.
```
\( mathematical material \)
$ mathematical material $
\begin{math} mathematical material \end{math}

```

The first form is preferred and the second is quite common, but the third form is rarely used. You can sometimes use one and sometimes another, as in `\(x\) and $y$`. You can use these in paragraph mode or in LR mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)).
To make displayed formulas, put LaTeX into display math mode with either:
```
\begin{displaymath}
  mathematical material
\end{displaymath}

```

or
```
\begin{equation}
  mathematical material
\end{equation}

```

(see [`displaymath`](https://latexref.xyz/dev/latex2e.html#displaymath), see [`equation`](https://latexref.xyz/dev/latex2e.html#equation)). The only difference is that with the `equation` environment, LaTeX puts a formula number alongside the formula. The construct `\[ math \]` is equivalent to `\begin{displaymath} math \end{displaymath}`. These environments can only be used in paragraph mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)).
The American Mathematical Society has made freely available a set of packages that greatly expand your options for writing mathematics, `amsmath` and `amssymb` (also be aware of the `mathtools` package that is an extension to, and loads, `amsmath`). New documents that will have mathematical text should use these packages. Descriptions of these packages is outside the scope of this document; see their documentation on CTAN.
  * [Subscripts & superscripts](https://latexref.xyz/dev/latex2e.html#Subscripts-_0026-superscripts)
  * [Math symbols](https://latexref.xyz/dev/latex2e.html#Math-symbols)
  * [Math functions](https://latexref.xyz/dev/latex2e.html#Math-functions)
  * [Math accents](https://latexref.xyz/dev/latex2e.html#Math-accents)
  * [Over- or under math](https://latexref.xyz/dev/latex2e.html#Over_002d-or-under-math)
  * [Spacing in math mode](https://latexref.xyz/dev/latex2e.html#Spacing-in-math-mode)
  * [Math styles](https://latexref.xyz/dev/latex2e.html#Math-styles)
  * [Math miscellany](https://latexref.xyz/dev/latex2e.html#Math-miscellany)

### 16.1 Subscripts & superscripts
Synopsis (in math mode or display math mode), one of:
```
base^exp
base^{exp}

```

or, one of:
```
base_exp
base_{exp}

```

Make exp appear as a superscript of base (with the caret character, `^`) or a subscript (with underscore, `_`).
In this example the `0`’s and `1`’s are subscripts while the `2`’s are superscripts.
```
\( (x_0+x_1)^2 \leq (x_0)^2+(x_1)^2 \)

```

To have the subscript or superscript contain more than one character, surround the expression with curly braces, as in `e^{-2x}`. This example’s fourth line shows curly braces used to group an expression for the exponent.
```
\begin{displaymath}
  (3^3)^3=27^3=19\,683
  \qquad
  3^{(3^3)}=3^{27}=7\,625\,597\,484\,987
\end{displaymath}

```

LaTeX knows how to handle a superscript on a superscript, or a subscript on a subscript, or supers on subs, or subs on supers. So, expressions such as `e^{x^2}` and `x_{i_0}` give correct output. Note the use in those expressions of curly braces to give the base a determined exp. If you enter `\(3^3^3\)`, this interpreted as `\(3^{3}^{3}\)` and then you get TeX error ‘Double superscript’.
LaTeX does the right thing when something has both a subscript and a superscript. In this example the integral has both. They come out in the correct place without any author intervention.
```
\begin{displaymath}
  \int_{x=a}^b f'(x)\,dx = f(b)-f(a)
\end{displaymath}

```

Note the curly braces around `x=a` to make the entire expression a subscript.
To put a superscript or subscript before a symbol, use a construct like `{}_t K^2`. The empty curly braces `{}` give the subscript something to attach to and keeps it from accidentally attaching to a prior symbols.
Using the subscript or superscript character outside of math mode or display math mode, as in `the expression x^2`, will get you the TeX error ‘Missing $ inserted’.
A common reason to want subscripts outside of a mathematics mode is to typeset chemical formulas. There are packages for that, such as `mhchem`; see CTAN.
