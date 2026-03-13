# 24 Splitting the input

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 24.1 `\endinput`
- 24.2 `\include` & `\includeonly`
- 24.3 `\input`

## 24 Splitting the input
LaTeX lets you split a large document into several smaller ones. This can simplify editing or allow multiple authors to work on the document. It can also speed processing.
Regardless of how many separate files you use, there is always one  _root file_ , on which LaTeX compilation starts. This shows such a file with five included files.
```
\documentclass{book}
\includeonly{  % comment out lines below to omit compiling
  pref,
  chap1,
  chap2,
  append,
  bib
  }
\begin{document}
\frontmatter
\include{pref}
\mainmatter
\include{chap1}
\include{chap2}
\appendix
\include{append}
\backmatter
\include{bib}
\end{document}

```

This will bring in material from pref.tex, chap1.tex, chap2.tex, append.tex, and bib.tex. If you compile this file, and then comment out all of the lines inside `\includeonly{...}` except for `chap1`, and compile again, then LaTeX will only process the material in the first chapter. Thus, your output will appear more quickly and be shorter to print. However, the advantage of the `\includeonly` command is that LaTeX will retain the page numbers and all of the cross reference information from the other parts of the document so these will appear in your output correctly.
See [Larger `book` template](https://latexref.xyz/dev/latex2e.html#Larger-book-template), for another example of `\includeonly`.
  * [`\endinput`](https://latexref.xyz/dev/latex2e.html#g_t_005cendinput)
  * [`\include` & `\includeonly`](https://latexref.xyz/dev/latex2e.html#g_t_005cinclude-_0026-_005cincludeonly)
  * [`\input`](https://latexref.xyz/dev/latex2e.html#g_t_005cinput)

### 24.1 `\endinput`
Synopsis:
```
\endinput

```

When you `\include{filename}`, inside filename.tex the material after `\endinput` will not be included. This command is optional; if filename.tex has no `\endinput` then LaTeX will read all of the file.
For example, suppose that a document’s root file has `\input{chap1}` and this is chap1.tex.
```
\chapter{One}
This material will appear in the document.
\endinput
This will not appear.

```

This can be useful for putting documentation or comments at the end of a file, or for avoiding junk characters that can be added if the file is transmitted in the body of an email. It is also useful for debugging: one strategy to localize errors is to put `\endinput` halfway through the included file and see if the error disappears. Now, knowing which half contains the error, moving `\endinput` to halfway through that area further narrows down the location. This process rapidly finds the offending line.
After reading `\endinput`, LaTeX continues to read to the end of the line, so something can follow this command and be read nonetheless. This allows you, for instance, to close an `\if...` with a `\fi`.
### 24.2 `\include` & `\includeonly`
Synopsis:
```
\includeonly{  % in document preamble
  ...
  filename,
  ...
  }
  ...
\include{filename}  % in document body

```

Bring material from the external file filename.tex into a LaTeX document.
The `\include` command does three things: it executes `\clearpage` (see [`\clearpage` & `\cleardoublepage`](https://latexref.xyz/dev/latex2e.html#g_t_005cclearpage-_0026-_005ccleardoublepage)), then it inputs the material from filename.tex into the document, and then it does another `\clearpage`. This command can only appear in the document body.
The `\includeonly` command controls which files will be read by LaTeX under subsequent `\include` commands. Its list of filenames is comma-separated. It must appear in the preamble or even earlier, e.g., the command line; it can’t appear in the document body.
This example root document, constitution.tex, brings in three files, preamble.tex, articles.tex, and amendments.tex.
```
\documentclass{book}
\includeonly{
  preamble,
  articles,
  amendments
  }
\begin{document}
\include{preamble}
\include{articles}
\include{amendments}
\end{document}

```

The file preamble.tex contains no special code; you have just excerpted the chapter from consitution.tex and put it in a separate file just for editing convenience.
```
\chapter{Preamble}
We the People of the United States,
in Order to form a more perfect Union, ...

```

Running LaTeX on constitution.tex makes the material from the three files appear in the document but also generates the auxiliary files preamble.aux, articles.aux, and amendments.aux. These contain information such as page numbers and cross-references (see [Cross references](https://latexref.xyz/dev/latex2e.html#Cross-references)). If you now comment out `\includeonly`’s lines with `preamble` and `amendments` and run LaTeX again then the resulting document shows only the material from articles.tex, not the material from preamble.tex or amendments.tex. Nonetheless, all of the auxiliary information from the omitted files is still there, including the starting page number of the chapter.
If the document preamble does not have `\includeonly` then LaTeX will include all the files you call for with `\include` commands.
The `\include` command makes a new page. To avoid that, see [`\input`](https://latexref.xyz/dev/latex2e.html#g_t_005cinput) (which, however, does not retain the auxiliary information).
See [Larger `book` template](https://latexref.xyz/dev/latex2e.html#Larger-book-template), for another example using `\include` and `\includeonly`. That example also uses `\input` for some material that will not necessarily start on a new page.
File names can involve paths.
```
\documentclass{book}
\includeonly{
  chapters/chap1,
  }
\begin{document}
\include{chapters/chap1}
\end{document}

```

To make your document portable across distributions and platforms you should avoid spaces in the file names. The tradition is to instead use dashes or underscores. Nevertheless, for the name ‘amo amas amat’, this works under TeX Live on GNU/Linux:
```
\documentclass{book}
\includeonly{
  "amo\space amas\space amat"
  }
\begin{document}
\include{"amo\space amas\space amat"}
\end{document}

```

and this works under MiKTeX on Windows:
```
\documentclass{book}
\includeonly{
  {"amo amas amat"}
  }
\begin{document}
\include{{"amo amas amat"}}
\end{document}

```

You cannot use `\include` inside a file that is being included or you get ‘LaTeX Error: \include cannot be nested.’ The `\include` command cannot appear in the document preamble; you will get ‘LaTeX Error: Missing \begin{document}’.
If a file that you `\include` does not exist, for instance if you `\include{athiesm}` but you meant `\include{atheism}`, then LaTeX does not give you an error but will warn you ‘No file athiesm.tex.’ (It will also create athiesm.aux.)
If you `\include` the root file in itself then you first get ‘LaTeX Error: Can be used only in preamble.’ Later runs get ‘TeX capacity exceeded, sorry [text input levels=15]’. To fix this, you must remove the inclusion `\include{root}` but also delete the file root.aux and rerun LaTeX.
### 24.3 `\input`
Synopsis:
```
\input{filename}

```

LaTeX processes the file as if its contents were inserted in the current file. For a more sophisticated inclusion mechanism see [`\include` & `\includeonly`](https://latexref.xyz/dev/latex2e.html#g_t_005cinclude-_0026-_005cincludeonly).
If filename does not end in ‘.tex’ then LaTeX first tries the filename with that extension; this is the usual case. If filename ends with ‘.tex’ then LaTeX looks for the filename as it is.
For example, this
```
\input{macros}

```

will cause LaTeX to first look for macros.tex. If it finds that file then it processes its contents as thought they had been copy-pasted in. If there is no file of the name macros.tex then LaTeX tries the name macros, without an extension. (This may vary by distribution.)
To make your document portable across distributions and platforms you should avoid spaces in the file names. The tradition is to instead use dashes or underscores. Nevertheless, for the name ‘amo amas amat’, this works under TeX Live on GNU/Linux:
```
\input{"amo\space amas\space amat"}

```

and this works under MiKTeX on Windows:
```
\input{{"amo amas amat"}}

```
