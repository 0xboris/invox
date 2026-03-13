# 09 Line breaking

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 9.1 `\\`
- 9.2 `\obeycr` & `\restorecr`
- 9.3 `\newline`
- 9.4 `\-` (discretionary hyphen)
- 9.5 `\slash`: breakable ‘/’
- 9.6 `\discretionary` (generalized hyphenation point)
- 9.7 `\fussy` & `\sloppy`
- 9.8 `\hyphenation`
- 9.9 `\linebreak` & `\nolinebreak`

## 9 Line breaking
The first thing LaTeX does when processing ordinary text is to translate your input file into a sequence of glyphs and spaces. To produce a printed document, this sequence must be broken into lines (and these lines must be broken into pages).
LaTeX usually does the line (and page) breaking in the text body for you but in some environments you manually force line breaks.
A common workflow is to get a final version of the document content before taking a final pass through and considering line breaks (and page breaks). This differs from word processing, where you are formatting text as you input it. Putting these off until the end prevents a lot of fiddling with breaks that will change anyway.
  * [`\\`](https://latexref.xyz/dev/latex2e.html#g_t_005c_005c)
  * [`\obeycr` & `\restorecr`](https://latexref.xyz/dev/latex2e.html#g_t_005cobeycr-_0026-_005crestorecr)
  * [`\newline`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewline)
  * [`\-` (discretionary hyphen)](https://latexref.xyz/dev/latex2e.html#g_t_005c_002d-_0028hyphenation_0029)
  * [`\slash`: breakable ‘/’](https://latexref.xyz/dev/latex2e.html#g_t_005cslash)
  * [`\discretionary` (generalized hyphenation point)](https://latexref.xyz/dev/latex2e.html#g_t_005cdiscretionary)
  * [`\fussy` & `\sloppy`](https://latexref.xyz/dev/latex2e.html#g_t_005cfussy-_0026-_005csloppy)
  * [`\hyphenation`](https://latexref.xyz/dev/latex2e.html#g_t_005chyphenation)
  * [`\linebreak` & `\nolinebreak`](https://latexref.xyz/dev/latex2e.html#g_t_005clinebreak-_0026-_005cnolinebreak)

### 9.1 `\\`
Synopsis, one of:
```
\\
\\[morespace]

```

or one of:
```
\\*
\\*[morespace]

```

End the current line. The optional argument morespace specifies extra vertical space to be inserted before the next line. This is a rubber length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)) and can be negative. The text before the line break is set at its normal length, that is, it is not stretched to fill out the line width. This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
```
\title{My story: \\[0.25in]
       a tale of woe}

```

The starred form, `\\*`, tells LaTeX not to start a new page between the two lines, by issuing a `\nobreak`.
Explicit line breaks in the main text body are unusual in LaTeX. In particular, don’t start new paragraphs with `\\`. Instead leave a blank line between the two paragraphs. And don’t put in a sequence of `\\`’s to make vertical space. Instead use `\vspace{length}`, or `\leavevmode\vspace{length}`, or `\vspace*{length}` if you want the space to not be thrown out at the top of a new page (see [`\vspace`](https://latexref.xyz/dev/latex2e.html#g_t_005cvspace)).
The `\\` command is mostly used outside of the main flow of text such as in a `tabular` or `array` environment or in an equation environment.
The `\\` command is a synonym for `\newline` (see [`\newline`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewline)) under ordinary circumstances (an example of an exception is the `p{...}` column in a `tabular` environment; see [`tabular`](https://latexref.xyz/dev/latex2e.html#tabular)).
The `\\` command is a macro, and its definition changes by context so that its definition in normal text, a `center` environment, a `flushleft` environment, and a `tabular` are all different. In normal text when it forces a linebreak it is essentially a shorthand for `\newline`. It does not end horizontal mode or end the paragraph, it just inserts some glue and penalties so that when the paragraph does end a linebreak will occur at that point, with the short line padded with white space.
You get ‘LaTeX Error: There's no line here to end’ if you use `\\` to ask for a new line, rather than to end the current line. An example is if you have `\begin{document}\\` or, more likely, something like this.
```
\begin{center}
  \begin{minipage}{0.5\textwidth}
  \\
  In that vertical space put your mark.
  \end{minipage}
\end{center}

```

Fix it by replacing the double backslash with something like `\vspace{\baselineskip}`.
### 9.2 `\obeycr` & `\restorecr`
The `\obeycr` command makes a return in the input file (‘^^M’, internally) the same as `\\`, followed by `\relax`. So each new line in the input will also be a new line in the output. The `\restorecr` command restores normal line-breaking behavior.
This is not the way to show verbatim text or computer code. Use `verbatim` (see [`verbatim`](https://latexref.xyz/dev/latex2e.html#verbatim)) instead.
With LaTeX’s usual defaults, this
```
aaa
bbb

\obeycr
ccc
ddd
   eee

\restorecr
fff
ggg

hhh
iii

```

produces output like this.
```
  aaa bbb
  ccc
ddd
eee

fff ggg
  hhh iii

```

The indents are paragraph indents.
### 9.3 `\newline`
In ordinary text, this ends a line in a way that does not right-justify it, so the text before the end of line is not stretched. That is, in paragraph mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)), the `\newline` command is equivalent to double-backslash (see [`\\`](https://latexref.xyz/dev/latex2e.html#g_t_005c_005c)). This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
However, the two commands are different inside a `tabular` or `array` environment. In a column with a specifier producing a paragraph box such as typically `p{...}`, `\newline` will insert a line end inside of the column; that is, it does not break the entire tabular row. To break the entire row use `\\` or its equivalent `\tabularnewline`.
This will print ‘Name:’ and ‘Address:’ as two lines in a single cell of the table.
```
\begin{tabular}{p{1in}@{\hspace{2in}}p{1in}}
  Name: \newline Address: &Date: \\ \hline
\end{tabular}

```

The ‘Date:’ will be baseline-aligned with ‘Name:’.
### 9.4 `\-` (discretionary hyphen)
Tell LaTeX that it may hyphenate the word at the given point. When you insert `\-` commands in a word, the word will only be hyphenated at those points and not at any of the other hyphenation points that LaTeX might otherwise have chosen. This command is robust (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
LaTeX is good at hyphenating and usually finds most of the correct hyphenation points, while almost never using an incorrect one. The `\-` command is for exceptional cases.
For example, LaTeX does not ordinarily hyphenate words containing a hyphen. Below, the long and hyphenated word means LaTeX has to put in unacceptably large spaces to set the narrow column.
```
\begin{tabular}{rp{1.75in}}
  Isaac Asimov &The strain of
               anti-intellectualism
               % an\-ti-in\-tel\-lec\-tu\-al\-ism
               has been a constant thread winding its way through our
               political and cultural life, nurtured by
               the false notion that democracy means that
               `my ignorance is just as good as your knowledge'.
\end{tabular}

```

Commenting out the third line and uncommenting the fourth makes a much better fit.
The `\-` command only allows LaTeX to break there, it does not require that it break there. You can force a split with something like `Hef-\linebreak feron`. Of course, if you later change the text then this forced break may look out of place, so this approach requires care.
### 9.5 `\slash`: breakable ‘/’
The `\slash` command produces a ‘/’ character and then a penalty of the same value as an explicit ‘-’ character (`\exhyphenpenalty`). This allows TeX to break a line at the ‘/’, similar to a hyphen. Hyphenation is allowed in the word part preceding the ‘/’, but not after. For example:
```
The input\slash output of the program is complicated.

```

### 9.6 `\discretionary` (generalized hyphenation point)
Synopsis:
```
\discretionary{pre-break}{post-break}{no-break}

```

Handle word changes around hyphens. This command is not often used in LaTeX documents.
If a line break occurs at the point where `\discretionary` appears then TeX puts pre-break at the end of the current line and puts post-break at the start of the next line. If there is no line break here then TeX puts no-break.
In ‘difficult’ the three letters `ffi` form a ligature. But TeX can nonetheless break between the two ‘f’’s with this.
```
di\discretionary{f-}{fi}{ffi}cult

```

Note that users do not have to do this. It is typically handled automatically by TeX’s hyphenation algorithm.
### 9.7 `\fussy` & `\sloppy`
Declarations to make TeX more picky or less picky about line breaking. Declaring `\fussy` usually avoids too much space between words, at the cost of an occasional overfull box. Conversely, `\sloppy` avoids overfull boxes while suffering loose interword spacing.
The default is `\fussy`. Line breaking in a paragraph is controlled by whichever declaration is current at the end of the paragraph, i.e., at the blank line or `\par` or displayed equation ending that paragraph. So to affect the line breaks, include that paragraph-ending material in the scope of the command.
  * [`sloppypar`](https://latexref.xyz/dev/latex2e.html#sloppypar)

#### 9.7.1 `sloppypar`
Synopsis:
```
\begin{sloppypar}
  ... paragraphs ...
\end{sloppypar}

```

Typeset the paragraphs with `\sloppy` in effect (see [`\fussy` & `\sloppy`](https://latexref.xyz/dev/latex2e.html#g_t_005cfussy-_0026-_005csloppy)). Use this to locally adjust line breaking, to avoid ‘Overfull box’ or ‘Underfull box’ errors.
The example is simple.
```
\begin{sloppypar}
  Her plan for the morning thus settled, she sat quietly down to her
  book after breakfast, resolving to remain in the same place and the
  same employment till the clock struck one; and from habitude very
  little incommoded by the remarks and ejaculations of Mrs.\ Allen,
  whose vacancy of mind and incapacity for thinking were such, that
  as she never talked a great deal, so she could never be entirely
  silent; and, therefore, while she sat at her work, if she lost her
  needle or broke her thread, if she heard a carriage in the street,
  or saw a speck upon her gown, she must observe it aloud, whether
  there were anyone at leisure to answer her or not.
\end{sloppypar}

```

### 9.8 `\hyphenation`
Synopsis:
```
\hyphenation{word1 ...}

```

Declares allowed hyphenation points within the words in the list. The words in that list are separated by spaces. Show permitted points for hyphenation with an ASCII dash character, `-`.
Here is an example:
```
\hyphenation{hat-er il-lit-e-ra-ti tru-th-i-ness}

```

Use lowercase letters. TeX will only hyphenate if the word matches exactly; no inflections are tried. Multiple `\hyphenation` commands accumulate.
### 9.9 `\linebreak` & `\nolinebreak`
Synopses, one of:
```
\linebreak
\linebreak[zero-to-four]

```

or one of these.
```
\nolinebreak
\nolinebreak[zero-to-four]

```

Encourage or discourage a line break. The optional zero-to-four is an integer lying between 0 and 4 that allows you to soften the instruction. The default is 4, so that without the optional argument these commands entirely force or prevent the break. But for instance, `\nolinebreak[1]` is a suggestion that another place may be better. The higher the number, the more insistent the request. Both commands are fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
Here we tell LaTeX that a good place to put a linebreak is after the standard legal text.
```
\boilerplatelegal{} \linebreak[2]
We especially encourage applications from members of traditionally
underrepresented groups.

```

When you issue `\linebreak`, the spaces in the line are stretched out so that the break point reaches the right margin. See [`\\`](https://latexref.xyz/dev/latex2e.html#g_t_005c_005c) and [`\newline`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewline), to have the spaces not stretched out.
