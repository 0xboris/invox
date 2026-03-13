# 12 Definitions: Protection, spacing, and xspace

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 12.11 `\protect`
- 12.12 `\ignorespaces & \ignorespacesafterend`
- 12.13 `xspace` package

### 12.11 `\protect`
All LaTeX commands are either _fragile_ or _robust_. A fragile command can break when it is used in the argument to certain other commands, typically those that write material to the table of contents, the cross-reference file, etc. To prevent fragile commands from causing errors, one solution is to precede them with the command `\protect`.
For example, when LaTeX runs the `\section{section name}` command it writes the section name text to the .aux auxiliary file, moving it there for use elsewhere in the document such as in the table of contents. Such an argument that is used in multiple places is referred to as a  _moving argument_. A command is fragile if it can expand during this process into invalid TeX code. Some examples of moving arguments are those that appear in the `\caption{...}` command (see [`figure`](https://latexref.xyz/dev/latex2e.html#figure)), in the `\thanks{...}` command (see [`\maketitle`](https://latexref.xyz/dev/latex2e.html#g_t_005cmaketitle)), and in @-expressions in the `tabular` and `array` environments (see [`tabular`](https://latexref.xyz/dev/latex2e.html#tabular)).
If you get strange errors from commands used in moving arguments, try preceding it with `\protect`. Each fragile command must be protected with their own `\protect`.
Although usually a `\protect` command doesn’t hurt, length commands such as `\parindent` should not be preceded by a `\protect` command (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths). Nor can a `\protect` command be used in the argument to `\addtocounter` or `\setcounter` command (see [`\setcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005csetcounter) and [`\addtocounter`](https://latexref.xyz/dev/latex2e.html#g_t_005caddtocounter). These commands are already robust.
As of the October 2019 release of LaTeX (<https://www.latex-project.org/news/latex2e-news/ltnews30.pdf>), most commands that had been previously fragile were fixed to be robust. For example, any command taking an optional argument, such as `\root` or `\raisebox`, was fragile, but is now robust. Similarly, `\(...\)` math was fragile and is now robust (`$...$` has always been robust).
Perhaps the most commonly used remaining fragile command is `\verb`; for example,
```
\begin{figure}
  ...
  \caption{This \verb|\command| causes an error.}
\end{figure}

```

Adding `\protect` does not help here. It’s usually feasible to rewrite the caption (or section heading or whatever) to use `\texttt`, often the simplest solution.
Alternatively, to use `\verb`, you can apply the `\cprotect` command from `cprotect` package (<https://ctan.org/pkg/cprotect>) to the `\caption`:
```
\cprotect\caption{This \verb|\command| is ok with \verb|\cprotect|.}

```

`\cprotect` also allows use of `\begin...\end` environments in moving arguments, where they are normally not allowed, via a similar prefix command `\cprotEnv`.
### 12.12 `\ignorespaces & \ignorespacesafterend`
Synopsis:
```
\ignorespaces

```

or
```
\ignorespacesafterend

```

Both commands cause LaTeX to ignore blanks (that is, characters of catcode 10 such as space or tabulation) after the end of the command up to the first box or non-blank character. The first is a primitive command of TeX, and the second is LaTeX-specific.
The `\ignorespaces` is often used when defining commands via `\newcommand`, or `\newenvironment`, or `\def`. The example below illustrates. It allows a user to show the points values for quiz questions in the margin but it is inconvenient because, as shown in the `enumerate` list, users must not put any space between the command and the question text.
```
\newcommand{\points}[1]{\makebox[0pt]{\makebox[10em][l]{#1~pts}}
\begin{enumerate}
  \item\points{10}no extra space output here
  \item\points{15} extra space between the number and the `extra'
\end{enumerate}

```

The solution is to change to this.
```
\newcommand{\points}[1]{%
  \makebox[0pt]{\makebox[10em][l]{#1~pts}}\ignorespaces}

```

A second example shows blanks being removed from the front of text. The commands below allow a user to uniformly attach a title to names. But, as given, if a title accidentally starts with a space then `\fullname` will reproduce that.
```
\newcommand{\honorific}[1]{\def\honorific{#1}} % remember title
\newcommand{\fullname}[1]{\honorific~#1}       % put title before name

\begin{tabular}{|l|}
\honorific{Mr/Ms}  \fullname{Jones} \\  % no extra space here
\honorific{ Mr/Ms} \fullname{Jones}     % extra space before title
\end{tabular}

```

To fix this, change to `\newcommand{\fullname}[1]{\ignorespaces\honorific~#1}`.
The `\ignorespaces` is also often used in a `\newenvironment` at the end of the begin clause, as in `\begin{newenvironment}{env name}{... \ignorespaces}{...}`.
To strip blanks off the end of an environment use `\ignorespacesafterend`. An example is that this will show a much larger vertical space between the first and second environments than between the second and third.
```
\newenvironment{eq}{\begin{equation}}{\end{equation}}
\begin{eq}
e=mc^2
\end{eq}
\begin{equation}
F=ma
\end{equation}
\begin{equation}
E=IR
\end{equation}

```

Putting a comment character `%` immediately after the `\end{eq}` will make the vertical space disappear, but that is inconvenient. The solution is to change to `\newenvironment{eq}{\begin{equation}}{\end{equation}\ignorespacesafterend}`.
### 12.13 `xspace` package
This is an add-on package, not part of core LaTeX. Synopsis:
```
\usepackage{xspace}
  ...
\newcommand{...}{...\xspace}

```

The `\xspace` macro, when used at the end of a command definition, adds a space unless the command is followed by certain punctuation characters.
After a control sequence that is a control word (see [Control sequence, control word and control symbol](https://latexref.xyz/dev/latex2e.html#Control-sequences), as opposed to control symbols such as `\$`), TeX gobbles blank characters. Thus, in the first sentence below, the output has ‘Vermont’ placed snugly against the period, without any intervening space, despite the space in the input.
```
\newcommand{\VT}{Vermont}
Our college is in \VT .
\VT{} summers are nice.

```

But because of the gobbling, the second sentence needs the empty curly braces or else there would be no space separating ‘Vermont’ from ‘summers’. (Many authors instead use a backslash-space `\ ` for this. See [Backslash-space, `\ `](https://latexref.xyz/dev/latex2e.html#g_t_005c_0028SPACE_0029).)
The `xspace` package provides `\xspace`. It is for writing commands which are designed to be used mainly in text. It must be placed at the very end of the definition of these commands. It inserts a space after that command unless what immediately follows is in a list of exceptions. In this example, the empty braces are not needed.
```
\newcommand{\VT}{Vermont\xspace}
Our college is in \VT .
\VT summers are nice.

```

The default exception list contains the characters `,.'/?;:!~-)`, the open curly brace and the backslash-space command discussed above, and the commands `\footnote` or `\footnotemark`. You can add to that list as with `\xspaceaddexceptions{\myfni \myfnii}` which adds `\myfni` and `\myfnii` to the list; and you can remove from that list as with `\xspaceremoveexception{!}`.
A comment: many experts prefer not to use `\xspace`. Putting it in a definition means that the command will usually get the spacing right. But it isn’t easy to predict when to enter empty braces because `\xspace` will get it wrong, such as when it is followed by another command, and so `\xspace` can make editing material harder and more error-prone than instead of always inserting the empty braces.
