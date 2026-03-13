# 28 Command line interface

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 28.1 Command line options
- 28.2 Command line input
- 28.3 Jobname
- 28.4 Recovering from errors

## 28 Command line interface
Synopsis (from a terminal command line):
```
pdflatex options argument

```

Run LaTeX on argument. In place of `pdflatex` you can also use (for PDF output) `xelatex` or `lualatex`, or (for DVI output) `latex` or `dvilualatex`, among others (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)).
For example, this will run LaTeX on the file thesis.tex, creating the output thesis.pdf.
```
pdflatex thesis

```

Note that .tex is the default file name extension.
pdfTeX is an extension of the original TeX program, as are XeTeX and LuaTeX (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)). The first two are completely backward compatible and the latter, almost so. Perhaps the most fundamental new feature for all three is that the original TeX output its own DVI format, while the newer ones can output directly to PDF. This allows them to take advantage of the extra features in PDF such as hyperlinks, support for modern image formats such as JPG and PNG, and ubiquitous viewing programs. In short, if you run `pdflatex` or `xelatex` or `lualatex` then you will by default get PDF and have access to all its modern features. If you run `latex`, or `dvilualatex`, then you will get DVI. The description here assumes `pdflatex`.
See [Command line options](https://latexref.xyz/dev/latex2e.html#Command-line-options), for a selection of the most useful command line options. As to argument, the usual case is that it does not begin with a backslash, so the system takes it to be the name of a file and it compiles that file. If argument begins with a backslash then the system will interpret it as a line of LaTeX input, which can be used for special effects (see [Command line input](https://latexref.xyz/dev/latex2e.html#Command-line-input)).
If you gave no arguments or options then `pdflatex` prompts for input from the terminal. You can escape from this by entering `CTRL-D`.
If LaTeX finds an error in your document then by default it stops and asks you about it. See [Recovering from errors](https://latexref.xyz/dev/latex2e.html#Recovering-from-errors), for an outline of what to do.
  * [Command line options](https://latexref.xyz/dev/latex2e.html#Command-line-options)
  * [Command line input](https://latexref.xyz/dev/latex2e.html#Command-line-input)
  * [Jobname](https://latexref.xyz/dev/latex2e.html#Jobname)
  * [Recovering from errors](https://latexref.xyz/dev/latex2e.html#Recovering-from-errors)

### 28.1 Command line options
These are the command-line options relevant to ordinary document authoring. For a full list, try running ‘latex --help’ from the command line.
With many implementations you can specify command line options by prefixing them with ‘-’ or ‘--’. This is the case for both TeX Live (including MacTeX) and MiKTeX. We will use both conventions interchangeably. If an option takes a value, it can be specified either as a separate argument (‘--foo val’), or as one argument with an ‘=’ sign (‘--foo=val’), but there can be no spaces around the ‘=’. We will generally use the ‘=’ syntax.

`-version`

Show the current version, like ‘pdfTeX 3.14159265-2.6-1.40.16 (TeX Live 2015/Debian)’ along with a small amount of additional information, and exit.

`-help`

Give a brief usage message that is useful as a prompt and exit.

`-interaction=mode`

TeX compiles a document in one of four interaction modes: `batchmode`, `nonstopmode`, `scrollmode`, `errorstopmode`. In _errorstopmode_ (the default), TeX stops at each error and asks for user intervention. In _batchmode_ it prints nothing on the terminal, errors are scrolled as if the user hit `RETURN` at every error, and missing files cause the job to abort. In _nonstopmode_ , diagnostic message appear on the terminal but as in batch mode there is no user interaction. In _scrollmode_ , TeX stops for missing files or keyboard input, but nothing else.
For instance, starting LaTeX with this command line
```
pdflatex -interaction=batchmode filename

```

eliminates most terminal output.

`-jobname=string`

Set the value of TeX’s _jobname_ to the string. The log file and output file will then be named string.log and string.pdf. see [Jobname](https://latexref.xyz/dev/latex2e.html#Jobname).

`-output-directory=directory`

Write files in the directory directory. It must already exist. This applies to all external files created by TeX or LaTeX, such as the .log file for the run, the .aux, .toc, etc., files created by LaTeX, as well as the main .pdf or .dvi output file itself.
When specified, the output directory directory is also automatically checked first for any file that it is input, so that the external files can be read back in, if desired. The true current directory (in which LaTeX was run) remains unchanged, and is also checked for input files.

`--enable-write18`

`--disable-write18`

`--shell-escape`

`--no-shell-escape`

Enable or disable `\write18{shell_command}` (see [`\write18`](https://latexref.xyz/dev/latex2e.html#g_t_005cwrite18)). The first two options are supported by both TeX Live and MiKTeX, while the second two are synonyms supported by TeX Live.
Enabling this functionality has major security implications, since it allows a LaTeX file to run any command whatsoever. Thus, by default, unrestricted `\write18` is not allowed. (The default for TeX Live, MacTeX, and MiKTeX is to allow the execution of a limited number of TeX-related programs, which they distribute.)
For example, if you invoke LaTeX with the option `no-shell-escape`, and in your document you call `\write18{ls -l}`, then you do not get an error but the log file says ‘runsystem(ls -l)...disabled’.

`-halt-on-error`

Stop processing at the first error.

`-file-line-error`

`-no-file-line-error`

Enable or disable `filename:lineno:error`-style error messages. These are only available with TeX Live or MacTeX.
### 28.2 Command line input
As part of the command line invocation
```
latex-engine options argument

```

you can specify arbitrary LaTeX input by starting argument with a backslash. (All the engines support this.) This allows you to do some special effects.
For example, this file (which uses the `hyperref` package for hyperlinks) can produce two kinds of output, one to be read on physical paper and one to be read online.
```
\ifdefined\paperversion        % in preamble
\newcommand{\urlcolor}{black}
\else
\newcommand{\urlcolor}{blue}
\fi
\usepackage[colorlinks=true,urlcolor=\urlcolor]{hyperref}
  ...
\href{https://www.ctan.org}{CTAN}  % in body
  ...

```

Compiling this document book.tex with the command line `pdflatex book` will give the ‘CTAN’ link in blue. But compiling it with
```
pdflatex "\def\paperversion{}\input book.tex"

```

has the link in black. We use double quotes to prevent interpretation of the symbols by the command line shell. (This usually works on both Unix and Windows systems, but there are many peculiarities to shell quoting, so read your system documentation if need be.)
In a similar way, from the single file main.tex you can compile two different versions.
```
pdflatex -jobname=students "\def\student{}\input{main}"
pdflatex -jobname=teachers "\def\teachers{}\input{main}"

```

The `jobname` option is there because otherwise both files would be called main.pdf and the second would overwrite the first (see [Jobname](https://latexref.xyz/dev/latex2e.html#Jobname)).
In this example we use the command line to select which parts of a document to include. For a book named mybook.tex and structured like this.
```
\documentclass{book}
\begin{document}
   ...
\include{chap1}
\include{chap2}
  ...
\end{document}

```

the command line
```
pdflatex "\includeonly{chap1}\input{mybook}"

```

will give output that has the first chapter but no other chapter. See [Splitting the input](https://latexref.xyz/dev/latex2e.html#Splitting-the-input).
### 28.3 Jobname
Running LaTeX creates a number of files, including the main PDF (or DVI) output but also including others. These files are named with the so-called _jobname_. The most common case is also the simplest, where for instance the command `pdflatex thesis` creates `thesis.pdf` and also `thesis.log` and `thesis.aux`. Here the job name is `thesis`.
In general, LaTeX is invoked as `latex-engine options argument`, where latex-engine is `pdflatex`, `lualatex`, etc. (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)). If argument does not start with a backslash, as is the case above with `thesis`, then TeX considers it to be the name of the file to input as the main document. This file is referred to as the _root file_ (see [Splitting the input](https://latexref.xyz/dev/latex2e.html#Splitting-the-input), and [`\input`](https://latexref.xyz/dev/latex2e.html#g_t_005cinput)). The name of that root file, without the .tex extension if any, is the jobname. If argument does start with a backslash, or if TeX is in interactive mode, then it waits for the first `\input` command, and the jobname is the argument to `\input`.
There are two more possibilities for the jobname. It can be directly specified with the `-jobname` option, as in `pdflatex -jobname=myname` (see [Command line input](https://latexref.xyz/dev/latex2e.html#Command-line-input) for a practical example).
The final possibility is texput, which is the final fallback default if no other name is available to TeX. That is, if no `-jobname` option was specified, and the compilation stops before any input file is met, then the log file will be named texput.log.
A special case of this is that in LaTeX versions of (approximately) 2020 or later, the jobname is also texput if the first `\input` occurs as a result of being called by either `\documentclass` or `\RequirePackage`. So this will produce a file named texput.pdf:
```
pdflatex "\documentclass{minimal}\begin{document}Hello!\end{document}"

```

However, this special case only applies to those two commands. Thus, with
```
pdflatex "\documentclass{article}\usepackage{lipsum}\input{thesis}"

```

the output file is lipsum.pdf, as `\usepackage` calls `\input`.
Within the document, the macro `\jobname` expands to the jobname. (When you run LaTeX on a file whose name contains spaces, the string returned by `\jobname` contains matching start and end quotes.) In the expansion of that macro, all characters are of catcode 12 (other) except that spaces are category 10, including letters that are normally catcode 11.
Because of this catcode situation, using the jobname in a conditional can become complicated. One solution is to use the macro `\IfBeginWith` from the xstring package in its star variant, which is insensitive to catcode. For example, in the following text the footnote “Including Respublica Bananensis Francorum.” is only present if the task name starts with my-doc.
```
If a democracy is just a regime where citizens vote then
all banana republics \IfBeginWith*{\jobname}{my-doc}%
{\footnote{Including Respublica Bananensis Francorum.}}{} are
democracies.

```

Manipulating the value of `\jobname` inside of a document does not change the name of the output file or the log file.
### 28.4 Recovering from errors
If LaTeX finds an error in your document then it gives you an error message and prompts you with a question mark, `?`. For instance, running LaTeX on this file
```
\newcommand{\NP}{\ensuremath{\textbf{NP}}}
The \PN{} problem is a million dollar one.

```

causes it show this, and wait for input.
```
! Undefined control sequence.
l.5 The \PN
           {} problem is a million dollar one.
?

```

The simplest thing is to enter `x` and `RETURN` and fix the typo. You could instead enter `?` and `RETURN` to see other options.
There are two other error scenarios. The first is that you forgot to include the `\end{document}` or misspelled it. In this case LaTeX gives you a ‘*’ prompt. You can get back to the command line by typing `\stop` and `RETURN`; this command does its best to exit LaTeX immediately, whatever state it may be in.
The last scenario is that you mistyped the filename. For instance, instead of `pdflatex test` you might type `pdflatex tste`.
```
! I can't find file `tste'.
<*> tste

(Press Enter to retry, or Control-D to exit)
Please type another input file name:

```

The simplest thing is to enter `CTRL d` (holding the Control and d keys down at the same time), and then retype the correct command line.
