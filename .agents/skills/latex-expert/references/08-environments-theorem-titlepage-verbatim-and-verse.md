# 08 Environments: Theorem, titlepage, verbatim, and verse

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 8.25 `theorem`
- 8.26 `titlepage`
- 8.27 `verbatim`
- 8.28 `verse`

### 8.25 `theorem`
Synopsis:
```
\begin{theorem}
  theorem body
\end{theorem}

```

Produces ‘Theorem n’ in boldface followed by theorem body in italics. The numbering possibilities for n are described under `\newtheorem` (see [`\newtheorem`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewtheorem)).
```
\newtheorem{lem}{Lemma}      % in preamble
\newtheorem{thm}{Theorem}
  ...
\begin{lem}                  % in  document body
  text of lemma
\end{lem}

The next result follows immediately.
\begin{thm}[Gauss]   %  put `Gauss' in parens after theorem head
  text of theorem
\end{thm}

```

Most new documents use the packages `amsthm` and `amsmath` from the American Mathematical Society. Among other things these packages include a large number of options for theorem environments, such as styling options.
### 8.26 `titlepage`
Synopsis:
```
\begin{titlepage}
  ... text and spacing ...
\end{titlepage}

```

Create a title page, a page with no printed page number or heading and with succeeding pages numbered starting with page one.
In this example all formatting, including vertical spacing, is left to the author.
```
\begin{titlepage}
\vspace*{\stretch{1}}
\begin{center}
  {\huge\bfseries Thesis \\[1ex]
                  title}                  \\[6.5ex]
  {\large\bfseries Author name}           \\
  \vspace{4ex}
  Thesis  submitted to                    \\[5pt]
  \textit{University name}                \\[2cm]
  in partial fulfilment for the award of the degree of \\[2cm]
  \textsc{\Large Doctor of Philosophy}    \\[2ex]
  \textsc{\large Mathematics}             \\[12ex]
  \vfill
  Department of Mathematics               \\
  Address                                 \\
  \vfill
  \today
\end{center}
\vspace{\stretch{2}}
\end{titlepage}

```

To instead produce a standard title page without a `titlepage` environment, use `\maketitle` (see [`\maketitle`](https://latexref.xyz/dev/latex2e.html#g_t_005cmaketitle)).
### 8.27 `verbatim`
Synopsis:
```
\begin{verbatim}
literal-text
\end{verbatim}

```

A paragraph-making environment in which LaTeX produces as output exactly what you type as input. For instance inside literal-text the backslash `\` character does not start commands, it produces a printed ‘\’, and carriage returns and blanks are taken literally. The output appears in a monospaced typewriter-like font (`\tt`).
```
\begin{verbatim}
Symbol swearing: %&$#?!.
\end{verbatim}

```

The only restriction on `literal-text` is that it cannot include the string `\end{verbatim}`.
You cannot use the verbatim environment in the argument to macros, for instance in the argument to a `\section`. This is not the same as commands being fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)), instead it just cannot work, as the `verbatim` environment changes the catcode regime before processing its contents, and restore it immediately afterward, nevertheless with a macro argument the content of the argument has already be converted to a token list along the catcode regime in effect when the macro was called. However, the `cprotect` package can help with this.
One common use of verbatim input is to typeset computer code. Some packages offer many features not provided by the `verbatim` environment; two of the most popular are `listings` and `minted`. For example, they are capable of pretty-printing, line numbering, and selecting parts of files for a continuing listing.
A package that provides many more options for verbatim environments is `fancyvrb`. Another is `verbatimbox`.
For a list of all the relevant packages, see CTAN (see [CTAN: The Comprehensive TeX Archive Network](https://latexref.xyz/dev/latex2e.html#CTAN)), particularly the topics `listing` (<https://ctan.org/topic/listing>) and `verbatim` (<https://ctan.org/topic/verbatim>).
  * [`\verb`](https://latexref.xyz/dev/latex2e.html#g_t_005cverb)

#### 8.27.1 `\verb`
Synopsis:
```
\verb char literal-text char
\verb* char literal-text char

```

Typeset literal-text as it is input, including special characters and spaces, using the typewriter (`\tt`) font.
This example shows two different invocations of `\verb`.
```
This is \verb!literally! the biggest pumpkin ever.
And this is the best squash, \verb+literally!+

```

The first `\verb` has its literal-text surrounded with exclamation point, `!`. The second instead uses plus, `+`, because the exclamation point is part of `literal-text`.
The single-character delimiter char surrounds literal-text—it must be the same character before and after. No spaces come between `\verb` or `\verb*` and char, or between char and literal-text, or between literal-text and the second occurrence of char (the synopsis shows a space only to distinguish one component from the other). The delimiter must not appear in literal-text. The literal-text cannot include a line break.
The `*`-form differs only in that spaces are printed with a visible space character.
The output from this will include a visible space on both side of word ‘with’:
```
The command's first argument is \verb*!filename with extension! and ...

```

For typesetting Internet addresses, urls, the package `url` is a better option than the `\verb` command, since it allows line breaks.
You cannot use `\verb` in the argument to a macro, for instance in the argument to a `\section`. It is not a question of `\verb` being fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)), instead it just cannot work, as the `\verb` command changes the catcode regime before reading its argument, and restore it immediately afterward, nevertheless with a macro argument the content of the argument has already be converted to a token list along the catcode regime in effect when the macro was called. However, the `cprotect` package can help with this.
### 8.28 `verse`
Synopsis:
```
\begin{verse}
  line1 \\
  line2 \\
  ...
\end{verse}

```

An environment for poetry.
Here are two lines from Shakespeare’s Romeo and Juliet.
```
Then plainly know my heart's dear love is set \\
On the fair daughter of rich Capulet.

```

Separate the lines of each stanza with `\\`, and use one or more blank lines to separate the stanzas.
```
\begin{verse}
\makebox[\linewidth][c]{\textit{Shut Not Your Doors} ---Walt Whitman}
  \\[1\baselineskip]
Shut not your doors to me proud libraries,                  \\
For that which was lacking on all your well-fill'd shelves, \\
\qquad yet needed most, I bring,                             \\
Forth from the war emerging, a book I have made,            \\
The words of my book nothing, the drift of it every thing,  \\
A book separate, not link'd with the rest nor felt by the intellect, \\
But you ye untold latencies will thrill to every page.
\end{verse}

```

The output has margins indented on the left and the right, paragraphs are not indented, and the text is not right-justified.
