# 15 Making paragraphs

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 15.1 `\par`
- 15.2 `\indent` & `\noindent`
- 15.3 `\parindent` & `\parskip`
- 15.4 Marginal notes

## 15 Making paragraphs
To start a paragraph, just type some text. To end the current paragraph, put an empty line. This is three paragraphs, the separation of which is made by two empty lines.
```
It is a truth universally acknowledged, that a single man in possession
of a good fortune, must be in want of a wife.

However little known the feelings or views of such a man may be on his
first entering a neighbourhood, this truth is so well fixed in the minds
of the surrounding families, that he is considered the rightful property
of some one or other of their daughters.

``My dear Mr. Bennet,'' said his lady to him one day,
``have you heard that Netherfield Park is let at last?''

```

A paragraph separator can be made of a sequence of at least one blank line, at least one of which is not terminated by a comment. A blank line is a line that is empty or made only of blank characters such as space or tab. Comments in source code are started with a `%` and span up to the end of line. In the following example the two columns are identical:
```
\documentclass[twocolumn]{article}
\begin{document}
First paragraph.

Second paragraph.
\newpage
First paragraph.

  % separator lines may contain blank characters.

Second paragraph.
\end{document}

```

Once LaTeX has gathered all of a paragraph’s contents it divides that content into lines in a way that is optimized over the entire paragraph (see [Line breaking](https://latexref.xyz/dev/latex2e.html#Line-breaking)).
There are places where a new paragraph is not permitted. Don’t put a blank line in math mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)); here the blank line before the `\end{equation}`
```
\begin{equation}
  2^{|S|} > |S|

\end{equation}

```

will get you the error ‘Missing $ inserted’. Similarly, the blank line in this `\section` argument
```
\section{aaa

bbb}

```

gets ‘Runaway argument? {aaa ! Paragraph ended before \@sect was complete’.
  * [`\par`](https://latexref.xyz/dev/latex2e.html#g_t_005cpar)
  * [`\indent` & `\noindent`](https://latexref.xyz/dev/latex2e.html#g_t_005cindent-_0026-_005cnoindent)
  * [`\parindent` & `\parskip`](https://latexref.xyz/dev/latex2e.html#g_t_005cparindent-_0026-_005cparskip)
  * [Marginal notes](https://latexref.xyz/dev/latex2e.html#Marginal-notes)

### 15.1 `\par`
Synopsis (note that while reading the input TeX converts any sequence of one or more blank lines to a `\par`, [Making paragraphs](https://latexref.xyz/dev/latex2e.html#Making-paragraphs)):
```
\par

```

End the current paragraph. The usual way to separate paragraphs is with a blank line but the `\par` command is entirely equivalent. This command is robust (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
This example uses `\par` rather than a blank line simply for readability.
```
\newcommand{\syllabusLegalese}{%
  \whatCheatingIs\par\whatHappensWhenICatchYou}

```

In LR mode the `\par` command does nothing and is ignored. In paragraph mode, the `\par` command terminates paragraph mode, switching LaTeX to vertical mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)).
You cannot use the `\par` command in a math mode. You also cannot use it in the argument of many commands, such as the sectioning commands, e.g. `\section` (see [Making paragraphs](https://latexref.xyz/dev/latex2e.html#Making-paragraphs) and [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand)).
The `\par` command is not the same as the `\paragraph` command. The latter is, like `\section` or `\subsection`, a sectioning command used by the LaTeX document standard classes (see [`\subsubsection`, `\paragraph`, `\subparagraph`](https://latexref.xyz/dev/latex2e.html#g_t_005csubsubsection-_0026-_005cparagraph-_0026-_005csubparagraph)).
The `\par` command is not the same as `\newline` or the line break double backslash, `\\`. The difference is that `\par` ends the paragraph, not just the line, and also triggers the addition of the between-paragraph vertical space `\parskip` (see [`\parindent` & `\parskip`](https://latexref.xyz/dev/latex2e.html#g_t_005cparindent-_0026-_005cparskip)).
The output from this example
```
xyz

\setlength{\parindent}{3in}
\setlength{\parskip}{5in}
\noindent test\indent test1\par test2

```

is: after ‘xyz’ there is a vertical skip of 5 inches and then ‘test’ appears, aligned with the left margin. On the same line, there is an empty horizontal space of 3 inches and then ‘test1’ appears. Finally. there is a vertical space of 5 inches, followed by a fresh paragraph with a paragraph indent of 3 inches, and then LaTeX puts the text ‘test2’.
### 15.2 `\indent` & `\noindent`
Synopsis:
```
\indent

```

or
```
\noindent

```

Go into horizontal mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). The `\indent` command first outputs an empty box whose width is `\parindent`. These commands are robust (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
Ordinarily you create a new paragraph by putting in a blank line. See [`\par`](https://latexref.xyz/dev/latex2e.html#g_t_005cpar), for the difference between this command and `\par`. To start a paragraph without an indent, or to continue an interrupted paragraph, use `\noindent`.
In the middle of a paragraph the `\noindent` command has no effect, because LaTeX is already in horizontal mode there. The `\indent` command’s only effect is to output a space.
This example starts a fresh paragraph.
```
... end of the prior paragraph.

\noindent This paragraph is not indented.

```

and this continues an interrupted paragraph.
```
The data

\begin{center}
  \begin{tabular}{rl} ... \end{tabular}
\end{center}

\noindent shows this clearly.

```

To omit indentation in the entire document put `\setlength{\parindent}{0pt}` in the preamble. If you do that, you may want to also set the length of spaces between paragraphs, `\parskip` (see [`\parindent` & `\parskip`](https://latexref.xyz/dev/latex2e.html#g_t_005cparindent-_0026-_005cparskip)).
Default LaTeX styles have the first paragraph after a section that is not indented, as is traditional typesetting in English. To change that, look on CTAN for the package `indentfirst`.
### 15.3 `\parindent` & `\parskip`
Synopsis:
```
\setlength{\parindent}{horizontal len}
\setlength{\parskip}{vertical len}

```

Both are rubber lengths (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). They affect the indentation of ordinary paragraphs, not paragraphs inside minipages (see [`minipage`](https://latexref.xyz/dev/latex2e.html#minipage)), and the vertical space between paragraphs, respectively.
For example, if this is put in the preamble:
```
\setlength{\parindent}{0em}
\setlength{\parskip}{1ex}

```

The document will have paragraphs that are not indented, but instead are vertically separated by about the height of a lowercase ‘x’.
In LaTeX standard class documents, the default value for `\parindent` in one-column documents is `15pt` when the default text size is `10pt`, `17pt` for `11pt`, and `1.5em` for `12pt`. In two-column documents it is `1em`. (These values are set before LaTeX calls `\normalfont` so `em` is derived from the default font, Computer Modern. If you use a different font then to set `\parindent` to 1em matching that font, put `\AtBeginDocument{\setlength{\parindent}{1em}}` in the preamble.)
The default value for `\parskip` in LaTeX’s standard document classes is `0pt plus1pt`.
### 15.4 Marginal notes
Synopsis, one of:
```
\marginpar{right}
\marginpar[left]{right}

```

Create a note in the margin. The first line of the note will have the same baseline as the line in the text where the `\marginpar` occurs.
The margin that LaTeX uses for the note depends on the current layout (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)) and also on `\reversemarginpar` (see below). If you are using one-sided layout (document option `oneside`) then it goes in the right margin. If you are using two-sided layout (document option `twoside`) then it goes in the outside margin. If you are in two-column layout (document option `twocolumn`) then it goes in the nearest margin.
If you declare `\reversemarginpar` then LaTeX will place subsequent marginal notes in the opposite margin to that given in the prior paragraph. Revert that to the default position with `\normalmarginpar`.
When you specify the optional argument left then it is used for a note in the left margin, while the mandatory argument right is used for a note in the right margin.
Normally, a note’s first word will not be hyphenated. You can enable hyphenation there by beginning left or right with `\hspace{0pt}`.
These parameters affect the formatting of the note:

`\marginparpush`

Minimum vertical space between notes; default ‘7pt’ for ‘12pt’ documents, ‘5pt’ else. See also [page layout parameters marginparpush](https://latexref.xyz/dev/latex2e.html#page-layout-parameters-marginparpush).

`\marginparsep`

Horizontal space between the main text and the note; default ‘11pt’ for ‘10pt’ documents, ‘10pt’ else.

`\marginparwidth`

Width of the note itself; default for a one-sided ‘10pt’ document is ‘90pt’, ‘83pt’ for ‘11pt’, and ‘68pt’ for ‘12pt’; ‘17pt’ more in each case for a two-sided document. In two column mode, the default is ‘48pt’.
The standard LaTeX routine for marginal notes does not prevent notes from falling off the bottom of the page.
