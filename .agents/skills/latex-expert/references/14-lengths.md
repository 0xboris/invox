# 14 Lengths

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 14.1 Units of length
- 14.2 `\setlength`
- 14.3 `\addtolength`
- 14.4 `\settodepth`
- 14.5 `\settoheight`
- 14.6 `\settowidth`
- 14.7 `\stretch`
- 14.8 Expressions

## 14 Lengths
A _length_ is a measure of distance. Many LaTeX commands take a length as an argument.
Lengths come in two types. A _rigid length_ such as `10pt` does not contain a `plus` or `minus` component. (Plain TeX calls this a _dimen_.) A _rubber length_ (what plain TeX calls a _skip_ or _glue_) such as with `1cm plus0.05cm minus0.01cm` can contain either or both of those components. In that rubber length, the `1cm` is the _natural length_ while the other two, the `plus` and `minus` components, allow TeX to stretch or shrink the length to optimize placement.
The illustrations below use these two commands.
```
% make a black bar 10pt tall and #1 wide
\newcommand{\blackbar}[1]{\rule{#1}{10pt}}

% Make a box around #2 that is #1 wide (excluding the border)
\newcommand{\showhbox}[2]{%
  \fboxsep=0pt\fbox{\hbox to #1{#2}}}

```

This next example uses those commands to show a black bar 100 points long between ‘ABC’ and ‘XYZ’. This length is rigid.
```
ABC\showhbox{100pt}{\blackbar{100pt}}XYZ

```

As for rubber lengths, shrinking is simpler one: with `1cm minus 0.05cm`, the natural length is 1cm but TeX can shrink it down as far as 0.95cm. Beyond that, TeX refuses to shrink any more. Thus, below the first one works fine, producing a space of 98 points between the two bars.
```
ABC\showhbox{300pt}{%
  \blackbar{101pt}\hspace{100pt minus 2pt}\blackbar{101pt}}YYY

ABC\showhbox{300pt}{%
  \blackbar{105pt}\hspace{100pt minus 1pt}\blackbar{105pt}}YYY

```

But the second one gets a warning like ‘Overfull \hbox (1.0pt too wide) detected at line 17’. In the output the first ‘Y’ is overwritten by the end of the black bar, because the box’s material is wider than the 300pt allocated, as TeX has refused to shrink the total to less than 309 points.
Stretching is like shrinking except that if TeX is asked to stretch beyond the given amount, it will do it. Here the first line is fine, producing a space of 110 points between the bars.
```
ABC\showhbox{300pt}{%
  \blackbar{95pt}\hspace{100pt plus 10pt}\blackbar{95pt}}YYY

ABC\showhbox{300pt}{%
  \blackbar{95pt}\hspace{100pt plus 1pt}\blackbar{95pt}}YYY

```

In the second line TeX needs a stretch of 10 points and only 1 point was specified. TeX stretches the space to the required length but it gives you a warning like ‘Underfull \hbox (badness 10000) detected at line 22’. (We won’t discuss badness.)
You can put both stretch and shrink in the same length, as in `1ex plus 0.05ex minus 0.02ex`.
If TeX is setting two or more rubber lengths then it allocates the stretch or shrink in proportion.
```
ABC\showhbox{300pt}{%
  \blackbar{100pt}%  left
  \hspace{0pt plus 50pt}\blackbar{80pt}\hspace{0pt plus 10pt}%  middle
  \blackbar{100pt}}YYY  % right

```

The left and right bars take up 100 points, so the middle needs another 100. The middle bar is 80 points so the two `\hspace`’s must stretch 20 points. Because the two are `plus 50pt` and `plus 10pt`, TeX gets 5/6 of the stretch from the first space and 1/6 from the second.
The `plus` or `minus` component of a rubber length can contain a _fill_ component, as in `1in plus2fill`. This gives the length infinite stretchability or shrinkability so that TeX could set it to any distance. Here the two figures will be equally spaced across the page.
```
\begin{minipage}{\linewidth}
  \hspace{0pt plus 1fill}\includegraphics{godel.png}%
  \hspace{0pt plus 1fill}\includegraphics{einstein.png}%
  \hspace{0pt plus 1fill}
\end{minipage}

```

TeX has three levels of infinity for glue components: `fil`, `fill`, and `filll`. The later ones are more infinite than the earlier ones. Ordinarily document authors only use the middle one (see [`\hfill`](https://latexref.xyz/dev/latex2e.html#g_t_005chfill) and see [`\vfill`](https://latexref.xyz/dev/latex2e.html#g_t_005cvfill)).
Multiplying a rubber length by a number turns it into a rigid length, so that after `\setlength{\ylength}{1in plus 0.2in}` and `\setlength{\zlength}{3\ylength}` then the value of `\zlength` is `3in`.
  * [Units of length](https://latexref.xyz/dev/latex2e.html#Units-of-length)
  * [`\setlength`](https://latexref.xyz/dev/latex2e.html#g_t_005csetlength)
  * [`\addtolength`](https://latexref.xyz/dev/latex2e.html#g_t_005caddtolength)
  * [`\settodepth`](https://latexref.xyz/dev/latex2e.html#g_t_005csettodepth)
  * [`\settoheight`](https://latexref.xyz/dev/latex2e.html#g_t_005csettoheight)
  * [`\settowidth`](https://latexref.xyz/dev/latex2e.html#g_t_005csettowidth)
  * [`\stretch`](https://latexref.xyz/dev/latex2e.html#g_t_005cstretch)
  * [Expressions](https://latexref.xyz/dev/latex2e.html#Expressions)

### 14.1 Units of length
TeX and LaTeX know about these units both inside and outside of math mode.

`pt`

Point, 1/72.27 inch. The (approximate) conversion to metric units is 1point = .35146mm = .035146cm.

`pc`

Pica, 12 pt

`in`

Inch, 72.27 pt

`bp`

Big point, 1/72 inch. This length is the definition of a point in PostScript and many desktop publishing systems.

`mm`

Millimeter, 2.845pt

`cm`

Centimeter, 10mm

`dd`

Didot point, 1.07 pt

`cc`

Cicero, 12 dd

`sp`

Scaled point, 1/65536 pt
Three other units are defined according to the current font, rather than being an absolute dimension.

`ex`

The x-height of the current font _ex_ , traditionally the height of the lowercase letter x, is often used for vertical lengths.

`em`

Similarly _em_ , traditionally the width of the capital letter M, is often used for horizontal lengths. This is also often the size of the current font, e.g., a nominal 10pt font will have 1em = 10pt. LaTeX has several commands to produce horizontal spaces based on the em (see [`\enspace` & `\quad` & `\qquad`](https://latexref.xyz/dev/latex2e.html#g_t_005censpace-_0026-_005cquad-_0026-_005cqquad)).

`mu`

Finally, in math mode, many definitions are expressed in terms of the math unit _mu_ , defined by 1em = 18mu, where the em is taken from the current math symbols family. See [Spacing in math mode](https://latexref.xyz/dev/latex2e.html#Spacing-in-math-mode).
Using these units can help make a definition work better across font changes. For example, a definition of the vertical space between list items given as `\setlength{\itemsep}{1ex plus 0.05ex minus 0.01ex}` is more likely to still be reasonable if the font is changed than a definition given in points.
### 14.2 `\setlength`
Synopsis:
```
\setlength{\len}{amount}

```

Set the length \len to amount. The length name `\len` has to be a control sequence (see [Control sequence, control word and control symbol](https://latexref.xyz/dev/latex2e.html#Control-sequences)), and as such must begin with a backslash, `\` under normal circumstances. The amount can be a rubber length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). It can be positive, negative or zero, and can be in any units that LaTeX understands (see [Units of length](https://latexref.xyz/dev/latex2e.html#Units-of-length)).
Below, with LaTeX’s defaults the first paragraph will be indented while the second will not.
```
I told the doctor I broke my leg in two places.

\setlength{\parindent}{0em}
He said stop going to those places.

```

If you did not declare \len with `\newlength`, for example if you mistype it as in `\newlength{\specparindent}\setlength{\sepcparindent}{...}`, then you get an error like ‘Undefined control sequence. <argument> \sepcindent’. If you omit the backslash at the start of the length name then you get an error like ‘Missing number, treated as zero.’.
### 14.3 `\addtolength`
Synopsis:
```
\addtolength{\len}{amount}

```

Increment the length \len by amount. The length name `\len` has to be a control sequence (see [Control sequence, control word and control symbol](https://latexref.xyz/dev/latex2e.html#Control-sequences)), and as such must begin with a backslash, `\` under normal circumstances. The amount is a rubber length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). It can be positive, negative or zero, and can be in any units that LaTeX understands (see [Units of length](https://latexref.xyz/dev/latex2e.html#Units-of-length)).
Below, if `\parskip` starts with the value `0pt plus 1pt`
```
Doctor: how is the boy who swallowed the silver dollar?
\addtolength{\parskip}{1pt}

Nurse: no change.

```

then it has the value `1pt plus 1pt` for the second paragraph.
If you did not declare \len with `\newlength`, for example if you mistype it as in `\newlength{\specparindent}\addtolength{\sepcparindent}{...}`, then you get an error like ‘Undefined control sequence. <argument> \sepcindent’. If the amount uses some length that has not been declared, for instance if for example you mistype the above as `\addtolength{\specparindent}{0.6\praindent}`, then you get something like ‘Undefined control sequence. <argument> \praindent’. If you leave off the backslash at the start of \len, as in `\addtolength{parindent}{1pt}`, then you get something like ‘You can't use `the letter p' after \advance’.
### 14.4 `\settodepth`
Synopsis:
```
\settodepth{\len}{text}

```

Set the length \len to the depth of box that LaTeX gets on typesetting the text argument. The length name `\len` has to be a control sequence (see [Control sequence, control word and control symbol](https://latexref.xyz/dev/latex2e.html#Control-sequences)), and as such must begin with a backslash, `\` under normal circumstances.
This will print how low the character descenders go.
```
\newlength{\alphabetdepth}
\settodepth{\alphabetdepth}{abcdefghijklmnopqrstuvwxyz}
\the\alphabetdepth

```

If you did not declare \len with `\newlength`, if for example you mistype the above as `\settodepth{\aplhabetdepth}{abc...}`, then you get something like ‘Undefined control sequence. <argument> \aplhabetdepth’. If you leave the backslash out of \len, as in `\settodepth{alphabetdepth}{...}` then you get something like ‘Missing number, treated as zero. <to be read again> \setbox’.
### 14.5 `\settoheight`
Synopsis:
```
\settoheight{\len}{text}

```

Sets the length \len to the height of box that LaTeX gets on typesetting the `text` argument. The length name `\len` has to be a control sequence (see [Control sequence, control word and control symbol](https://latexref.xyz/dev/latex2e.html#Control-sequences)), and as such must begin with a backslash, `\` under normal circumstances.
This will print how high the characters go.
```
\newlength{\alphabetheight}
\settoheight{\alphabetheight}{abcdefghijklmnopqrstuvwxyz}
\the\alphabetheight

```

If no such length \len has been declared with `\newlength`, if for example you mistype as `\settoheight{\aplhabetheight}{abc...}`, then you get something like ‘Undefined control sequence. <argument> \alphabetheight’. If you leave the backslash out of \len, as in `\settoheight{alphabetheight}{...}` then you get something like ‘Missing number, treated as zero. <to be read again> \setbox’.
### 14.6 `\settowidth`
Synopsis:
```
\settowidth{\len}{text}

```

Set the length \len to the width of the box that LaTeX gets on typesetting the text argument. The length name `\len` has to be a control sequence (see [Control sequence, control word and control symbol](https://latexref.xyz/dev/latex2e.html#Control-sequences)), and as such must begin with a backslash, `\` under normal circumstances.
This prints the width of the lowercase ASCII alphabet.
```
\newlength{\alphabetwidth}
\settowidth{\alphabetwidth}{abcdefghijklmnopqrstuvwxyz}
\the\alphabetwidth

```

If no such length \len has been declared with `\newlength`, if for example you mistype the above as `\settowidth{\aplhabetwidth}{abc...}`, then you get something like ‘Undefined control sequence. <argument> \aplhabetwidth’. If you leave the backslash out of \len, as in `\settoheight{alphabetwidth}{...}` then you get something like ‘Missing number, treated as zero. <to be read again> \setbox’.
### 14.7 `\stretch`
Synopsis:
```
\stretch{number}

```

Produces a rubber length with zero natural length and number times `\fill` units of stretchability (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). The number can be positive or negative. This command is robust (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
It works for both vertical and horizontal spacing. In this horizontal example, LaTeX produces three tick marks, and the distance between the first and second is half again as long as the distance between the second and third.
```
\rule{0.4pt}{1ex}\hspace{\stretch{1.5}}%
  \rule{0.4pt}{1ex}\hspace{\stretch{1}}%
  \rule{0.4pt}{1ex}

```

In this vertical example, the ‘We dedicate …’ will have three times as much space under it as above it.
```
\newenvironment{dedication}{% in document preamble
  \clearpage\thispagestyle{empty}%
  \vspace*{\stretch{1}} % stretchable space at top
  \it
}{%
  \vspace{\stretch{3}}  % space at bot is 3x as at top
  \clearpage
}
  ...
\begin{dedication}  % in document body
We dedicate this book to our wives.
\end{dedication}

```

### 14.8 Expressions
Synopsis, one of:
```
\numexpr expression
\dimexpr expression
\glueexpr expression
\muglue expression

```

Any place where you may write an integer, or a TeX dimen, or TeX glue, or muglue, you can instead write an expression to compute that type of quantity.
An example is that `\the\dimexpr\linewidth-4pt\relax` will produce as output the length that is four points less than width of a line (the only purpose of `\the` is to show the result in the document). Analogously, `\romannumeral\numexpr6+3\relax` will produce ‘ix’, and `\the\glueexpr 5pt plus 1pt * 2 \relax` will produce ‘10.0pt plus 2.0pt’.
A convenience here over doing calculations by allocating registers and then using `\advance`, etc., is that the evaluation of expressions does not involve assignments and can therefore be performed in places where assignments are not allowed. The next example computes the width of the `\parbox`.
```
\newlength{\offset}\setlength{\offset}{2em}
\begin{center}
\parbox{\dimexpr\linewidth-\offset*3}{With malice toward none
with charity for all with firmness in the right as God gives us to see
the right let us strive on to finish the work we are in to bind up the
nation's wounds, to care for him who shall have borne the battle and
for his widow and his orphan \textasciitilde\ to do all which may
achieve and cherish a just and lasting peace among ourselves and with
all nations.  ---Abraham Lincoln, Second Inaugural Address, from the
memorial}
\end{center}

```

The expression consists of one or more terms of the same type (integer, dimension, etc.) that are added or subtracted. A term that is a type of number, dimension, etc., consists of a factor of that type, optionally multiplied or divided by factors. A factor of a type is either a quantity of that type or a parenthesized subexpression. The expression produces a result of the given type, so that `\numexpr` produces an integer, `\dimexpr` produces a dimension, etc.
In the quotation example above, changing to `\dimexpr\linewidth-3*\offset` gives the error `Illegal unit of measure (pt inserted)`. This is because for `\dimexpr` and `\glueexpr`, the input consists of a dimension or glue value followed by an optional multiplication factor, and not the other way around. Thus `\the\dimexpr 1pt*10\relax` is valid and produces ‘10.0pt’, but `\the\dimexpr 10*1pt\relax` gives the `Illegal unit` error.
The expressions absorb tokens and carry out appropriate mathematics up to a `\relax` (which will be absorbed), or up to the first non-valid token. Thus, `\the\numexpr2+3px` will print ‘5px’, because LaTeX reads the `\numexpr2+3`, which is made up of numbers, and then finds the letter `p`, which cannot be part of a number. It therefore terminates the expression and produces the ‘5’, followed by the regular text ‘px’.
This termination behavior is useful in comparisons. In `\ifnum\numexpr\parindent*2 < 10pt Yes\else No\fi`, the less than sign terminates the expression and the result is ‘No’ (in a standard LaTeX article).
Expressions may use the operators `+`, `-`, `*` and `/` along with parentheses for subexpressions, `(...)`. In glue expressions the `plus` and `minus` parts do not need parentheses to be affected by a factor. So `\the\glueexpr 5pt plus 1pt * 2 \relax` results in ‘10pt plus 2pt’.
TeX will coerce other numerical types in the same way as it does when doing register assignment. Thus `\the\numexpr\dimexpr 1pt\relax\relax` will result in ‘65536’, which is `1pt` converted to scaled points (see [`sp`](https://latexref.xyz/dev/latex2e.html#units-of-length-sp), TeX’s internal unit) and then coerced into an integer. With a `\glueexpr` here, the stretch and shrink would be dropped. Going the other way, a `\numexpr` inside a `\dimexpr` or `\glueexpr` will need appropriate units, as in `\the\dimexpr\numexpr 1 + 2\relax pt\relax`, which produces ‘3.0pt’.
The details of the arithmetic: each factor is checked to be in the allowed range, numbers must be less than _2^{31}_ in absolute value, and dimensions or glue components must be less than _2^{14}_ points, or `mu`, or `fil`, etc. The arithmetic operations are performed individually, except for a scaling operation (a multiplication immediately followed by a division) which is done as one combined operation with a 64-bit product as intermediate value. The result of each operation is again checked to be in the allowed range.
Finally, division and scaling take place with rounding (unlike TeX’s `\divide`, which truncates). Thus `\the\dimexpr 5pt*(3/2)\relax` puts ‘10.0pt’ in the document, because it rounds `3/2` to `2`, while `\the\dimexpr 5pt*(4/3)\relax` produces ‘5.0pt’.
