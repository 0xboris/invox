# 25 Front/back matter: Contents, list of figures, and list of tables

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 25.1.1 `\@dottedtocline`
- 25.1.2 `\addcontentsline`
- 25.1.3 `\addtocontents`
- 25.1.4 `\contentsline`
- 25.1.5 `\nofiles`
- 25.1.6 `\numberline`

## 25 Front/back matter
  * [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)
  * [Indexes](https://latexref.xyz/dev/latex2e.html#Indexes)
  * [Glossaries](https://latexref.xyz/dev/latex2e.html#Glossaries)

### 25.1 Table of contents, list of figures, list of tables
Synopsis, one of:
```
\tableofcontents
\listoffigures
\listoftables

```

Produce a table of contents, or list of figures, or list of tables. Put the command in the input file where you want the table or list to go. You do not type the entries; for example, typically the table of contents entries are automatically generated from the sectioning commands `\chapter`, etc.
This example illustrates the first command, `\tableofcontents`. LaTeX will produce a table of contents on the book’s first page.
```
\documentclass{book}
% \setcounter{tocdepth}{1}
\begin{document}
\tableofcontents\newpage
  ...
\chapter{...}
  ...
\section{...}
  ...
\subsection{...}
  ...
\end{document}

```

Uncommenting the second line would cause that table to contain chapter and section listings but not subsection listings, because the `\section` command has level 1. See [Sectioning](https://latexref.xyz/dev/latex2e.html#Sectioning), for level numbers of the sectioning units. For more on the `tocdepth` see [Sectioning/tocdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002ftocdepth).
Another example of the use of `\tableofcontents` is in [Larger `book` template](https://latexref.xyz/dev/latex2e.html#Larger-book-template).
If you want a page break after the table of contents, write a `\newpage` command after the `\tableofcontents` command, as above.
To make the table of contents, LaTeX stores the information in an auxiliary file named root-file.toc (see [Splitting the input](https://latexref.xyz/dev/latex2e.html#Splitting-the-input)). For example, this LaTeX file test.tex
```
\documentclass{article}
\begin{document}
\tableofcontents\newpage
\section{First section}
\subsection{First subsection}
  ...

```

writes these lines to test.toc.
```
\contentsline {section}{\numberline {1}First section}{2}
\contentsline {subsection}{\numberline {1.1}First subsection}{2}

```

Each line contains a single command, `\contentsline` (see [`\contentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005ccontentsline)). The first argument, the `section` or `subsection`, is the sectioning unit. The second argument has two components. The hook `\numberline` determines how the sectioning number, `1` or `1.1`, appears in the table of contents (see [`\numberline`](https://latexref.xyz/dev/latex2e.html#g_t_005cnumberline)). The remainder of the second argument of `\contentsline`, ‘First section’ or ‘First subsection’, is the sectioning title text. Finally, the third argument, ‘2’, is the page number on which this sectioning unit starts.
To typeset these lines, the document class provides `\l@section-unit` commands such as `\l@section{text}{pagenumber}` and `\l@subsection{text}{pagenumber}`. These commands often use the `\@dottedtocline` command (see [`\@dottedtocline`](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040dottedtocline)).
A consequence of LaTeX’s strategy of using auxiliary files is that to get the correct information in the document you must run LaTeX twice, once to store the information and the second time to retrieve it. In the ordinary course of writing a document authors run LaTeX a number of times, but you may notice that the first time that you compile a new document, the table of contents page will be empty except for its ‘Contents’ header. Just run LaTeX again.
The commands `\listoffigures` and `\listoftables` produce a list of figures and a list of tables. Their information is stored in files with extension .lof and .lot. They work the same way as `\tableofcontents` but the latter is more common, so we use it for most examples.
You can manually add material to the table of contents, the list of figures, and the list of tables. For instance, add a line about a section to the table of contents with `\addcontentsline{toc}{section}{text}`. (see [`\addcontentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005caddcontentsline)). Add arbitrary material, that is, non-line material, with `\addtocontents`, as with the command `\addtocontents{lof}{\protect\vspace{2ex}}`, which adds vertical space to the list of figures (see [`\addtocontents`](https://latexref.xyz/dev/latex2e.html#g_t_005caddtocontents)).
Lines in the table of contents, the list of figures, and the list of tables, have four parts. First is an indent. Next is a box into which sectioning numbers are placed, and then the third box holds the title text, such as ‘First section’. Finally there is a box up against the right margin, inside of which LaTeX puts the page number box. For the indent and the width of the number box, see [`\@dottedtocline`](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040dottedtocline). The right margin box has width `\@tocrmarg` and the page number is flush right in that space, inside a box of width `\@pnumwidth`. By default `\@tocrmarg` is `2.55em` and `\@pnumwidth` is `1.55em`. Change these as with `\renewcommand{\@tocrmarg}{3.5em}`.
CTAN has many packages for the table of contents and lists of figures and tables (see [CTAN: The Comprehensive TeX Archive Network](https://latexref.xyz/dev/latex2e.html#CTAN)). The package `tocloft` is convenient for adjusting some aspects of the default such as spacing. And, `tocbibbind` will automatically add the bibliography, index, etc. to the table of contents.
To change the header for the table of contents page, do something like these commands before you call `\tableofcontents`, etc.
```
\renewcommand{\contentsname}{Table of Contents}
\renewcommand{\listfigurename}{Plots}
\renewcommand{\listtablename}{Specifications}

```

Internationalization packages such as `babel` or `polyglossia` will change these headers depending on the chosen base language.
  * [`\@dottedtocline`](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040dottedtocline)
  * [`\addcontentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005caddcontentsline)
  * [`\addtocontents`](https://latexref.xyz/dev/latex2e.html#g_t_005caddtocontents)
  * [`\contentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005ccontentsline)
  * [`\nofiles`](https://latexref.xyz/dev/latex2e.html#g_t_005cnofiles)
  * [`\numberline`](https://latexref.xyz/dev/latex2e.html#g_t_005cnumberline)

#### 25.1.1 `\@dottedtocline`
Synopsis:
```
\@dottedtocline{section-level-num}{indent}{numwidth}{text}{pagenumber}

```

Used internally by LaTeX to format an entry line in the table of contents, list of figures, or list of tables. Authors do not directly enter `\@dottedtocline` commands.
This command is typically used by `\l@section`, `\l@subsection`, etc., to format the content lines. For example, the article.cls file contains these definitions:
```
\newcommand*\l@section{\@dottedtocline{1}{1.5em}{2.3em}}
\newcommand*\l@subsection{\@dottedtocline{2}{3.8em}{3.2em}}
\newcommand*\l@subsubsection{\@dottedtocline{3}{7.0em}{4.1em}}

```

In this example, `\@dottedcline` appears to have been given only three arguments. But tracing the internal code shows that it picks up the final text and pagenumber arguments in the synopsis from a call to `\contentsline` (see [`\contentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005ccontentsline)).
Between the box for the title text of a section and the right margin box, these `\@dottedtocline` commands insert _leaders_ , that is, evenly-spaced dots. The dot-to-dot space is given by the command `\@dotsep`. By default it is 4.5 (it is in math units, aka. `mu`, which are `1/18` em. Change it using `\renewcommand`, as in `\renewcommand{\@dotsep}{3.5}`.
In the standard book class, LaTeX does not use dotted leaders for the Part and Chapter table entries, and in the standard article class it does not use dotted leaders for Section entries.
#### 25.1.2 `\addcontentsline`
Synopsis:
```
\addcontentsline{ext}{unit}{text}

```

Add an entry to the auxiliary file with extension ext.
The following will result in an ‘Appendices’ line in the table of contents.
```
\addcontentsline{toc}{section}{\protect\textbf{Appendices}}

```

It will appear at the same indentation level as the sections, will be in boldface, and will be assigned the page number associated with the point where the command appears in the input file.
The `\addcontentsline` command writes information to the file root-name.ext, where root-name is the file name of the root file (see [Splitting the input](https://latexref.xyz/dev/latex2e.html#Splitting-the-input)). It writes that information as the text of the command `\contentsline{unit}{text}{num}`, where `num` is the current value of counter `unit` (see [`\contentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005ccontentsline)). The most common case is the table of contents and there num is the page number of the first page of unit.
This command is invoked by the sectioning commands `\chapter`, etc. (see [Sectioning](https://latexref.xyz/dev/latex2e.html#Sectioning)), and also by `\caption` inside a float environment (see [Floats](https://latexref.xyz/dev/latex2e.html#Floats)). But it is also directly used by authors. For example, an author writing a book whose style is to have an unnumbered preface may use the starred `\chapter*`. But that command leaves out table of contents information, which can be entered manually, as here.
```
\chapter*{Preface}
\addcontentsline{toc}{chapter}{\protect\numberline{}Preface}

```

In the root-name.toc file LaTeX will put the line `\contentsline {chapter}{\numberline {}Preface}{3}`; note that the page number ‘3’ is automatically generated by the system, not entered manually.
All of the arguments for `\addcontentsline` are required.

ext

Typically one of the strings `toc` for the table of contents, `lof` for the list of figures, or `lot` for the list of tables. The filename extension of the information file.

unit

A string that depends on the value of the ext argument, typically one of:

`toc`

For the table of contents, this is the name of a sectional unit: `part`, `chapter`, `section`, `subsection`, etc.

`lof`

For the list of figures: `figure`.

`lot`

For the list of tables: `table`.

text

The text of the entry. You must `\protect` any fragile commands (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)) used in it.
The `\addcontentsline` command has an interaction with `\include` (see [`\include` & `\includeonly`](https://latexref.xyz/dev/latex2e.html#g_t_005cinclude-_0026-_005cincludeonly)). If you use them at the same level, as with `\addcontentsline{...}{...}{...}\include{...}` then lines in the table of contents can come out in the wrong order. The solution is to move `\addcontentsline` into the file being included.
If you use a unit that LaTeX does not recognize, as with the typo here
```
\addcontentsline{toc}{setcion}{\protect\textbf{Appendices}}

```

then you don’t get an error but the formatting in the table of contents will not make sense.
#### 25.1.3 `\addtocontents`
Synopsis:
```
\addtocontents{ext}{text}

```

Add text, which may be text or formatting commands, directly to the auxiliary file with extension ext. This is most commonly used for the table of contents so that is the discussion here, but it also applies to the list of figures and list of tables.
This will put some vertical space in the table of contents after the ‘Contents’ header.
```
\tableofcontents\newpage
\addtocontents{toc}{\protect\vspace*{3ex}}

```

This puts the word ‘Page’, in boldface, above the column of page numbers and after the header.
```
\tableofcontents
\addtocontents{toc}{~\hfill\textbf{Page}\par}
\chapter{...}

```

This adds a line announcing work by a new author.
```
\addtocontents{toc}{%
  \protect\vspace{2ex}
  \textbf{Chapters by N. Other Author}\par}

```

The difference between `\addtocontents` and `\addcontentsline` is that the latter is strictly for lines, such as with a line giving the page number for the start of a new subset of the chapters. As the above examples show, `\addtocontents` is for material such as spacing.
The `\addtocontents` command has two arguments. Both are required.

ext

Typically one of: toc for the table of contents, lof for the list of figures, or lot for the list of tables. The extension of the file holding the information.

text

The text, and possibly commands, to be written.
The sectioning commands such as `\chapter` use the `\addcontentsline` command to store information. This command creates lines in the .toc auxiliary file containing the `\contentsline` command (see [`\addcontentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005caddcontentsline)). In contrast, the command `\addtocontents` puts material directly in that file.
The `\addtocontents` command has an interaction with `\include` (see [`\include` & `\includeonly`](https://latexref.xyz/dev/latex2e.html#g_t_005cinclude-_0026-_005cincludeonly)). If you use them at the same level, as with `\addtocontents{...}{...}\include{...}` then lines in the table of contents can come out in the wrong order. The solution is to move `\addtocontents` into the file being included.
#### 25.1.4 `\contentsline`
Synopsis:
```
\contentsline{unit}{text}{pagenumber}

```

Used internally by LaTeX to typeset an entry of the table of contents, list of figures, or list of tables (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)). Authors do not directly enter `\contentsline` commands.
Usually adding material to these lists is done automatically by the commands `\chapter`, `\section`, etc. for the table of contents, or by the `\caption` command inside of a `\figure` or `\table` environment (see [`figure`](https://latexref.xyz/dev/latex2e.html#figure) and see [`table`](https://latexref.xyz/dev/latex2e.html#table)). Thus, where the root file is thesis.tex, and contains the declaration `\tableofcontents`, the command `\chapter{Chapter One}` produces something like this in the file thesis.toc.
```
\contentsline {chapter}{\numberline {1}Chapter One}{3}

```

If the file contains the declaration `\listoffigures` then a figure environment involving `\caption{Test}` will produce something like this in thesis.lof.
```
\contentsline {figure}{\numberline {1.1}{\ignorespaces Test}}{6}

```

To manually add material, use `\addcontentsline{filetype}{unit}{text}`, where filetype is `toc`, `lof`, or `lot` (see [`\addcontentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005caddcontentsline)).
For manipulating how the `\contentline` material is typeset, see the `tocloft` package.
Note that the `hyperref` package changes the definition of `\contentsline` (and `\addcontentsline`) to add more arguments, to make hyperlinks. This is the source of the error `Argument of \contentsline has an extra }` when one adds/remove the use of package `hyperref` and a compilation was already run. Fix this error by deleting the .toc or .lof or .lot file, and running LaTeX again.
#### 25.1.5 `\nofiles`
Synopsis:
```
\nofiles

```

Prevent LaTeX from writing any auxiliary files. The only output will be the .log and .pdf (or .dvi) files. This command must go in the preamble.
Because of the `\nofiles` command this example will not produce a .toc file.
```
\documentclass{book}
\nofiles
\begin{document}
\tableofcontents\newpage
\chapter{...}
  ...

```

LaTeX will not erase any existing auxiliary files, so if you insert the `\nofiles` command after you have run the file and gotten a .toc then the table of contents page will continue to show the old information.
#### 25.1.6 `\numberline`
Synopsis:
```
\numberline{number}

```

Typeset its argument flush left in a box. This is used in a `\contentsline` command to typeset the section number (see [`\contentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005ccontentsline)).
For example, this line in a .toc file causes the `1.1` to be typeset flush left.
```
\contentsline {subsection}{\numberline {1.1}Motivation}{2}

```

By default, LaTeX typesets the section numbers in a box of length `\@tempdima`. That length is set by the commands `\l@section`, `\l@subsection`, etc. Put section numbers inside a natural-width box with `\renewcommand{\numberline}[1]{#1~}` before `\tableofcontents`.
This command is fragile so you may need to precede it with `\protect` (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)). An example is the use of `\protect` in this command,
```
\addcontentsline{toc}{section}{\protect\numberline{}Summary}

```

to get the `\numberline` into the `\contentsline` command in the .toc file: `\contentsline {section}{\numberline {}Summary}{6}` (the page number ‘6’ is automatically added by LaTeX; see [`\addcontentsline`](https://latexref.xyz/dev/latex2e.html#g_t_005caddcontentsline)).
