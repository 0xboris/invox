# 25 Front/back matter: Indexes

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 25.2.1 Produce the index manually
- 25.2.2 `\index`
- 25.2.3 `makeindex`
- 25.2.4 `\printindex`

### 25.2 Indexes
If you tell LaTeX what terms you want to appear in an index then it can produce that index, alphabetized and with the page numbers automatically maintained. This illustrates the basics.
```
\documentclass{article}
\usepackage{makeidx}  % Provide indexing commands
  \makeindex
% \usepackage{showidx}  % Show marginal notes of index entries
  ...
\begin{document}
  ...
Wilson's Theorem\index{Wilson's Theorem}
says that a number $n>1$ is prime if and only if the factorial
of $n-1$ is congruent to $-1$
modulo~$n$.\index{congruence!and Wilson's Theorem}
  ...
\printindex
\end{document}

```

As that shows, declare index entries with the `\index` command (see [`\index`](https://latexref.xyz/dev/latex2e.html#g_t_005cindex)). When you run LaTeX, the `\index` writes its information, such as ‘Wilson's Theorem’ and the page number, to an auxiliary file whose name ends in .idx. Next, to alphabetize and do other manipulations, run an external command, typically `makeindex` (see [`makeindex`](https://latexref.xyz/dev/latex2e.html#makeindex)), which writes a file whose name ends in .ind. Finally, `\printindex` brings this manipulated information into the output (see [`\printindex`](https://latexref.xyz/dev/latex2e.html#g_t_005cprintindex)).
Thus, if the above example is in the file numth.tex then running ‘pdflatex numth’ will save index entry and page number information to numth.idx. Then running ‘makeindex numth’ will alphabetize and save the results to numth.ind. Finally, again running ‘pdflatex numth’ will show the desired index, at the place where the `\printindex` command is in the source file.
There are many options for the output. An example is that the exclamation point in `\index{congruence!and Wilson's Theorem}` produces a main entry of ‘congruence’ with a subentry of ‘and Wilson's Theorem’. For more, see [`makeindex`](https://latexref.xyz/dev/latex2e.html#makeindex).
The `\makeindex` and `\printindex` commands are independent. Leaving out the `\makeindex` will stop LaTeX from saving the index entries to the auxiliary file. Leaving out the `\printindex` will cause LaTeX to not show the index in the document output.
There are many packages in the area of indexing. The `showidx` package causes each index entries to be shown in the margin on the page where the `\index` appears. This can help in preparing the index. The `multind` package, among others, supports multiple indexes. See also the TeX FAQ entry on this topic, <https://www.texfaq.org/FAQ-multind>, and the CTAN topic, <https://ctan.org/topic/index-multi>.
  * [Produce the index manually](https://latexref.xyz/dev/latex2e.html#Produce-the-index-manually)
  * [`\index`](https://latexref.xyz/dev/latex2e.html#g_t_005cindex)
  * [`makeindex`](https://latexref.xyz/dev/latex2e.html#makeindex)
  * [`\printindex`](https://latexref.xyz/dev/latex2e.html#g_t_005cprintindex)

#### 25.2.1 Produce the index manually
Documents that are small and static can have a manually produced index. This will make a separate page labeled ‘Index’, in twocolumn format.
```
\begin{theindex}
\item acorn squash, 1
\subitem maple baked, 2
\indexspace
\item bacon, 3
\subitem maple baked, 4
\end{theindex}

```

Note that the author must enter the page numbers, which is tedious and which will result in wrong numbers if the document changes. This is why in most cases automated methods such as `makeindex` are best. See [Indexes](https://latexref.xyz/dev/latex2e.html#Indexes).
However we cover the commands for completeness, and because the automated methods are based on these commands. There are three levels of entries. As the example shows, a main entry uses `\item`, subentries use `\subitem`, and the lowest level uses `\subsubitem`. Blank lines between entries have no effect. The example above includes `\indexspace` to produce vertical space in the output that some index styles use before the first entry starting with a new letter.
#### 25.2.2 `\index`
Synopsis:
```
\index{index-entry-string}

```

Declare an entry in the index. This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
For example, as described in [Indexes](https://latexref.xyz/dev/latex2e.html#Indexes), one way to get an index from what’s below is to compile the document with `pdflatex test`, then process the index entries with `makeindex test`, and then compile again with `pdflatex test`.
```
% file test.tex
  ...
W~Ackermann (1896--1962).\index{Ackermann}
  ...
Ackermann function\index{Ackermann!function}
  ...
rate of growth\index{Ackermann!function!growth rate}

```

All three index entries will get a page number, such as ‘Ackermann, 22’. LaTeX will format the second as a subitem of the first, on the line below it and indented, and the third as a subitem of the second. Three levels deep is as far as you can nest subentries. (If you add `\index{Ackermann!function!growth rate!comparison}` then `makeindex` says ‘Scanning input file test.idx....done (4 entries accepted, 1 rejected)’ and the fourth level is silently missing from the index.)
If you enter a second `\index` with the same index-entry-string then you will get a single index entry with two page numbers (unless they happen to fall on the same page). Thus, adding `as for Ackermann.\index{Ackermann}` later in the same document as above will give an index entry like ‘Ackermann, 22, 151’. Also, you can enter the index entries in any order, so for instance `\index{Ackermann!function}` could come before `\index{Ackermann}`.
Get a page range in the output, like ‘Hilbert, 23--27’, as here.
```
W~Ackermann (1896--1962).\index{Ackermann}
  ...
D~Hilbert (1862--1943)\index{Ackermann!Hilbert|(}
  ...
disapproved of his marriage.\index{Ackermann!Hilbert|)}

```

If the beginning and ending of the page range are equal then the system just gives a single page number, not a range.
If you index subentries but not a main entry, as with `\index{Jones!program}` and `\index{Jones!results}`, then the output is the item ‘Jones’ with no comma or page number, followed by two subitems, like ‘program, 50’ and ‘results, 51’.
Generate a index entry that says ‘see’ by using a vertical bar character: `\index{Ackermann!function|see{P\'eter's function}}`. You can instead get ‘see also’ with `seealso`. (The text ‘see’ is defined by `\seename`, and ‘see also’ by `\alsoname`. You can redefine these either by using an internationalization package such as `babel` or `polyglossia`, or directly as with `\renewcommand{\alsoname}{Also see}`.)
The ‘see’ feature is part of a more general functionality. After the vertical bar you can put the name of a one-input command, as in `\index{group|textit}` (note the missing backslash on the `\textit` command) and the system will apply that command to the page number, here giving something like `\textit{7}`. You can define your own one-input commands, such as `\newcommand{\definedpage}[1]{{\color{blue}#1}}` and then `\index{Ackermann!function|definedpage}` will give a blue page number (see [Color](https://latexref.xyz/dev/latex2e.html#Color)). Another, less practical, example is this,
```
\newcommand\indexownpage[1]{#1, \thepage}
  ... Epimenides.\index{self-reference|indexownpage}

```

which creates an entry citing the page number of its own index listing.
The two functions just described combine, as here
```
\index{Ackermann!function|(definedpage}
  ...
\index{Ackermann!function|)}

```

which outputs an index entry like ‘function, 23--27’ where the page number range is in blue.
Consider an index entry such as ‘α-ring’. Entering it as `$\alpha$-ring` will cause it to be alphabetized according to the dollar sign. You can instead enter it using an at-sign, as `\index{alpha-ring@$\alpha$-ring}`. If you specify an entry with an at-sign separating two strings, `pos@text`, then pos gives the alphabetical position of the entry while text produces the text of the entry. Another example is that `\index{Saint Michael's College@SMC}` produces an index entry ‘SMC’ alphabetized into a different location than its spelling would naturally give it.
To put a `!`, or `@`, or `|`, or `"` character in an index entry, escape it by preceding it with a double quote, `"`. (The double quote gets deleted before alphabetization.)
A number of packages on CTAN have additional functionality beyond that provided by `makeidx`. One is `index`, which allows for multiple indices and contains a command `\index*{index-entry-string}` that prints the index-entry-string as well as indexing it.
The `\index` command writes the indexing information to the file root-name.idx file. Specifically, it writes text of the command `\indexentry{index-entry-string}{page-num}`, where page-num is the value of the `\thepage` counter. On occasion, when the `\printindex` command is confused, you have to delete this file to start with a fresh slate.
If you omit the closing brace of an `\index` command then you get a message like this.
```
Runaway argument?  {Ackermann!function
!  Paragraph ended before \@wrindex was complete.

```

#### 25.2.3 `makeindex`
Synopsis, one of:
```
makeindex filename
makeindex -s style-file filename
makeindex options filename0 ...

```

Sort, and otherwise process, the index information in the auxiliary file filename. This is a command line program. It takes one or more raw index files, filename.idx files, and produces the actual index file, the filename.ind file that is input by `\printindex` (see [`\printindex`](https://latexref.xyz/dev/latex2e.html#g_t_005cprintindex)).
The first form of the command suffices for many uses. The second allows you to format the index by using an _index style file_ , a .isty file. The third form is the most general; see the full documentation on CTAN.
This is a simple .isty file.
```
% book.isty
%   $ makeindex -s book.isty -p odd book.idx
% creates the index as book.ind, starting on an odd page.
preamble
"\\pagestyle{empty}
\\small
\\begin{theindex}
\\thispagestyle{empty}"

postamble
"\n
\\end{theindex}"

```

The description here covers only some of the index formatting possibilities in style-file. For a full list see the documentation on CTAN.
A style file consists of a list of pairs: specifier and attribute. These can appear in the file in any order. All of the attributes are strings, except where noted. Strings are surrounded with double quotes, `"`, and the maximum length of a string is 144 characters. The `\n` is for a newline and `\t` is for a tab. Backslashes are escaped with another backslash, `\\`. If a line begins with a percent sign, `%`, then it is a comment.

`preamble`

Preamble of the output index file. Defines the context in which the index is formatted. Default: `"\\begin{theindex}\n"`.

`postamble`

Postamble of the output index file. Default: `"\n\n\\end{theindex}\n"`.

`group_skip`

Traditionally index items are broken into groups, typically a group for entries starting with letter ‘a’, etc. This specifier gives what is inserted when a new group begins. Default: `"\n\n \\indexspace\n"` (`\indexspace` is a command inserting a rubber length, by default `10pt plus5pt minus3pt`).

`lethead_flag`

An integer. It governs what is inserted for a new group or letter. If it is 0 (which is the default) then other than `group_skip` nothing will be inserted before the group. If it is positive then at a new letter the `lethead_prefix` and `lethead_suffix` will be inserted, with that letter in uppercase between them. If it is negative then what will be inserted is the letter in lowercase. The default is 0.

`lethead_prefix`

If a new group begins with a different letter then this is the prefix inserted before the new letter header. Default: `""`

`lethead_suffix`

If a group begins with a different letter then this is the suffix inserted after the new letter header. Default: `""`.

`item_0`

What is put between two level 0 items. Default: `"\n \\item "`.

`item_1`

Put between two level 1 items. Default: `"\n \\subitem "`.

`item_2`

put between two level 2 items. Default: `"\n \\subsubitem "`.

`item_01`

What is put between a level 0 item and a level 1 item. Default: `"\n \\subitem "`.

`item_x1`

What is put between a level 0 item and a level 1 item in the case that the level 0 item doesn’t have any page numbers (as in `\index{aaa|see{bbb}}`). Default: `"\n \\subitem "`.

`item_12`

What is put between a level 1 item and a level 2 item. Default: `"\n \\subsubitem "`.

`item_x2`

What is put between a level 1 item and a level 2 item, if the level 1 item doesn’t have page numbers. Default: `"\n \\subsubitem "`.

`delim_0`

Delimiter put between a level 0 key and its first page number. Default: a comma followed by a blank, `", "`.

`delim_1`

Delimiter put between a level 1 key and its first page number. Default: a comma followed by a blank, `", "`.

`delim_2`

Delimiter between a level 2 key and its first page number. Default: a comma followed by a blank, `", "`.

`delim_n`

Delimiter between two page numbers for the same key (at any level). Default: a comma followed by a blank, `", "`.

`delim_r`

What is put between the starting and ending page numbers of a range. Default: `"--"`.

`line_max`

An integer. Maximum length of an index entry’s line in the output, beyond which the line wraps. Default: `72`.

`indent_space`

What is inserted at the start of a wrapped line. Default: `"\t\t"`.

`indent_length`

A number. The length of the wrapped line indentation. The default `indent_space` is two tabs and each tab is eight spaces so the default here is `16`.

`page_precedence`

A document may have pages numbered in different ways. For example, a book may have front matter pages numbered in lowercase roman while main matter pages are in arabic. This string specifies the order in which they will appear in the index. The `makeindex` command supports five different types of numerals: lowercase roman `r`, and numeric or arabic `n`, and lowercase alphabetic `a`, and uppercase roman `R`, and uppercase alphabetic `A`. Default: `"rnaRA"`.
There are a number of other programs that do the job `makeindex` does. One is `xindy` (<https://ctan.org/pkg/xindy>), which does internationalization and can process indexes for documents marked up using LaTeX and a number of other languages. It is written in Lisp, highly configurable, both in markup terms and in terms of the collating order of the text, as described in its documentation.
A more recent indexing program supporting Unicode is `xindex`, written in Lua (<https://ctan.org/pkg/xindex>).
#### 25.2.4 `\printindex`
Synopsis:
```
\printindex

```

Place the index into the output.
To get an index you must first include `\usepackage{makeidx}\makeindex` in the document preamble and compile the document, then run the system command `makeindex`, and then compile the document again. See [Indexes](https://latexref.xyz/dev/latex2e.html#Indexes), for further discussion and an example of the use of `\printindex`.
