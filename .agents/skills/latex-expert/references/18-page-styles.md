# 18 Page styles

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 18.1 `\maketitle`
- 18.2 `\pagenumbering`
- 18.3 `\pagestyle`
- 18.4 `\thispagestyle`
- 18.5 `\thepage`

## 18 Page styles
The style of a page determines where LaTeX places the components of that page, such as headers and footers, and the text body. This includes pages in the main part of the document but also includes special pages such as the title page of a book, a page from an index, or the first page of an article.
The package `fancyhdr` is commonly used for constructing page styles. See its documentation.
  * [`\maketitle`](https://latexref.xyz/dev/latex2e.html#g_t_005cmaketitle)
  * [`\pagenumbering`](https://latexref.xyz/dev/latex2e.html#g_t_005cpagenumbering)
  * [`\pagestyle`](https://latexref.xyz/dev/latex2e.html#g_t_005cpagestyle)
  * [`\thispagestyle`](https://latexref.xyz/dev/latex2e.html#g_t_005cthispagestyle)
  * [`\thepage`](https://latexref.xyz/dev/latex2e.html#g_t_005cthepage)

### 18.1 `\maketitle`
Synopsis:
```
\maketitle

```

Generate a title. In the standard classes the title appears on a separate page, except in the `article` class where it is at the top of the first page. (See [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options), for information about the `titlepage` document class option.)
This example shows `\maketitle` appearing in its usual place, immediately after `\begin{document}`.
```
\documentclass{article}
\title{Constructing a Nuclear Reactor Using Only Coconuts}
\author{Jonas Grumby\thanks{%
    With the support of a Ginger Grant from the Roy Hinkley Society.} \\
  Skipper, \textit{Minnow}
  \and
  Willy Gilligan\thanks{%
    Thanks to the Mary Ann Summers foundation
    and to Thurston and Lovey Howell.}           \\
  Mate, \textit{Minnow}
  }
\date{1964-Sep-26}
\begin{document}
\maketitle
Just sit right back and you'll hear a tale, a tale of a fateful trip.
That started from this tropic port, aboard this tiny ship. The mate was
a mighty sailin' man, the Skipper brave and sure. Five passengers set
sail that day for a three hour tour. A three hour tour.
  ...

```

You tell LaTeX the information used to produce the title by making the following declarations. These must come before the `\maketitle`, either in the preamble or in the document body.

`\author{name1 \and name2 \and ...}`

Required. Declare the document author or authors. The argument is a list of authors separated by `\and` commands. To separate lines within a single author’s entry, for instance to give the author’s institution or address, use a double backslash, `\\`. If you omit the `\author` declaration then you get ‘LaTeX Warning: No \author given’.

`\date{text}`

Optional. Declare text to be the document’s date. The text doesn’t need to be in a date format; it can be any text at all. If you omit `\date` then LaTeX uses the current date (see [`\today`](https://latexref.xyz/dev/latex2e.html#g_t_005ctoday)). To have no date, instead use `\date{}`.

`\thanks{text}`

Optional. Produce a footnote. You can use it in the author information for acknowledgements as illustrated above, but you can also use it in the title, or anywhere that a footnote mark makes sense. It can be any text at all so you can use it for any purpose, such as to print an email address.

`\title{text}`

Required. Declare text to be the title of the document. Get line breaks inside text with a double backslash, `\\`. If you omit the `\title` declaration then the `\maketitle` command yields error ‘LaTeX Error: No \title given’.
To make your own title page, see [`titlepage`](https://latexref.xyz/dev/latex2e.html#titlepage). You can either create this as a one-off or you can include it as part of a renewed `\maketitle` command. Many publishers will provide a class to use in place of `article` that formats the title according to their house requirements.
### 18.2 `\pagenumbering`
Synopsis:
```
\pagenumbering{number-style}

```

Specifies the style of page numbers, and resets the page number. The numbering style is reflected on the page, and also in the table of contents and other page references. This declaration has global scope so its effect is not stopped by an end of group such as a closing brace or an end of environment.
By default, LaTeX numbers pages starting at 1, using Arabic numerals.
The argument number-style is one of the following (see also [`\alph \Alph \arabic \roman \Roman \fnsymbol`: Printing counters](https://latexref.xyz/dev/latex2e.html#g_t_005calph-_005cAlph-_005carabic-_005croman-_005cRoman-_005cfnsymbol)).

`arabic`

Arabic numerals: 1, 2, …

`roman`

lowercase Roman numerals: i, ii, …

`Roman`

uppercase Roman numerals: I, II, …

`alph`

lowercase letters: a, b, … If you have more than 26 pages then you get ‘LaTeX Error: Counter too large’.

`Alph`

uppercase letters: A, B, … If you have more than 26 pages then you get ‘LaTeX Error: Counter too large’.

`gobble`

no page number is output, though the number is still reset. References to that page also are blank.
This setting does not work with the popular package `hyperref`, so to omit page numbers you may want to instead use `\pagestyle{empty}` or `\thispagestyle{empty}`.
If you want to typeset the page number in some other way, or change where the page number appears on the page, see [`\pagestyle`](https://latexref.xyz/dev/latex2e.html#g_t_005cpagestyle) (in short: use the `fancyhdr` package). The list above of LaTeX’s built-in numbering styles cannot be extended.
Traditionally, if a document has front matter—preface, table of contents, etc.—then it is numbered with lowercase Roman numerals. The main matter of a document uses arabic. LaTeX implements this, by providing explicit commands for the different parts (see [`\frontmatter`, `\mainmatter`, `\backmatter`](https://latexref.xyz/dev/latex2e.html#g_t_005cfrontmatter-_0026-_005cmainmatter-_0026-_005cbackmatter)).
As an explicit example, before the ‘Main’ section the pages are numbered ‘a’, etc. Starting on the page containing the `\pagenumbering` call in that section, the pages are numbered ‘1’, etc.
```
\begin{document}\pagenumbering{alph}
  ...
\section{Main}\pagenumbering{arabic}
  ...

```

If you want to change the value of the page number, then you manipulate the `page` counter (see [Counters](https://latexref.xyz/dev/latex2e.html#Counters)).
### 18.3 `\pagestyle`
Synopsis:
```
\pagestyle{style}

```

Declaration that specifies how the page headers and footers are typeset, from the current page onwards.
A discussion with an example is below. First, however: the package `fancyhdr` is now the standard way to manipulate headers and footers. New documents that need to do anything other than one of the standard options below should use this package. See its documentation (<https://ctan.org/pkg/fancyhdr>).
Values for style:

`plain`

The header is empty. The footer contains only a page number, centered.

`empty`

The header and footer are both empty.

`headings`

Put running headers and footers on each page. The document style specifies what goes in there; see the discussion below.

`myheadings`

Custom headers, specified via the `\markboth` or the `\markright` commands.
Some discussion of the motivation for LaTeX’s mechanism will help you work with the options `headings` or `myheadings`. The document source below produces an article, two-sided, with the pagestyle `headings`. On this document’s left hand pages, LaTeX wants (in addition to the page number) the title of the current section. On its right hand pages LaTeX wants the title of the current subsection. When it makes up a page, LaTeX gets this information from the commands `\leftmark` and `\rightmark`. So it is up to `\section` and `\subsection` to store that information there.
```
\documentclass[twoside]{article}
\pagestyle{headings}
\begin{document}
  ... \section{Section 1} ... \subsection{Subsection 1.1} ...
\section{Section 2}
  ...
\subsection{Subsection 2.1}
  ...
\subsection{Subsection 2.2}
  ...

```

Suppose that the second section falls on a left page. Although when the page starts it is in the first section, LaTeX will put ‘Section 2’ in the left page header. As to the right header, if no subsection starts before the end of the right page then LaTeX blanks the right hand header. If a subsection does appear before the right page finishes then there are two cases. If at least one subsection starts on the right hand page then LaTeX will put in the right header the title of the first subsection starting on that right page. If at least one of 2.1, 2.2, …, starts on the left page but none starts on the right then LaTeX puts in the right hand header the title of the last subsection to start, that is, the one in effect during the right hand page.
To accomplish this, in a two-sided article, LaTeX has `\section` issue a command `\markboth`, setting `\leftmark` to ‘Section 2’ and setting `\rightmark` to an empty content. And, LaTeX has `\subsection` issue a command `\markright`, setting `\rightmark` to ‘Subsection 2.1’, etc.
Here are the descriptions of `\markboth` and `\markright`:

`\markboth{left-head}{right-head}`

Sets both the right hand and left hand heading information for either a page style of `headings` or `myheadings`. A left hand page heading left-head is generated by the last `\markboth` command before the end of the page. A right hand page heading right-head is generated by the first `\markboth` or `\markright` that comes on the page if there is one, otherwise by the last one that came before that page.

`\markright{right-head}`

Sets the right hand page heading, leaving the left unchanged.
### 18.4 `\thispagestyle`
Synopsis:
```
\thispagestyle{style}

```

Works in the same way as the `\pagestyle` (see [`\pagestyle`](https://latexref.xyz/dev/latex2e.html#g_t_005cpagestyle)), except that it changes to style for the current page only. This declaration has global scope, so its effect is not delimited by braces or environments.
Often the first page of a chapter or section has a different style. For example, this LaTeX book document has the first page of the first chapter in `plain` style, as is the default (see [Page styles](https://latexref.xyz/dev/latex2e.html#Page-styles)).
```
\documentclass{book}
\pagestyle{headings}
\begin{document}
\chapter{First chapter}
  ...
\chapter{Second chapter}\thispagestyle{empty}
  ...

```

The `plain` style has a page number on it, centered in the footer. To make the page entirely empty, the command `\thispagestyle{empty}` immediately follows the second `\chapter`.
### 18.5 `\thepage`
If you want to change the appearance of page numbers only in the page headers, for example by adding an ornament, typesetting in small caps, etc., then the `fancyhdr` package, as mentioned in a previous section, is the best approach.
On the other hand, you may want to change how page numbers are denoted everywhere, including the table of contents and cross-references, as well as the page headers. In this case, you should redefine `\thepage`, which is the command LaTeX uses for the representation of page numbers.
However, `\thepage` should do any typesetting or other complicated maneuvers, but merely expand to the intended page number representation. The results of a complicated redefinition of `\thepage` are not predictable, but LaTeX’s report of page numbers in diagnostic messages, at least, will become unusable.
There is some discussion of this issue at <https://tex.stackexchange.com/questions/687258>.
