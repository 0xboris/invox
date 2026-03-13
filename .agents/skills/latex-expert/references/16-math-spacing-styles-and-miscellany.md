# 16 Math formulas: Spacing, styles, and miscellany

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 16.6 Spacing in math mode
- 16.7 Math styles
- 16.8 Math miscellany

### 16.6 Spacing in math mode
When typesetting mathematics, LaTeX puts in spacing according to the normal rules for mathematics texts. If you enter `y=m x` then LaTeX ignores the space and in the output the m is next to the x, as _y=mx_.
But LaTeX’s rules occasionally need tweaking. For example, in an integral the tradition is to put a small extra space between the `f(x)` and the `dx`, here done with the `\,` command:
```
\int_0^1 f(x)\,dx

```

LaTeX provides the commands that follow for use in math mode. Many of these spacing definitions are expressed in terms of the math unit _mu_. It is defined as 1/18em, where the em is taken from the current math symbols family (see [Units of length](https://latexref.xyz/dev/latex2e.html#Units-of-length)). Thus, a `\thickspace` is something like 5/18 times the width of a ‘M’.

`\;`

Synonym: `\thickspace`. Normally `5.0mu plus 5.0mu`. With the `amsmath` package, or as of the 2020-10-01 LaTeX release, can be used in text mode as well as math mode; otherwise, in math mode only.

`\negthickspace`

Normally `-5.0mu plus 2.0mu minus 4.0mu`. With the `amsmath` package, or as of the 2020-10-01 LaTeX release, can be used in text mode as well as math mode; otherwise, in math mode only.

`\:`

`\>`

Synonym: `\medspace`. Normally `4.0mu plus 2.0mu minus 4.0mu`. With the `amsmath` package, or as of the 2020-10-01 LaTeX release, can be used in text mode as well as math mode; before that, in math mode only.

`\negmedspace`

Normally `-4.0mu plus 2.0mu minus 4.0mu`. With the `amsmath` package, or as of the 2020-10-01 LaTeX release, can be used in text mode as well as math mode; before that, in math mode only.

`\,`

Synonym: `\thinspace`. Normally `3mu`, which is 1/6em. Can be used in both math mode and text mode (see [`\thinspace` & `\negthinspace`](https://latexref.xyz/dev/latex2e.html#g_t_005cthinspace-_0026-_005cnegthinspace)).
This space is widely used, for instance between the function and the infinitesimal in an integral `\int f(x)\,dx` and, if an author does this, before punctuation in a displayed equation.
```
The antiderivative is
\begin{equation}
  3x^{-1/2}+3^{1/2}\,.
\end{equation}

```

`\!`

Synonym: `\negthinspace`. A negative thin space. Normally `-3mu`. With the `amsmath` package, or as of the 2020-10-01 LaTeX release, can be used in text mode as well as math mode; otherwise, the `\!` command is math mode only but the `\negthinspace` command has always also worked in text mode (see [`\thinspace` & `\negthinspace`](https://latexref.xyz/dev/latex2e.html#g_t_005cthinspace-_0026-_005cnegthinspace)).

`\quad`

This is 18mu, that is, 1em. This is often used for space surrounding equations or expressions, for instance for the space between two equations inside a `displaymath` environment. It is available in both text and math mode.

`\qquad`

A length of 2 quads, that is, 36mu = 2em. It is available in both text and math mode.
  * [`\smash`](https://latexref.xyz/dev/latex2e.html#g_t_005csmash)
  * [`\phantom` & `\vphantom` & `\hphantom`](https://latexref.xyz/dev/latex2e.html#g_t_005cphantom-_0026-_005cvphantom-_0026-_005chphantom)
  * [`\mathstrut`](https://latexref.xyz/dev/latex2e.html#g_t_005cmathstrut)

#### 16.6.1 `\smash`
Synopsis:
```
\smash{subformula}

```

Typeset subformula as if its height and depth were zero.
In this example the exponential is so tall that without the `\smash` command LaTeX would separate its line from the line above it, and the uneven line spacing might be unsightly.
```
To compute the tetration $\smash{2^{2^{2^2}}}$,
evaluate from the top down, as $2^{2^4}=2^{16}=65536$.

```

(Because of the `\smash` the printed expression could run into the line above so you may want to wait until the final version of the document to make such adjustments.)
This pictures the effect of `\smash` by using `\fbox` to surround the box that LaTeX will put on the line. The `\blackbar` command makes a bar extending from 10 points below the baseline to 20 points above.
```
\newcommand{\blackbar}{\rule[-10pt]{5pt}{30pt}}
\fbox{\blackbar}
\fbox{\smash{\blackbar}}

```

The first box that LaTeX places is 20 points high and 10 points deep. But the second box is treated by LaTeX as having zero height and zero depth, despite that the ink printed on the page still extends well above and below the line.
The `\smash` command appears often in mathematics to adjust the size of an element that surrounds a subformula. Here the first radical extends below the baseline while the second lies just on the baseline.
```
\begin{equation}
\sqrt{\sum_{0\leq k< n} f(k)}
\sqrt{\vphantom{\sum}\smash{\sum_{0\leq k< n}} f(k)}
\end{equation}

```

Note the use of `\vphantom` to give the `\sqrt` command an argument with the height of the `\sum` (see [`\phantom` & `\vphantom` & `\hphantom`](https://latexref.xyz/dev/latex2e.html#g_t_005cphantom-_0026-_005cvphantom-_0026-_005chphantom)).
While most often used in mathematics, the `\smash` command can appear in other contexts. However, it doesn’t change into horizontal mode. So if it starts a paragraph then you should first put a `\leavevmode`, as in the bottom line below.
```
Text above.

\smash{smashed, no indent}  % no paragraph indent

\leavevmode\smash{smashed, with indent}  % usual paragraph indent

```

The package `mathtools` has operators that provide even finer control over smashing a subformula box.
#### 16.6.2 `\phantom` & `\vphantom` & `\hphantom`
Synopsis:
```
\phantom{subformula}

```

or
```
\vphantom{subformula}

```

or
```
\hphantom{subformula}

```

The `\phantom` command creates a box with the same height, depth, and width as subformula, but empty. That is, this command causes LaTeX to typeset the space but not fill it with the material. Here LaTeX will put a blank line that is the correct width for the answer, but will not show that answer.
```
\begin{displaymath}
  \int x^2\,dx=\mbox{\underline{$\phantom{(1/3)x^3+C}$}}
\end{displaymath}

```

The `\vphantom` variant produces an invisible box with the same vertical size as subformula, the same height and depth, but having zero width. And `\hphantom` makes a box with the same width as subformula but with zero height and depth.
In this example, the tower of exponents in the second summand expression is so tall that TeX places this expression further down than its default. Without adjustment, the two summand expressions would be at different levels. The `\vphantom` in the first expression tells TeX to leave as much vertical room as it does for the tower, so the two expressions come out at the same level.
```
\begin{displaymath}
    \sum_{j\in\{0,\ldots\, 10\}\vphantom{3^{3^{3^j}}}}
      \sum_{i\in\{0,\ldots\, 3^{3^{3^j}}\}} i\cdot j
\end{displaymath}

```

These commands are often used in conjunction with `\smash`. See [`\smash`](https://latexref.xyz/dev/latex2e.html#g_t_005csmash), which includes another example of `\vphantom`.
The three phantom commands appear often but note that LaTeX provides a suite of other commands to work with box sizes that may be more convenient, including `\makebox` (see [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox)) as well as `\settodepth` (see [`\settodepth`](https://latexref.xyz/dev/latex2e.html#g_t_005csettodepth)), `\settoheight` (see [`\settoheight`](https://latexref.xyz/dev/latex2e.html#g_t_005csettoheight)), and `\settowidth` (see [`\settowidth`](https://latexref.xyz/dev/latex2e.html#g_t_005csettowidth)). In addition, the `mathtools` package has many commands that offer fine-grained control over spacing.
All three commands produce an ordinary box, without any special mathematics status. So to do something like attaching a superscript you should give it such a status, for example with the `\operatorname` command from the package `amsmath`.
While most often used in mathematics, these three can appear in other contexts. However, they don’t cause LaTeX to change into horizontal mode. So if one of these starts a paragraph then you should prefix it with `\leavevmode`.
#### 16.6.3 `\mathstrut`
Synopsis:
```
\mathstrut

```

The analogue of `\strut` for mathematics. See [`\strut`](https://latexref.xyz/dev/latex2e.html#g_t_005cstrut).
The input `$\sqrt{x} + \sqrt{x^i}$` gives output where the second radical is taller than the first. To add extra vertical space without any horizontal space, so that the two have the same height, use `$\sqrt{x\mathstrut} + \sqrt{x^i\mathstrut}$`.
The `\mathstrut` command adds the vertical height of an open parenthesis, `(`, but no horizontal space. It is defined as `\vphantom{(}`, so see [`\phantom` & `\vphantom` & `\hphantom`](https://latexref.xyz/dev/latex2e.html#g_t_005cphantom-_0026-_005cvphantom-_0026-_005chphantom) for more. An advantage over `\strut` is that `\mathstrut` adds no depth, which is often the right thing for formulas. Using the height of an open parenthesis is just a convention; for complete control over the amount of space, use `\rule` with a width of zero. See [`\rule`](https://latexref.xyz/dev/latex2e.html#g_t_005crule).
### 16.7 Math styles
TeX’s rules for typesetting a formula depend on the context. For example, inside a displayed equation, the input `\sum_{0\leq i<n}k^m=\frac{n^{m+1}}{m+1}+\mbox{lower order terms}` will give output with the summation index centered below the summation symbol. But if that input is inline then the summation index is off to the right rather than below, so it won’t push the lines apart. Similarly, in a displayed context, the symbols in the numerator and denominator will be larger than for an inline context, and in display math subscripts and superscripts are further apart then they are in inline math.
TeX uses four math styles.
  * Display style is for a formula displayed on a line by itself, such as with `\begin{equation} ... \end{equation}`.
  * Text style is for an inline formula, as with ‘so we have $ ... $’.
  * Script style is for parts of a formula in a subscript or superscript.
  * Scriptscript style is for parts of a formula at a second level (or more) of subscript or superscript.

TeX determines a default math style but you can override it with a declaration of `\displaystyle`, or `\textstyle`, or `\scriptstyle`, or `\scriptscriptstyle`.
In this example, the ‘Arithmetic’ line’s fraction will look scrunched.
```
\begin{tabular}{r|cc}
  \textsc{Name}  &\textsc{Series}  &\textsc{Sum}  \\  \hline
  Arithmetic     &$a+(a+b)+(a+2b)+\cdots+(a+(n-1)b)$
                   &$na+(n-1)n\cdot\frac{b}{2}$  \\
  Geometric      &$a+ab+ab^2+\cdots+ab^{n-1}$
                   &$\displaystyle a\cdot\frac{1-b^n}{1-b}$  \\
\end{tabular}

```

But because of the `\displaystyle` declaration, the ‘Geometric’ line’s fraction will be easy to read, with characters the same size as in the rest of the line.
Another example is that, compared to the same input without the declaration, the result of
```
we get
$\pi=2\cdot{\displaystyle\int_{x=0}^1 \sqrt{1-x^2}\,dx}$

```

will have an integral sign that is much taller. Note that here the `\displaystyle` applies to only part of the formula, and it is delimited by being inside curly braces, as ‘{\displaystyle ...}’.
The last example is a continued fraction.
```
\begin{equation}
a_0+\frac{1}{
       \displaystyle a_1+\frac{\mathstrut 1}{
       \displaystyle a_2+\frac{\mathstrut 1}{
       \displaystyle a_3}}}
\end{equation}

```

Without the `\displaystyle` declarations, the denominators would be set in script style and scriptscript style. (The `\mathstrut` improves the height of the denominators; see [`\mathstrut`](https://latexref.xyz/dev/latex2e.html#g_t_005cmathstrut).)
### 16.8 Math miscellany
LaTeX contains a wide variety of mathematics facilities. Here are some that don’t fit into other categories.
  * [Colon character `:` & `\colon`](https://latexref.xyz/dev/latex2e.html#Colon-character-_0026-_005ccolon)
  * [`\*`](https://latexref.xyz/dev/latex2e.html#g_t_005c_002a)
  * [`\frac`](https://latexref.xyz/dev/latex2e.html#g_t_005cfrac)
  * [`\sqrt`](https://latexref.xyz/dev/latex2e.html#g_t_005csqrt)
  * [`\stackrel`](https://latexref.xyz/dev/latex2e.html#g_t_005cstackrel)

#### 16.8.1 Colon character `:` & `\colon`
Synopsis, one of:
```
:
\colon

```

In mathematics, the colon character, `:`, is a relation.
```
With side ratios \( 3:4 \) and \( 4:5 \), the triangle is right.

```

Ordinary LaTeX defines `\colon` to produce the colon character with the spacing appropriate for punctuation, as in set-builder notation `\{x\colon 0\leq x<1\}`.
But the widely-used `amsmath` package defines `\colon` for use in the definition of functions `f\colon D\to C`. So if you want the colon character as a punctuation then use `\mathpunct{:}`.
#### 16.8.2 `\*`
Synopsis:
```
\*

```

A multiplication symbol that allows a line break. If there is a break then LaTeX puts a `\times` symbol, ×, before that break.
In `\( A_1\* A_2\* A_3\* A_4 \)`, if there is no line break then LaTeX outputs it as though it were `\( A_1 A_2 A_3 A_4 \)`. If a line break does happen, for example between the two middle ones, then LaTeX sets it like `\( A_1 A_2 \times \)`, followed by the break, followed by `\( A_3 A_4 \)`.
#### 16.8.3 `\frac`
Synopsis:
```
\frac{numerator}{denominator}

```

Produces the fraction. Used as: `\begin{displaymath} \frac{1}{\sqrt{2\pi\sigma}} \end{displaymath}`. In inline math mode it comes out small; see the discussion of `\displaystyle` (see [Math formulas](https://latexref.xyz/dev/latex2e.html#Math-formulas)).
#### 16.8.4 `\sqrt`
Synopsis, one of:
```
\sqrt{arg}
\sqrt[root-number]{arg}

```

The square root, or optionally other roots, of arg. The optional argument root-number gives the root, i.e., enter the cube root of `x+y` as `\sqrt[3]{x+y}`. The size of the radical grows with that of arg (as the height of the radical grows, the angle on the leftmost part gets steeper, until for a tall enough `arg`, it is vertical).
LaTeX has a separate `\surd` symbol for making a square root without arg (see [Math symbols](https://latexref.xyz/dev/latex2e.html#Math-symbols)).
#### 16.8.5 `\stackrel`
Synopsis:
```
\stackrel{text}{relation}

```

Put text above relation. To put a function name above an arrow enter `\stackrel{f}{\longrightarrow}`.
