# 02 Overview of LaTeX

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 2.1 Starting and ending
- 2.2 Output files
- 2.3 TeX engines
- 2.4 Input text
- 2.5 LaTeX command syntax
- 2.6 Environment syntax
- 2.7 `\DocumentMetadata`: Producing tagged PDF output
- 2.8 CTAN: The Comprehensive TeX Archive Network

## 2 Overview of LaTeX
LaTeX is a system for typesetting documents. It was originally created by Leslie Lamport in 1984, but has been maintained by a group of volunteers for many years now (<https://latex-project.org>). It is widely used, particularly but not exclusively for mathematical and technical documents.
A LaTeX user writes an input file containing text to be typeset along with interspersed commands. The default encoding for the text is UTF-8 (as of 2018). The commands specify, for example, how the text should be formatted.
LaTeX is implemented as a set of so-called “macros” (a TeX _format_) which use Donald E. Knuth’s TeX typesetting program or one of its derivatives, collectively known as “engines”. Thus, the user produces output, typically PDF, by giving the input file to a TeX engine. The following sections describe all this in more detail.
The term LaTeX is also sometimes used to mean the language in which the input document is marked up, that is, to mean the set of commands available to a LaTeX user.
The name LaTeX is short for “Lamport TeX”. It is pronounced LAH-teck or LAY-teck, or sometimes LAY-tecks. Inside a document, produce the logo with `\LaTeX`. Where use of the logo is not sensible, such as in plain text, write it as ‘LaTeX’.
  * [Starting and ending](https://latexref.xyz/dev/latex2e.html#Starting-and-ending)
  * [Output files](https://latexref.xyz/dev/latex2e.html#Output-files)
  * [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)
  * [Input text](https://latexref.xyz/dev/latex2e.html#Input-text)
  * [LaTeX command syntax](https://latexref.xyz/dev/latex2e.html#LaTeX-command-syntax)
  * [Environment syntax](https://latexref.xyz/dev/latex2e.html#Environment-syntax)
  * [`\DocumentMetadata`: Producing tagged PDF output](https://latexref.xyz/dev/latex2e.html#g_t_005cDocumentMetadata)
  * [CTAN: The Comprehensive TeX Archive Network](https://latexref.xyz/dev/latex2e.html#CTAN)

### 2.1 Starting and ending
LaTeX files have a simple global structure, with a standard beginning and ending. Here is a small example:
```
\documentclass{article}
\begin{document}
Hello, \LaTeX\ world.
\end{document}

```

Every LaTeX document has a `\begin{document}` line and an `\end{document}` line.
Here, the ‘article’ is the _document class_. It is implemented in a file article.cls. You can use any document class available on your system. A few document classes are defined by LaTeX itself, and a vast array of others are available. See [Document classes](https://latexref.xyz/dev/latex2e.html#Document-classes).
You can include other LaTeX commands between the `\documentclass` and the `\begin{document}` commands. This area is called the _preamble_.
The `\begin{document}` … `\end{document}` pair defines an _environment_ ; the ‘document’ environment (and no others) is required in all LaTeX documents (see [`document`](https://latexref.xyz/dev/latex2e.html#document)). LaTeX provides many environments that are documented here (see [Environments](https://latexref.xyz/dev/latex2e.html#Environments)). Many more are available to you from external packages, most importantly those available at CTAN (see [CTAN: The Comprehensive TeX Archive Network](https://latexref.xyz/dev/latex2e.html#CTAN)).
The following sections discuss how to produce PDF or other output from a LaTeX input file.
### 2.2 Output files
LaTeX produces a main output file and at least two auxiliary files. The main output file’s name ends in either .dvi or .pdf.

`.dvi`

If LaTeX is invoked with the system command `latex` then it produces a DeVice Independent file, with extension .dvi. You can view this file with a command such as `xdvi`, or convert it to a PostScript `.ps` file with `dvips` or to a Portable Document Format `.pdf` file with `dvipdfmx`. The contents of the file can be dumped in human-readable form with `dvitype`. A vast array of other DVI utility programs are available (<https://mirror.ctan.org/dviware>).

`.pdf`

If LaTeX is invoked via the system command `pdflatex`, among other commands (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)), then the main output is a Portable Document Format (PDF) file. Typically this is a self-contained file, with all fonts and images included.
LaTeX always produces at least two additional files.

`.log`

This transcript file contains summary information such as a list of loaded packages. It also includes diagnostic messages and perhaps additional information for any errors.

`.aux`

Auxiliary information is used by LaTeX for things such as cross references. For example, the first time that LaTeX finds a forward reference—a cross reference to something that has not yet appeared in the source—it will appear in the output as a doubled question mark `??`. When the referred-to spot does eventually appear in the source then LaTeX writes its location information to this `.aux` file. On the next invocation, LaTeX reads the location information from this file and uses it to resolve the reference, replacing the double question mark with the remembered location.
LaTeX may produce yet more files, characterized by the filename ending. These include a `.lof` file that is used to make a list of figures, a `.lot` file used to make a list of tables, and a `.toc` file used to make a table of contents (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)). A particular class may create others; the list is open-ended.
### 2.3 TeX engines
LaTeX is a large set of commands (macros) that is executed by a TeX program (see [Overview of LaTeX](https://latexref.xyz/dev/latex2e.html#Overview)). Such a set of commands is called a _format_ , and is embodied in a binary `.fmt` file, which can be read much more quickly than the corresponding TeX source.
This section gives a terse overview of the TeX programs that are commonly available (see also [Command line interface](https://latexref.xyz/dev/latex2e.html#Command-line-interface)).

`latex`

`pdflatex`

In TeX Live (<https://tug.org/texlive>), if LaTeX is invoked via either the system command `latex` or `pdflatex`, then the pdfTeX engine is run (<https://ctan.org/pkg/pdftex>). When invoked as `latex`, the main output is a .dvi file; as `pdflatex`, the main output is a .pdf file.
pdfTeX incorporates the e-TeX extensions to Knuth’s original program (<https://ctan.org/pkg/etex>), including additional programming features and bi-directional typesetting, and has plenty of extensions of its own. e-TeX is available on its own as the system command `etex`, but this is plain TeX (and produces .dvi).
In other TeX distributions, `latex` may invoke e-TeX rather than pdfTeX. In any case, the e-TeX extensions can be assumed to be available in LaTeX, and a few extensions beyond e-TeX, particularly for file manipulation.

`lualatex`

If LaTeX is invoked via the system command `lualatex`, the LuaTeX engine is run (<https://ctan.org/pkg/luatex>). This program allows code written in the scripting language Lua (<http://luatex.org>) to interact with TeX’s typesetting. LuaTeX handles UTF-8 Unicode input natively, can handle OpenType and TrueType fonts, and produces a .pdf file by default. There is also `dvilualatex` to produce a .dvi file.

`xelatex`

If LaTeX is invoked with the system command `xelatex`, the XeTeX engine is run (<https://tug.org/xetex>). Like LuaTeX, XeTeX natively supports UTF-8 Unicode and TrueType and OpenType fonts, though the implementation is completely different, mainly using external libraries instead of internal code. XeTeX produces a .pdf file as output; it does not support DVI output.
Internally, XeTeX creates an `.xdv` file, a variant of DVI, and translates that to PDF using the (`x`)`dvipdfmx` program, but this process is automatic. The `.xdv` file is only useful for debugging.

`hilatex`

If LaTeX is invoked via the system command `hilatex`, the HiTeX engine is run (<https://ctan.org/pkg/hitex>). This program produces its own format, named HINT, designed especially for high-quality typesetting on mobile devices.

`platex`

`uplatex`

These commands provide significant additional support for Japanese and other languages; the `u` variant supports Unicode. See <https://ctan.org/pkg/ptex> and <https://ctan.org/pkg/uptex>.
As of 2019, there is a companion `-dev` command and format for all of the above, except `hitex`:

`dvilualatex-dev`

`latex-dev`

`lualatex-dev`

`pdflatex-dev`

`platex-dev`

`uplatex-dev`

`xelatex-dev`

These are candidates for an upcoming LaTeX release. The main purpose is to find and address compatibility problems before an official release.
These `-dev` formats make it easy for anyone to help test documents and code: you can run, say, `pdflatex-dev` instead of `pdflatex`, without changing anything else in your environment. Indeed, it is easiest and most helpful to always run the `-dev` versions instead of bothering to switch back and forth. During quiet times after a release, the commands will be equivalent.
These are not daily snapshots or untested development code. They undergo the same extensive regression testing by the LaTeX team before being released.
For more information, see “The LaTeX release workflow and the LaTeX `dev` formats” by Frank Mittelbach, TUGboat 40:2, <https://tug.org/TUGboat/tb40-2/tb125mitt-dev.pdf>.
### 2.4 Input text
To a first approximation, most input characters in LaTeX print as themselves. But there are exceptions, as discussed in the following sections.
  * [Input encodings](https://latexref.xyz/dev/latex2e.html#Input-encodings)
  * [Ligatures](https://latexref.xyz/dev/latex2e.html#Ligatures)
  * [Special characters: `\ { } % $ & _ ^ # ~`](https://latexref.xyz/dev/latex2e.html#Special-characters)

#### 2.4.1 Input encodings
The input to TeX (or any computer program) ultimately consists of a sequence of bytes. (Nowadays, a byte is almost universally an eight-bit number, i.e., an integer between 0 and 255, inclusive.) The input encoding defines how to interpret that sequence of bytes, and thus how LaTeX behaves.
Today, by far the most common way to encode text is with _UTF-8_ , a so-called “Unicode Transformation Format” which specifies how to transform a sequence of 8-bit bytes to Unicode code points, which are defined independent of any particular representation. The Unicode encoding defines code points for virtually all characters used today in written text.
When TeX was created, Unicode and UTF-8 did not exist and the 7-bit ASCII encoding was by far the most widely used. So TeX does not require Unicode for text input. UTF-8 is a superset of ASCII, so a pure 7-bit ASCII document is also UTF-8.
Since 2018, the default input encoding for LaTeX is UTF-8. Some methods for handling documents written in some other encoding, such as ISO-8859-1 (Latin 1), are explained in [`inputenc` package](https://latexref.xyz/dev/latex2e.html#inputenc-package).
You can easily find more about all these topics in any introductory computer text or online. For example, you might start at: <https://en.wikipedia.org/wiki/Unicode>.
#### 2.4.2 Ligatures
A _ligature_ combines two or more letters (more generally, characters) into a single glyph. For example, in Latin-based typography, the two letters ‘f’ and ‘i’ are often combined into the glyph ‘fi’.
TeX supports ligatures automatically. To continue the example, if the input has the word ‘fine’, written as four separate ASCII characters, TeX will output the word ‘fine’ (with the default fonts), with three typeset glyphs.
In traditional TeX, the available ligatures, if any, are defined by the current font. TeX also uses the ligature mechanism to produce a few typographical characters which were not available in any computer encoding when TeX was invented. In all, in the original Computer Modern fonts, the following input character sequences are defined to lead to ligatures:

‘ff’

ff (ff ligature, U+FB00)

‘fi’

fi (fi ligature, U+FB01)

‘fl’

fl (fl ligature, U+FB02)

‘ffi’

ffi (ffi ligature, U+FB03)

‘ffl’

ffl (ffl ligature, U+FB04)

‘``’

“ (left double quotation mark, U+201C)

‘''’

” (right double quotation mark, U+201D)

‘--’

– (en-dash, U+2013)

‘---’

— (em-dash, U+2014)

‘!`’

!‘ (inverted exclamation mark, U+00A1)

‘?`’

?‘ (inverted question mark, U+00BF)
(For the f-ligatures above, the text in parentheses shows the individual characters, so in the typeset output you can easily see the difference between the ligature and the original character sequence.)
Nowadays it’s usually possible to directly input the punctuation characters as Unicode characters, and LaTeX supports that (see previous section). But even today, it can still often be useful to use the ASCII ligature input form; for example, the difference between an en-dash and em-dash, as a single glyph, can be all but impossible to discern, but the difference between two and three ASCII hyphen characters is clear. Similarly with quotation marks, in some fonts.
Thus, even the engines with native support for UTF-8, namely LuaTeX and XeTeX, also support the ASCII ligature input sequences by default, independent of the font used. They also need to do so for compatibility.
By the way, the f-ligatures are also available in Unicode (the “Alphabetic Presentation Forms” block starting at U+FB00), but it’s almost never desirable to use them as input characters, since in principle it should be up to the typesetter and the current font whether to use ligatures. Also, in practice, using them will typically cause searches to fail, that is, a search for the two characters ‘fi’ will not be matched by the ligature ‘fi’ at U+FB01.
#### 2.4.3 Special characters: `\ { } % $ & _ ^ # ~`
Besides ligatures (see previous section), a few individual characters have special meaning to LaTeX. They are called _reserved characters_ or _special characters_. Here they are:

‘\’

Introduces a command name, as seen throughout this manual.

‘{’

‘}’

Delimits a required argument to a command or a level of grouping, as seen throughout this manual.

‘%’

Starts a comment: the ‘%’ and all remaining characters on the current line are ignored.

‘$’

Starts and ends math mode (see [Math formulas](https://latexref.xyz/dev/latex2e.html#Math-formulas)).

‘&’

Separates cells in a table (see [`tabular`](https://latexref.xyz/dev/latex2e.html#tabular)).

‘_’

‘^’

Introduce a subscript or superscript, respectively, in math (see [Subscripts & superscripts](https://latexref.xyz/dev/latex2e.html#Subscripts-_0026-superscripts)); they produce an error outside math mode. As a little-used special feature, two superscript characters in a row can introduce special notation for an arbitrary character.

‘#’

Stands for arguments in a macro definition (see [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand)).

‘~’

Produces a nonbreakable interword space (see [`~`, `\nobreakspace`](https://latexref.xyz/dev/latex2e.html#g_t_007e)).
See [Printing special characters](https://latexref.xyz/dev/latex2e.html#Printing-special-characters), for how to typeset these characters when you need them literally.
### 2.5 LaTeX command syntax
In the LaTeX input file, a command name starts with a backslash character, `\`. The name itself then consists of either (a) a string of letters or (b) a single non-letter.
LaTeX commands names are case sensitive; for example, `\pagebreak` differs from `\Pagebreak` (the latter is not a standard command). Most command names are lowercase, but in any event you must enter all commands in the same case as they are defined.
A command may be followed by zero, one, or more arguments. These arguments may be either required or optional. Required arguments are contained in curly braces, `{...}`. Optional arguments are contained in square brackets, `[...]`. Generally, but not universally, if the command accepts an optional argument, it comes first, before any required arguments; optional arguments could come after required arguments, or both before and after.
Inside of an optional argument, to use the character close square bracket (`]`) hide it inside curly braces, as in `\item[closing bracket {]}]`. Similarly, if an optional argument comes last, with no required argument after it, then to make the first character of the following text be an open square bracket, hide it inside curly braces.
LaTeX has the convention that some commands have a `*` form that is closely related to the form without a `*`, such as `\chapter` and `\chapter*`. The difference in behavior varies from command to command.
This manual describes all accepted options and `*`-forms for the commands it covers (barring unintentional omissions, a.k.a. bugs).
As of the 2020-10-01 release of LaTeX, the `expl3` and `xparse` packages are part of the LaTeX2e format. They provide a completely different underlying programming language syntax. We won’t try to cover that in this document; see the related package documentation and other LaTeX manuals.
### 2.6 Environment syntax
Synopsis:
```
\begin{environment-name}
  ...
\end{environment-name}

```

An _environment_ is an area of LaTeX source, inside of which there is a distinct behavior. For instance, for poetry in LaTeX put the lines between `\begin{verse}` and `\end{verse}`.
```
\begin{verse}
  There once was a man from Nantucket \\
  ...
\end{verse}

```

See [Environments](https://latexref.xyz/dev/latex2e.html#Environments), for a list of environments. Particularly notable is that every LaTeX document must have a `document` environment, a `\begin{document} ... \end{document}` pair.
The environment-name at the beginning must exactly match that at the end. This includes the case where environment-name ends in a star (`*`); both the `\begin` and `\end` texts must include the star.
Environments may have arguments, including optional arguments. This example produces a table. The first argument is optional (and causes the table to be aligned on its top row) while the second argument is required (it specifies the formatting of columns).
```
\begin{tabular}[t]{r|l}
  ... rows-of-table ...
\end{tabular}

```

### 2.7 `\DocumentMetadata`: Producing tagged PDF output
The `\DocumentMetadata` command was added to LaTeX in 2022. It enables so-called “tagging” of the PDF output, aiding accessibility of the PDF. It is supported best with LuaLaTeX; pdfLaTeX and XeLaTeX are supported as well as possible (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)).
It is unlike nearly any other command in LaTeX in that it must occur before the `\documentclass` command that starts a LaTeX document proper (see [\documentclass](https://latexref.xyz/dev/latex2e.html#g_t_005cdocumentclass)). Therefore it must be called with `\RequirePackage` rather than `\usepackage` (see [\RequirePackage](https://latexref.xyz/dev/latex2e.html#g_t_005cRequirePackage)).
This support is still in development, so we will not try to list all the possible settings. Please see the `documentmetadata-support-doc` document, part of the `latex-lab` package (<https://ctan.org/pkg/latex-lab>). Here is a simple example which enables most tagging currently implemented:
```
\DocumentMetadata{testphase={phase-III,firstaid}}
\documentclass{article}
...

```

As you can see from the key name `testphase`, this is all still in an experimental phase. The LaTeX developers strongly encourage users to give it a try and report problems, so it can be improved.
### 2.8 CTAN: The Comprehensive TeX Archive Network
The Comprehensive TeX Archive Network, CTAN, is the TeX and LaTeX community’s repository of free material. It is a set of Internet sites around the world that offer material related to LaTeX for download. Visit CTAN on the web at <https://ctan.org>.
This material is organized into packages, discrete bundles that typically offer some coherent functionality and are maintained by one person or a small number of people. For instance, many publishers have a package that allows authors to format papers to that publisher’s specifications.
In addition to its massive holdings, the `ctan.org` web site offers features such as search by name or by functionality.
CTAN is not a single host, but instead is a set of hosts, one of which is the so-called “master”. The master host actively manages the material, for instance, by accepting uploads of new or updated packages. For many years, it has been hosted by the German TeX group, DANTE e.V.
Other sites around the world help out by mirroring, that is, automatically syncing their collections with the master site and then in turn making their copies publicly available. This gives users close to their location better access and relieves the load on the master site. The list of mirrors is at <https://ctan.org/mirrors>.
