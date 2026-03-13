# Appendix A Document templates

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- A.1 `beamer` template
- A.2 `article` template
- A.3 `book` template
- A.4 Larger `book` template

## Appendix A Document templates
Although illustrative material, perhaps these document templates will be useful. Additional template resources are listed at <https://tug.org/interest.html#latextemplates>.
  * [`beamer` template](https://latexref.xyz/dev/latex2e.html#beamer-template)
  * [`article` template](https://latexref.xyz/dev/latex2e.html#article-template)
  * [`book` template](https://latexref.xyz/dev/latex2e.html#book-template)
  * [Larger `book` template](https://latexref.xyz/dev/latex2e.html#Larger-book-template)

### A.1 `beamer` template
The `beamer` class creates presentation slides. It has a vast array of features, but here is a basic template:
```
\documentclass{beamer}

\title{Beamer Class template}
\author{Alex Author}
\date{July 31, 2020}

\begin{document}

\maketitle

% without [fragile], any {verbatim} code gets mysterious errors.
\begin{frame}[fragile]
 \frametitle{First Slide}

\begin{verbatim}
  This is \verbatim!
\end{verbatim}

\end{frame}

\end{document}

```

The Beamer package on CTAN: <https://ctan.org/pkg/beamer>.
### A.2 `article` template
A simple template for an article.
```
\documentclass{article}
\title{Article Class Template}
\author{Alex Author}

\begin{document}
\maketitle

\section{First section}
Some text.

\subsection{First section, first subsection}
Additional text.

\section{Second section}
Some more text.

\end{document}

```

### A.3 `book` template
This is a straightforward template for a book. See [Larger `book` template](https://latexref.xyz/dev/latex2e.html#Larger-book-template), for a more elaborate one.
```
\documentclass{book}
\title{Book Class Template}
\author{Alex Author}

\begin{document}
\maketitle

\chapter{First}
Some text.

\chapter{Second}
Some other text.

\section{A subtopic}
The end.

\end{document}

```

### A.4 Larger `book` template
This is a somewhat elaborate template for a book. See [`book` template](https://latexref.xyz/dev/latex2e.html#book-template), for a simpler one.
This template uses `\frontmatter`, `\mainmatter`, and `\backmatter` to control the typography of the three main areas of a book (see [`\frontmatter`, `\mainmatter`, `\backmatter`](https://latexref.xyz/dev/latex2e.html#g_t_005cfrontmatter-_0026-_005cmainmatter-_0026-_005cbackmatter)). The book has a bibliography and an index.
Also notable is that it uses `\include` and `\includeonly` (see [Splitting the input](https://latexref.xyz/dev/latex2e.html#Splitting-the-input)). While you are working on a chapter you can comment out all the other chapter entries from the argument to `\includeonly`. That will speed up compilation without losing any information such as cross-references. (Material that does not need to come on a new page is brought in with `\input` instead of `\include`. You don’t get the cross-reference benefit with `\input`.)
```
\documentclass[titlepage]{book}
\usepackage{makeidx}\makeindex

\title{Book Class Template}
\author{Alex Author}

\includeonly{%
% frontcover,
  preface,
  chap1,
% appenA,
  }

\begin{document}
\frontmatter
\include{frontcover}
  % maybe comment out while drafting:
\maketitle \input{dedication} \input{copyright}
\tableofcontents
\include{preface}

\mainmatter
\include{chap1}
...
\appendix
\include{appenA}
...

\backmatter
\bibliographystyle{apalike}
\addcontentsline{toc}{chapter}{Bibliography}
\bibliography
\addcontentsline{toc}{chapter}{Index}
\printindex

\include{backcover}
\end{document}

```
