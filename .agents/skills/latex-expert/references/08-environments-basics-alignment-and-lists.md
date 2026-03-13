# 08 Environments: Basics, alignment, and lists

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 8.1 `abstract`
- 8.2 `array`
- 8.3 `center`
- 8.4 `description`
- 8.5 `displaymath`
- 8.6 `document`
- 8.7 `enumerate`
- 8.8 `eqnarray`
- 8.9 `equation`
- 8.10 `figure`
- 8.11 `filecontents`
- 8.12 `flushleft`
- 8.13 `flushright`
- 8.14 `itemize`

## 8 Environments
LaTeX provides many environments for delimiting certain behavior. An environment begins with `\begin` and ends with `\end`, like this:
```
\begin{environment-name}
  ...
\end{environment-name}

```

The environment-name at the beginning must exactly match that at the end. For instance, the input `\begin{table*}...\end{table}` will cause an error like: ‘! LaTeX Error: \begin{table*} on input line 5 ended by \end{table}.’
Environments are executed within a group.
  * [`abstract`](https://latexref.xyz/dev/latex2e.html#abstract)
  * [`array`](https://latexref.xyz/dev/latex2e.html#array)
  * [`center`](https://latexref.xyz/dev/latex2e.html#center)
  * [`description`](https://latexref.xyz/dev/latex2e.html#description)
  * [`displaymath`](https://latexref.xyz/dev/latex2e.html#displaymath)
  * [`document`](https://latexref.xyz/dev/latex2e.html#document)
  * [`enumerate`](https://latexref.xyz/dev/latex2e.html#enumerate)
  * [`eqnarray`](https://latexref.xyz/dev/latex2e.html#eqnarray)
  * [`equation`](https://latexref.xyz/dev/latex2e.html#equation)
  * [`figure`](https://latexref.xyz/dev/latex2e.html#figure)
  * [`filecontents`](https://latexref.xyz/dev/latex2e.html#filecontents)
  * [`flushleft`](https://latexref.xyz/dev/latex2e.html#flushleft)
  * [`flushright`](https://latexref.xyz/dev/latex2e.html#flushright)
  * [`itemize`](https://latexref.xyz/dev/latex2e.html#itemize)
  * [`letter` environment: writing letters](https://latexref.xyz/dev/latex2e.html#letter)
  * [`list`](https://latexref.xyz/dev/latex2e.html#list)
  * [`math`](https://latexref.xyz/dev/latex2e.html#math)
  * [`minipage`](https://latexref.xyz/dev/latex2e.html#minipage)
  * [`picture`](https://latexref.xyz/dev/latex2e.html#picture)
  * [`quotation` & `quote`](https://latexref.xyz/dev/latex2e.html#quotation-_0026-quote)
  * [`tabbing`](https://latexref.xyz/dev/latex2e.html#tabbing)
  * [`table`](https://latexref.xyz/dev/latex2e.html#table)
  * [`tabular`](https://latexref.xyz/dev/latex2e.html#tabular)
  * [`thebibliography`](https://latexref.xyz/dev/latex2e.html#thebibliography)
  * [`theorem`](https://latexref.xyz/dev/latex2e.html#theorem)
  * [`titlepage`](https://latexref.xyz/dev/latex2e.html#titlepage)
  * [`verbatim`](https://latexref.xyz/dev/latex2e.html#verbatim)
  * [`verse`](https://latexref.xyz/dev/latex2e.html#verse)

### 8.1 `abstract`
Synopsis:
```
\begin{abstract}
...
\end{abstract}

```

Produce an abstract, possibly of multiple paragraphs. This environment is only defined in the `article` and `report` document classes (see [Document classes](https://latexref.xyz/dev/latex2e.html#Document-classes)).
Using the example below in the `article` class produces a displayed paragraph. Document class option `titlepage` causes the abstract to be on a separate page (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)); this is the default only in the `report` class.
```
\begin{abstract}
  We compare all known accounts of the proposal made by Porter Alexander
  to Robert E Lee at the Appomattox Court House that the army continue
  in a guerrilla war, which Lee refused.
\end{abstract}

```

The next example produces a one column abstract in a two column document (for a more flexible solution, use the package `abstract`).
```
\documentclass[twocolumn]{article}
  ...
\begin{document}
\title{Babe Ruth as Cultural Progenitor: a Atavistic Approach}
\author{Smith \\ Jones \\ Robinson\thanks{Railroad tracking grant.}}
\twocolumn[
  \begin{@twocolumnfalse}
     \maketitle
     \begin{abstract}
       Ruth was not just the Sultan of Swat, he was the entire swat
       team.
     \end{abstract}
   \end{@twocolumnfalse}
   ]
{   % by-hand insert a footnote at page bottom
 \renewcommand{\thefootnote}{\fnsymbol{footnote}}
 \footnotetext[1]{Thanks for all the fish.}
}

```

### 8.2 `array`
Synopsis:
```
\begin{array}{cols}
  column 1 entry &column 2 entry ... &column n entry \\
  ...
\end{array}

```

or:
```
\begin{array}[pos]{cols}
  column 1 entry &column 2 entry ... &column n entry \\
  ...
\end{array}

```

Produce a mathematical array. This environment can only be used in math mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)), and normally appears within a displayed mathematics environment such as `equation` (see [`equation`](https://latexref.xyz/dev/latex2e.html#equation)). Inside of each row the column entries are separated by an ampersand, (`&`). Rows are terminated with double-backslashes (see [`\\`](https://latexref.xyz/dev/latex2e.html#g_t_005c_005c)).
This example shows a three by three array.
```
\begin{displaymath}
  \chi(x) =
  \left|              % vertical bar fence
    \begin{array}{ccc}
      x-a  &-b  &-c  \\
      -d   &x-e &-f  \\
      -g   &-h  &x-i
    \end{array}
 \right|
\end{displaymath}

```

The required argument cols describes the number of columns, their alignment, and the formatting of the intercolumn regions. For instance, `\begin{array}{rcl}...\end{array}` gives three columns: the first flush right, the second centered, and the third flush left. See [`tabular`](https://latexref.xyz/dev/latex2e.html#tabular) for the complete description of cols and of the other common features of the two environments, including the optional pos argument.
There are two ways that `array` diverges from `tabular`. The first is that `array` entries are typeset in math mode, in textstyle (see [Math styles](https://latexref.xyz/dev/latex2e.html#Math-styles)) except if the cols definition specifies the column with `p{...}`, which causes the entry to be typeset in text mode. The second is that, instead of `tabular`’s parameter `\tabcolsep`, LaTeX’s intercolumn space in an `array` is governed by  `\arraycolsep`, which gives half the width between columns. The default for this is ‘5pt’ so that between two columns comes 10pt of space.
To obtain arrays with braces the standard is to use the `amsmath` package. It comes with environments `pmatrix` for an array surrounded by parentheses `(...)`, `bmatrix` for an array surrounded by square brackets `[...]`, `Bmatrix` for an array surrounded by curly braces `{...}`, `vmatrix` for an array surrounded by vertical bars `|...|`, and `Vmatrix` for an array surrounded by double vertical bars `||...||`, along with a number of other array constructs.
The next example uses the `amsmath` package.
```
\usepackage{amsmath}  % in preamble

\begin{equation}
  \begin{vmatrix}{cc}  % array with vert lines
    a  &b \\
    c  &d
  \end{vmatrix}=ad-bc
\end{equation}

```

There are many packages concerning arrays. The `array` package has many useful extensions, including more column types. The `dcolumn` package adds a column type to center on a decimal point. For both see the documentation on CTAN.
### 8.3 `center`
Synopsis:
```
\begin{center}
  line1 \\
  line2 \\
  ...
\end{center}

```

Create a new paragraph consisting of a sequence of lines that are centered within the left and right margins. Use double-backslash, `\\`, to get a line break (see [`\\`](https://latexref.xyz/dev/latex2e.html#g_t_005c_005c)).  If some text is too long to fit on a line then LaTeX will insert line breaks that avoid hyphenation and avoid stretching or shrinking any interword space.
This environment inserts space above and below the text body. See [`\centering`](https://latexref.xyz/dev/latex2e.html#g_t_005ccentering) to avoid such space, for example inside a `figure` environment.
This example produces three centered lines. There is extra vertical space between the last two lines.
```
\begin{center}
  A Thesis Submitted in Partial Fufillment \\
  of the Requirements of \\[0.5ex]
  the School of Environmental Engineering
\end{center}

```

In this example, depending on the page’s line width, LaTeX may choose a line break for the part before the double backslash. If so, it will center each of the two lines and if not it will center the single line. Then LaTeX will break at the double backslash, and will center the ending.
```
\begin{center}
  My father considered that anyone who went to chapel and didn't drink
  alcohol was not to be tolerated.\\
  I grew up in that belief.  ---Richard Burton
\end{center}

```

A double backslash after the final line is optional. If present it doesn’t add any vertical space.
In a two-column document the text is centered in a column, not in the entire page.
  * [`\centering`](https://latexref.xyz/dev/latex2e.html#g_t_005ccentering)

#### 8.3.1 `\centering`
Synopsis:
```
{\centering ... }

```

or
```
\begin{group}
  \centering ...
\end{group}

```

Center the material in its scope. It is most often used inside an environment such as `figure`, or in a `parbox`.
This example’s `\centering` declaration causes the graphic to be horizontally centered.
```
\begin{figure}
  \centering
  \includegraphics[width=0.6\textwidth]{ctan_lion.png}
  \caption{CTAN Lion}  \label{fig:CTANLion}
\end{figure}

```

The scope of this `\centering` ends with the `\end{figure}`.
Unlike the `center` environment, the `\centering` command does not add vertical space above and below the text. That’s its advantage in the above example; there is not an excess of space.
It also does not start a new paragraph; it simply changes how LaTeX formats paragraph units. If `ww {\centering xx \\ yy} zz` is surrounded by blank lines then LaTeX will create a paragraph whose first line ‘ww xx’ is centered and whose second line, not centered, contains ‘yy zz’. Usually what is desired is for the scope of the declaration to contain a blank line or the `\end` command of an environment such as `figure` or `table` that ends the paragraph unit. Thus, if `{\centering xx \\ yy\par} zz` is surrounded by blank lines then it makes a new paragraph with two centered lines ‘xx’ and ‘yy’, followed by a new paragraph with ‘zz’ that is formatted as usual.
### 8.4 `description`
Synopsis:
```
\begin{description}
  \item[label of first item] text of first item
  \item[label of second item] text of second item
   ...
\end{description}

```

Environment to make a list of labeled items. Each item’s label is typeset in bold and is flush left, so that long labels continue into the first line of the item text. There must be at least one item; having none causes the LaTeX error ‘Something's wrong--perhaps a missing \item’.
This example shows the environment used for a sequence of definitions.
```
\begin{description}
  \item[lama] A priest.
  \item[llama] A beast.
\end{description}

```

The labels ‘lama’ and ‘llama’ are output in boldface, with the left edge on the left margin.
Start list items with the `\item` command (see [`\item`: An entry in a list](https://latexref.xyz/dev/latex2e.html#g_t_005citem)). Use the optional labels, as in `\item[Main point]`, because there is no sensible default. Following the `\item` is optional text, which may contain multiple paragraphs.
Since the labels are in bold style, if the label text calls for a font change given in argument style (see [Font styles](https://latexref.xyz/dev/latex2e.html#Font-styles)) then it will come out bold. For instance, if the label text calls for typewriter with `\item[\texttt{label text}]` then it will appear in bold typewriter, if that is available. If you want to avoid this, and get non-bold typewriter, you can use declarative style: `\item[{\tt label text}]`. Similarly, you can get the standard roman font, instead of bold, with `\item[{\rm label text}]`.
For other major LaTeX list environments, see [`itemize`](https://latexref.xyz/dev/latex2e.html#itemize) and [`enumerate`](https://latexref.xyz/dev/latex2e.html#enumerate). Unlike those environments, nesting `description` environments does not change the default label; it is boldface and flush left at all levels.
For information about list layout parameters, including the default values, and for information about customizing list layout, see [`list`](https://latexref.xyz/dev/latex2e.html#list). The package `enumitem` is useful for customizing lists.
This example changes the description labels to small caps.
```
\renewcommand{\descriptionlabel}[1]{%
  {\hspace{\labelsep}\textsc{#1}}}

```

### 8.5 `displaymath`
Synopsis:
```
\begin{displaymath}
  mathematical text
\end{displaymath}

```

Environment to typeset the mathematical text on its own line, in display style and centered. To make the text be flush-left use the global option `fleqn` (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)).
In the `displaymath` environment no equation number is added to the math text. One way to get an equation number is to use the `equation` environment (see [`equation`](https://latexref.xyz/dev/latex2e.html#equation)).
LaTeX will not break the math text across lines.
The `amsmath` package defines an `equation*` environment which is functionally identical to `displaymath` but allows use of other `amsmath` facilities. In general, `amsmath`has significantly more extensive displayed equation facilities. For example, there are a number of ways in that package for having math text broken across lines; see also the `breqn` package for that (<https://ctan.org/pkg/breqn>).
The construct `\[ math \]` is a synonym for the environment `\begin{displaymath} math \end{displaymath}` but the latter is easier to work with in the source; for instance, searching for a square bracket may get false positives but the word `displaymath` will likely be unique.
The construct `$$math$$` from plain TeX is sometimes used instead of LaTeX’s `displaymath`. Although the output is similar, but is not officially supported in LaTeX at all; `$$` doesn’t support the `fleqn` option, has different vertical spacing, and doesn’t perform consistency checks.
The output from this example is centered and alone on its line.
```
\begin{displaymath}
  \int_1^2 x^2\,dx=7/3
\end{displaymath}

```

Also, the integral sign is larger than the inline version `\( \int_1^2 x^2\,dx=7/3 \)` produces.
### 8.6 `document`
The `document` environment encloses the entire body of a document. It is required in every LaTeX document. See [Starting and ending](https://latexref.xyz/dev/latex2e.html#Starting-and-ending).
  * [`\AtBeginDocument`](https://latexref.xyz/dev/latex2e.html#g_t_005cAtBeginDocument)
  * [`\AtEndDocument`](https://latexref.xyz/dev/latex2e.html#g_t_005cAtEndDocument)

#### 8.6.1 `\AtBeginDocument`
Synopsis:
```
\AtBeginDocument{code}

```

Save code and execute it when `\begin{document}` is executed, at the very end of the preamble. The code is executed after the font selection tables have been set up, so the normal font for the document is the current font. However, the code is executed as part of the preamble so you cannot do any typesetting with it.
You can issue this command more than once; the successive code lines will be executed in the order that you gave them.
#### 8.6.2 `\AtEndDocument`
Synopsis:
```
\AtEndDocument{code}

```

Save code and execute it near the end of the document. Specifically, it is executed when `\end{document}` is executed, before the final page is finished and before any leftover floating environments are processed. If you want some of the code to be executed after these two processes then include a `\clearpage` at the appropriate point in code.
You can issue this command more than once; the successive code lines will be executed in the order that you gave them.
### 8.7 `enumerate`
Synopsis:
```
\begin{enumerate}
  \item[optional label of first item] text of first item
  \item[optional label of second item] text of second item
  ...
\end{enumerate}

```

Environment to produce a numbered list of items. The format of the label numbering depends on the nesting level of this environment; see below. The default top-level numbering is ‘1.’, ‘2.’, etc. Each `enumerate` list environment must have at least one item; having none causes the LaTeX error ‘Something's wrong--perhaps a missing \item’.
This example gives the first two finishers in the 1908 Olympic marathon. As a top-level list the labels would come out as ‘1.’ and ‘2.’.
```
\begin{enumerate}
 \item Johnny Hayes (USA)
 \item Charles Hefferon (RSA)
\end{enumerate}

```

Start list items with the `\item` command (see [`\item`: An entry in a list](https://latexref.xyz/dev/latex2e.html#g_t_005citem)). If you give `\item` an optional argument by following it with square brackets, as in `\item[Interstitial label]`, then the next item will continue the interrupted sequence (see [`\item`: An entry in a list](https://latexref.xyz/dev/latex2e.html#g_t_005citem)). That is, you will get labels like ‘1.’, then ‘Interstitial label’, then ‘2.’. Following the `\item` is optional text, which may contain multiple paragraphs.
Enumerations may be nested within other `enumerate` environments, or within any paragraph-making environment such as `itemize` (see [`itemize`](https://latexref.xyz/dev/latex2e.html#itemize)), up to four levels deep. This gives LaTeX’s default for the format at each nesting level, where 1 is the top level, the outermost level.
  1. arabic number followed by a period: ‘1.’, ‘2.’, …
  2. lowercase letter inside parentheses: ‘(a)’, ‘(b)’ …
  3. lowercase roman numeral followed by a period: ‘i.’, ‘ii.’, …
  4. uppercase letter followed by a period: ‘A.’, ‘B.’, …

The `enumerate` environment uses the counters `\enumi` through `\enumiv` (see [Counters](https://latexref.xyz/dev/latex2e.html#Counters)).
For other major LaTeX labeled list environments, see [`description`](https://latexref.xyz/dev/latex2e.html#description) and [`itemize`](https://latexref.xyz/dev/latex2e.html#itemize). For information about list layout parameters, including the default values, and for information about customizing list layout, see [`list`](https://latexref.xyz/dev/latex2e.html#list). The package `enumitem` is useful for customizing lists.
To change the format of the label use `\renewcommand` (see [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand)) on the commands `\labelenumi` through `\labelenumiv`. For instance, this first level list will be labelled with uppercase letters, in boldface, and without a trailing period.
```
\renewcommand{\labelenumi}{\textbf{\Alph{enumi}}}
\begin{enumerate}
  \item Shows as boldface A
  \item Shows as boldface B
\end{enumerate}

```

For a list of counter-labeling commands see [`\alph \Alph \arabic \roman \Roman \fnsymbol`: Printing counters](https://latexref.xyz/dev/latex2e.html#g_t_005calph-_005cAlph-_005carabic-_005croman-_005cRoman-_005cfnsymbol).
### 8.8 `eqnarray`
The `eqnarray` environment is obsolete. It has infelicities, including spacing that is inconsistent with other mathematics elements. (See “Avoid eqnarray!” by Lars Madsen <https://tug.org/TUGboat/tb33-1/tb103madsen.pdf>). New documents should include the `amsmath` package and use the displayed mathematics environments provided there, such as the `align` environment. We include a description only for completeness and for working with old documents.
Synopsis:
```
\begin{eqnarray}
  first formula left  &first formula middle  &first formula right \\
  ...
\end{eqnarray}

```

or
```
\begin{eqnarray*}
  first formula left  &first formula middle  &first formula right \\
  ...
\end{eqnarray*}

```

Display a sequence of equations or inequalities. The left and right sides are typeset in display mode, while the middle is typeset in text mode.
It is similar to a three-column `array` environment, with items within a row separated by an ampersand (`&`), and with rows separated by double backslash `\\`).  The starred form of line break (`\\*`) can also be used to separate equations, and will disallow a page break there (see [`\\`](https://latexref.xyz/dev/latex2e.html#g_t_005c_005c)).
The unstarred form `eqnarray` places an equation number on every line (using the `equation` counter), unless that line contains a `\nonumber` command. The starred form `eqnarray*` omits equation numbering, while otherwise being the same.
The command `\lefteqn` is used for splitting long formulas across lines. It typesets its argument in display style flush left in a box of zero width.
This example shows three lines. The first two lines make an inequality, while the third line has not entry on the left side.
```
\begin{eqnarray*}
  \lefteqn{x_1+x_2+\cdots+x_n}     \\
    &\leq &y_1+y_2+\cdots+y_n      \\
    &=    &z+y_3+\cdots+y_n
\end{eqnarray*}

```

### 8.9 `equation`
Synopsis:
```
\begin{equation}
  mathematical text
\end{equation}

```

The same as a `displaymath` environment (see [`displaymath`](https://latexref.xyz/dev/latex2e.html#displaymath)) except that LaTeX puts an equation number flush to the right margin. The equation number is generated using the `equation` counter.
You should have no blank lines between `\begin{equation}` and `\end{equation}`, or LaTeX will tell you that there is a missing dollar sign.
The package `amsmath` package has extensive displayed equation facilities. New documents should include this package.
### 8.10 `figure`
Synopsis:
```
\begin{figure}[placement]
  figure body
  \caption[loftitle]{title}  % optional
  \label{label}              % optional
\end{figure}

```

or:
```
\begin{figure*}[placement]
  figure body
  \caption[loftitle]{title}  % optional
  \label{label}              % optional
\end{figure*}

```

Figures are for material that is not part of the normal text. An example is material that you cannot have split between two pages, such as a graphic. Because of this, LaTeX does not typeset figures in sequence with normal text but instead “floats” them to a convenient place, such as the top of a following page (see [Floats](https://latexref.xyz/dev/latex2e.html#Floats)).
The figure body can consist of imported graphics (see [Graphics](https://latexref.xyz/dev/latex2e.html#Graphics)), or text, LaTeX commands, etc. It is typeset in a `parbox` of width `\textwidth`.
The possible values of placement are `h` for ‘here’, `t` for ‘top’, `b` for ‘bottom’, and `p` for ‘on a separate page of floats’. For the effect of these options on the float placement algorithm, see [Floats](https://latexref.xyz/dev/latex2e.html#Floats).
The starred form `figure*` is used when a document is in double-column mode (see [`\twocolumn`](https://latexref.xyz/dev/latex2e.html#g_t_005ctwocolumn)). It produces a figure that spans both columns, at the top of the page. To add the possibility of placing at a page bottom see the discussion of placement `b` in [Floats](https://latexref.xyz/dev/latex2e.html#Floats).
The label is optional; it is used for cross references (see [Cross references](https://latexref.xyz/dev/latex2e.html#Cross-references)). The optional `\caption` command specifies caption text for the figure (see [`\caption`](https://latexref.xyz/dev/latex2e.html#g_t_005ccaption)). By default it is numbered. If loftitle is present, it is used in the list of figures instead of title (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)).
This example makes a figure out of a graphic. LaTeX will place that graphic and its caption at the top of a page or, if it is pushed to the end of the document, on a page of floats.
```
\usepackage{graphicx}  % in preamble
  ...
\begin{figure}[t]
  \centering
  \includegraphics[width=0.5\textwidth]{CTANlion.png}
  \caption{The CTAN lion, by Duane Bibby}
\end{figure}

```

### 8.11 `filecontents`
Synopsis:
```
\begin{filecontents}[option]{filename}
  text
\end{filecontents}

```

or
```
\begin{filecontents*}[option]{filename}
  text
\end{filecontents*}

```

Create a file named filename in the current directory (or the output directory, if specified; see [output directory](https://latexref.xyz/dev/latex2e.html#output-directory)) and write text to it. By default, an existing file is not overwritten.
The unstarred version of the environment `filecontents` prefixes the content of the created file with a header of TeX comments; see the example below. The starred version `filecontents*` does not include the header.
The possible options are:

`force`

`overwrite`

Overwrite an existing file.

`noheader`

Omit the header. Equivalent to using `filecontents*`.

`nosearch`

Only check the current directory (and the output directory, if specified) for an existing file, not the entire search path.
These options were added in a 2019 release of LaTeX.
This environment can be used anywhere in the preamble, although it often appears before the `\documentclass` command. It is commonly used to create a `.bib` or other such data file from the main document source, to make the source file self-contained. Similarly, it can be used to create a custom style or class file, again making the source self-contained.
For example, this document:
```
\documentclass{article}
\begin{filecontents}{JH.sty}
\newcommand{\myname}{Jim Hef{}feron}
\end{filecontents}
\usepackage{JH}
\begin{document}
Article by \myname.
\end{document}

```

produces this file JH.sty:
```
%% LaTeX2e file `JH.sty'
%% generated by the `filecontents' environment
%% from source `test' on 2015/10/12.
%%
\newcommand{\myname}{Jim Hef{}feron}

```

### 8.12 `flushleft`
Synopsis:
```
\begin{flushleft}
  line1 \\
  line2 \\
  ...
\end{flushleft}

```

An environment that creates a paragraph whose lines are flush to the left-hand margin, and ragged right. If you have lines that are too long then LaTeX will linebreak them in a way that avoids hyphenation and stretching or shrinking interword spaces. To force a new line use a double backslash, `\\`. For the declaration form see [`\raggedright`](https://latexref.xyz/dev/latex2e.html#g_t_005craggedright).
This creates a box of text that is at most 3 inches wide, with the text flush left and ragged right.
```
\noindent\begin{minipage}{3in}
\begin{flushleft}
  A long sentence that will be broken by \LaTeX{}
    at a convenient spot. \\
  And, a fresh line forced by the double backslash.
\end{flushleft}
\end{minipage}

```

  * [`\raggedright`](https://latexref.xyz/dev/latex2e.html#g_t_005craggedright)

#### 8.12.1 `\raggedright`
Synopses:
```
{\raggedright  ... }

```

or
```
\begin{environment} \raggedright
  ...
\end{environment}

```

A declaration which causes lines to be flush to the left margin and ragged right. It can be used inside an environment such as `quote` or in a `parbox`. For the environment form see [`flushleft`](https://latexref.xyz/dev/latex2e.html#flushleft).
Unlike the `flushleft` environment, the `\raggedright` command does not start a new paragraph; it only changes how LaTeX formats paragraph units. To affect a paragraph unit’s format, the scope of the declaration must contain the blank line or `\end` command that ends the paragraph unit.
Here `\raggedright` in each second column keeps LaTeX from doing awkward typesetting to fit the text into the narrow column. Note that `\raggedright` is inside the curly braces `{...}` to delimit its effect.
```
\begin{tabular}{rp{2in}}
  Team alpha  &{\raggedright This team does all the real work.} \\
  Team beta   &{\raggedright This team ensures that the water
                cooler is never empty.}                         \\
\end{tabular}

```

### 8.13 `flushright`
```
\begin{flushright}
  line1 \\
  line2 \\
  ...
\end{flushright}

```

An environment that creates a paragraph whose lines are flush to the right-hand margin and ragged left. If you have lines that are too long to fit the margins then LaTeX will linebreak them in a way that avoids hyphenation and stretching or shrinking inter-word spaces. To force a new line use a double backslash, `\\`. For the declaration form see [`\raggedleft`](https://latexref.xyz/dev/latex2e.html#g_t_005craggedleft).
For an example related to this environment, see [`flushleft`](https://latexref.xyz/dev/latex2e.html#flushleft), where one just have mutatis mutandis to replace `flushleft` by `flushright`.
  * [`\raggedleft`](https://latexref.xyz/dev/latex2e.html#g_t_005craggedleft)

#### 8.13.1 `\raggedleft`
Synopses:
```
{\raggedleft  ... }

```

or
```
\begin{environment} \raggedleft
  ...
\end{environment}

```

A declaration which causes lines to be flush to the right margin and ragged left. It can be used inside an environment such as `quote` or in a `parbox`. For the environment form see [`flushright`](https://latexref.xyz/dev/latex2e.html#flushright).
Unlike the `flushright` environment, the `\raggedleft` command does not start a new paragraph; it only changes how LaTeX formats paragraph units. To affect a paragraph unit’s formatting, the scope of the declaration must contain the blank line or `\end` command that ends the paragraph unit.
See [`\raggedright`](https://latexref.xyz/dev/latex2e.html#g_t_005craggedright), for an example related to this environment; just replace `\raggedright` there by `\raggedleft`.
### 8.14 `itemize`
Synopsis:
```
\begin{itemize}
  \item[optional label of first item] text of first item
  \item[optional label of second item] text of second item
  ...
\end{itemize}

```

Produce an _unordered list_ , sometimes called a bullet list. There must be at least one `\item` within the environment; having none causes the LaTeX error ‘Something's wrong--perhaps a missing \item’.
This gives a two-item list:
```
\begin{itemize}
 \item Pencil and watercolor sketch by Cassandra
 \item Rice portrait
\end{itemize}

```

By default, in a top-level list each label would come out as a bullet, •. The format of the labeling depends on the nesting level; see below.
Many language adaptations change list formatting, in which case this section may apply only partially or not at all. For instance, after this:
```
\usepackage[french]{babel} % changes list formatting!

```

the margins are smaller and the item markers are different.
Start list items with the `\item` command (see [`\item`: An entry in a list](https://latexref.xyz/dev/latex2e.html#g_t_005citem)). If you give `\item` an optional argument by following it with square brackets, as in `\item[Optional label]`, then by default Optional label will appear in bold and be flush right, so it could extend into the left margin. For labels that are flush left see the [`description`](https://latexref.xyz/dev/latex2e.html#description) environment. Following the `\item` is the text of the item, which may be empty or contain multiple paragraphs.
Unordered lists can be nested within one another, up to four levels deep. They can also be nested within other paragraph-making environments, such as `enumerate` (see [`enumerate`](https://latexref.xyz/dev/latex2e.html#enumerate)).
The `itemize` environment uses the commands `\labelitemi` through `\labelitemiv` to produce the default label (note the convention of lowercase roman numerals at the end of the command names that signify the nesting level). These are the default marks at each level.
  1. • (bullet, from `\textbullet`)
  2. **–** (bold en-dash, from `\normalfont\bfseries\textendash`)
  3. * (asterisk, from `\textasteriskcentered`)
  4. _\cdot_ (vertically centered dot, rendered here as a period, from `\textperiodcentered`)

Change the labels with `\renewcommand`. For instance, this makes the first level use diamonds.
```
\renewcommand{\labelitemi}{$\diamond$}

```

The distance between the left margin of the enclosing environment and the left margin of the `itemize` list is determined by the parameters `\leftmargini` through `\leftmarginvi`. (This also uses the convention of using lowercase roman numerals at the end of the command name to denote the nesting level.) The defaults are: `2.5em` in level 1 (`2em` in two-column mode), `2.2em` in level 2, `1.87em` in level 3, and `1.7em` in level 4, with smaller values for more deeply nested levels. The margin parameters must be overridden before the list starts.
For other major LaTeX labeled list environments, see [`description`](https://latexref.xyz/dev/latex2e.html#description) and [`enumerate`](https://latexref.xyz/dev/latex2e.html#enumerate). The `itemize`, `enumerate` and `description` environment use the same list layout parameters. For a description, including the default values, and for information about customizing list layout, see [`list`](https://latexref.xyz/dev/latex2e.html#list). The package `enumitem` is useful for customizing lists.
This example greatly reduces the margin space for an outermost itemized list:
```
\setlength{\leftmargini}{1.25em} % default 2.5em
\begin{itemize}
\item ...
\end{itemize}

```

Especially for lists with short items, it may be desirable to elide space between items. Here is an example defining an `itemize*` environment with no extra spacing between items, or between paragraphs within a single item (`\parskip` is not list-specific; see [`\parindent` & `\parskip`](https://latexref.xyz/dev/latex2e.html#g_t_005cparindent-_0026-_005cparskip)):
```
\newenvironment{itemize*}%
  {\begin{itemize}%
    \setlength{\itemsep}{0pt}%
    \setlength{\parsep}{0pt}%
    \setlength{\parskip}{0pt}%
  }%
  {\end{itemize}}

```
