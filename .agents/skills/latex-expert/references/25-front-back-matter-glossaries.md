# 25 Front/back matter: Glossaries

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 25.3.1 `\newglossaryentry`
- 25.3.2 `\gls`

### 25.3 Glossaries
Synopsis:
```
\usepackage{glossaries} \makeglossaries
  ...
\newglossaryentry{label}{settings}
  ...
\gls{label}.
  ...
\printglossaries

```

The glossaries package allows you to make glossaries, including multiple glossaries, as well as lists of acronyms.
To get the output from this example, compile the document (for instance with `pdflatex filename`), then run the command line command `makeglossaries filename`, and then compile the document again.
```
\documentclass{...}
\usepackage{glossaries} \makeglossaries
\newglossaryentry{tm}{%
  name={Turing machine},
  description={A model of a machine that computes.  The model is simple
               but can compute anything any existing device can compute.
               It is the standard model used in Computer Science.},
  }
\begin{document}
Everything begins with the definition of a \gls{tm}.
  ...
\printglossaries
\end{document}

```

That gives two things. In the main text it outputs ‘... definition of a Turing machine’. In addition, in a separate sectional unit headed ‘Glossary’ there appears a description list. In boldface it says ‘Turing machine’ and the rest of the item says in normal type ‘A model of a machine … Computer Science’.
The command `\makeglossary` opens the file that will contain the entry information, root-file.glo. Put the `\printglossaries` command where you want the glossaries to appear in your document.
The glossaries package is very powerful. For instance, besides the commands `\newglossaryentry` and `\gls`, there are similar commands for a list of acronyms. See the package documentations on CTAN.
  * [`\newglossaryentry`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewglossaryentry)
  * [`\gls`](https://latexref.xyz/dev/latex2e.html#g_t_005cgls)

#### 25.3.1 `\newglossaryentry`
Synopsis, one of:
```
\newglossaryentry{label}
{
  name={name},
  description={description},
  other options, ...
}

```

or
```
\longnewglossaryentry{label}
{
  name={name},
  other options ...,
}
{description}

```

Declare a new entry for a glossary. The label must be unique for the document. The settings associated with the label are pairs: `key=value`.
This puts the blackboard bold symbol for the real numbers ℝ, in the glossary.
```
\newglossaryentry{R}
{
  name={\ensuremath{\mathbb{R}}},
  description={the real numbers},
}

```

Use the second command form if the description spans more than one paragraph.
For a full list of keys see the package documentation on CTAN but here are a few.

`name`

(Required.) The word, phrase, or symbol that you are defining.

`description`

(Required.) The description that will appear in the glossary. If this has more than one paragraph then you must use the second command form given in the synopsis.

`plural`

The plural form of name. Refer to the plural form using `\glspl` or `\Glspl` (see [`\gls`](https://latexref.xyz/dev/latex2e.html#g_t_005cgls)).

`sort`

How to place this entry in the list of entries that the glossary holds.

`symbol`

A symbol, such as a mathematical symbol, besides the name.
#### 25.3.2 `\gls`
Synopsis, one of:
```
\gls{label}
\glspl{label}
\Gls{label}
\Glspl{label}

```

Refer to a glossary entry. The entries are declared with `\newglossaryentry` (see [`\newglossaryentry`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewglossaryentry)).
This
```
\newglossaryentry{N}{%
  name={the natural numbers},
  description={The numbers $0$, $1$, $2$, $\ldots$\@},
  symbol={\ensuremath{\mathbb{N}}},
  }
  ...
Consider \gls{N}.

```

gives the output ‘Consider the natural numbers’.
The second command form `\glspl{label}` produces the plural of name (by default it tries adding an ‘s’). The third form capitalizes the first letter of name, as does the fourth form, which also takes the plural.
