# 16 Math formulas: Functions, accents, and over/under constructs

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 16.3 Math functions
- 16.4 Math accents
- 16.5 Over- or under math

### 16.3 Math functions
These commands produce roman function names in math mode with proper spacing.

`\arccos`

Inverse cosine

`\arcsin`

Inverse sine

`\arctan`

Inverse tangent

`\arg`

Angle between the real axis and a point in the complex plane

`\bmod`

Binary modulo operator, used as in `\( 5\bmod 3=2 \)`

`\cos`

Cosine

`\cosh`

Hyperbolic cosine

`\cot`

Cotangent

`\coth`

Hyperbolic cotangent

`\csc`

Cosecant

`\deg`

Degrees

`\det`

Determinant

`\dim`

Dimension

`\exp`

Exponential

`\gcd`

Greatest common divisor

`\hom`

Homomorphism

`\inf`

Infimum

`\ker`

Kernel

`\lg`

Base 2 logarithm

`\lim`

Limit

`\liminf`

Limit inferior

`\limsup`

Limit superior

`\ln`

Natural logarithm

`\log`

Logarithm

`\max`

Maximum

`\min`

Minimum

`\pmod`

Parenthesized modulus, as used in `\( 5\equiv 2\pmod 3 \)`

`\Pr`

Probability

`\sec`

Secant

`\sin`

Sine

`\sinh`

Hyperbolic sine

`\sup`

Supremum sup

`\tan`

Tangent

`\tanh`

Hyperbolic tangent
The `amsmath` package adds improvements on some of these, and also allows you to define your own. The full documentation is on CTAN, but briefly, you can define an identity operator with `\DeclareMathOperator{\identity}{id}` that is like the ones above but prints as ‘id’. The starred form `\DeclareMathOperator*{\op}{op}` sets any superscript or subscript to be above and below, as is traditional with `\lim`, `\sup`, or `\max`.
### 16.4 Math accents
LaTeX provides a variety of commands for producing accented letters in math. These are different from accents in normal text (see [Accents](https://latexref.xyz/dev/latex2e.html#Accents)).

`\acute`

Math acute accent

`\bar`

Math bar-over accent

`\breve`

Math breve accent

`\check`

Math háček (check) accent

`\ddot`

Math dieresis accent

`\dot`

Math dot accent

`\grave`

Math grave accent

`\hat`

Math hat (circumflex) accent

`\mathring`

Math ring accent

`\tilde`

Math tilde accent

`\vec`

Math vector symbol

`\widehat`

Math wide hat accent

`\widetilde`

Math wide tilde accent
When you are putting an accent on an i or a j, the tradition is to use one without a dot, `\imath` or `jmath` (see [Math symbols](https://latexref.xyz/dev/latex2e.html#Math-symbols)).
### 16.5 Over- or under math
LaTeX provides commands for putting lines, braces, and arrows over or under math material.

`\underline{math}`

Underline math. For example: `\underline{x+y}`. The line is always completely below the text, taking account of descenders, so in `\(\underline{y}\)` the line is lower than in `\(\underline{x}\)`. As of approximately 2019, this command and others in this section are robust; before that, they were fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
The package `ulem` (<https://ctan.org/pkg/uelem>) does text mode underlining and allows line breaking as well as a number of other features. See also [`\hrulefill` & `\dotfill`](https://latexref.xyz/dev/latex2e.html#g_t_005chrulefill-_0026-_005cdotfill) for producing a line for such things as a signature or placeholder.

`\overline{math}`

Put a horizontal line over math. For example: `\overline{x+y}`. This differs from the accent command `\bar` (see [Math accents](https://latexref.xyz/dev/latex2e.html#Math-accents)).

`\underbrace{math}`

Put a brace under math. For example: `(1-\underbrace{1/2)+(1/2}-1/3)`.
You can attach text to the brace as a subscript (`_`) or superscript (`^`) as here:
```
\begin{displaymath}
  1+1/2+\underbrace{1/3+1/4}_{>1/2}+
       \underbrace{1/5+1/6+1/7+1/8}_{>1/2}+\cdots
\end{displaymath}

```

The superscript appears on top of the expression, and so can look unconnected to the underbrace.

`\overbrace{math}`

Put a brace over math. For example:
`\overbrace{x+x+\cdots+x}^{\mbox{\(k\) times}}`.

`\overrightarrow{math}`

Put a right arrow over math. For example: `\overrightarrow{x+y}`.

`\overleftarrow{math}`

Put a left arrow over math. For example: `\overleftarrow{a+b}`.
The package `mathtools` (<https://ctan.org/pkg/mathtools>) adds an over- and underbracket, as well as some improvements on the braces.
