# 03 Document classes

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 3.1 Document class options
- 3.2 `\usepackage`: Additional packages
- 3.3 Class and package creation

## 3 Document classes
The document’s overall class is defined with the `\documentclass` command, which is normally the first command in a LaTeX source file.
```
\documentclass[options]{class}

```

The following document class names are built into LaTeX. Many other document classes are available as separate packages (see [Overview of LaTeX](https://latexref.xyz/dev/latex2e.html#Overview)).

`article`

For a journal article, a presentation, and miscellaneous general use.

`book`

Full-length books, including chapters and possibly including front matter, such as a preface, and back matter, such as an appendix (see [Front/back matter](https://latexref.xyz/dev/latex2e.html#Front_002fback-matter)).

`letter`

Mail, optionally including mailing labels (see [Letters](https://latexref.xyz/dev/latex2e.html#Letters)).

`report`

For documents of length between an `article` and a `book`, such as technical reports or theses, which may contain several chapters.

`slides`

For slide presentations—rarely used nowadays. The `beamer` package is perhaps the most prevalent replacement (<https://ctan.org/pkg/beamer>). See [`beamer` template](https://latexref.xyz/dev/latex2e.html#beamer-template), for a small template for a beamer document.
Standard options are described in the next section.
  * [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)
  * [`\usepackage`: Additional packages](https://latexref.xyz/dev/latex2e.html#g_t_005cusepackage)
  * [Class and package creation](https://latexref.xyz/dev/latex2e.html#Class-and-package-creation)

### 3.1 Document class options
You can specify _global options_ or _class options_ to the `\documentclass` command by enclosing them in square brackets. To specify more than one option, separate them with a comma.
```
\documentclass[option1,option2,...]{class}

```

LaTeX automatically passes options specified for `\documentclass` on to any other loaded classes that can handle them.
Here is the list of the standard class options.
All of the standard classes except `slides` accept the following options for selecting the typeface size; the default is `10pt`:
```
10pt  11pt  12pt

```

All of the standard classes accept these options for selecting the paper size (dimensions are listed height by width):

`a4paper`

210 by 297mm (about 8.25 by 11.75 inches)

`a5paper`

148 by 210mm (about 5.8 by 8.3 inches)

`b5paper`

176 by 250mm (about 6.9 by 9.8 inches)

`executivepaper`

7.25 by 10.5 inches

`legalpaper`

8.5 by 14 inches

`letterpaper`

8.5 by 11 inches (the default)
When using one of the engines pdfLaTeX, LuaLaTeX, or XeLaTeX (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)), options other than `letterpaper` set the print area but you must also set the physical paper size. Usually, the `geometry` package is the best way to do that; it provides flexible ways of setting the print area and physical page size. Otherwise, setting the paper size is engine-dependent. For example, with pdfLaTeX, you could include `\pdfpagewidth=\paperwidth` and `\pdfpageheight=\paperheight` in the preamble.
Miscellaneous other options:

`draft`

`final`

Mark (`draft`) or do not mark (`final`) overfull boxes with a black box in the margin; default is `final`.

`fleqn`

Put displayed formulas flush left; default is centered.

`landscape`

Selects landscape format; default is portrait.

`leqno`

Put equation numbers on the left side of equations; default is the right side.

`openbib`

Use “open” bibliography format.

`titlepage`

`notitlepage`

Specifies whether there is a separate page for the title information and for the abstract also, if there is one. The default for the `report` class is `titlepage`, for the other classes it is `notitlepage`.
The following options are not available with the `slides` class.

`onecolumn`

`twocolumn`

Typeset in one or two columns; default is `onecolumn`.

`oneside`

`twoside`

Selects one- or two-sided layout; default is `oneside`, except that in the `book` class the default is `twoside`.
For one-sided printing, the text is centered on the page. For two-sided printing, the `\evensidemargin` (`\oddsidemargin`) parameter determines the distance on even (odd) numbered pages between the left side of the page and the text’s left margin, with `\oddsidemargin` being 40% of the difference between `\paperwidth` and `\textwidth`, and `\evensidemargin` is the remainder.

`openright`

`openany`

Specifies whether a chapter (or appendix, etc.) should start on a right-hand page, by inserting a blank page if necessary. The default is `openright` for `book`, and `openany` for `report`.
The `slides` class offers the option `clock` for printing the time at the bottom of each note.
### 3.2 `\usepackage`: Additional packages
To load a package pkg, with the package options given in the comma-separated list options:
```
\usepackage[options]{pkg}[mindate]

```

To specify more than one package you can separate them with a comma, as in `\usepackage{pkg1,pkg2,...}`, or use multiple `\usepackage` commands.
If the mindate optional argument is given, LaTeX gives a warning if the loaded package has an earlier date, i.e., is too old. The mindate argument must be in the form `YYYY/MM/DD`. More info on this: <https://tex.stackexchange.com/questions/47743>.
`\usepackage` must be used in the document preamble, between the `\documentclass` declaration and the `\begin{document}`. Occasionally it is necessary to load packages before the `\documentclass`; see `\RequirePackage` for that (see [\RequirePackage](https://latexref.xyz/dev/latex2e.html#g_t_005cRequirePackage)).
Any options given in the global `\documentclass` command that are unknown to the selected document class are passed on to the packages loaded with `\usepackage`.
### 3.3 Class and package creation
You can create new document classes and new packages. For instance, if your memos must satisfy some local requirements, such as a standard header for each page, then you could create a new class `smcmemo.cls` and begin your documents with `\documentclass{smcmemo}`.
What separates a package from a document class is that the commands in a package are useful across classes while those in a document class are specific to that class. Thus, a command to set page headers is for a package while a command to make the page headers be `Memo from the SMC Math Department` is for a class.
Inside of a class or package definition you can use the at-sign `@` as a character in command names without having to surround the code containing that command with `\makeatletter` and `\makeatother` (see [`\makeatletter` & `\makeatother`](https://latexref.xyz/dev/latex2e.html#g_t_005cmakeatletter-_0026-_005cmakeatother)). This allows you to create commands that users will not accidentally redefine.
It is also highly desirable to prefix class- or package-specific commands with your package name or similar string, to prevent your definitions from clashing with those from other packages. For instance, the class `smcmemo` might have commands `\smc@tolist`, `\smc@fromlist`, etc.
  * [Class and package structure](https://latexref.xyz/dev/latex2e.html#Class-and-package-structure)

#### 3.3.1 Class and package structure
A class file or package file typically has four parts.
  1. In the _identification part_ , the file says that it is a LaTeX package or class and describes itself, using the `\NeedsTeXFormat` and `\ProvidesClass` or `\ProvidesPackage` commands.
  2. The _preliminary declarations part_ declares some commands and can also load other files. Usually these commands will be those needed for the code used in the next part. For example, an `smcmemo` class might be called with an option to read in a file with a list of people for the to-head, as `\documentclass[mathto]{smcmemo}`, and therefore needs to define a command `\newcommand{\setto}[1]{\def\@tolist{#1}}` used in that file.
  3. In the _handle options part_ the class or package declares and processes its options. Class options allow a user to start their document as `\documentclass[option list]{class name}`, to modify the behavior of the class. An example is when you declare `\documentclass[11pt]{article}` to set the default document font size.
  4. Finally, in the _more declarations part_ the class or package usually does most of its work: declaring new variables, commands and fonts, and loading other files.

Here is a starting class file, which should be saved as stub.cls where LaTeX can find it, for example in the same directory as the .tex file.
```
\NeedsTeXFormat{LaTeX2e}
\ProvidesClass{stub}[2017/07/06 stub to start building classes from]
\DeclareOption*{\PassOptionsToClass{\CurrentOption}{article}}
\ProcessOptions\relax
\LoadClass{article}

```

It identifies itself, handles the class options via the default of passing them all to the `article` class, and then loads the `article` class to provide the basis for this class’s code.
For more, see the official guide for class and package writers, the Class Guide, at <https://ctan.org/pkg/clsguide> (much of the description here derives from this document), or the tutorial at <https://tug.org/TUGboat/tb26-3/tb84heff.pdf>.
See [Class and package commands](https://latexref.xyz/dev/latex2e.html#Class-and-package-commands), for some of the commands specifically intended for class and package writers.
