# 05 Layout

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 5.1 `\onecolumn`
- 5.2 `\twocolumn`
- 5.3 `\flushbottom`
- 5.4 `\raggedbottom`
- 5.5 Page layout parameters
- 5.6 `\baselineskip` & `\baselinestretch`
- 5.7 Floats

## 5 Layout
Commands for controlling the general page layout.
  * [`\onecolumn`](https://latexref.xyz/dev/latex2e.html#g_t_005conecolumn)
  * [`\twocolumn`](https://latexref.xyz/dev/latex2e.html#g_t_005ctwocolumn)
  * [`\flushbottom`](https://latexref.xyz/dev/latex2e.html#g_t_005cflushbottom)
  * [`\raggedbottom`](https://latexref.xyz/dev/latex2e.html#g_t_005craggedbottom)
  * [Page layout parameters](https://latexref.xyz/dev/latex2e.html#Page-layout-parameters)
  * [`\baselineskip` & `\baselinestretch`](https://latexref.xyz/dev/latex2e.html#g_t_005cbaselineskip-_0026-_005cbaselinestretch)
  * [Floats](https://latexref.xyz/dev/latex2e.html#Floats)

### 5.1 `\onecolumn`
Synopsis:
```
\onecolumn

```

Start a new page and produce single-column output. If the document is given the class option `onecolumn` then this is the default behavior (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)). This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
### 5.2 `\twocolumn`
Synopses:
```
\twocolumn
\twocolumn[prelim one column text]

```

Start a new page and produce two-column output. If the document is given the class option `twocolumn` then this is the default (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)). This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
If the optional prelim one column text argument is present, it is typeset in one-column mode before the two-column typesetting starts.
These parameters control typesetting in two-column output:

`\columnsep`

The distance between columns. The default is 35pt. Change it with a command such as `\setlength{\columnsep}{40pt}`. You must change it before the two column mode starts; in the preamble is a good place.

`\columnseprule`

The width of the rule between columns. The default is 0pt, meaning that there is no rule. Otherwise, the rule appears halfway between the two columns. Change it with a command such as `\setlength{\columnseprule}{0.4pt}`, before the two-column mode starts.

`\columnwidth`

The width of a single column. In one-column mode this is equal to `\textwidth`. In two-column mode by default LaTeX sets the width of each of the two columns, `\columnwidth`, to be half of `\textwidth` minus `\columnsep`.
In a two-column document, the starred environments `table*` and `figure*` are two columns wide, whereas the unstarred environments `table` and `figure` take up only one column (see [`figure`](https://latexref.xyz/dev/latex2e.html#figure) and see [`table`](https://latexref.xyz/dev/latex2e.html#table)). LaTeX places starred floats at the top of a page. The following parameters control float behavior of two-column output.

`\dbltopfraction`

The maximum fraction at the top of a two-column page that may be occupied by two-column wide floats. The default is 0.7, meaning that the height of a `table*` or `figure*` environment must not exceed `0.7\textheight`. If the height of your starred float environment exceeds this then you can take one of the following actions to prevent it from floating all the way to the back of the document:
  * Use the `[tp]` location specifier to tell LaTeX to try to put the bulky float on a page by itself, as well as at the top of a page.
  * Use the `[t!]` location specifier to override the effect of `\dbltopfraction` for this particular float.
  * Increase the value of `\dbltopfraction` to a suitably large number, to avoid going to float pages so soon.

You can redefine it, as with `\renewcommand{\dbltopfraction}{0.9}`.

`\dblfloatpagefraction`

For a float page of two-column wide floats, this is the minimum fraction that must be occupied by floats, limiting the amount of blank space. LaTeX’s default is `0.5`. Change it with `\renewcommand`.

`\dblfloatsep`

On a float page of two-column wide floats, this length is the distance between floats, at both the top and bottom of the page. The default is `12pt plus2pt minus2pt` for a document set at `10pt` or `11pt`, and `14pt plus2pt minus4pt` for a document set at `12pt`.

`\dbltextfloatsep`

This length is the distance between a multi-column float at the top or bottom of a page and the main text. The default is `20pt plus2pt minus4pt`.

`\dbltopnumber`

On a float page of two-column wide floats, this counter gives the maximum number of floats allowed at the top of the page. The LaTeX default is `2`.
This example uses `\twocolumn`’s optional argument of to create a title that spans the two-column article:
```
\documentclass[twocolumn]{article}
\newcommand{\authormark}[1]{\textsuperscript{#1}}
\begin{document}
\twocolumn[{% inside this optional argument goes one-column text
  \centering
  \LARGE The Title \\[1.5em]
  \large Author One\authormark{1},
         Author Two\authormark{2},
         Author Three\authormark{1} \\[1em]
  \normalsize
  \begin{tabular}{p{.2\textwidth}@{\hspace{2em}}p{.2\textwidth}}
    \authormark{1}Department one  &\authormark{2}Department two \\
     School one                   &School two
  \end{tabular}\\[3em] % space below title part
  }]

Two column text here.

```

### 5.3 `\flushbottom`
Make all pages in the document after this declaration have the same height, by stretching the vertical space where necessary to fill out the page. This is most often used when making two-sided documents since the differences in facing pages can be glaring.
If TeX cannot satisfactorily stretch the vertical space in a page then you get a message like ‘Underfull \vbox (badness 10000) has occurred while \output is active’. If you get that, one option is to change to `\raggedbottom` (see [`\raggedbottom`](https://latexref.xyz/dev/latex2e.html#g_t_005craggedbottom)). Alternatively, you can adjust the `textheight` to make compatible pages, or you can add some vertical stretch glue between lines or between paragraphs, as in `\setlength{\parskip}{0ex plus0.1ex}`. Your last option is to, in a final editing stage, adjust the height of individual pages (see [`\enlargethispage`](https://latexref.xyz/dev/latex2e.html#g_t_005cenlargethispage)).
The `\flushbottom` state is the default only if you select the `twocolumn` document class option (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)), and for indexes made using `makeidx`.
### 5.4 `\raggedbottom`
Make all later pages the natural height of the material on that page; no rubber vertical lengths will be stretched. Thus, in a two-sided document the facing pages may be different heights. This command can go at any point in the document body. See [`\flushbottom`](https://latexref.xyz/dev/latex2e.html#g_t_005cflushbottom).
This is the default unless you select the `twocolumn` document class option (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)).
### 5.5 Page layout parameters

`\columnsep`

`\columnseprule`

`\columnwidth`

The distance between the two columns, the width of a rule between the columns, and the width of the columns, when the document class option `twocolumn` is in effect (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)). See [`\twocolumn`](https://latexref.xyz/dev/latex2e.html#g_t_005ctwocolumn).

`\headheight`

Height of the box that contains the running head. The default in the `article`, `report`, and `book` classes is ‘12pt’, at all type sizes.

`\headsep`

Vertical distance between the bottom of the header line and the top of the main text. The default in the `article` and `report` classes is ‘25pt’. In the `book` class the default is: if the document is set at 10pt then it is ‘0.25in’, and at 11pt or 12pt it is ‘0.275in’.

`\footskip`

Distance from the baseline of the last line of text to the baseline of the page footer. The default in the `article` and `report` classes is ‘30pt’. In the `book` class the default is: when the type size is 10pt the default is ‘0.35in’, while at 11pt it is ‘0.38in’, and at 12pt it is ‘30pt’.

`\linewidth`

Width of the current line, decreased for each nested `list` (see [`list`](https://latexref.xyz/dev/latex2e.html#list)). That is, the nominal value for `\linewidth` is to equal `\textwidth` but for each nested list the `\linewidth` is decreased by the sum of that list’s `\leftmargin` and `\rightmargin` (see [`itemize`](https://latexref.xyz/dev/latex2e.html#itemize)).

`\marginparpush`

`\marginsep`

`\marginparwidth`

The minimum vertical space between two marginal notes, the horizontal space between the text body and the marginal notes, and the horizontal width of the notes.
Normally marginal notes appear on the outside of the page, but the declaration `\reversemarginpar` changes that (and `\normalmarginpar` changes it back).
The defaults for `\marginparpush` in both `book` and `article` classes are: ‘7pt’ if the document is set at 12pt, and ‘5pt’ if the document is set at 11pt or 10pt.
For `\marginsep`, in `article` class the default is ‘10pt’ except if the document is set at 10pt and in two-column mode where the default is ‘11pt’.
For `\marginsep` in `book` class the default is ‘10pt’ in two-column mode and ‘7pt’ in one-column mode.
For `\marginparwidth` in both `book` and `article` classes, in two-column mode the default is 60% of `\paperwidth − \textwidth`, while in one-column mode it is 50% of that distance.

`\oddsidemargin`

`\evensidemargin`

The `\oddsidemargin` length is the extra distance between the left side of the page and the text’s left margin, on odd-numbered pages when the document class option `twoside` is chosen and on all pages when `oneside` is in effect. When `twoside` is in effect, on even-numbered pages the extra distance on the left is `\evensidemargin`.
LaTeX’s default is that `\oddsidemargin` is 40% of the difference between `\paperwidth` and `\textwidth`, and `\evensidemargin` is the remainder.

`\paperheight`

The height of the paper, as distinct from the height of the print area. Normally set with a document class option, as in `\documentclass[a4paper]{article}` (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)).

`\paperwidth`

The width of the paper, as distinct from the width of the print area. Normally set with a document class option, as in `\documentclass[a4paper]{article}` (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)).

`\textheight`

The normal vertical height of the page body. If the document is set at a nominal type size of 10pt then for an `article` or `report` the default is ‘43\baselineskip’, while for a `book` it is ‘41\baselineskip’. At a type size of 11pt the default is ‘38\baselineskip’ for all document classes. At 12pt it is ‘36\baselineskip’ for all classes.

`\textwidth`

The full horizontal width of the entire page body. For an `article` or `report` document, the default is ‘345pt’ when the chosen type size is 10pt, the default is ‘360pt’ at 11pt, and it is ‘390pt’ at 12pt. For a `book` document, the default is ‘4.5in’ at a type size of 10pt, and ‘5in’ at 11pt or 12pt.
In multi-column output, `\textwidth` remains the width of the entire page body, while `\columnwidth` is the width of one column (see [`\twocolumn`](https://latexref.xyz/dev/latex2e.html#g_t_005ctwocolumn)).
In lists (see [`list`](https://latexref.xyz/dev/latex2e.html#list)), `\textwidth` remains the width of the entire page body (and `\columnwidth` the width of the entire column), while `\linewidth` may decrease for nested lists.
Inside a minipage (see [`minipage`](https://latexref.xyz/dev/latex2e.html#minipage)) or `\parbox` (see [`\parbox`](https://latexref.xyz/dev/latex2e.html#g_t_005cparbox)), all the width-related parameters are set to the specified width, and revert to their normal values at the end of the `minipage` or `\parbox`.

`\hsize`

This entry is included for completeness: `\hsize` is the TeX primitive parameter used when text is broken into lines. It should not be used in normal LaTeX documents.

`\topmargin`

Space between the top of the TeX page (one inch from the top of the paper, by default) and the top of the header. The value is computed based on many other parameters: `\paperheight − 2in − \headheight − \headsep − \textheight − \footskip`, and then divided by two.

`\topskip`

Minimum distance between the top of the page body and the baseline of the first line of text. For the standard classes, the default is the same as the font size, e.g., ‘10pt’ at a type size of 10pt.
### 5.6 `\baselineskip` & `\baselinestretch`
The `\baselineskip` is a rubber length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). It gives the _leading_ , the normal distance between lines in a paragraph, from baseline to baseline.
Ordinarily document authors do not directly change `\baselineskip` while writing. Instead, it is set by the low level font selection command `\fontsize` (see [low level font commands fontsize](https://latexref.xyz/dev/latex2e.html#low-level-font-commands-fontsize)). The `\baselineskip`’s value is reset every time a font change happens and so any direct change to `\baselineskip` would vanish the next time there was a font switch. For how to influence line spacing, see the discussion of `\baselinestretch` below.
Usually, a font’s size and baseline skip is assigned by the font designer. These numbers are nominal in the sense that if, for instance, a font’s style file has the command `\fontsize{10pt}{12pt}` then that does not mean that the characters in the font are 10pt tall; for instance, parentheses and accented capitals may be taller. Nor does it mean that if the lines are spaced less than 12pt apart then they risk touching. Rather these numbers are typographic judgements. (Often, the `\baselineskip` is about twenty percent larger than the font size.)
The `\baselineskip` is not a property of each line but of the entire paragraph. As a result, large text in the middle of a paragraph, such as a single `{\Huge Q}`, will be squashed into its line. TeX will make sure it doesn’t scrape up against the line above but won’t change the `\baselineskip` for that one line to make extra room above. For the fix, use a `\strut` (see [`\strut`](https://latexref.xyz/dev/latex2e.html#g_t_005cstrut)).
The value of `\baselineskip` that TeX uses for the paragraph is the value in effect at the blank line or command that ends the paragraph unit. So if a document contains this paragraph then its lines will be scrunched together, compared to lines in surrounding paragraphs.
```
Many people see a page break between text and a displayed equation as
bad style, so in effect the display is part of the paragraph.
Because this display is in footnotesize, the entire paragraph has the
baseline spacing matching that size.
{\footnotesize $$a+b = c$$}

```

The process for making paragraphs is that when a new line is added, if the depth of the previous line plus the height of the new line is less than `\baselineskip` then TeX inserts vertical glue to make up the difference. There are two fine points. The first is that if the lines would be too close together, closer than `\lineskiplimit`, then TeX instead uses `\lineskip` as the interline glue. The second is that TeX doesn’t actually use the depth of the previous line. Instead it uses `\prevdepth`, which usually contains that depth. But at the beginning of the paragraph (or any vertical list) or just after a rule, `\prevdepth` has the value -1000pt and this special value tells TeX not to insert any interline glue at the paragraph start.
In the standard classes `\lineskiplimit` is 0pt and `\lineskip` is 1pt. By the prior paragraph then, the distance between lines can approach zero but if it becomes zero (or less than zero) then the lines jump to 1pt apart.
Sometimes authors must, for editing purposes, put the document in double space or one-and-a-half space. The right way to influence the interline distance is via `\baselinestretch`. It scales `\baselineskip`, and has a default value of 1.0. It is a command, not a length, and does not take effect until a font change happens, so set the scale factor like this: `\renewcommand{\baselinestretch}{1.5}\selectfont`.
The most straightforward way to change the line spacing for an entire document is to put `\linespread{factor}` in the preamble. For double spacing, take factor to be 1.6 and for one-and-a-half spacing use 1.3. These numbers are rough: for instance, since the `\baselineskip` is about 1.2 times the font size, multiplying by 1.6 gives a baseline skip to font size ratio of about 2. (The `\linespread` command is defined as `\renewcommand{\baselinestretch}{factor}` so it also won’t take effect until a font setting happens. But that always takes place at the start of a document, so there you don’t need to follow it with `\selectfont`.)
A simpler approach is the `setspace` package. The basic example:
```
\usepackage{setspace}
\doublespacing  % or \onehalfspacing for 1.5

```

In the preamble these will start the document off with that sizing. But you can also use these declarations in the document body to change the spacing from that point forward, and consequently there is `\singlespacing` to return the spacing to normal. In the document body, a better practice than using the declarations is to use environments, such as `\begin{doublespace} ... \end{doublespace}`. The package also has commands to do arbitrary spacing: `\setstretch{factor}` and `\begin{spacing}{factor} ... \end{spacing}`. This package also keeps the line spacing single-spaced in places where that is typically desirable, such as footnotes and figure captions. See the package documentation.
### 5.7 Floats
Some typographic elements, such as figures and tables, cannot be broken across pages. They must be typeset outside of the normal flow of text, for instance floating to the top of a later page.
LaTeX can have a number of different classes of floating material. The default is the two classes, `figure` (see [`figure`](https://latexref.xyz/dev/latex2e.html#figure)) and `table` (see [`table`](https://latexref.xyz/dev/latex2e.html#table)), but you can create a new class with the package `float`.
Within any one float class LaTeX always respects the order, so that the first figure in a document source must be typeset before the second figure. However, LaTeX may mix the classes, so it can happen that while the first table appears in the source before the first figure, it appears in the output after it.
The placement of floats is subject to parameters, given below, that limit the number of floats that can appear at the top of a page, and the bottom, etc. If so many floats are queued that the limits prevent them all from fitting on a page then LaTeX places what it can and defers the rest to the next page. In this way, floats may end up being typeset far from their place in the source. In particular, a float that is big may migrate to the end of the document. In which event, because all floats in a class must appear in sequential order, every following float in that class also appears at the end.
In addition to changing the parameters, for each float you can tweak where the float placement algorithm tries to place it by using its placement argument. The possible values are a sequence of the letters below. The default for both `figure` and `table`, in both `article` and `book` classes, is `tbp`.

`t`

(Top)—at the top of a text page.

`b`

(Bottom)—at the bottom of a text page. (However, `b` is not allowed for full-width floats (`figure*`) with double-column output. To ameliorate this, use the stfloats or dblfloatfix package, but see the discussion at caveats in the FAQ: <https://www.texfaq.org/FAQ-2colfloat>.

`h`

(Here)—at the position in the text where the `figure` environment appears. However, `h` is not allowed by itself; `t` is automatically added.
To absolutely force a float to appear “here”, you can `\usepackage{float}` and use the `H` specifier which it defines. For further discussion, see the FAQ entry at <https://www.texfaq.org/FAQ-figurehere>.

`p`

(Page of floats)—on a separate _float page_ , which is a page containing no text, only floats.

`!`

Used in addition to one of the above; for this float only, LaTeX ignores the restrictions on both the number of floats that can appear and the relative amounts of float and non-float text on the page. The `!` specifier does _not_ mean “put the float here”; see above.
Note: the order in which letters appear in the placement argument does not change the order in which LaTeX tries to place the float; for instance, `btp` has the same effect as `tbp`. All that placement does is that if a letter is not present then the algorithm does not try that location. Thus, LaTeX’s default of `tbp` is to try every location except placing the float where it occurs in the source.
To prevent LaTeX from moving floats to the end of the document or a chapter you can use a `\clearpage` command to start a new page and insert all pending floats. If a pagebreak is undesirable then you can use the afterpage package and issue `\afterpage{\clearpage}`. This will wait until the current page is finished and then flush all outstanding floats.
LaTeX can typeset a float before where it appears in the source (although on the same output page) if there is a `t` specifier in the placement parameter. If this is not desired, and deleting the `t` is not acceptable as it keeps the float from being placed at the top of the next page, then you can prevent it by either using the `flafter` package or using the command  `\suppressfloats[t]`, which causes floats for the top position on this page to moved to the next page.
Parameters relating to fractions of pages occupied by float and non-float text (change them with `\renewcommand{parameter}{decimal between 0 and 1}`):

`\bottomfraction`

The maximum fraction of the page allowed to be occupied by floats at the bottom; default ‘.3’.

`\floatpagefraction`

The minimum fraction of a float page that must be occupied by floats; default ‘.5’.

`\textfraction`

Minimum fraction of a page that must be text; if floats take up too much space to preserve this much text, floats will be moved to a different page. The default is ‘.2’.

`\topfraction`

Maximum fraction at the top of a page that may be occupied before floats; default ‘.7’.
Parameters relating to vertical space around floats (change them with a command of the form `\setlength{parameter}{length expression}`):

`\floatsep`

Space between floats at the top or bottom of a page; default ‘12pt plus2pt minus2pt’.

`\intextsep`

Space above and below a float in the middle of the main text; default ‘12pt plus2pt minus2pt’ for 10 point and 11 point documents, and ‘14pt plus4pt minus4pt’ for 12 point documents.

`\textfloatsep`

Space between the last (first) float at the top (bottom) of a page; default ‘20pt plus2pt minus4pt’.
Counters relating to the number of floats on a page (change them with a command of the form `\setcounter{ctrname}{natural number}`):

`bottomnumber`

Maximum number of floats that can appear at the bottom of a text page; default 1.

`dbltopnumber`

Maximum number of full-sized floats that can appear at the top of a two-column page; default 2.

`topnumber`

Maximum number of floats that can appear at the top of a text page; default 2.

`totalnumber`

Maximum number of floats that can appear on a text page; default 3.
The principal TeX FAQ entry relating to floats <https://www.texfaq.org/FAQ-floats> contains suggestions for relaxing LaTeX’s default parameters to reduce the problem of floats being pushed to the end. A full explanation of the float placement algorithm is in Frank Mittelbach’s article “How to influence the position of float environments like figure and table in LaTeX?” (<https://www.latex-project.org/publications/2014-FMi-TUB-tb111mitt-float-placement.pdf>).
  * [`\caption`](https://latexref.xyz/dev/latex2e.html#g_t_005ccaption)

#### 5.7.1 `\caption`
Synopsis:
```
\caption{caption-text}

```

or
```
\caption[short-caption-text]{caption-text}

```

Make a caption for a floating environment, such as a `figure` or `table` environment (see [`figure`](https://latexref.xyz/dev/latex2e.html#figure) or [`table`](https://latexref.xyz/dev/latex2e.html#table)).
In this example, LaTeX places a caption below the vertical blank space that is left by the author for the later inclusion of a picture.
```
\begin{figure}
  \vspace*{1cm}
  \caption{Alonzo Cushing, Battery A, 4th US Artillery.}
  \label{fig:CushingPic}
\end{figure}

```

The `\caption` command will label the caption-text with something like ‘Figure 1:’ for an article or ‘Figure 1.1:’ for a book. The text is centered if it is shorter than the text width, or set as an unindented paragraph if it takes more than one line.
In addition to placing the caption-text in the output, the `\caption` command also saves that information for use in a list of figures or list of tables (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)).
Here the `\caption` command uses the optional short-caption-text, so that the shorter text appears in the list of tables, rather than the longer caption-text.
```
\begin{table}
  \centering
  \begin{tabular}{|*{3}{c}|}
    \hline
    4  &9  &2 \\
    3  &5  &7 \\
    8  &1  &6 \\
    \hline
  \end{tabular}
  \caption[\textit{Lo Shu} magic square]{%
    The \textit{Lo Shu} magic square, which is unique among
    squares of order three up to rotation and reflection.}
  \label{tab:LoShu}
\end{table}

```

LaTeX will label the caption-text with something like ‘Table 1:’ for an article or ‘Table 1.1:’ for a book.
The caption can appear at the top of the `figure` or `table`. For instance, that would happen in the prior example by putting the `\caption` between the `\centering` and the `\begin{tabular}`.
Different floating environments are numbered separately, by default. It is `\caption` that updates the counter, and so any `\label` must come after the `\caption`. The counter for the `figure` environment is named `figure`, and similarly the counter for the `table` environment is `table`.
The text that will be put in the list of figures or list of tables is moving argument. If you get the LaTeX error ‘! Argument of \@caption has an extra }’ then you must put `\protect` in front of any fragile commands. See [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect).
The `caption` package has many options to adjust how the caption appears, for example changing the font size, making the caption be hanging text rather than set as a paragraph, or making the caption always set as a paragraph rather than centered when it is short.
