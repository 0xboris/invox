# 12 Definitions: Macro basics and branching

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 12.1 `\newcommand` & `\renewcommand`
- 12.2 `\providecommand`
- 12.3 `\makeatletter` & `\makeatother`
- 12.4 `\@ifstar`

## 12 Definitions
LaTeX has support for making new commands of many different kinds.
  * [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand)
  * [`\providecommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cprovidecommand)
  * [`\makeatletter` & `\makeatother`](https://latexref.xyz/dev/latex2e.html#g_t_005cmakeatletter-_0026-_005cmakeatother)
  * [`\@ifstar`](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040ifstar)
  * [`\newcounter`: Allocating a counter](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcounter)
  * [`\newlength`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewlength)
  * [`\newsavebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewsavebox)
  * [`\newenvironment` & `\renewenvironment`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewenvironment-_0026-_005crenewenvironment)
  * [`\newtheorem`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewtheorem)
  * [`\newfont`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewfont)
  * [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)
  * [`\ignorespaces & \ignorespacesafterend`](https://latexref.xyz/dev/latex2e.html#g_t_005cignorespaces-_0026-_005cignorespacesafterend)
  * [`xspace` package](https://latexref.xyz/dev/latex2e.html#xspace-package)
  * [Class and package commands](https://latexref.xyz/dev/latex2e.html#Class-and-package-commands)

### 12.1 `\newcommand` & `\renewcommand`
Synopses, one of (three regular forms, three starred forms):
```
\newcommand{\cmd}{defn}
\newcommand{\cmd}[nargs]{defn}
\newcommand{\cmd}[nargs][optargdefault]{defn}
\newcommand*{\cmd}{defn}
\newcommand*{\cmd}[nargs]{defn}
\newcommand*{\cmd}[nargs][optargdefault]{defn}

```

or the same six possibilities with `\renewcommand` instead of `\newcommand`:
```
\renewcommand{\cmd}{defn}
\renewcommand{\cmd}[nargs]{defn}
\renewcommand{\cmd}[nargs][optargdefault]{defn}
\renewcommand*{\cmd}{defn}
\renewcommand*{\cmd}[nargs]{defn}
\renewcommand*{\cmd}[nargs][optargdefault]{defn}

```

Define or redefine a command (see also `\DeclareRobustCommand` in [Class and package commands](https://latexref.xyz/dev/latex2e.html#Class-and-package-commands)).
The starred form of these two forbids the arguments from containing multiple paragraphs of text (i.e., a `\par` token; in plain TeX terms: the commands are not `\long`). With the default form, arguments can be multiple paragraphs.
These are the parameters (examples follow):

cmd

Required; `\cmd` is the command name. It must begin with a backslash, `\`, and must not begin with the four character string `\end`. For `\newcommand`, it must not be already defined. For `\renewcommand`, this name must already be defined.

nargs

Optional; an integer from 0 to 9, specifying the number of arguments that the command takes, including any optional argument. Omitting this argument is the same as specifying 0, meaning that the command has no arguments. If you redefine a command, the new version can have a different number of arguments than the old version.

optargdefault

Optional; if this argument is present then the first argument of `\cmd` is optional, with default value optargdefault (which may be the empty string). If optargdefault is not present then `\cmd` does not take an optional argument.
That is, if `\cmd` is called with a following argument in square brackets, as in `\cmd[optval]{...}...`, then within defn the parameter `#1` is set to optval. On the other hand, if `\cmd` is called without following square brackets then within defn the parameter `#1` is set to optargdefault. In either case, the required arguments start with `#2`.
Omitting `[optargdefault]` from the definition is entirely different from giving the square brackets with empty contents, as in `[]`. The former says the command being defined takes no optional argument, so `#1` is the first required argument (if _nargs ≥ 1_); the latter sets the optional argument `#1` to the empty string as the default, if no optional argument was given in the call.
Similarly, omitting `[optval]` from a call is also entirely different from giving the square brackets with empty contents. The former sets `#1` to the value of optval (assuming the command was defined to take an optional argument); the latter sets `#1` to the empty string, just as with any other value.
If a command is not defined to take an optional argument, but is called with an optional argument, the results are unpredictable: there may be a LaTeX error, there may be incorrect typeset output, or both.

defn

Required; the text to be substituted for every occurrence of `\cmd`. The parameters `#1`, `#2`, …, `#nargs` are replaced by the values supplied when the command is called (or by optargdefault in the case of an optional argument not specified in the call, as just explained).
TeX ignores blanks in the source following a control word (see [Control sequence, control word and control symbol](https://latexref.xyz/dev/latex2e.html#Control-sequences)), as in ‘\cmd ’. If you want a space there, one solution is to type `{}` after the command (‘\cmd{} ’), and another solution is to use an explicit control space (‘\cmd\ ’).
A simple example of defining a new command: `\newcommand{\RS}{Robin Smith}` results in `\RS` being replaced by the longer text. Redefining an existing command is similar: `\renewcommand{\qedsymbol}{{\small QED}}`.
If you use `\newcommand` and the command name has already been used then you get something like ‘LaTeX Error: Command \fred already defined. Or name \end... illegal, see p.192 of the manual’. Similarly, If you use `\renewcommand` and the command name has not been defined then you get something like ‘LaTeX Error: \hank undefined’.
Here the first definition creates a command with no arguments, and the second, a command with one required argument:
```
\newcommand{\student}{Ms~O'Leary}
\newcommand{\defref}[1]{Definition~\ref{#1}}

```

Use the first as in `I highly recommend \student{} to you`. The second has a variable argument, so that `\defref{def:basis}` expands to `Definition~\ref{def:basis}`, which ultimately expands to something like ‘Definition~3.14’.
Similarly, but with two required arguments: `\newcommand{\nbym}[2]{$#1 \times #2$}` is invoked as `\nbym{2}{k}`.
This example has an optional argument.
```
\newcommand{\salutation}[1][Sir or Madam]{Dear #1:}

```

Then `\salutation` gives ‘Dear Sir or Madam:’ while `\salutation[John]` gives ‘Dear John:’. And `\salutation[]` gives ‘Dear :’.
This example has an optional argument and two required arguments.
```
\newcommand{\lawyers}[3][company]{#2, #3, and~#1}
I employ \lawyers[Howe]{Dewey}{Cheatem}.

```

The output is ‘I employ Dewey, Cheatem, and Howe.’. The optional argument, `Howe`, is associated with `#1`, while `Dewey` and `Cheatem` are associated with `#2` and `#3`. Because of the optional argument, `\lawyers{Dewey}{Cheatem}` will give the output ‘I employ Dewey, Cheatem, and company.’.
The braces around defn do not define a group, that is, they do not delimit the scope of the result of expanding defn. For example, with `\newcommand{\shipname}[1]{\it #1}`, in this sentence,
```
The \shipname{Monitor} met the \shipname{Merrimac}.

```

the words ‘met the’, and the period, would incorrectly be in italics. The solution is to put another pair of braces inside the definition: `\newcommand{\shipname}[1]{{\it #1}}`.
  * [Control sequence, control word and control symbol](https://latexref.xyz/dev/latex2e.html#Control-sequences)

#### 12.1.1 Control sequence, control word and control symbol
When reading input TeX converts the stream of read characters into a sequence of _tokens_. When TeX sees a backslash `\`, it will handle the following characters in a special way in order to make a _control sequence_ token.
The control sequences fall into two categories:
  * _control word_ , when the control sequence is gathered from a `\` followed by at least one ASCII letter (`A-Z` and `a-z`), followed by at least one non-letter.
  * _control symbol_ , when the control sequence is gathered from a `\` followed by one non-letter character.

The sequence of characters so found after the `\` is also called the _control sequence name_.
Blanks after a control word are ignored and do not produce any whitespace in the output (see [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand) and [Backslash-space, `\ `](https://latexref.xyz/dev/latex2e.html#g_t_005c_0028SPACE_0029)).
Just as the `\relax` command does nothing, the following input will simply print ‘Hello!’ :
```
Hel\relax
   lo!

```

This is because blanks after `\relax`, including the newline, are ignored, and blanks at the beginning of a line are also ignored (see [Leading blanks](https://latexref.xyz/dev/latex2e.html#Leading-blanks)).
### 12.2 `\providecommand`
Synopses, one of:
```
\providecommand{\cmd}{defn}
\providecommand{\cmd}[nargs]{defn}
\providecommand{\cmd}[nargs][optargdefault]{defn}
\providecommand*{\cmd}{defn}
\providecommand*{\cmd}[nargs]{defn}
\providecommand*{\cmd}[nargs][optargdefault]{defn}

```

Defines a command, as long as no command of this name already exists. If no command of this name already exists then this has the same effect as `\newcommand`. If a command of this name already exists then this definition does nothing. This is particularly useful in a file that may be loaded more than once, such as a style file. See [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand), for the description of the arguments.
This example
```
\providecommand{\myaffiliation}{Saint Michael's College}
\providecommand{\myaffiliation}{Lyc\'ee Henri IV}
From \myaffiliation.

```

outputs ‘From Saint Michael's College.’. Unlike `\newcommand`, the repeated use of `\providecommand` to (try to) define `\myaffiliation` does not give an error.
### 12.3 `\makeatletter` & `\makeatother`
Synopsis:
```
\makeatletter
  ... definition of commands with @ in their name ..
\makeatother

```

Use this pair when you redefine LaTeX commands that are named with an at-sign character ‘`@`’. The `\makeatletter` declaration makes the at-sign character have the category code of a letter, code 11. The `\makeatother` declaration sets the category code of the at-sign to code 12, its default value.
As TeX reads characters, it assigns each one a category code, or  _catcode_. For instance, it assigns the backslash character ‘`\`’ the catcode 0. Command names consist of a category 0 character, ordinarily backslash, followed by letters, category 11 characters (except that a command name can also consist of a category 0 character followed by a single non-letter symbol).
LaTeX’s source code has the convention that some commands use `@` in their name. These commands are mainly intended for package or class writers. The convention prevents authors who are just using a package or class from accidentally replacing such a command with one of their own, because by default the at-sign has catcode 12.
Use the pair `\makeatletter` and `\makeatother` inside a .tex file, typically in the preamble, when you are defining or redefining commands named with `@`, by having them surround your definition. Don’t use these inside .sty or .cls files since the `\usepackage` and `\documentclass` commands already arrange that the at-sign has the character code of a letter, catcode 11.
For a comprehensive list of macros with an at-sign in their names see <https://ctan.org/pkg/macros2e>.
In this example the class file has a command `\thesis@universityname` that the user wants to change. These three lines should go in the preamble, before the `\begin{document}`.
```
\makeatletter
\renewcommand{\thesis@universityname}{Saint Michael's College}
\makeatother

```

### 12.4 `\@ifstar`
Synopsis:
```
\newcommand{\mycmd}{\@ifstar{\mycmd@star}{\mycmd@nostar}}
\newcommand{\mycmd@nostar}[nostar-num-args]{nostar-body}
\newcommand{\mycmd@star}[star-num-args]{star-body}

```

Many standard LaTeX environments or commands have a variant with the same name but ending with a star character `*`, an asterisk. Examples are the `table` and `table*` environments and the `\section` and `\section*` commands.
When defining environments, following this pattern is straightforward because `\newenvironment` and `\renewenvironment` allow the environment name to contain a star. So you just have to write `\newenvironment{myenv}` or `\newenvironment{myenv*}` and continue the definition as usual. For commands the situation is more complex as the star not being a letter cannot be part of the command name. As in the synopsis above, there will be a user-called command, given above as `\mycmd`, which peeks ahead to see if it is followed by a star. For instance, LaTeX does not really have a `\section*` command; instead, the `\section` command peeks ahead. This command does not accept arguments but instead expands to one of two commands that do accept arguments. In the synopsis these two are `\mycmd@nostar` and `\mycmd@star`. They could take the same number of arguments or a different number, or no arguments at all. As always, in a LaTeX document a command using an at-sign `@` in its name must be enclosed inside a `\makeatletter ... \makeatother` block (see [`\makeatletter` & `\makeatother`](https://latexref.xyz/dev/latex2e.html#g_t_005cmakeatletter-_0026-_005cmakeatother)).
This example of `\@ifstar` defines the command `\ciel` and a variant `\ciel*`. Both have one required argument. A call to `\ciel{blue}` will return "not starry blue sky" while `\ciel*{night}` will return "starry night sky".
```
\makeatletter
\newcommand*{\ciel@unstarred}[1]{not starry #1 sky}
\newcommand*{\ciel@starred}[1]{starry #1 sky}
\newcommand*{\ciel}{\@ifstar{\ciel@starred}{\ciel@unstarred}}
\makeatother

```

In the next example, the starred variant takes a different number of arguments than the unstarred one. With this definition, Agent 007’s ```My name is \agentsecret*{Bond}, \agentsecret{James}{Bond}.''` is equivalent to entering the commands ```My name is \textsc{Bond}, \textit{James} textsc{Bond}.''`
```
\newcommand*{\agentsecret@unstarred}[2]{\textit{#1} \textsc{#2}}
\newcommand*{\agentsecret@starred}[1]{\textsc{#1}}
\newcommand*{\agentsecret}{%
  \@ifstar{\agentsecret@starred}{\agentsecret@unstarred}}

```

After a command name, a star is handled similarly to an optional argument. (This differs from environment names in which the star is part of the name itself and as such could be in any position.) Thus, it is technically possible to put any number of spaces between the command and the star. Thus `\agentsecret*{Bond}` and `\agentsecret *{Bond}` are equivalent. However, the standard practice is not to insert any such spaces.
There are two alternative ways to accomplish the work of `\@ifstar`. (1) The `suffix` package allows the construct `\newcommand\mycommand{unstarred-variant}` followed by `\WithSuffix\newcommand\mycommand*{starred-variant}`. (2) LaTeX provides the `xparse` package, which allows this code:
```
\NewDocumentCommand\foo{s}{\IfBooleanTF#1
  {starred-variant}%
  {unstarred-variant}%
  }

```
