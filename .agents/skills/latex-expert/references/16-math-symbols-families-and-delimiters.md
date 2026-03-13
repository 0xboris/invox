# 16 Math formulas: Symbol families, delimiters, dots, and Greek letters

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 16.2.1 Arrows
- 16.2.2 `\boldmath` & `\unboldmath`
- 16.2.2.1 `bm`: Individual bold math symbols
- 16.2.2.2 OpenType bold math
- 16.2.3 Blackboard bold
- 16.2.4 Calligraphic
- 16.2.5 Delimiters
- 16.2.5.1 `\left` & `\right`
- 16.2.5.2 `\bigl`, `\bigr`, etc.
- 16.2.6 Dots, horizontal or vertical
- 16.2.7 Greek letters

#### 16.2.1 Arrows
These are the arrows that come with standard LaTeX. The `latexsym` and `amsfonts` packages contain many more.
Symbol | Command |
---|---|---
⇓ | `\Downarrow` |
↓ | `\downarrow` |
↩ | `\hookleftarrow` |
↪ | `\hookrightarrow` |
← | `\leftarrow` |
⇐ | `\Leftarrow` |
⇔ | `\Leftrightarrow` |
↔ | `\leftrightarrow` |
⟵ | `\longleftarrow` |
⟸ | `\Longleftarrow` |
⟷ | `\longleftrightarrow` |
⟺ | `\Longleftrightarrow` |
⟼ | `\longmapsto` |
⟹ | `\Longrightarrow` |
⟶ | `\longrightarrow` |
↦ | `\mapsto` |
↗ | `\nearrow` |
↖ | `\nwarrow` |
⇒ | `\Rightarrow` |
→ |  `\rightarrow`, or `\to` |
↘ | `\searrow` |
↙ | `\swarrow` |
↑ | `\uparrow` |
⇑ | `\Uparrow` |
↕ | `\updownarrow` |
⇕ | `\Updownarrow` |
An example of the difference between `\to` and `\mapsto` is: `\( f\colon D\to C \) given by \( n\mapsto n^2 \)`.
For commutative diagrams there are a number of packages, including `tikz-cd` and `amscd`.
#### 16.2.2 `\boldmath` & `\unboldmath`
Synopsis (used in paragraph mode or LR mode):
```
\boldmath \( math \)

```

or
```
\unboldmath \( math \)

```

Declarations to change the letters and symbols in math to be in a bold font, or to countermand that and bring back the regular (non-bold) default, respectively. They must be used when _not_ in math mode or display math mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). Both commands are fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
In this example each `\boldmath` command takes place inside an `\mbox`,
```
we have $\mbox{\boldmath \( v \)} = 5\cdot\mbox{\boldmath \( u \)$}$

```

which means `\boldmath` is only called in a text mode, here LR mode, and explains why we must switch LaTeX into math mode to set `v` and `u`.
If you use either command inside math mode, as with `Trouble: \( \boldmath x \)`, then you get something like ‘LaTeX Font Warning: Command \boldmath invalid in math mode’ and ‘LaTeX Font Warning: Command \mathversion invalid in math mode’.
  * [`bm`: Individual bold math symbols](https://latexref.xyz/dev/latex2e.html#bm)
  * [OpenType bold math](https://latexref.xyz/dev/latex2e.html#OpenType-bold-math)

#### 16.2.2.1 `bm`: Individual bold math symbols
Specifying `\boldmath` is the best method for typesetting a whole math expression in bold. But to typeset individual symbols within an expression in bold, the `bm` package provided by the LaTeX Project team is better. Its usage is outside the scope of this document (see its documentation at <https://ctan.org/pkg/bm> or in your installation) but the spacing in the output of this small example will show that it is an improvement over `\boldmath` within an expression:
```
\usepackage{bm}   % in preamble
...
we have $\bm{v} = 5\cdot\bm{u}$

```

#### 16.2.2.2 OpenType bold math
Unfortunately, when using the Unicode engines (XeLaTeX, LuaLaTeX), neither `\boldmath` nor `bm` usually work well, because the OpenType math fonts normally used with those engines rarely come with a bold companion, and both `\boldmath` and `bm` require this. (The implementation of `bm` relies on `\boldmath`, so the requirements are the same.) If you do have a bold math font, though, then `\boldmath` and `bm` work fine.
If no such font is available, one alternative is to construct fake bold fonts with the `fontspec` package’s `FakeBold=1` parameter (see its documentation, <https://ctan.org/pkg/fontspec>). This may be acceptable for drafting or informal distribution, but the results are far from a true bold font.
Another alternative to handling bold for OpenType math fonts is to use the `\symbf` (bold), `\symbfit` (bold italic), and related commands from the `unicode-math` package. These do not change the current font, but rather change the (Unicode) “alphabet” used, which in practice is more widely supported than a separate bold font. Many variations are possible, and so there are subtleties to getting the desired output. As usual, see the package documentation (<https://ctan.org/pkg/unicode-math>).
#### 16.2.3 Blackboard bold
Synopsis:
```
\usepackage{amssymb}   % in preamble
  ...
\mathbb{uppercase-letter}

```

Provide blackboard bold symbols, sometimes also known as doublestruck letters, used to denote number sets such as the natural numbers, the integers, etc.
Here
```
\( \forall n \in \mathbb{N}, n^2 \geq 0 \)

```

the `\mathbb{N}` gives blackboard bold symbol ℕ, representing the natural numbers.
If the argument contains something other than an uppercase letter, you do not get an error but you do get strange results, including unexpected characters.
There are packages that give access to symbols other than just the capital letters; look on CTAN.
#### 16.2.4 Calligraphic
Synopsis:
```
\mathcal{uppercase-letters}

```

Use a script-like font.
In this example the graph identifier is output in a cursive font.
```
Let the graph be \( \mathcal{G} \).

```

If you use something other than an uppercase letter then you do not get an error but you also do not get math calligraphic output. For instance, `\mathcal{g}` outputs a close curly brace symbol.
#### 16.2.5 Delimiters
Delimiters are parentheses, braces, or other characters used to mark the start and end of subformulas. This formula has three sets of parentheses delimiting the three subformulas.
```
(z-z_0)^2 = (x-x_0)^2 + (y-y_0)^2

```

The delimiters do not need to match, so you can enter `\( [0,1) \)`.
Here are the common delimiters:
Delimiter | Command | Name
---|---|---
( | `(` | Left parenthesis
) | `)` | Right parenthesis
\\{ |  `{` or `\lbrace` | Left brace
\\} |  `}` or `\rbrace` | Right brace
[ |  `[` or `\lbrack` | Left bracket
] |  `]` or `\rbrack` | Right bracket
⌊ | `\lfloor` | Left floor bracket
⌋ | `\rfloor` | Right floor bracket
⌈ | `\lceil` | Left ceiling bracket
⌉ | `\rceil` | Right ceiling bracket
⟨ | `\langle` | Left angle bracket
⟩ | `\rangle` | Right angle bracket
/ | `/` | Slash, or forward slash
\ | `\backslash` | Reverse slash, or backslash
| |  `|` or `\vert` | Vertical bar
‖ |  `\|` or `\Vert` | Double vertical bar
The `mathtools` package allows you to create commands for paired delimiters. For instance, if you put `\DeclarePairedDelimiter\abs{\lvert}{\rvert}` in your preamble then you get two commands for single-line vertical bars (they only work in math mode). The starred form, such as `\abs*{\frac{22}{7}}`, has the height of the vertical bars match the height of the argument. The unstarred form, such as `\abs{\frac{22}{7}}`, has the bars fixed at a default height. This form accepts an optional argument, as in `\abs[size command]{\frac{22}{7}}`, where the height of the bars is given in size command, such as `\Bigg`. Using instead `\lVert` and `\rVert` as the symbols will give you a norm symbol with the same behavior.
  * [`\left` & `\right`](https://latexref.xyz/dev/latex2e.html#g_t_005cleft-_0026-_005cright)
  * [`\bigl`, `\bigr`, etc.](https://latexref.xyz/dev/latex2e.html#g_t_005cbigl-_0026-_005cbigr-etc_002e)

#### 16.2.5.1 `\left` & `\right`
Synopsis:
```
\left delimiter1 ... \right delimiter2

```

Make matching parentheses, braces, or other delimiters. LaTeX makes the delimiters tall enough to just cover the size of the formula that they enclose.
This makes a unit vector surrounded by parentheses tall enough to cover the entries.
```
\begin{equation}
  \left(\begin{array}{c}
    1   \\
    0   \\
  \end{array}\right)
\end{equation}

```

See [Delimiters](https://latexref.xyz/dev/latex2e.html#Delimiters), for a list of the common delimiters.
Every `\left` must have a matching `\right`. In the above example, leaving out the `\left(` gets the error message ‘Extra \right’. Leaving out the `\right)` gets ‘You can't use `\eqno' in math mode’.
However, delimiter1 and delimiter2 need not match. A common case is that you want an unmatched brace, as below. Use a period, ‘.’, as a _null delimiter_.
```
\begin{equation}
  f(n)=\left\{\begin{array}{ll}
                1             &\mbox{--if \(n=0\)} \\
                f(n-1)+3n^2   &\mbox{--else}
       \end{array}\right.
\end{equation}

```

Note that to get a curly brace as a delimiter you must prefix it with a backslash, `\{` (see [Reserved characters](https://latexref.xyz/dev/latex2e.html#Reserved-characters)). (The packages `amsmath` and `mathtools` allow you to get the above construct through in a `cases` environment.)
The `\left ... \right` pair make a group. One consequence is that the formula enclosed in the `\left ... \right` pair cannot have line breaks in the output. This includes both manual line breaks and LaTeX-generated automatic ones. In this example, LaTeX breaks the equation to make the formula fit the margins.
```
Lorem ipsum dolor sit amet
\( (a+b+c+d+e+f+g+h+i+j+k+l+m+n+o+p+q+r+s+t+u+v+w+x+y+z) \)

```

But with `\left` and `\right`
```
Lorem ipsum dolor sit amet
\( \left(a+b+c+d+e+f+g+h+i+j+k+l+m+n+o+p+q+r+s+t+u+v+w+x+y+z\right) \)

```

LaTeX won’t break the line, causing the formula to extend into the margin.
Because `\left ... \right` make a group, all the usual grouping rules hold. Here, the value of `\testlength` set inside the equation will be forgotten, and the output is ‘1.2pt’.
```
\newlength{\testlength} \setlength{\testlength}{1.2pt}
\begin{equation}
  \left( a+b=c \setlength{\testlength}{3.4pt} \right)
  \the\testlength
\end{equation}

```

The `\left ... \right` pair affect the horizontal spacing of the enclosed formula, in two ways. The first is that in `\( \sin(x) = \sin\left(x\right) \)` the one after the equals sign has more space around the `x`. That’s because `\left( ... \right)` inserts an inner node while `( ... )` inserts an opening node. The second way that the pair affect the horizontal spacing is that because they form a group, the enclosed subformula will be typeset at its natural width, with no stretching or shrinking to make the line fit better.
TeX scales the delimiters according to the height and depth of the enclosed formula. Here LaTeX grows the brackets to extend the full height of the integral.
```
\begin{equation}
  \left[ \int_{x=r_0}^{\infty} -G\frac{Mm}{r^2}\, dr \right]
\end{equation}

```

Manual sizing is often better. For instance, although below the rule has no depth, TeX will create delimiters that extend far below the rule.
```
\begin{equation}
  \left( \rule{1pt}{1cm} \right)
\end{equation}

```

TeX can choose delimiters that are too small, as in `\( \left| |x|+|y| \right| \)`. It can also choose delimiters that are too large, as here.
```
\begin{equation}
  \left( \sum_{0\leq i<n} i^k \right)
\end{equation}

```

A third awkward case is when a long displayed formula is on more than one line and you must match the sizes of the opening and closing delimiter; you can’t use `\left` on the first line and `\right` on the last because they must be paired.
To size the delimiters manually, see [`\bigl`, `\bigr`, etc.](https://latexref.xyz/dev/latex2e.html#g_t_005cbigl-_0026-_005cbigr-etc_002e).
#### 16.2.5.2 `\bigl`, `\bigr`, etc.
Synopsis, one of:
```
\bigldelimiter1 ... \bigrdelimiter2
\Bigldelimiter1 ... \bigrdelimiter2
\biggldelimiter1 ... \biggrdelimiter2
\Biggldelimiter1 ... \Biggrdelimiter2

```

(as with `\bigl[...\bigr]`; strictly speaking they need not be paired, see below), or one of:
```
\bigmdelimiter
\Bigmdelimiter
\biggmdelimiter
\Biggmdelimiter

```

(as with `\bigm|`), or one of:
```
\bigdelimiter
\Bigdelimiter
\biggdelimiter
\Biggdelimiter

```

(as with `\big[`).
Produce manually-sized delimiters. For delimiters that are automatically sized see [`\left` & `\right`](https://latexref.xyz/dev/latex2e.html#g_t_005cleft-_0026-_005cright)).
This produces slightly larger outer vertical bars.
```
  \bigl| |x|+|y| \bigr|

```

The commands above are listed in order of increasing size. You can use the smallest size such as `\bigl...\bigr` in a paragraph without causing LaTeX to spread the lines apart. The larger sizes are meant for displayed equations.
See [Delimiters](https://latexref.xyz/dev/latex2e.html#Delimiters), for a list of the common delimiters. In the family of commands with ‘l’ or ‘r’, delimiter1 and delimiter2 need not match together.
The ‘l’ and ‘r’ commands produce open and close delimiters that insert no horizontal space between a preceding atom and the delimiter, while the commands without ‘l’ and ‘r’ insert some space (because each delimiter is set as an ordinary variable). Compare these two.
```
\begin{tabular}{l}
  \(\displaystyle \sin\biggl(\frac{1}{2}\biggr) \)  \\  % good
  \(\displaystyle \sin\bigg(\frac{1}{2}\bigg)  \)   \\  % bad
\end{tabular}

```

The traditional typographic treatment is on the first line. On the second line the output will have some extra space between the `\sin` and the open parenthesis.
Commands without ‘l’ or ‘r’ do give correct spacing in some circumstances, as with this large vertical line
```
\begin{equation}
  \int_{x=a}^b x^2\,dx = \frac{1}{3} x^3 \Big|_{x=a}^b
\end{equation}

```

(many authors would replace `\frac` with the `\tfrac` command from the `amsmath` package), and as with this larger slash.
```
\begin{equation}
  \lim_{n\to\infty}\pi(n) \big/ (n/\log n) = 1
\end{equation}

```

Unlike the `\left...\right` pair (see [`\left` & `\right`](https://latexref.xyz/dev/latex2e.html#g_t_005cleft-_0026-_005cright)), the commands here with ‘l’ or ‘r’ do not make a group. Strictly speaking they need not be matched so you can write something like this.
```
\begin{equation}
  \Biggl[ \pi/6 ]
\end{equation}

```

The commands with ‘m’ are for relations, which are in the middle of formulas, as here.
```
\begin{equation}
  \biggl\{ a\in B \biggm| a=\sum_{0\leq i<n}3i^2+4 \biggr\}
\end{equation}

```

#### 16.2.6 Dots, horizontal or vertical
Ellipses are the three dots (usually three) indicating that a pattern continues.
```
\begin{array}{cccc}
  a_{0,0}    &a_{0,1}   &a_{0,2} &\ldots \\
  a_{1,0}    &\ddots                     \\
  \vdots
\end{array}

```

LaTeX provides these.

`\cdots`

Horizontal ellipsis with the dots raised to the center of the line, as in ⋯. Used as: `\( a_0\cdot a_1\cdots a_{n-1} \)`.

`\ddots`

Diagonal ellipsis, ⋱. See the above array example for a usage.

`\ldots`

`\mathellipsis`

`\dots`

Ellipsis on the baseline, …. Used as: `\( x_0,\ldots x_{n-1} \)`. Another example is the above array example. Synonyms are `\mathellipsis` and `\dots`. A synonym from the `amsmath` package is `\hdots`.
You can also use this command outside of mathematical text, as in `The gears, brakes, \ldots{} are all broken`.

`\vdots`

Vertical ellipsis, ⋮. See the above array example for a usage.
The `amsmath` package has the command `\dots` to semantically mark up ellipses. This example produces two different-looking outputs for the first two uses of the `\dots` command.
```
\usepackage{amsmath}  % in preamble
  ...
Suppose that \( p_0, p_1, \dots, p_{n-1} \) lists all of the primes.
Observe that \( p_0\cdot p_1 \dots \cdot p_{n-1} +1 \) is not a
  multiple of any \( p_i \).
Conclusion: there are infinitely many primes \( p_0, p_1, \dotsc \).

```

In the first line LaTeX looks to the comma following `\dots` to determine that it should output an ellipsis on the baseline. The second line has a `\cdot` following `\dots` so LaTeX outputs an ellipsis that is on the math axis, vertically centered. However, the third usage has no follow-on character so you have to tell LaTeX what to do. You can use one of the commands: `\dotsc` if you need the ellipsis appropriate for a comma following, `\dotsb` if you need the ellipses that fits when the dots are followed by a binary operator or relation symbol, `\dotsi` for dots with integrals, or `\dotso` for others.
The `\dots` command from `amsmath` differs from the LaTeX kernel’s `\dots` command in another way: it outputs a thin space after the ellipsis. Furthermore, the `unicode-math` package automatically loads `amsmath`, so `amsmath`’s `\dots` may be active even when you did not explicitly load it, thus changing the output from `\dots` in both text and math mode.
Yet more about the ellipsis commands: when running under Unicode engines (`lualatex`, `xelatex`), LaTeX will use the Unicode ellipsis character (U+2026) in the font if it’s available; under traditional TeX engines (`pdflatex`, `latex`), it will typeset three spaced periods. Generally, the Unicode single-character ellipsis has almost no space between the three periods, while the spacing of the non-Unicode ellipsis is looser, more in accordance with traditional typography.
#### 16.2.7 Greek letters
The upper case versions of these Greek letters are only shown when they differ from Roman upper case letters.
Symbol | Command | Name |
---|---|---|---
α | `\alpha` | Alpha
β | `\beta` | Beta
γ, Γ |  `\gamma`, `\Gamma` | Gamma
δ, Δ |  `\delta`, `\Delta` | Delta
ε, ϵ |  `\varepsilon`, `\epsilon` | Epsilon
ζ | `\zeta` | Zeta
η | `\eta` | Eta
θ, ϑ |  `\theta`, `\vartheta` | Theta
ι | `\iota` | Iota
κ | `\kappa` | Kappa
λ, Λ |  `\lambda`, `\Lambda` | Lambda
μ | `\mu` | Mu
ν | `\nu` | Nu
ξ, Ξ |  `\xi`, `\Xi` | Xi
π, Π |  `\pi`, `\Pi` | Pi
ρ, ϱ |  `\rho`, `\varrho` | Rho
σ, Σ |  `\sigma`, `\Sigma` | Sigma
τ | `\tau` | Tau
ϕ, φ, Φ |  `\phi`, `\varphi`, `\Phi` | Phi
χ | `\chi` | chi
ψ, Ψ |  `\psi`, `\Psi` | Psi
ω, Ω |  `\omega`, `\Omega` | Omega
For omicron, if you are using LaTeX’s default Computer Modern font then enter omicron just as ‘o’ or ‘O’. If you like having the name or if your font shows a difference then you can use something like `\newcommand\omicron{o}`. The package `unicode-math` has `\upomicron` for upright omicron and `\mitomicron` for math italic.
While the set membership relation symbol ∈ generated by `\in` is related to epsilon, it is never used for a variable.
