# 27 Input/output

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 27.1 `\openin` & `\openout`
- 27.2 `\read`
- 27.3 `\typein`
- 27.4 `\typeout`
- 27.5 `\write`

## 27 Input/output
LaTeX uses the ability to write to a file and later read it back in to build document components such as a table of contents or index. You can also read a file that other programs written, or write a file for others to read. You can communicate with users through the terminal. And, you can issue instructions for the operating system.
  * [`\openin` & `\openout`](https://latexref.xyz/dev/latex2e.html#g_t_005copenin-_0026-_005copenout)
  * [`\read`](https://latexref.xyz/dev/latex2e.html#g_t_005cread)
  * [`\typein`](https://latexref.xyz/dev/latex2e.html#g_t_005ctypein)
  * [`\typeout`](https://latexref.xyz/dev/latex2e.html#g_t_005ctypeout)
  * [`\write`](https://latexref.xyz/dev/latex2e.html#g_t_005cwrite)

### 27.1 `\openin` & `\openout`
Synopsis:
```
\openin number=filename

```

or:
```
\openout number=filename

```

Open a file for reading material, or for writing it. In most engines, the number must be between 0 and 15, as in `\openin3`; in LuaLaTeX, number can be between 0 and 127.
Here TeX opens the file presidents.tex for reading.
```
\newread\presidentsfile
\openin\presidentsfile=presidents
\typeout{presidentsfile is \the\presidentsfile}
\read\presidentsfile to\presidentline
\typeout{\presidentline}

```

The `\newread` command allocates input stream numbers from 0 to 15 (there is also a `\newwrite`). The `\presidentsfile` is more memorable but under the hood it is still a number; the first `\typeout` gives something like ‘presidentsfile is 1’. In addition, `\newread` keeps track of the allocation so that if you use too many then you get an error like ‘! No room for a new \read’. The second `\typeout` gives the first line of the file, something like ‘1 Washington, George’.
Ordinarily TeX will not try to open the file until the next page shipout. To change this, use `\immediate\openin number=filename` or `\immediate\openout number=filename`.
Close files with `\closein number` and `\closeout number`.
How LaTeX handles filenames varies among distributions, and even can vary among versions of a distribution. If the file does not have an extension then TeX will add a .tex. This creates presidents.tex, writes one line to it, and closes it.
```
\newwrite\presidentsfile
\openout\presidentsfile=presidents
\write\presidentsfile{1 Washington, George}
\closeout\presidentsfile

```

But filenames with a period can cause trouble: if TeX finds a filename of presidents.dat it could look first for presidents.dat.tex and later for presidents.dat, or it could do the opposite. Your distribution’s documentation should say more, and if you find something that works for you then you are good, but to ensure complete portability the best thing is to use file names containing only the twenty six ASCII letters (not case-sensitive) and the ten digits, along with underscore and dash, and in particular with no dot or space.
For `\openin`, if TeX cannot find the file then it does not give an error. It just considers that the stream is not open (test for this with `\ifeof`; one recourse is the command `\InputIfFileExists`, see [Class and package commands](https://latexref.xyz/dev/latex2e.html#Class-and-package-commands)). If you try to use the same number twice, LaTeX won’t give you an error. If you try to use a bad number then you get an error message like ‘! Bad number (16). <to be read again> = l.30 \openin16=test.jh’.
### 27.2 `\read`
Synopsis:
```
\read number tomacro

```

Make the command macro contain the next line of input from text stream number, as in `\read5 to\data`.
This opens the file email.tex for reading, puts the contents of the first line into the command `\email`, and then closes the file.
```
\newread\recipientfile
\openin\recipientfile=email
\read\recipientfile to\email
\typeout{Email address: \email}
\closein\recipientfile

```

If number is outside the range from 0 to 15 or if no file of that number is open, or if the file has ended, then `\read` will take input from the terminal (or exit if interaction is not allowed, e.g., `\nonstopmode`; see [interaction modes](https://latexref.xyz/dev/latex2e.html#interaction-modes)). (However, the natural way in LaTeX to take input from the terminal is `\typein` (see [`\typein`](https://latexref.xyz/dev/latex2e.html#g_t_005ctypein).)
To read an entire file as additional LaTeX source, use `\input` (see [`\input`](https://latexref.xyz/dev/latex2e.html#g_t_005cinput)) or `\include` (see [`\include` & `\includeonly`](https://latexref.xyz/dev/latex2e.html#g_t_005cinclude-_0026-_005cincludeonly)).
A common reason to want to read from a data file is to do mail merges. CTAN has a number of packages for that; one is `datatool`.
### 27.3 `\typein`
Synopsis, one of:
```
\typein{prompt-msg}
\typein[cmd]{prompt-msg}

```

Print prompt-msg on the terminal and cause LaTeX to stop and wait for you to type a line of input. This line of input ends when you hit the return key.
For example, this
```
As long as I live I shall never forget \typein{Enter student name:}

```

coupled with this command line interaction
```
Enter student name:

\@typein=Aphra Behn

```

gives the output ‘... never forget Aphra Behn’.
The first command version, `\typein{prompt-msg}`, causes the input you typed to be processed as if it had been included in the input file in place of the `\typein` command.
In the second command version the optional argument `cmd` argument must be a command name, that is, it must begin with a backslash, \\. This command name is then defined or redefined to be the input that you typed. For example, this
```
\typein[\student]{Enter student name:}
\typeout{Recommendation for \student .}

```

gives this output on the command line,
```
Enter student name:

\student=John Dee
Recommendation for John Dee.

```

where the user has entered ‘John Dee.’
### 27.4 `\typeout`
Synopsis:
```
\typeout{msg}

```

Print `msg` on the terminal and in the `log` file.
This
```
\newcommand{\student}{John Dee}
\typeout{Recommendation for \student .}

```

outputs ‘Recommendation for John Dee’. Like what happens here with `\student`, commands that are defined with `\newcommand` or `\renewcommand` (among others) are replaced by their definitions before being printed.
LaTeX’s usual rules for treating multiple spaces as a single space and ignoring spaces after a command name apply to `msg`. Use the command `\space` to get a single space, independent of surrounding spaces. Use `^^J` to get a newline. Get a percent character with `\csname @percentchar\endcsname`.
This command can be useful for simple debugging, as here:
```
\newlength{\jhlength}
\setlength{\jhlength}{5pt}
\typeout{The length is \the\jhlength.}

```

produces on the command line ‘The length is 5.0pt’.
### 27.5 `\write`
Synopsis:
```
\write number{string}

```

Write string to the log file, to the terminal, or to a file opened by `\openout`. For instance, `\write6` writes to text stream number 6.
If the following appears in basefile.tex then it opens basefile.jh, writes ‘Hello World!’ and a newline to it, and closes that file.
```
\newwrite\myfile
\immediate\openout\myfile=\jobname.jh  % \jobname is root file basename
...
\immediate\write\myfile{Hello world!}
...
\immediate\closeout\myfile

```

The `\newwrite` allocates a stream number, giving it a symbolic name to make life easier, so that `stream \newwrite\myfile\the\myfile` produces something like ‘stream 3’. Then `\openout` associates the stream number with the given file name. TeX ultimately executed `\write3` which puts the string in the file.
Typically number is between 0 and 15, because typically LaTeX authors follow the prior example and the number is allocated by the system. If number is outside the range from 0 to 15 or if it is not associated with an open file then LaTeX writes string to the log file. If number is positive then in addition LaTeX writes string to the terminal.
Thus, `test \write-1{Hello World!}` puts ‘Hello World!’ followed by a newline in the log file. (This is what the `\wlog` command does; see [`\wlog`](https://latexref.xyz/dev/latex2e.html#g_t_005cwlog)). And `\write100{Hello World!}` puts the same in the log file but also puts ‘Hello World!’ followed by a newline in the terminal output. (But 16, 17, and 18 are special as number; see below.)
In LuaTeX, instead of 16 output streams there are 256 (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)).
Use `\write\@auxout{string}` to write to the current .aux file, which is associated with either the root file or with the current include file; and use `\write\@mainaux{string}` to write to the main .aux. These symbolic names are defined by LaTeX.
By default LaTeX does not write string to the file right away. This is because, for example, you may need `\write` to save the current page number, but when TeX comes across a `\write` it typically does not know what the page number is, since it has not yet done the page breaking. So, you use `\write` in one of three contexts:
```
\immediate\write\@auxout{string}      %1
\write\@auxout{string}                %2
\protected@write\@auxout{}{string}    %3

```

  1. With the first, LaTeX writes string to the file immediately. Any macros in string are fully expanded (just as in `\edef`) so to prevent expansion you must use `\noexpand`, `toks`, etc., except that you should use `#` instead of `##`).
  2. With the second, string is stored on the current list of things (as a TeX “whatsit” item) and kept until the page is shipped out and likewise the macros are unexpanded until `\shipout`. At `\shipout`, string is fully expanded.
  3. The third, `\protected@write`, is like the second except that you can use `\protect` to avoid expansion. The extra first argument allows you to locally insert extra definitions to make more macros protected or to have some other special definition for the write.

As a simple example of expansion with `\write`, string here contains a control sequence `\triplex` which we’ve defined to be the text ‘XYZ’:
```
\newwrite\jhfile
\openout\jhfile=test.jh
\newcommand{\triplex}{XYZ}
\write\jhfile{test \triplex test}

```

This results in the file test.jh containing the text ‘test XYZtest’ followed by a newline.
The cases where number is 16, 17, or 18 are special. Because of `\write`’s behavior when number is outside the range from 0 to 15 described above, in plain TeX `\write16` and `\write17` were sometimes used to write to the log file and the terminal; however, in LaTeX, the natural way to do that is with `\typeout` (see [`\typeout`](https://latexref.xyz/dev/latex2e.html#g_t_005ctypeout)). The `\write18` command is even more special; modern TeX systems use it to run an external operating system command (see [`\write18`](https://latexref.xyz/dev/latex2e.html#g_t_005cwrite18)).
Ordinarily `\write` outputs a single line. You can include a newline with `^^J`. Thus, this produces two lines in the log file:
```
\wlog{Parallel lines have a lot in common.^^JBut they never meet.}

```

A common case where authors need to write their own file is for answers to exercises, or another situation where you want to write out verbatim, without expanding the macros. CTAN has a number of packages for this; one is `answers`.
  * [`\write` and security](https://latexref.xyz/dev/latex2e.html#g_t_005cwrite-and-security)
  * [`\message`](https://latexref.xyz/dev/latex2e.html#g_t_005cmessage)
  * [`\wlog`](https://latexref.xyz/dev/latex2e.html#g_t_005cwlog)
  * [`\write18`](https://latexref.xyz/dev/latex2e.html#g_t_005cwrite18)

#### 27.5.1 `\write` and security
The ability to write files raises security issues. If you compiled a downloaded LaTeX file and it overwrote your password file then you would be justifiably troubled.
Thus, by default TeX systems only allow you to open files for writing that are in the current directory or output directory, if specified (see [output directory](https://latexref.xyz/dev/latex2e.html#output-directory)), or in a subdirectory of those. So, this code
```
\newwrite\jhfile
\openout\jhfile=../test.jh

```

gives an error like:
```
Not writing to ../test.jh (openout_any = p).
! I can't write on file `../test.jh'

```

You can get just such an error when using commands such as `\include{../filename}` because LaTeX will try to open ../filename.aux. The simplest solution is to put the included files in the same directory as the root file, or in subdirectories.
#### 27.5.2 `\message`
Synopsis:
```
\message{string}

```

Write string to the log file and the terminal.
Typically, LaTeX authors use `\typeout` (see [`\typeout`](https://latexref.xyz/dev/latex2e.html#g_t_005ctypeout)). It allows you to use `\protect` on any fragile commands in string (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)). But `\typeout` always inserts a newline at the end of string while `\message` does not, so the latter can be useful.
With this example document body.
```
before\message{One Two}\message{Three}\message{Four^^JI}
\message{declare a thumb war.}After

```

under some circumstances (see below) LaTeX writes the following to both the terminal and the log file.
```
One Two Three Four
I declare a thumb war.

```

The `^^J` produces a newline. Also, in the output document, between ‘before’ and ‘After’ will be a single space (from the end of line following ‘I}’).
While `\message` allows you more control over formatting, a gotcha is that LaTeX may mess up that formatting because it inserts line breaks depending on what it has already written. Contrast this document body, where the ‘Two’ has moved, to the one given above.
```
before\message{One}\message{Two Three}\message{Four^^JI}
\message{declare a thumb war.}After

```

This can happen: when LaTeX is outputting the messages to the terminal, now the message with ‘One’ is shorter and it fits at the end of the output terminal line, and so LaTeX breaks the line between it and the ‘Two Three’. That line break appears also in the log file. This line break insertion can depend on, for instance, the length of the full path names of included files. So producing finely-formatted lines in a way that is portable is hard, likely requiring starting your message at the beginning of a line.
#### 27.5.3 `\wlog`
Synopsis:
```
\wlog{string}

```

Write string to the log file.
```
\wlog{Did you hear about the mathematician who hates negatives?}
\wlog{He'll stop at nothing to avoid them.}

```

Ordinarily string appears in a single separate line. Use `^^J` to insert a newline.
```
\wlog{Helvetica and Times Roman walk into a bar.}
\wlog{The barman says,^^JWe don't serve your type.}

```

#### 27.5.4 `\write18`
Synopsis:
```
\write18{shell_command}

```

Issue a command to the operating system shell. The operating system runs the command and LaTeX’s execution is blocked until that finishes.
This sequence (on Unix)
```
\usepackage{graphicx}  % in preamble
  ...
\newcommand{\fignum}{1}
\immediate\write18{cd pix && asy figure\fignum}
\includegraphics{pix/figure\fignum.pdf}

```

will run Asymptote (the `asy` program) on pix/figure1.asy, so that the document can later read in the resulting graphic (see [`\includegraphics`](https://latexref.xyz/dev/latex2e.html#g_t_005cincludegraphics)). Like any `\write`, here LaTeX expands macros in shell_command so that `\fignum` is replaced by ‘1’.
Another example is that you can automatically run BibTeX at the start of each LaTeX run (see [Using BibTeX](https://latexref.xyz/dev/latex2e.html#Using-BibTeX)) by including `\immediate\write18{bibtex8 \jobname}` as the first line of the file. Note that `\jobname` expands to the basename of the root file unless the `--jobname` option is passed on the command line, in which case this is the option argument.
You sometimes need to do a multi-step process to get the information that you want. This will insert into the input a list of all PDF files in the current directory (but see `texosquery` below):
```
\immediate\write18{ls *.pdf > tmp.dat}
\input{tmp.dat}

```

The standard behavior of any `\write` is to wait until a page is being shipped out before expanding the macros or writing to the stream (see [`\write`](https://latexref.xyz/dev/latex2e.html#g_t_005cwrite)). But sometimes you want it done now. For this, use `\immediate\write18{shell_command}`.
There are obvious security issues with allowing system commands inside a LaTeX file. If you download a file off the net and it contains commands to delete all your files then you would be unhappy. The standard settings in modern distributions turn off full shell access. You can turn it on, if you are sure the shell commands are safe, by compiling with `latex --enable-write18 filename` (see [Command line options](https://latexref.xyz/dev/latex2e.html#Command-line-options)). (The `--shell-escape` option is a synonym, in TeX Live.)
In the place of full shell access, modern distributions by default use a restricted version that allows some commands to work, such as those that run Metafont to generate missing fonts, even if you do not use the `enable-write18` option. By default this list of allowed commands is short and features only commands that are under the control of the distribution maintainers (see [Command line options](https://latexref.xyz/dev/latex2e.html#Command-line-options)).
The shell_command text is always passed to /bin/sh on Unix-like operating systems, and the DOS command interpreter cmd.exe on Windows. Any different shell set by the user, and the `SHELL` environment variable, is ignored.
If what you need is system information, such as the operating system name, locale information, or directory contents, take a look at the `texosquery` package, which provides a convenient and secure interface for this, unlike the above examples using the raw `\write18`: <https://ctan.org/pkg/texosquery>.
LaTeX provides a package `shellesc` on top of the primitive `\write18` command. Its primary purpose is to provide a command `\ShellEscape` which works identically on all TeX engines; LuaTeX intentionally did not retain `\write18` as a way to invoke a shell command, so some engine-specific code is needed. The `shellesc` package also provides a command `\DelayedShellEscape`, executed at `\output` time, for the same reason.
