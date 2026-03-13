# 12 Definitions: Counters, lengths, boxes, environments, and theorems

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 12.5 `\newcounter`: Allocating a counter
- 12.6 `\newlength`
- 12.7 `\newsavebox`
- 12.8 `\newenvironment` & `\renewenvironment`
- 12.9 `\newtheorem`
- 12.10 `\newfont`

### 12.5 `\newcounter`: Allocating a counter
Synopsis, one of:
```
\newcounter{countername}
\newcounter{countername}[supercounter]

```

Globally defines a new counter named countername and initialize it to zero (see [Counters](https://latexref.xyz/dev/latex2e.html#Counters)).
The name countername must consist of letters only. It does not begin with a backslash. This name must not already be in use by another counter.
When you use the optional argument `[supercounter]` then the counter countername will be reset to zero whenever supercounter is incremented. For example, ordinarily `subsection` is numbered within `section` so that any time you increment section, either with `\stepcounter` (see [`\stepcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005cstepcounter)) or `\refstepcounter` (see [`\refstepcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005crefstepcounter)), then LaTeX will reset subsection to zero.
This example
```
\newcounter{asuper}  \setcounter{asuper}{1}
\newcounter{asub}[asuper] \setcounter{asub}{3}   % Note `asuper'
The value of asuper is \arabic{asuper} and of asub is \arabic{asub}.
\stepcounter{asuper}
Now asuper is \arabic{asuper} while asub is \arabic{asub}.

```

produces ‘The value of asuper is 1 and that of asub is 3’ and ‘Now asuper is 2 while asub is 0’.
If the counter already exists, for instance by entering `asuper` twice, then you get something like ‘LaTeX Error: Command \c@asuper already defined. Or name \end... illegal, see p.192 of the manual.’.
If you use the optional argument then the super counter must already exist. Entering `\newcounter{jh}[lh]` when `lh` is not a defined counter will get you ‘LaTeX Error: No counter 'lh' defined.’
### 12.6 `\newlength`
Synopsis:
```
\newlength{\len}

```

Allocate a new length register (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). The required argument `\len` has to be a control sequence (see [Control sequence, control word and control symbol](https://latexref.xyz/dev/latex2e.html#Control-sequences)), and as such must begin with a backslash, `\` under normal circumstances. The new register holds rubber lengths such as `72.27pt` or `1in plus.2in minus.1in` (a LaTeX length register is what plain TeX calls a `skip` register). The initial value is zero. The control sequence `\len` must not be already defined.
An example:
```
\newlength{\graphichgt}

```

If you forget the backslash then you get ‘Missing control sequence inserted’. If the control sequence already exists then you get something like ‘LaTeX Error: Command \graphichgt already defined. Or name \end... illegal, see p.192 of the manual’.
### 12.7 `\newsavebox`
Synopsis:
```
\newsavebox{\cmd}

```

Define \cmd, the string consisting of a backslash followed by cmd, to refer to a new bin for storing material. These bins hold material that has been typeset, to use multiple times or to measure or manipulate (see [Boxes](https://latexref.xyz/dev/latex2e.html#Boxes)). The bin name \cmd is required, must start with a backslash, \, and must not already be a defined command. This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
This allocates a bin and then puts typeset material into it.
```
\newsavebox{\logobox}
\savebox{\logobox}{LoGo}
Our logo is \usebox{\logobox}.

```

The output is ‘Our logo is LoGo’.
If there is an already defined bin then you get something like ‘LaTeX Error: Command \logobox already defined. Or name \end... illegal, see p.192 of the manual’.
The allocation of a box is global.
### 12.8 `\newenvironment` & `\renewenvironment`
Synopses, one of:
```
\newenvironment{env}{begdef}{enddef}
\newenvironment{env}[nargs]{begdef}{enddef}
\newenvironment{env}[nargs][optargdefault]{begdef}{enddef}
\newenvironment*{env}{begdef}{enddef}
\newenvironment*{env}[nargs]{begdef}{enddef}
\newenvironment*{env}[nargs][optargdefault]{begdef}{enddef}

```

or one of these.
```
\renewenvironment{env}{begdef}{enddef}
\renewenvironment{env}[nargs]{begdef}{enddef}
\renewenvironment{env}[nargs][optargdefault]{begdef}{enddef}
\renewenvironment*{env}{begdef}{enddef}
\renewenvironment*{env}[nargs]{begdef}{enddef}
\renewenvironment*{env}[nargs][optargdefault]{begdef}{enddef}

```

Define or redefine the environment env, that is, create the construct `\begin{env} ... body ... \end{env}`.
The starred form of these commands requires that the arguments not contain multiple paragraphs of text. However, the body of these environments can contain multiple paragraphs.

env

Required; the environment name. It consists only of letters or the `*` character, and thus does not begin with backslash, `\`. It must not begin with the string `end`. For `\newenvironment`, the name env must not be the name of an already existing environment, and also the command `\env` must be undefined. For `\renewenvironment`, env must be the name of an existing environment.

nargs

Optional; an integer from 0 to 9 denoting the number of arguments of that the environment takes. When you use the environment these arguments appear after the `\begin`, as in `\begin{env}{arg1} ... {argn}`. Omitting this is equivalent to setting it to 0; the environment will have no arguments. When redefining an environment, the new version can have a different number of arguments than the old version.

optargdefault

Optional; if this is present then the first argument of the defined environment is optional, with default value optargdefault (which may be the empty string). If this is not in the definition then the environment does not take an optional argument.
That is, when optargdefault is present in the definition of the environment then you can start the environment with square brackets, as in `\begin{env}[optval]{...} ... \end{env}`. In this case, within begdefn the parameter `#1` is set to the value of optval. If you call `\begin{env}` without square brackets, then within begdefn the parameter `#1` is set to the value of the default optargdefault. In either case, any required arguments start with `#2`.
Omitting `[myval]` in the call is different than having the square brackets with no contents, as in `[]`. The former results in `#1` expanding to optargdefault; the latter results in `#1` expanding to the empty string.

begdef

Required; the text expanded at every occurrence of `\begin{env}`. Within begdef, the parameters `#1`, `#2`, ... `#nargs`, are replaced by the values that you supply when you call the environment; see the examples below.

enddef

Required; the text expanded at every occurrence of `\end{env}`. This may not contain any parameters, that is, you cannot use `#1`, `#2`, etc., here (but see the final example below).
All environments, that is to say the begdef code, the environment body, and the enddef code, are processed within a group. Thus, in the first example below, the effect of the `\small` is limited to the quote and does not extend to material following the environment.
If you try to define an environment and the name has already been used then you get something like ‘LaTeX Error: Command \fred already defined. Or name \end... illegal, see p.192 of the manual’. If you try to redefine an environment and the name has not yet been used then you get something like ‘LaTeX Error: Environment hank undefined.’.
This example gives an environment like LaTeX’s `quotation` except that it will be set in smaller type.
```
\newenvironment{smallquote}{%
  \small\begin{quotation}
}{%
  \end{quotation}
}

```

This has an argument, which is set in boldface at the start of a paragraph.
```
\newenvironment{point}[1]{%
  \noindent\textbf{#1}
}{%
}

```

This one shows the use of a optional argument; it gives a quotation environment that cites the author.
```
\newenvironment{citequote}[1][Shakespeare]{%
  \begin{quotation}
  \noindent\textit{#1}:
}{%
  \end{quotation}
}

```

The author’s name is optional, and defaults to ‘Shakespeare’. In the document, use the environment like this.
```
\begin{citequote}[Lincoln]
  ...
\end{citequote}

```

The final example shows how to save the value of an argument to use in enddef, in this case in a box (see [`\sbox` & `\savebox`](https://latexref.xyz/dev/latex2e.html#g_t_005csbox-_0026-_005csavebox)).
```
\newsavebox{\quoteauthor}
\newenvironment{citequote}[1][Shakespeare]{%
  \sbox\quoteauthor{#1}%
  \begin{quotation}
}{%
  \hspace{1em plus 1fill}---\usebox{\quoteauthor}
  \end{quotation}
}

```

### 12.9 `\newtheorem`
Synopses:
```
\newtheorem{name}{title}
\newtheorem{name}{title}[numbered_within]
\newtheorem{name}[numbered_like]{title}

```

Define a new theorem-like environment. You can specify one of numbered_within and numbered_like, or neither, but not both.
The first form, `\newtheorem{name}{title}`, creates an environment that will be labelled with title; see the first example below.
The second form, `\newtheorem{name}{title}[numbered_within]`, creates an environment whose counter is subordinate to the existing counter numbered_within, so this counter will be reset when numbered_within is reset. See the second example below.
The third form `\newtheorem{name}[numbered_like]{title}`, with optional argument between the two required arguments, creates an environment whose counter will share the previously defined counter numbered_like. See the third example.
This command creates a counter named name. In addition, unless the optional argument numbered_like is used, inside of the theorem-like environment the current `\ref` value will be that of `\thenumbered_within` (see [`\ref`](https://latexref.xyz/dev/latex2e.html#g_t_005cref)).
This declaration is global. It is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
Arguments:

name

The name of the environment. It is a string of letters. It must not begin with a backslash, `\`. It must not be the name of an existing environment, and the command name `\name` must not already be defined.

title

The text to be printed at the beginning of the environment, before the number. For example, ‘Theorem’.

numbered_within

Optional; the name of an already defined counter, usually a sectional unit such as `chapter` or `section`. When the numbered_within counter is reset then the name environment’s counter will also be reset.
If this optional argument is not used then the command `\thename` is set to `\arabic{name}`.

numbered_like

Optional; the name of an already defined theorem-like environment. The new environment will be numbered in sequence with numbered_like.
Without any optional arguments the environments are numbered sequentially. The example below has a declaration in the preamble that results in ‘Definition 1’ and ‘Definition 2’ in the output.
```
\newtheorem{defn}{Definition}
\begin{document}
\section{...}
\begin{defn}
  First def
\end{defn}

\section{...}
\begin{defn}
  Second def
\end{defn}

```

This example has the same document body as the prior one. But here `\newtheorem`’s optional argument numbered_within is given as `section`, so the output is like ‘Definition 1.1’ and ‘Definition 2.1’.
```
\newtheorem{defn}{Definition}[section]
\begin{document}
\section{...}
\begin{defn}
  First def
\end{defn}

\section{...}
\begin{defn}
  Second def
\end{defn}

```

In the next example there are two declarations in the preamble, the second of which calls for the new `thm` environment to use the same counter as `defn`. It gives ‘Definition 1.1’, followed by ‘Theorem 2.1’ and ‘Definition 2.2’.
```
\newtheorem{defn}{Definition}[section]
\newtheorem{thm}[defn]{Theorem}
\begin{document}
\section{...}
\begin{defn}
  First def
\end{defn}

\section{...}
\begin{thm}
  First thm
\end{thm}

\begin{defn}
  Second def
\end{defn}

```

### 12.10 `\newfont`
This command is obsolete. This description is here only to help with old documents. New documents should define fonts in families through the New Font Selection Scheme which allows you to, for example, associate a boldface with a roman (see [Fonts](https://latexref.xyz/dev/latex2e.html#Fonts)).
Synopsis:
```
\newfont{\cmd}{font description}

```

Define a command `\cmd` that will change the current font. The control sequence must not already be defined. It must begin with a backslash, `\`.
The font description consists of a fontname and an optional _at clause_. LaTeX will look on your system for a file named fontname.tfm. The at clause can have the form either `at dimen` or `scaled factor`, where a factor of ‘1000’ means no scaling. For LaTeX’s purposes, all this does is scale all the character and other font dimensions relative to the font’s design size, which is a value defined in the .tfm file.
This defines two equivalent fonts and typesets a few characters in each.
```
\newfont{\testfontat}{cmb10 at 11pt}
\newfont{\testfontscaled}{cmb10 scaled 1100}
\testfontat abc
\testfontscaled abc

```
