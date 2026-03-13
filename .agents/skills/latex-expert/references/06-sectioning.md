# 06 Sectioning

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 6.1 `\part`
- 6.2 `\chapter`
- 6.3 `\section`
- 6.4 `\subsection`
- 6.5 `\subsubsection`, `\paragraph`, `\subparagraph`
- 6.6 `\appendix`
- 6.7 `\frontmatter`, `\mainmatter`, `\backmatter`
- 6.8 `\@startsection`: Typesetting sectional unit headings

## 6 Sectioning
Structure your text into divisions: parts, chapters, sections, etc. All sectioning commands have the same form, one of:
```
sectioning-command{title}
sectioning-command*{title}
sectioning-command[toc-title]{title}

```

For instance, declare the start of a subsection as with `\subsection{Motivation}`.
The table has each sectioning-command in LaTeX. All are available in all of LaTeX’s standard document classes `book`, `report`, and `article`, except that `\chapter` is not available in `article`.
Sectioning unit | Command | Level
---|---|---
Part | `\part` | -1 (`book`, `report`), 0 (`article`)
Chapter | `\chapter` | 0
Section | `\section` | 1
Subsection | `\subsection` | 2
Subsubsection | `\subsubsection` | 3
Paragraph | `\paragraph` | 4
Subparagraph | `\subparagraph` | 5
All these commands have a `*`-form that prints title as usual but does not number it and does not make an entry in the table of contents. An example of using this is for an appendix in an `article`. The input `\appendix\section{Appendix}` gives the output ‘A Appendix’ (see [`\appendix`](https://latexref.xyz/dev/latex2e.html#g_t_005cappendix)). You can lose the numbering ‘A’ by instead entering `\section*{Appendix}` (articles often omit a table of contents and have simple page headers so the other differences from the `\section` command may not matter).
The section title title provides the heading in the main text, but it may also appear in the table of contents and in the running head or foot (see [Page styles](https://latexref.xyz/dev/latex2e.html#Page-styles)). You may not want the same text in these places as in the main text. All of these commands have an optional argument toc-title for these other places.
The level number in the table above determines which sectional units are numbered, and which appear in the table of contents. If the sectioning command’s level is less than or equal to the value of the counter `secnumdepth` then the titles for this sectioning command will be numbered (see [Sectioning/secnumdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002fsecnumdepth)). And, if level is less than or equal to the value of the counter `tocdepth` then the table of contents will have an entry for this sectioning unit (see [Sectioning/tocdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002ftocdepth)).
LaTeX expects that before you have a `\subsection` you will have a `\section` and, in a `book` class document, that before a `\section` you will have a `\chapter`. Otherwise you can get something like a subsection numbered ‘3.0.1’.
LaTeX lets you change the appearance of the sectional units. As a simple example, you can change the section numbering to uppercase letters with this (in the preamble):
`\renewcommand\thesection{\Alph{section}}` . (See [`\alph \Alph \arabic \roman \Roman \fnsymbol`: Printing counters](https://latexref.xyz/dev/latex2e.html#g_t_005calph-_005cAlph-_005carabic-_005croman-_005cRoman-_005cfnsymbol).) CTAN has many packages that make this adjustment easier, notably `titlesec`.
Two counters relate to the appearance of headings made by sectioning commands.

`secnumdepth`

Controls which sectioning unit are numbered. Setting the counter with `\setcounter{secnumdepth}{level}` will suppress numbering of sectioning at any depth greater than level (see [`\setcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005csetcounter)). See the above table for the level numbers. For instance, if the `secnumdepth` is 1 in an `article` then a `\section{Introduction}` command will produce output like ‘1 Introduction’ while `\subsection{Discussion}` will produce output like ‘Discussion’, without the number. LaTeX’s default `secnumdepth` is 3 in article class and 2 in the book and report classes.

`tocdepth`

Controls which sectioning units are listed in the table of contents. The setting `\setcounter{tocdepth}{level}` makes the sectioning units at level be the smallest ones listed (see [`\setcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005csetcounter)). See the above table for the level numbers. For instance, if `tocdepth` is 1 then the table of contents will list sections but not subsections. LaTeX’s default `tocdepth` is 3 in article class and 2 in the book and report classes.
  * [`\part`](https://latexref.xyz/dev/latex2e.html#g_t_005cpart)
  * [`\chapter`](https://latexref.xyz/dev/latex2e.html#g_t_005cchapter)
  * [`\section`](https://latexref.xyz/dev/latex2e.html#g_t_005csection)
  * [`\subsection`](https://latexref.xyz/dev/latex2e.html#g_t_005csubsection)
  * [`\subsubsection`, `\paragraph`, `\subparagraph`](https://latexref.xyz/dev/latex2e.html#g_t_005csubsubsection-_0026-_005cparagraph-_0026-_005csubparagraph)
  * [`\appendix`](https://latexref.xyz/dev/latex2e.html#g_t_005cappendix)
  * [`\frontmatter`, `\mainmatter`, `\backmatter`](https://latexref.xyz/dev/latex2e.html#g_t_005cfrontmatter-_0026-_005cmainmatter-_0026-_005cbackmatter)
  * [`\@startsection`: Typesetting sectional unit headings](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection)

### 6.1 `\part`
Synopsis, one of:
```
\part{title}
\part*{title}
\part[toc-title]{title}

```

Start a document part. The standard LaTeX classes `book`, `report`, and `article`, all have this command.
This produces a document part, in a book.
```
\part{VOLUME I \\
       PERSONAL MEMOIRS OF  U.\ S.\ GRANT}
\chapter{ANCESTRY--BIRTH--BOYHOOD.}
My family is American, and has been for generations,
in all its branches, direct and collateral.

```

In each standard class the `\part` command outputs a part number such as ‘Part I’, alone on its line, in boldface, and in large type. Then LaTeX outputs title, also alone on its line, in bold and in even larger type. In class `book`, the LaTeX default puts each part alone on its own page. If the book is two-sided then LaTeX will skip a page if needed to have the new part on an odd-numbered page. In `report` it is again alone on a page, but LaTeX won’t force it onto an odd-numbered page. In an `article` LaTeX does not put it on a fresh page, but instead outputs the part number and part title onto the main document page.
The `*` form shows title but it does not show the part number, does not increment the `part` counter, and produces no table of contents entry.
The optional argument toc-title will appear as the part title in the table of contents (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)) and in running headers (see [Page styles](https://latexref.xyz/dev/latex2e.html#Page-styles)). If it is not present then title will be there. This example puts a line break in title but omits the break in the table of contents.
```
\part[Up from the bottom; my life]{Up from the bottom\\ my life}

```

For determining which sectional units are numbered and which appear in the table of contents, the level number of a part is -1 (see [Sectioning/secnumdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002fsecnumdepth), and [Sectioning/tocdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002ftocdepth)).
In the class `article`, if a paragraph immediately follows the part title then it is not indented. To get an indent you can use the package `indentfirst`.
One package to change the behavior of `\part` is `titlesec`. See its documentation on CTAN.
### 6.2 `\chapter`
Synopsis, one of:
```
\chapter{title}
\chapter*{title}
\chapter[toc-title]{title}

```

Start a chapter. The standard LaTeX classes `book` and `report` have this command but `article` does not.
This produces a chapter.
```
\chapter{Loomings}
Call me Ishmael.
Some years ago---never mind how long precisely---having little or no
money in my purse, and nothing particular to interest me on shore, I
thought I would sail about a little and see the watery part of
the world.

```

The LaTeX default starts each chapter on a fresh page, an odd-numbered page if the document is two-sided. It produces a chapter number such as ‘Chapter 1’ in large boldface type (the size is `\huge`). It then puts title on a fresh line, in boldface type that is still larger (size `\Huge`). It also increments the `chapter` counter, adds an entry to the table of contents (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)), and sets the running header information (see [Page styles](https://latexref.xyz/dev/latex2e.html#Page-styles)).
The `*` form shows title on a fresh line, in boldface. But it does not show the chapter number, does not increment the `chapter` counter, produces no table of contents entry, and does not affect the running header. (If you use the page style `headings` in a two-sided document then the header will be from the prior chapter.) This example illustrates.
```
\chapter*{Preamble}

```

The optional argument toc-title will appear as the chapter title in the table of contents (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)) and in running headers (see [Page styles](https://latexref.xyz/dev/latex2e.html#Page-styles)). If it is not present then title will be there. This shows the full name in the chapter title,
```
\chapter[Weyl]{Hermann Klaus Hugo (Peter) Weyl (1885--1955)}

```

but only ‘Weyl’ on the contents page. This puts a line break in the title but that doesn’t work well with running headers so it omits the break in the contents
```
\chapter[Given it all; my story]{Given it all\\ my story}

```

For determining which sectional units are numbered and which appear in the table of contents, the level number of a chapter is 0 (see [Sectioning/secnumdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002fsecnumdepth) and see [Sectioning/tocdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002ftocdepth)).
The paragraph that follows the chapter title is not indented, as is a standard typographical practice. To get an indent use the package `indentfirst`.
You can change what is shown for the chapter number. To change it to something like ‘Lecture 1’, put in the preamble either `\renewcommand{\chaptername}{Lecture}` or this (see [`\makeatletter` & `\makeatother`](https://latexref.xyz/dev/latex2e.html#g_t_005cmakeatletter-_0026-_005cmakeatother)).
```
\makeatletter
\renewcommand{\@chapapp}{Lecture}
\makeatother

```

To make this change because of the primary language for the document, see the package `babel`.
In a two-sided document LaTeX puts a chapter on odd-numbered page, if necessary leaving an even-numbered page that is blank except for any running headers. To make that page completely blank, see [`\clearpage` & `\cleardoublepage`](https://latexref.xyz/dev/latex2e.html#g_t_005cclearpage-_0026-_005ccleardoublepage).
To change the behavior of the `\chapter` command, you can copy its definition from the LaTeX format file and make adjustments. But there are also many packages on CTAN that address this. One is `titlesec`. See its documentation, but the example below gives a sense of what it can do.
```
\usepackage{titlesec}   % in preamble
\titleformat{\chapter}
  {\Huge\bfseries}  % format of title
  {}                % label, such as 1.2 for a subsection
  {0pt}             % length of separation between label and title
  {}                % before-code hook

```

This omits the chapter number ‘Chapter 1’ from the page but unlike `\chapter*` it keeps the chapter in the table of contents and the running headers.
### 6.3 `\section`
Synopsis, one of:
```
\section{title}
\section*{title}
\section[toc-title]{title}

```

Start a section. The standard LaTeX classes `article`, `book`, and `report` all have this command.
This produces a section.
```
In this Part we tend to be more interested in the function,
in the input-output behavior,
than in the details of implementing that behavior.

\section{Turing machines}
Despite this desire to downplay implementation,
we follow the approach of A~Turing that the
first step toward defining the set of computable functions
is to reflect on the details of what mechanisms can do.

```

For the standard LaTeX classes `book` and `report` the default output is like ‘1.2 title’ (for chapter 1, section 2), alone on its line and flush left, in boldface and a larger type (the type size is `\Large`). The same holds in `article` except that there are no chapters in that class so it looks like ‘2 title’.
The `*` form shows title. But it does not show the section number, does not increment the `section` counter, produces no table of contents entry, and does not affect the running header. (If you use the page style `headings` in a two-sided document then the header will be from the prior section.)
The optional argument toc-title will appear as the section title in the table of contents (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)) and in running headers (see [Page styles](https://latexref.xyz/dev/latex2e.html#Page-styles)). If it is not present then title will be there. This shows the full name in the title of the section:
```
\section[Elizabeth~II]{Elizabeth the Second,
  by the Grace of God of the United Kingdom,
  Canada and Her other Realms and Territories Queen,
  Head of the Commonwealth, Defender of the Faith.}

```

but only ‘Elizabeth II’ on the contents page and in the headers. This has a line break in title but that does not work with headers so it is omitted from the contents and headers.
```
\section[Truth is, I cheated; my life story]{Truth is,
  I cheated\\my life story}

```

For determining which sectional units are numbered and which appear in the table of contents, the level number of a section is 1 (see [Sectioning/secnumdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002fsecnumdepth) and see [Sectioning/tocdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002ftocdepth)).
The paragraph that follows the section title is not indented, as is a standard typographical practice. One way to get an indent is to use the package `indentfirst`.
In general, to change the behavior of the `\section` command, there are a number of options. One is the `\@startsection` command (see [`\@startsection`: Typesetting sectional unit headings](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection)). There are also many packages on CTAN that address this, including `titlesec`. See the documentation but the example below gives a sense of what they can do.
```
\usepackage{titlesec}   % in preamble
\titleformat{\section}
  {\normalfont\Large\bfseries}  % format of title
  {\makebox[1pc][r]{\thesection\hspace{1pc}}} % label
  {0pt}                   % length of separation between label and title
  {}                      % before-code hook
\titlespacing*{\section}
  {-1pc}{18pt}{10pt}[10pc]

```

That puts the section number in the margin.
### 6.4 `\subsection`
Synopsis, one of:
```
\subsection{title}
\subsection*{title}
\subsection[toc-title]{title}

```

Start a subsection. The standard LaTeX classes `article`, `book`, and `report` all have this command.
This produces a subsection.
```
We will show that there are more functions than Turing machines and that
therefore some functions have no associated machine.

\subsection{Cardinality} We will begin with two paradoxes that
dramatize the challenge to our intuition posed by comparing the sizes of
infinite sets.

```

For the standard LaTeX classes `book` and `report` the default output is like ‘1.2.3 title’ (for chapter 1, section 2, subsection 3), alone on its line and flush left, in boldface and a larger type (the type size is `\large`). The same holds in `article` except that there are no chapters in that class so it looks like ‘2.3 title’.
The `*` form shows title. But it does not show the subsection number, does not increment the `subsection` counter, and produces no table of contents entry.
The optional argument toc-title will appear as the subsection title in the table of contents (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)). If it is not present then title will be there. This shows the full text in the title of the subsection:
```
\subsection[$\alpha,\beta,\gamma$ paper]{\textit{The Origin of
  Chemical Elements} by R.A.~Alpher, H.~Bethe, and G.~Gamow}

```

but only ‘α,β,γ paper’ on the contents page.
For determining which sectional units are numbered and which appear in the table of contents, the level number of a subsection is 2 (see [Sectioning/secnumdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002fsecnumdepth) and see [Sectioning/tocdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002ftocdepth)).
The paragraph that follows the subsection title is not indented, as is a standard typographical practice. One way to get an indent is to use the package `indentfirst`.
There are a number of ways to change the behavior of the `\subsection` command. One is the `\@startsection` command (see [`\@startsection`: Typesetting sectional unit headings](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection)). There are also many packages on CTAN that address this, including `titlesec`. See the documentation but the example below gives a sense of what they can do.
```
\usepackage{titlesec}   % in preamble
\titleformat{\subsection}[runin]
  {\normalfont\normalsize\bfseries}  % format of the title
  {\thesubsection}                   % label
  {0.6em}                            % space between label and title
  {}                                 % before-code hook

```

That puts the subsection number and title in the first line of text.
### 6.5 `\subsubsection`, `\paragraph`, `\subparagraph`
Synopsis, one of:
```
\subsubsection{title}
\subsubsection*{title}
\subsubsection[toc-title]{title}

```

or one of:
```
\paragraph{title}
\paragraph*{title}
\paragraph[toc-title]{title}

```

or one of:
```
\subparagraph{title}
\subparagraph*{title}
\subparagraph[toc-title]{title}

```

Start a subsubsection, paragraph, or subparagraph. The standard LaTeX classes `article`, `book`, and `report` all have these commands, although they are not commonly used.
This produces a subsubsection.
```
\subsubsection{Piston ring compressors: structural performance}
Provide exterior/interior wall cladding assemblies
capable of withstanding the effects of load and stresses from
consumer-grade gasoline engine piston rings.

```

The default output of each of the three does not change over the standard LaTeX classes `article`, `book`, and `report`. For `\subsubsection` the title is alone on its line, in boldface and normal size type. For `\paragraph` the title is inline with the text, not indented, in boldface and normal size type. For `\subparagraph` the title is inline with the text, with a paragraph indent, in boldface and normal size type (Because an `article` has no chapters its subsubsections are numbered and so it looks like ‘1.2.3 title’, for section 1, subsection 2, and subsubsection 3. The other two divisions are not numbered.)
The `*` form shows title. But it does not increment the associated counter and produces no table of contents entry (and does not show the number for `\subsubsection`).
The optional argument toc-title will appear as the division title in the table of contents (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)). If it is not present then title will be there.
For determining which sectional units are numbered and which appear in the table of contents, the level number of a subsubsection is 3, of a paragraph is 4, and of a subparagraph is 5 (see [Sectioning/secnumdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002fsecnumdepth) and see [Sectioning/tocdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002ftocdepth)).
The paragraph that follows the subsubsection title is not indented, as is a standard typographical practice. One way to get an indent is to use the package `indentfirst`.
There are a number of ways to change the behavior of the these commands. One is the `\@startsection` command (see [`\@startsection`: Typesetting sectional unit headings](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection)). There are also many packages on CTAN that address this, including `titlesec`. See the documentation on CTAN.
### 6.6 `\appendix`
Synopsis:
```
\appendix

```

This does not directly produce any output. But in a `book` or `report` document it declares that subsequent `\chapter` commands start an appendix. In an article it does the same, for `\section` commands. It also resets the `chapter` and `section` counters to 0 in a book or report, and in an article resets the `section` and `subsection` counters.
In this book
```
\chapter{One}  ...
\chapter{Two}  ...
 ...
\appendix
\chapter{Three}  ...
\chapter{Four}  ...

```

the first two will generate output numbered ‘Chapter 1’ and ‘Chapter 2’. After the `\appendix` the numbering will be ‘Appendix A’ and ‘Appendix B’. See [Larger `book` template](https://latexref.xyz/dev/latex2e.html#Larger-book-template), for another example.
The `appendix` package adds the command `\appendixpage` to put a separate ‘Appendices’ in the document body before the first appendix, and the command `\addappheadtotoc` to do the same in the table of contents. You can reset the name ‘Appendices’ with a command like `\renewcommand{\appendixname}{Specification}`, as well as a number of other features. See the documentation on CTAN.
### 6.7 `\frontmatter`, `\mainmatter`, `\backmatter`
Synopsis, one or more of:
```
\frontmatter
...
\mainmatter
...
\backmatter
...

```

Format a `book` class document differently according to which part of the document is being produced. All three commands are optional.
Traditionally, a book’s front matter contains such things as the title page, an abstract, a table of contents, a preface, a list of notations, a list of figures, and a list of tables. (Some of these front matter pages, such as the title page, are traditionally not numbered.) The back matter may contain such things as a glossary, notes, a bibliography, and an index.
The `\frontmatter` command makes the pages numbered in lowercase roman, and makes chapters not numbered, although each chapter’s title appears in the table of contents; if you use other sectioning commands here, use the `*`-version (see [Sectioning](https://latexref.xyz/dev/latex2e.html#Sectioning)).
The `\mainmatter` command changes the behavior back to the expected version, and resets the page number.
The `\backmatter` command leaves the page numbering alone but switches the chapters back to being not numbered.
See [Larger `book` template](https://latexref.xyz/dev/latex2e.html#Larger-book-template), for an example using these three commands.
### 6.8 `\@startsection`: Typesetting sectional unit headings
Synopsis:
```
\@startsection{name}{level}{indent}{beforeskip}{afterskip}{style}

```

Used to help redefine the behavior of commands that start sectioning divisions such as `\section` or `\subsection`.
The `titlesec` package makes manipulation of sectioning easier. Further, while most requirements for sectioning commands can be satisfied with `\@startsection`, some cannot. For instance, in the standard LaTeX `book` and `report` classes the commands `\chapter` and `\report` are not constructed using this. To make such a command you may want to use the `\secdef` command.
The `\@startsection` macro is used like this:
```
\@startsection{name}
  {level}
  {indent}
  {beforeskip}
  {afterskip}
  {style}*[toctitle]{title}

```

so that issuing
```
\renewcommand{\section}{\@startsection{name}
  {level}
  {indent}
  {beforeskip}
  {afterskip}
  {style}}

```

redefines `\section` while keeping its standard calling form `\section*[toctitle]{title}` (in which, as a reminder, the star `*` is optional). See [Sectioning](https://latexref.xyz/dev/latex2e.html#Sectioning). This implies that when you write a command like `\renewcommand{\section}{...}`, the `\@startsection{...}` must come last in the definition. See the examples below.

name

Name of the counter used to number the sectioning header. This counter must be defined separately. Most commonly this is either `section`, `subsection`, or `paragraph`. Although in those cases the counter name is the same as the sectioning command itself, you don’t have to use the same name.
Then `\the`name displays the title number and `\`name`mark` is for the page headers. See the third example below.

level

An integer giving the depth of the sectioning command. See [Sectioning](https://latexref.xyz/dev/latex2e.html#Sectioning), for the list of standard level numbers.
If level is less than or equal to the value of the counter `secnumdepth` then titles for this sectioning command will be numbered (see [Sectioning/secnumdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002fsecnumdepth)). For instance, if `secnumdepth` is 1 in an `article` then the command `\section{Introduction}` will produce output like “1 Introduction” while `\subsection{Discussion}` will produce output like “Discussion”, without the number prefix.
If level is less than or equal to the value of the counter tocdepth then the table of contents will have an entry for this sectioning unit (see [Sectioning/tocdepth](https://latexref.xyz/dev/latex2e.html#Sectioning_002ftocdepth)). For instance, in an `article`, if tocdepth is 1 then the table of contents will list sections but not subsections.

indent

A length giving the indentation of all of the title lines with respect to the left margin. To have the title flush with the margin use `0pt`. A negative indentation such as `-\parindent` will move the title into the left margin.

beforeskip

The absolute value of this length is the amount of vertical space that is inserted before this sectioning unit’s title. This space will be discarded if the sectioning unit happens to start at the beginning of a page. If this number is negative then the first paragraph following the header is not indented; if it is non-negative then the first paragraph is indented. (Example: the negative of `1pt plus 2pt minus 3pt` is `-1pt plus -2pt minus -3pt`.)
For example, if beforeskip is `-3.5ex plus -1ex minus -0.2ex` then to start the new sectioning unit, LaTeX will add about 3.5 times the height of a letter x in vertical space, and the first paragraph in the section will not be indented. Using a rubber length, with `plus` and `minus`, is good practice here since it gives LaTeX more flexibility in making up the page (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)).
The full accounting of the vertical space between the baseline of the line prior to this sectioning unit’s header and the baseline of the header is that it is the sum of the `\parskip` of the text font, the `\baselineskip` of the title font, and the absolute value of the beforeskip. This space is typically rubber so it may stretch or shrink. (If the sectioning unit starts on a fresh page so that the vertical space is discarded then the baseline of the header text will be where LaTeX would put the baseline of the first text line on that page.)

afterskip

This is a length. If afterskip is non-negative then this is the vertical space inserted after the sectioning unit’s title header. If it is negative then the title header becomes a run-in header, so that it becomes part of the next paragraph. In this case the absolute value of the length gives the horizontal space between the end of the title and the beginning of the following paragraph. (Note that the negative of `1pt plus 2pt minus 3pt` is `-1pt plus -2pt minus -3pt`.)
As with beforeskip, using a rubber length, with `plus` and `minus` components, is good practice here since it gives LaTeX more flexibility in putting together the page.
If `afterskip` is non-negative then the full accounting of the vertical space between the baseline of the sectioning unit’s header and the baseline of the first line of the following paragraph is that it is the sum of the `\parskip` of the title font, the `\baselineskip` of the text font, and the value of after. That space is typically rubber so it may stretch or shrink. (Note that because the sign of `afterskip` changes the sectioning unit header’s from standalone to run-in, you cannot use a negative `afterskip` to cancel part of the `\parskip`.)

style

Controls the styling of the title. See the examples below. Typical commands to use here are `\centering`, `\raggedright`, `\normalfont`, `\hrule`, or `\newpage`. The last command in style may be one that takes one argument, such as `\MakeUppercase` or `\fbox` that takes one argument. The section title will be supplied as the argument to this command. For instance, setting style to `\bfseries\MakeUppercase` would produce titles that are bold and uppercase.
These are LaTeX’s defaults for the first three sectioning units that are defined with `\@startsection`, for the article, book, and report classes.
  * For `section`: level is 1, indent is 0pt, beforeskip is `-3.5ex plus -1ex minus -0.2ex`, afterskip is `2.3ex plus 0.2ex`, and style is `\normalfont\Large\bfseries`.
  * For `subsection`: level is 2, indent is 0pt, beforeskip is `-3.25ex plus -1ex minus -0.2ex`, afterskip is `1.5ex plus 0.2ex`, and style is `\normalfont\large\bfseries`.
  * For `subsubsection`: level is 3, indent is 0pt, beforeskip is `-3.25ex plus -1ex minus -0.2ex`, afterskip is `1.5ex plus 0.2ex`, and style is `\normalfont\normalsize\bfseries`.

Some examples follow. These go either in a package or class file or in the preamble of a LaTeX document. If you put them in the preamble they must go between a `\makeatletter` command and a `\makeatother`. (Probably the error message `You can't use `\spacefactor' in vertical mode.` means that you forgot this.) See [`\makeatletter` & `\makeatother`](https://latexref.xyz/dev/latex2e.html#g_t_005cmakeatletter-_0026-_005cmakeatother).
This will put section titles in large boldface type, centered. It says `\renewcommand` because LaTeX’s standard classes have already defined a `\section`. For the same reason it does not define a `section` counter, or the commands `\thesection` and `\l@section`.
```
\renewcommand\section{%
  \@startsection{section}% [name](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002fname)
    {1}% [level](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002flevel)
    {0pt}% [indent](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002findent)
    {-3.5ex plus -1ex minus -.2ex}% [beforeskip](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002fbeforeskip)
    {2.3ex plus.2ex}% [afterskip](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002fafterskip)
    {\centering\normalfont\Large\bfseries}% [style](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002fstyle)
  }

```

This will put `subsection` titles in small caps type, inline with the paragraph.
```
\renewcommand\subsection{%
  \@startsection{subsection}%  [name](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002fname)
    {2}% [level](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002flevel)
    {0em}% [indent](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002findent)
    {-1ex plus 0.1ex minus -0.05ex}% [beforeskip](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002fbeforeskip)
    {-1em plus 0.2em}% [afterskip](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002fafterskip)
    {\scshape}% [style](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040startsection_002fstyle)
  }

```

The prior examples redefined existing sectional unit title commands. This defines a new one, illustrating the needed counter and macros to display that counter.
```
\setcounter{secnumdepth}{6}% show counters this far down
\newcounter{subsubparagraph}[subparagraph]% counter for numbering
\renewcommand{\thesubsubparagraph}%               how to display
  {\thesubparagraph.\@arabic\c@subsubparagraph}%  numbering
\newcommand{\subsubparagraph}{\@startsection
                         {subsubparagraph}%
                         {6}%
                         {0em}%
                         {\baselineskip}%
                         {0.5\baselineskip}%
                         {\normalfont\normalsize}}
\newcommand*\l@subsubparagraph{\@dottedtocline{6}{10em}{5em}}% for toc
\newcommand{\subsubparagraphmark}[1]{}% for page headers

```
