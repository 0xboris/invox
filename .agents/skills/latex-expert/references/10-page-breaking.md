# 10 Page breaking

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 10.1 `\clearpage` & `\cleardoublepage`
- 10.2 `\newpage`
- 10.3 `\enlargethispage`
- 10.4 `\pagebreak` & `\nopagebreak`

## 10 Page breaking
Ordinarily LaTeX automatically takes care of breaking output into pages with its usual aplomb. But if you are writing commands, or tweaking the final version of a document, then you may need to understand how to influence its actions.
LaTeX’s algorithm for splitting a document into pages is more complex than just waiting until there is enough material to fill a page and outputting the result. Instead, LaTeX typesets more material than would fit on the page and then chooses a break that is optimal in some way (it has the smallest _badness_). An example of the advantage of this approach is that if the page has some vertical space that can be stretched or shrunk, such as with rubber lengths between paragraphs, then LaTeX can use that to avoid widow lines (where a new page starts with the last line of a paragraph; LaTeX can squeeze the extra line onto the first page) and orphans (where the first line of paragraph is at the end of a page; LaTeX can stretch the material of the first page so the extra line falls on the second page). Another example is where LaTeX uses available vertical shrinkage to fit on a page not just the header for a new section but also the first two lines of that section.
But LaTeX does not optimize over the entire document’s set of page breaks. So it can happen that the first page break is great but the second one is lousy; to break the current page LaTeX doesn’t look as far ahead as the next page break. So occasionally you may want to influence page breaks while preparing a final version of a document.
See [Layout](https://latexref.xyz/dev/latex2e.html#Layout), for more material that is relevant to page breaking.
  * [`\clearpage` & `\cleardoublepage`](https://latexref.xyz/dev/latex2e.html#g_t_005cclearpage-_0026-_005ccleardoublepage)
  * [`\newpage`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewpage)
  * [`\enlargethispage`](https://latexref.xyz/dev/latex2e.html#g_t_005cenlargethispage)
  * [`\pagebreak` & `\nopagebreak`](https://latexref.xyz/dev/latex2e.html#g_t_005cpagebreak-_0026-_005cnopagebreak)

### 10.1 `\clearpage` & `\cleardoublepage`
Synopsis:
```
\clearpage

```

or
```
\cleardoublepage

```

End the current page and output all of the pending floating figures and tables (see [Floats](https://latexref.xyz/dev/latex2e.html#Floats)). If there are too many floats to fit on the page then LaTeX will put in extra pages containing only floats. In two-sided printing, `\cleardoublepage` also makes the next page of content a right-hand page, an odd-numbered page, if necessary inserting a blank page. The `\clearpage` command is robust while `\cleardoublepage` is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
LaTeX’s page breaks are optimized so ordinarily you only use this command in a document body to polish the final version, or inside commands.
The `\cleardoublepage` command will put in a blank page, but it will have the running headers and footers. To get a really blank page, use this command.
```
\let\origdoublepage\cleardoublepage
\newcommand{\clearemptydoublepage}{%
  \clearpage
  {\pagestyle{empty}\origdoublepage}%
}

```

If you want LaTeX’s standard `\chapter` command to do this then add the line `\let\cleardoublepage\clearemptydoublepage`. (Of course this affects all uses of `\cleardoublepage`, not just the one in `\chapter`.)
The command `\newpage` (see [`\newpage`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewpage)) also ends the current page, but without clearing pending floats. And, if LaTeX is in two-column mode then `\newpage` ends the current column while `\clearpage` and `\cleardoublepage` end the current page.
### 10.2 `\newpage`
Synopsis:
```
\newpage

```

End the current page. This command is robust (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
LaTeX’s page breaks are optimized so ordinarily you only use this command in a document body to polish the final version, or inside commands.
While the commands `\clearpage` and `\cleardoublepage` also end the current page, in addition they clear pending floats (see [`\clearpage` & `\cleardoublepage`](https://latexref.xyz/dev/latex2e.html#g_t_005cclearpage-_0026-_005ccleardoublepage)). And, if LaTeX is in two-column mode then `\clearpage` and `\cleardoublepage` end the current page, possibly leaving an empty column, while `\newpage` only ends the current column.
In contrast with `\pagebreak` (see [`\pagebreak` & `\nopagebreak`](https://latexref.xyz/dev/latex2e.html#g_t_005cpagebreak-_0026-_005cnopagebreak)), the `\newpage` command will cause the new page to start right where requested. This
```
Four score and seven years ago our fathers brought forth on this
continent,
\newpage
\noindent a new nation, conceived in Liberty, and dedicated to the
proposition that all men are created equal.

```

makes a new page start after ‘continent’, and the cut-off line is not right justified. In addition, `\newpage` does not vertically stretch out the page, as `\pagebreak` does.
### 10.3 `\enlargethispage`
Synopsis, one of:
```
\enlargethispage{size}
\enlargethispage*{size}

```

Enlarge the `\textheight` for the current page. The required argument size must be a rigid length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). It may be positive or negative. This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
A common strategy is to wait until you have the final text of a document, and then pass through it tweaking line and page breaks. This command allows you some page size leeway.
This will allow one extra line on the current page.
```
\enlargethispage{\baselineskip}

```

The starred form `\enlargesthispage*` tries to squeeze the material together on the page as much as possible, for the common use case of getting one more line on the page. This is often used together with an explicit `\pagebreak`.
### 10.4 `\pagebreak` & `\nopagebreak`
Synopses:
```
\pagebreak
\pagebreak[zero-to-four]

```

or
```
\nopagebreak
\nopagebreak[zero-to-four]

```

Encourage or discourage a page break. The optional zero-to-four is an integer that allows you to soften the request. The default is 4, so that without the optional argument these commands entirely force or prevent the break. But for instance `\nopagebreak[1]` suggests to LaTeX that another spot might be preferable. The higher the number, the more insistent the request. Both commands are fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
LaTeX’s page endings are optimized so ordinarily you only use these commands in a document body to polish the final version, or inside commands.
If you use these inside a paragraph, they apply to the point following the line in which they appear. So this
```
Four score and seven years ago our fathers brought forth on this
continent,
\pagebreak
a new nation, conceived in Liberty, and dedicated to the proposition
that all men are created equal.

```

does not give a page break at ‘continent’, but instead at ‘nation’, since that is where LaTeX breaks that line. In addition, with `\pagebreak` the vertical space on the page is stretched out where possible so that it extends to the normal bottom margin. This can look strange, and if `\flushbottom` is in effect this can cause you to get ‘Underfull \vbox (badness 10000) has occurred while \output is active’. See [`\newpage`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewpage), for a command that does not have these effects.
A declaration `\samepage` and corresponding `samepage` environment try to only allow breaks between paragraphs. They are not perfectly reliable. For more on keeping material on the same page, see the FAQ entry <https://texfaq.org/FAQ-nopagebrk>.)
