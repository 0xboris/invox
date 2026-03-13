# 08 Environments: Letters, generic lists, math, and minipages

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 8.15 `letter` environment: writing letters
- 8.16 `list`
- 8.17 `math`
- 8.18 `minipage`

### 8.15 `letter` environment: writing letters
This environment is used for creating letters. See [Letters](https://latexref.xyz/dev/latex2e.html#Letters).
### 8.16 `list`
Synopsis:
```
\begin{list}{labeling}{spacing}
  \item[optional label of first item] text of first item
  \item[optional label of second item] text of second item
  ...
\end{list}

```

An environment for constructing lists.
Note that this environment does not typically appear in the document body. Most lists created by LaTeX authors are the ones that come standard: the `description`, `enumerate`, and `itemize` environments (see [`description`](https://latexref.xyz/dev/latex2e.html#description), [`enumerate`](https://latexref.xyz/dev/latex2e.html#enumerate), and [`itemize`](https://latexref.xyz/dev/latex2e.html#itemize)).
Instead, the `list` environment is most often used in macros. For example, many standard LaTeX environments that do not immediately appear to be lists are in fact constructed using `list`, including `quotation`, `quote`, and `center` (see [`quotation` & `quote`](https://latexref.xyz/dev/latex2e.html#quotation-_0026-quote), see [`center`](https://latexref.xyz/dev/latex2e.html#center)).
This uses the `list` environment to define a new custom environment.
```
\newcounter{namedlistcounter}  % number the items
\newenvironment{named}
  {\begin{list}
     {Item~\Roman{namedlistcounter}.} % labeling
     {\usecounter{namedlistcounter}   % set counter
      \setlength{\leftmargin}{3.5em}} % set spacing
  }
  {\end{list}}

\begin{named}
  \item Shows as ``Item~I.''
  \item[Special label.] Shows as ``Special label.''
  \item Shows as ``Item~II.''
\end{named}

```

The mandatory first argument labeling specifies the default labeling of list items. It can contain text and LaTeX commands, as above where it contains both ‘Item’ and ‘\Roman{…}’. LaTeX forms the label by putting the labeling argument in a box of width `\labelwidth`. If the label is wider than that, the additional material extends to the right. When making an instance of a `list` you can override the default labeling by giving `\item` an optional argument by including square braces and the text, as in the above `\item[Special label.]`; see [`\item`: An entry in a list](https://latexref.xyz/dev/latex2e.html#g_t_005citem).
The mandatory second argument spacing has a list of commands. This list can be empty. A command that can go in here is `\usecounter{countername}` (see [`\usecounter`](https://latexref.xyz/dev/latex2e.html#g_t_005cusecounter)). Use this to tell LaTeX to number the items using the given counter. The counter will be reset to zero each time LaTeX enters the environment, and the counter is incremented by one each time LaTeX encounters an `\item` that does not have an optional argument.
Another command that can go in spacing is `\makelabel`, which constructs the label box. By default it puts the contents flush right. Its only argument is the label, which it typesets in LR mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). One example of changing its definition is that to the above `named` example, before the definition of the environment add `\newcommand{\namedmakelabel}[1]{\textsc{#1}}`, and between the `\setlength` command and the parenthesis that closes the spacing argument also add `\let\makelabel\namedmakelabel`. Then the labels will be typeset in small caps. Similarly, changing the second code line to `\let\makelabel\fbox` puts the labels inside a framed box. Another example of the `\makelabel` command is below, in the definition of the `redlabel` environment.
Also often in spacing are commands to redefine the spacing for the list. Below are the spacing parameters with their default values. (Default values for derived environments such as `itemize` can be different than the values shown here.) See also the figure that follows the list. Each is a length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). The vertical spaces are normally rubber lengths, with `plus` and `minus` components, to give TeX flexibility in setting the page. Change each with a command such as `\setlength{\itemsep}{2pt plus1pt minus1pt}`. For some effects these lengths should be zero or negative.

`\itemindent`

Extra horizontal space indentation, beyond `leftmargin`, of the each item’s first line. Its default value is `0pt`.

`\itemsep`

Vertical space between items, in addition to `\parsep` and `\parskip`. The defaults for the first three levels in LaTeX’s ‘article’, ‘book’, and ‘report’ classes at 10 point size are: `4pt plus2pt minus1pt`, `\parsep` (that is, `2pt plus1pt minus1pt`), and `\topsep` (that is, `2pt plus1pt minus1pt`). The defaults at 11 point are: `4.5pt plus2pt minus1pt`, `\parsep` (that is, `2pt plus1pt minus1pt`), and `\topsep` (that is, `2pt plus1pt minus1pt`). The defaults at 12 point are: `5pt plus2.5pt minus1pt`, `\parsep` (that is, `2.5pt plus1pt minus1pt`), and `\topsep` (that is, `2.5pt plus1pt minus1pt`).

`\labelsep`

Horizontal space between the label and text of an item. The default for LaTeX’s ‘article’, ‘book’, and ‘report’ classes is `0.5em`.

`\labelwidth`

Horizontal width. The box containing the label is nominally this wide. If `\makelabel` returns text that is wider than this then the first line of the item will be indented to make room for this extra material. If `\makelabel` returns text of width less than or equal to `\labelwidth` then LaTeX’s default is that the label is typeset flush right in a box of this width.
The left edge of the label box is `\leftmargin`+`\itemindent`-`\labelsep`-`\labelwidth` from the left margin of the enclosing environment.
The default for LaTeX’s ‘article’, ‘book’, and ‘report’ classes at the top level is `\leftmargini`-`\labelsep`, (which is `2em` in one column mode and `1.5em` in two column mode). At the second level it is `\leftmarginii`-`\labelsep`, and at the third level it is `\leftmarginiii`-`\labelsep`. These definitions make the label’s left edge coincide with the left margin of the enclosing environment.

`\leftmargin`

Horizontal space between the left margin of the enclosing environment (or the left margin of the page if this is a top-level list), and the left margin of this list. It must be non-negative.
In the standard LaTeX document classes, a top-level list has this set to the value of `\leftmargini`, while a list that is nested inside a top-level list has this margin set to `\leftmarginii`. More deeply nested lists get the values of `\leftmarginiii` through `\leftmarginvi`. (Nesting greater than level five generates the error message ‘Too deeply nested’.)
The defaults for the first three levels in LaTeX’s ‘article’, ‘book’, and ‘report’ classes are: `\leftmargini` is `2.5em` (in two column mode, `2em`), `\leftmarginii` is `2.2em`, and `\leftmarginiii` is `1.87em`.

`\listparindent`

Horizontal space of additional line indentation, beyond `\leftmargin`, for second and subsequent paragraphs within a list item. A negative value makes this an “outdent”. Its default value is `0pt`.

`\parsep`

Vertical space between paragraphs within an item. The defaults for the first three levels in LaTeX’s ‘article’, ‘book’, and ‘report’ classes at 10 point size are: `4pt plus2pt minus1pt`, `2pt plus1pt minus1pt`, and `0pt`. The defaults at 11 point size are: `4.5pt plus2pt minus1pt`, `2pt plus1pt minus1pt`, and `0pt`. The defaults at 12 point size are: `5pt plus2.5pt minus1pt`, `2.5pt plus1pt minus1pt`, and `0pt`.

`\partopsep`

Vertical space added, beyond `\topsep`+`\parskip`, to the top and bottom of the entire environment if the list instance is preceded by a blank line. (A blank line in the LaTeX source before the list changes spacing at both the top and bottom of the list; whether the line following the list is blank does not matter.)
The defaults for the first three levels in LaTeX’s ‘article’, ‘book’, and ‘report’ classes at 10 point size are: `2pt plus1 minus1pt`, `2pt plus1pt minus1pt`, and `1pt plus0pt minus1pt`. The defaults at 11 point are: `3pt plus1pt minus1pt`, `3pt plus1pt minus1pt`, and `1pt plus0pt minus1pt`). The defaults at 12 point are: `3pt plus2pt minus3pt`, `3pt plus2pt minus2pt`, and `1pt plus0pt minus1pt`.

`\rightmargin`

Horizontal space between the right margin of the list and the right margin of the enclosing environment. Its default value is `0pt`. It must be non-negative.

`\topsep`

Vertical space added to both the top and bottom of the list, in addition to `\parskip` (see [`\parindent` & `\parskip`](https://latexref.xyz/dev/latex2e.html#g_t_005cparindent-_0026-_005cparskip)). The defaults for the first three levels in LaTeX’s ‘article’, ‘book’, and ‘report’ classes at 10 point size are: `8pt plus2pt minus4pt`, `4pt plus2pt minus1pt`, and `2pt plus1pt minus1pt`. The defaults at 11 point are: `9pt plus3pt minus5pt`, `4.5pt plus2pt minus1pt`, and `2pt plus1pt minus1pt`. The defaults at 12 point are: `10pt plus4pt minus6pt`, `5pt plus2.5pt minus1pt`, and `2.5pt plus1pt minus1pt`.
This shows the horizontal and vertical distances.
![latex2e-figures/list](https://latexref.xyz/dev/latex2e-figures/list.png)
The lengths shown are listed below. The key relationship is that the right edge of the bracket for h1 equals the right edge of the bracket for h4, so that the left edge of the label box is at h3+h4-(h0+h1).

v0

_`\topsep`+`\parskip`_ if the list environment does not start a new paragraph, and `\topsep`+`\parskip`+`\partopsep` if it does

v1

`\parsep`

v2

`\itemsep`+`\parsep`

v3

Same as v0. (This space is affected by whether a blank line appears in the source above the environment; whether a blank line appears in the source below the environment does not matter.)

h0

`\labelwidth`

h1

`\labelsep`

h2

`\listparindent`

h3

`\leftmargin`

h4

`\itemindent`

h5

`\rightmargin`
The list’s left and right margins, shown above as h3 and h5, are with respect to the ones provided by the surrounding environment, or with respect to the page margins for a top-level list. The line width used for typesetting the list items is `\linewidth` (see [Page layout parameters](https://latexref.xyz/dev/latex2e.html#Page-layout-parameters)). For instance, set the list’s left margin to be one quarter of the distance between the left and right margins of the enclosing environment with `\setlength{\leftmargin}{0.25\linewidth}`.
Page breaking in a list structure is controlled by the three parameters below. For each, the LaTeX default is `-\@lowpenalty`, that is, `-51`. Because it is negative, it somewhat encourages a page break at each spot. Change it with, e.g., `\@beginparpenalty=9999`; a value of 10000 prohibits a page break.

`\@beginparpenalty`

The page breaking penalty for breaking before the list (default `-51`).

`\@itempenalty`

The page breaking penalty for breaking before a list item (default `-51`).

`\@endparpenalty`

The page breaking penalty for breaking after a list (default `-51`).
The package `enumitem` is useful for customizing lists.
This example has the labels in red. They are numbered, and the left edge of the label lines up with the left edge of the item text. See [`\usecounter`](https://latexref.xyz/dev/latex2e.html#g_t_005cusecounter).
```
\usepackage{color}
\newcounter{cnt}
\newcommand{\makeredlabel}[1]{\textcolor{red}{#1.}}
\newenvironment{redlabel}
  {\begin{list}
    {\arabic{cnt}}
    {\usecounter{cnt}
     \setlength{\labelwidth}{0em}
     \setlength{\labelsep}{0.5em}
     \setlength{\leftmargin}{1.5em}
     \setlength{\itemindent}{0.5em} % equals \labelwidth+\labelsep
     \let\makelabel=\makeredlabel
    }
  }
{\end{list}}

```

  * [`\item`: An entry in a list](https://latexref.xyz/dev/latex2e.html#g_t_005citem)
  * [`trivlist`: A restricted form of `list`](https://latexref.xyz/dev/latex2e.html#trivlist)

#### 8.16.1 `\item`: An entry in a list
Synopsis:
```
\item text of item

```

or
```
\item[optional-label] text of item

```

An entry in a list. The entries are prefixed by a label, whose default depends on the list type.
Because the optional label is surrounded by square brackets ‘[...]’, if you have an item whose text starts with [, you have to hide the bracket inside curly braces, as in: `\item {[} is an open square bracket`; otherwise, LaTeX will think it marks the start of an optional label.
Similarly, if the item does have the optional label and you need a close square bracket inside that label, you must hide it in the same way: `\item[Close square bracket, {]}]`. See [LaTeX command syntax](https://latexref.xyz/dev/latex2e.html#LaTeX-command-syntax).
In this example the enumerate list has two items that use the default label and one that uses the optional label.
```
\begin{enumerate}
  \item Moe
  \item[sometimes] Shemp
  \item Larry
\end{enumerate}

```

The first item is labelled ‘1.’, the second item is labelled ‘sometimes’, and the third item is labelled ‘2.’. Because of the optional label in the second item, the third item is not labelled ‘3.’.
#### 8.16.2 `trivlist`: A restricted form of `list`
Synopsis:
```
\begin{trivlist}
  ...
\end{trivlist}

```

A restricted version of the list environment, in which margins are not indented and an `\item` without an optional argument produces no text. It is most often used in macros, to define an environment where the `\item` command is part of the environment’s definition. For instance, the `center` environment is defined essentially like this:
```
\newenvironment{center}
  {\begin{trivlist}\centering\item\relax}
  {\end{trivlist}}

```

Using `trivlist` in this way allows the macro to inherit some common code: combining vertical space of two adjacent environments; detecting whether the text following the environment should be considered a new paragraph or a continuation of the previous one; adjusting the left and right margins for possible nested list environments.
Specifically, `trivlist` uses the current values of the list parameters (see [`list`](https://latexref.xyz/dev/latex2e.html#list)), except that `\parsep` is set to the value of `\parskip`, and `\leftmargin`, `\labelwidth`, and `\itemindent` are set to zero.
This example outputs the items as two paragraphs, except that (by default) they have no paragraph indent and are vertically separated.
```
\begin{trivlist}
\item The \textit{Surprise} is not old; no one would call her old.
\item She has a bluff bow, lovely lines.
\end{trivlist}

```

### 8.17 `math`
Synopsis:
```
\begin{math}
math
\end{math}

```

The `math` environment inserts given math material within the running text. `\(...\)` and `$...$` are synonyms. See [Math formulas](https://latexref.xyz/dev/latex2e.html#Math-formulas).
### 8.18 `minipage`
Synopses:
```
\begin{minipage}{width}
  contents
\end{minipage}

```

or
```
\begin{minipage}[position][height][inner-pos]{width}
  contents
\end{minipage}

```

Put contents into a box that is width wide. This is like a small version of a page; it can contain its own footnotes, itemized lists, etc. (There are some restrictions, including that it cannot have floats.) This box will not be broken across pages. So `minipage` is similar to `\parbox` (see [`\parbox`](https://latexref.xyz/dev/latex2e.html#g_t_005cparbox)) but here you can have paragraphs.
This example will be 3 inches wide, and has two paragraphs.
```
\begin{minipage}{3in}
  Stephen Kleene was a founder of the Theory of Computation.

  He was a student of Church, wrote three influential texts,
  was President of the Association for Symbolic Logic,
  and won the National Medal of Science.
\end{minipage}

```

See below for a discussion of the paragraph indent inside a `minipage`.
The required argument width is a rigid length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). It gives the width of the box into which contents are typeset.
There are three optional arguments, position, height, and inner-pos. You need not include all three. For example, get the default position and set the height with `\begin{minipage}[c][2.54cm]{\columnwidth} contents \end{minipage}`. (Get the natural height with an empty argument, `[]`.)
The optional argument position governs how the `minipage` vertically aligns with the surrounding material.

`c`

(synonym `m`) Default. Positions the `minipage` so its vertical center lines up with the center of the adjacent text line.

`t`

Align the baseline of the top line in the `minipage` with the baseline of the surrounding text (plain TeX’s `\vtop`).

`b`

Align the baseline of the bottom line in the `minipage` with the baseline of the surrounding text (plain TeX’s `\vbox`).
To see the effects of these, contrast running this
```
---\begin{minipage}[c]{0.25in}
  first\\ second\\ third
\end{minipage}

```

with the results of changing `c` to `b` or `t`.
The optional argument height is a rigid length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). It sets the height of the `minipage`. You can enter any value larger than, or equal to, or smaller than the `minipage`’s natural height and LaTeX will not give an error or warning. You can also set it to a height of zero or a negative value.
The final optional argument inner-pos controls the placement of contents inside the box. These are the possible values are (the default is the value of position).

`t`

Place contents at the top of the box.

`c`

Place it in the vertical center.

`b`

Place it at the box bottom.

`s`

Stretch contents out vertically; it must contain vertically stretchable space.
The inner-pos argument makes sense when the height option is set to a value larger than the `minipage`’s natural height. To see the effect of the options, run this example with the various choices in place of `b`.
```
Text before
\begin{center}
  ---\begin{minipage}[c][3in][b]{0.25\textwidth}
       first\\ second\\ third
  \end{minipage}
\end{center}
Text after

```

By default paragraphs are not indented in a `minipage`. Change that with a command such as `\setlength{\parindent}{1pc}` at the start of contents.
Footnotes in a `minipage` environment are handled in a way that is particularly useful for putting footnotes in figures or tables. A `\footnote` or `\footnotetext` command puts the footnote at the bottom of the minipage instead of at the bottom of the page, and it uses the `\mpfootnote` counter instead of the ordinary `footnote` counter (see [Counters](https://latexref.xyz/dev/latex2e.html#Counters)).
This puts the footnote at the bottom of the table, not the bottom of the page.
```
\begin{center}           % center the minipage on the line
\begin{minipage}{2.5in}
  \begin{center}         % center the table inside the minipage
    \begin{tabular}{ll}
      \textsc{Monarch}  &\textsc{Reign}             \\ \hline
      Elizabeth II      &63 years\footnote{to date} \\
      Victoria          &63 years                   \\
      George III        &59 years
    \end{tabular}
  \end{center}
\end{minipage}
\end{center}

```

If you nest minipages then there is an oddness when using footnotes. Footnotes appear at the bottom of the text ended by the next `\end{minipage}` which may not be their logical place.
This puts a table containing data side by side with a map graphic. They are vertically centered.
```
% siunitx to have the S column specifier,
% which aligns numbers on their decimal point.
\usepackage{siunitx}
\newcommand*{\vcenteredhbox}[1]{\begin{tabular}{@{}c@{}}#1\end{tabular}}
  ...
\begin{center}
  \vcenteredhbox{\includegraphics[width=0.3\textwidth]{nyc.png}}
  \hspace{0.1\textwidth}
  \begin{minipage}{0.5\textwidth}
    \begin{tabular}{r|S}
      % \multicolumn to remove vertical bar between column headers
      \multicolumn{1}{r}{Borough} &
      % braces to prevent siunitx from misinterpreting the
      % period as a decimal separator
      {Pop. (million)}  \\ \hline
      The Bronx      &1.5  \\
      Brooklyn       &2.6  \\
      Manhattan      &1.6  \\
      Queens         &2.3  \\
      Staten Island  &0.5
    \end{tabular}
  \end{minipage}
\end{center}

```
