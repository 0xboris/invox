# 07 Cross references

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 7.1 `\label`
- 7.2 `\pageref`
- 7.3 `\ref`
- 7.4 `xr` package

## 7 Cross references
We often want something like ‘See Theorem~31’. But by-hand typing the 31 is poor practice. Instead you should write a _label_ such as `\label{eq:GreensThm}` and then _reference_ it, as with `See equation~\ref{eq:GreensThm}`. LaTeX will automatically work out the number, put it into the output, and will change that number later if needed.
```
We will see this with Theorem~\ref{th:GreensThm}. % forward reference
...
\begin{theorem} \label{th:GreensThm}
  ...
\end{theorem}
...
See Theorem~\ref{th:GreensThm} on page~\pageref{th:GreensThm}.

```

LaTeX tracks cross reference information in a file having the extension .aux and with the same base name as the file containing the `\label`. So if `\label` is in calculus.tex then the information is in calculus.aux. LaTeX puts the information in that file every time it runs across a `\label`.
The behavior described in the prior paragraph results in a quirk that happens when your document has a _forward reference_ , a `\ref` that appears before the associated `\label`. If this is the first time that you are compiling the document then you will get ‘LaTeX Warning: Label(s) may have changed. Rerun to get cross references right’ and in the output the forward reference will appear as two question marks ‘??’, in boldface. A similar thing happens if you change some things so the references changes; you get the same warning and the output contains the old reference information. In both cases, resolve this by compiling the document a second time.
The `cleveref` package enhances LaTeX’s cross referencing features. You can arrange that if you enter `\begin{thm}\label{th:Nerode}...\end{thm}` then `\cref{th:Nerode}` will output ‘Theorem 3.21’, without you having to enter the “Theorem.”
  * [`\label`](https://latexref.xyz/dev/latex2e.html#g_t_005clabel)
  * [`\pageref`](https://latexref.xyz/dev/latex2e.html#g_t_005cpageref)
  * [`\ref`](https://latexref.xyz/dev/latex2e.html#g_t_005cref)
  * [`xr` package](https://latexref.xyz/dev/latex2e.html#xr-package)

### 7.1 `\label`
Synopsis:
```
\label{key}

```

Assign a reference number to key. In ordinary text `\label{key}` assigns to key the number of the current sectional unit. Inside an environment with numbering, such as a `table` or `theorem` environment, `\label{key}` assigns to key the number of that environment. Retrieve the assigned number with the `\ref{key}` command (see [`\ref`](https://latexref.xyz/dev/latex2e.html#g_t_005cref)).
A key name can consist of any sequence of letters, digits, or common punctuation characters. Upper and lowercase letters are distinguished, as usual.
A common convention is to use labels consisting of a prefix and a suffix separated by a colon or period. Thus, `\label{fig:Post}` is a label for a figure with a portrait of Emil Post. This helps to avoid accidentally creating two labels with the same name, and makes your source more readable. Some commonly-used prefixes:

`ch`

for chapters

`sec`

`subsec`

for lower-level sectioning commands

`fig`

for figures

`tab`

for tables

`eq`

for equations
In the auxiliary file the reference information is kept as the text of a command of the form `\newlabel{label}{{currentlabel}{pagenumber}}`. Here currentlabel is the current value of the macro `\@currentlabel` that is usually updated whenever you call `\refstepcounter{counter}`.
Below, the key `sec:test` will get the number of the current section and the key `fig:test` will get the number of the figure. (Incidentally, put labels after captions in figures and tables.)
```
\section{section name}
\label{sec:test}
This is Section~\ref{sec:test}.
\begin{figure}
  ...
  \caption{caption text}
  \label{fig:test}
\end{figure}
See Figure~\ref{fig:test}.

```

### 7.2 `\pageref`
Synopsis:
```
\pageref{key}

```

Produce the page number of the place in the text where the corresponding `\label`{key} command appears.
If there is no `\label{key}` then you get something like ‘LaTeX Warning: Reference `th:GrensThm' on page 1 undefined on input line 11.’
Below, the `\label{eq:main}` is used both for the formula number and for the page number. (Note that the two references are forward references so this document would need to be compiled twice to resolve those.)
```
The main result is formula~\ref{eq:main} on page~\pageref{eq:main}.
  ...
\begin{equation} \label{eq:main}
   \mathbf{P}=\mathbf{NP}
\end{equation}

```

### 7.3 `\ref`
Synopsis:
```
\ref{key}

```

Produces the number of the sectional unit, equation, footnote, figure, …, of the corresponding `\label` command (see [`\label`](https://latexref.xyz/dev/latex2e.html#g_t_005clabel)). It does not produce any text, such as the word ‘Section’ or ‘Figure’, just the bare number itself.
If there is no `\label{key}` then you get something like ‘LaTeX Warning: Reference `th:GrensThm' on page 1 undefined on input line 11.’
In this example the `\ref{popular}` produces ‘2’. Note that it is a forward reference since it comes before `\label{popular}` so this document would have to be compiled twice.
```
The most widely-used format is item number~\ref{popular}.
\begin{enumerate}
\item Plain \TeX
\item \label{popular} \LaTeX
\item Con\TeX t
\end{enumerate}

```

The `cleveref` package includes text such as ‘Theorem’ in the reference. See the documentation on CTAN.
### 7.4 `xr` package
Synopsis:
```
\usepackage{xr}
  \externaldocument{document-basename}

```

or
```
\usepackage{xr}
  \externaldocument[reference-prefix]{document-basename}

```

Make cross references to the external document document-basename.tex.
Here is an example. If lectures.tex has this in the preamble
```
\usepackage{xr}
  \externaldocument{exercises}
  \externaldocument[H-]{hints}
  \externaldocument{answers}

```

then it can use cross reference labels from the other three documents. Suppose that exercises.tex has an enumerated list containing this,
```
\item \label{exer:EulersThm} What if every vertex has odd degree?

```

and hints.tex has an enumerated list with this,
```
\item \label{exer:EulersThm} Distinguish the case of two vertices.

```

and answers.tex has an enumerated list with this,
```
\item \label{ans:EulersThm} There is no Euler path, except if there
  are exactly two vertices.

```

After compiling the exercises, hints, and answers documents, entering this in the body of lectures.tex will result in the lectures getting the reference numbers used in the other documents.
```
See Exercise~\ref{exer:EulersThm}, with Hint~\ref{H-exer:EulersThm}.
The solution is Answer~\ref{ans:EulersThm}.

```

The prefix `H-` for the reference from the hints file is needed because the label in the hints file is the same as the label in the exercises file. Without that prefix, both references would get the number from the later file.
Note: if the document uses the `hyperref` package then in place of `xr`, put `\usepackage{xr-hyper}` before the `\usepackage{hyperref}`. Also, if any of the multiple documents uses `hyperref` then they all must use it.
