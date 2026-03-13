# 08 Environments: Tables, tabular, and bibliography

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 8.22 `table`
- 8.23 `tabular`
- 8.24 `thebibliography`

### 8.22 `table`
Synopsis:
```
\begin{table}[placement]
  table body
  \caption[loftitle]{title}  % optional
  \label{label}              % also optional
\end{table}

```

A class of floats (see [Floats](https://latexref.xyz/dev/latex2e.html#Floats)). They cannot be split across pages and so they are not typeset in sequence with the normal text but instead are floated to a convenient place, such as the top of a following page.
This example `table` environment contains a `tabular`
```
\begin{table}
  \centering\small
  \begin{tabular}{ll}
    \multicolumn{1}{c}{\textit{Author}}
      &\multicolumn{1}{c}{\textit{Piece}}  \\ \hline
    Bach            &Cello Suite Number 1  \\
    Beethoven       &Cello Sonata Number 3 \\
    Brahms          &Cello Sonata Number 1
  \end{tabular}
  \caption{Top cello pieces}
  \label{tab:cello}
\end{table}

```

but you can put many different kinds of content in a `table`: the table body may contain text, LaTeX commands, graphics, etc. It is typeset in a `parbox` of width `\textwidth`.
For the possible values of placement and their effect on the float placement algorithm, see [Floats](https://latexref.xyz/dev/latex2e.html#Floats).
The label is optional; it is used for cross references (see [Cross references](https://latexref.xyz/dev/latex2e.html#Cross-references)).  The `\caption` command is also optional. It specifies caption text title for the table (see [`\caption`](https://latexref.xyz/dev/latex2e.html#g_t_005ccaption)). By default it is numbered. If its optional lottitle is present then that text is used in the list of tables instead of title (see [Table of contents, list of figures, list of tables](https://latexref.xyz/dev/latex2e.html#Table-of-contents-etc_002e)).
In this example the table and caption will float to the bottom of a page, unless it is pushed to a float page at the end.
```
\begin{table}[b]
  \centering
  \begin{tabular}{r|p{2in}} \hline
    One &The loneliest number \\
    Two &Can be as sad as one.
         It's the loneliest number since the number one.
  \end{tabular}
  \caption{Cardinal virtues}
  \label{tab:CardinalVirtues}
\end{table}

```

### 8.23 `tabular`
Synopsis:
```
\begin{tabular}[pos]{cols}
  column 1 entry  &column 2 entry  ...  &column n entry \\
  ...
\end{tabular}

```

or
```
\begin{tabular*}{width}[pos]{cols}
  column 1 entry  &column 2 entry  ...  &column n entry \\
  ...
\end{tabular*}

```

Produce a table, a box consisting of a sequence of horizontal rows. Each row consists of items that are aligned vertically in columns. This illustrates many of the features.
```
\begin{tabular}{l|l}
  \textit{Player name}  &\textit{Career home runs}  \\
  \hline
  Hank Aaron  &755 \\
  Babe Ruth   &714
\end{tabular}

```

The output will have two left-aligned columns with a vertical bar between them. This is specified in `tabular`’s argument `{l|l}`.  Put the entries into different columns by separating them with an ampersand, `&`. The end of each row is marked with a double backslash, `\\`. Put a horizontal rule below a row, after a double backslash, with `\hline`.  After the last row the `\\` is optional, unless an `\hline` command follows to put a rule below the table.
The required and optional arguments to `tabular` consist of:

pos

Optional. Specifies the table’s vertical position. The default is to align the table so its vertical center matches the baseline of the surrounding text. There are two other possible alignments: `t` aligns the table so its top row matches the baseline of the surrounding text, and `b` aligns on the bottom row.
This only has an effect if there is other text. In the common case of a `tabular` alone in a `center` environment this option makes no difference.

cols

Required. Specifies the formatting of columns. It consists of a sequence of the following specifiers, corresponding to the types of column and intercolumn material.

`l`

A column of left-aligned items.

`r`

A column of right-aligned items.

`c`

A column of centered items.

`|`

A vertical line the full height and depth of the environment.

`@{text or space}`

Insert text or space at this location in every row. The text or space material is typeset in LR mode. This text is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
If between two column specifiers there is no @-expression then LaTeX’s `book`, `article`, and `report` classes will put on either side of each column a space of width `\tabcolsep`, which by default is 6pt. That is, by default adjacent columns are separated by 12pt (so `\tabcolsep` is misleadingly named since it is only half of the separation between tabular columns). In addition, a space of `\tabcolsep` also comes before the first column and after the final column, unless you put a `@{...}` there.
If you override the default and use an @-expression then LaTeX does not insert `\tabcolsep` so you must insert any desired space yourself, as in `@{\hspace{1em}}`.
An empty expression `@{}` will eliminate the space. In particular, sometimes you want to eliminate the space before the first column or after the last one, as in the example below where the tabular lines need to lie on the left margin.
```
\begin{flushleft}
  \begin{tabular}{@{}l}
    ...
  \end{tabular}
\end{flushleft}

```

The next example shows text, a decimal point between the columns, arranged so the numbers in the table are aligned on it.
```
\begin{tabular}{r@{$.$}l}
  $3$ &$14$  \\
  $9$ &$80665$
\end{tabular}

```

An `\extracolsep{wd}` command in an @-expression causes an extra space of width wd to appear to the left of all subsequent columns, until countermanded by another `\extracolsep`. Unlike ordinary intercolumn space, this extra space is not suppressed by an @-expression. An `\extracolsep` command can be used only in an @-expression in the `cols` argument. Below, LaTeX inserts the right amount of intercolumn space to make the entire table 4 inches wide.
```
\begin{tabular*}{4in}{l@{\extracolsep{\fill}}l}
  Seven times down, eight times up \ldots
  &such is life!
\end{tabular*}

```

To insert commands that are automatically executed before a given column, load the `array` package and use the `>{...}` specifier.

`p{wd}`

Each item in the column is typeset in a parbox of width wd, as if it were the argument of a `\parbox[t]{wd}{...}` command.
A line break double backslash `\\` may not appear in the item, except inside an environment like `minipage`, `array`, or `tabular`, or inside an explicit `\parbox`, or in the scope of a `\centering`, `\raggedright`, or `\raggedleft` declaration (when used in a `p`-column element these declarations must appear inside braces, as with `{\centering .. \\ ..}`). Otherwise LaTeX will misinterpret the double backslash as ending the tabular row. Instead, to get a line break in there use `\newline` (see [`\newline`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewline)).

`*{num}{cols}`

Equivalent to num copies of cols, where num is a positive integer and cols is a list of specifiers. Thus the specifier `\begin{tabular}{|*{3}{l|r}|}` is equivalent to the specifier `\begin{tabular}{|l|rl|rl|r|}`. Note that cols may contain another `*`-expression.

width

Required for `tabular*`, not allowed for `tabular`. Specifies the width of the `tabular*` environment. The space between columns should be rubber, as with `@{\extracolsep{\fill}}`, to allow the table to stretch or shrink to make the specified width, or else you are likely to get the `Underfull \hbox (badness 10000) in alignment ...` warning.
Parameters that control formatting:

`\arrayrulewidth`

A length that is the thickness of the rule created by `|`, `\hline`, and `\vline` in the `tabular` and `array` environments. The default is ‘.4pt’. Change it as in `\setlength{\arrayrulewidth}{0.8pt}`.

`\arraystretch`

A factor by which the spacing between rows in the `tabular` and `array` environments is multiplied. The default is ‘1’, for no scaling. Change it as `\renewcommand{\arraystretch}{1.2}`.

`\doublerulesep`

A length that is the distance between the vertical rules produced by the `||` specifier. The default is ‘2pt’.

`\tabcolsep`

A length that is half of the space between columns. The default is ‘6pt’. Change it with `\setlength`.
The following commands can be used inside the body of a `tabular` environment, the first two inside an entry and the second two between lines:
  * [`\multicolumn`](https://latexref.xyz/dev/latex2e.html#g_t_005cmulticolumn)
  * [`\vline`](https://latexref.xyz/dev/latex2e.html#g_t_005cvline)
  * [`\cline`](https://latexref.xyz/dev/latex2e.html#g_t_005ccline)
  * [`\hline`](https://latexref.xyz/dev/latex2e.html#g_t_005chline)

#### 8.23.1 `\multicolumn`
Synopsis:
```
\multicolumn{numcols}{cols}{text}

```

Make an `array` or `tabular` entry that spans several columns. The first argument numcols gives the number of columns to span. The second argument cols specifies the formatting of the entry, with `c` for centered, `l` for flush left, or `r` for flush right. The third argument text gives the contents of that entry.
In this example, in the first row, the second and third columns are spanned by the single heading ‘Name’.
```
\begin{tabular}{lccl}
  \textit{ID}       &\multicolumn{2}{c}{\textit{Name}} &\textit{Age} \\
  \hline
  978-0-393-03701-2 &O'Brian &Patrick                  &55           \\
    ...
\end{tabular}

```

What counts as a column is: the column format specifier for the `array` or `tabular` environment is broken into parts, where each part (except the first) begins with `l`, `c`, `r`, or `p`. So from `\begin{tabular}{|r|ccp{1.5in}|}` the parts are `|r|`, `c`, `c`, and `p{1.5in}|`.
The cols argument overrides the `array` or `tabular` environment’s intercolumn area default adjoining this multicolumn entry. To affect that area, this argument can contain vertical bars `|` indicating the placement of vertical rules, and `@{...}` expressions. Thus if cols is ‘|c|’ then this multicolumn entry will be centered and a vertical rule will come in the intercolumn area before it and after it. This table details the exact behavior.
```
\begin{tabular}{|cc|c|c|}
  \multicolumn{1}{r}{w}       % entry one
    &\multicolumn{1}{|r|}{x}  % entry two
    &\multicolumn{1}{|r}{y}   % entry three
    &z                        % entry four
\end{tabular}

```

Before the first entry the output will not have a vertical rule because the `\multicolumn` has the cols specifier ‘r’ with no initial vertical bar. Between entry one and entry two there will be a vertical rule; although the first cols does not have an ending vertical bar, the second cols does have a starting one. Between entry two and entry three there is a single vertical rule; despite that the cols in both of the surrounding `multicolumn`’s call for a vertical rule, you only get one rule. Between entry three and entry four there is no vertical rule; the default calls for one but the cols in the entry three `\multicolumn` leaves it out, and that takes precedence. Finally, following entry four there is a vertical rule because of the default.
The number of spanned columns numcols can be 1. Besides giving the ability to change the horizontal alignment, this also is useful to override for one row the `tabular` definition’s default intercolumn area specification, including the placement of vertical rules.
In the example below, in the `tabular` definition the first column is specified to default to left justified but in the first row the entry is centered with `\multicolumn{1}{c}{\textsc{Period}}`. Also in the first row, the second and third columns are spanned by a single entry with `\multicolumn{2}{c}{\textsc{Span}}`, overriding the specification to center those two columns on the page range en-dash.
```
\begin{tabular}{l|r@{--}l}
  \multicolumn{1}{c}{\textsc{Period}}
    &\multicolumn{2}{c}{\textsc{Span}} \\ \hline
  Baroque          &1600           &1760         \\
  Classical        &1730           &1820         \\
  Romantic         &1780           &1910         \\
  Impressionistic  &1875           &1925
\end{tabular}

```

Although the `tabular` specification by default puts a vertical rule between the first and second columns, no such vertical rule appears in the first row here. That’s because there is no vertical bar in the cols part of the first row’s first `\multicolumn` command.
#### 8.23.2 `\vline`
Draw a vertical line in a `tabular` or `array` environment extending the full height and depth of an entry’s row. Can also be used in an @-expression, although its synonym vertical bar `|` is more common. This command is rarely used in the body of a table; typically a table’s vertical lines are specified in `tabular`’s cols argument and overridden as needed with `\multicolumn` (see [`tabular`](https://latexref.xyz/dev/latex2e.html#tabular)).
The example below illustrates some pitfalls. In the first row’s second entry the `\hfill` moves the `\vline` to the left edge of the cell. But that is different than putting it halfway between the two columns, so between the first and second columns there are two vertical rules, with the one from the `{c|cc}` specifier coming before the one produced by the `\vline\hfill`. In contrast, the first row’s third entry shows the usual way to put a vertical bar between two columns. In the second row, the `ghi` is the widest entry in its column so in the `\vline\hfill` the `\hfill` has no effect and the vertical line in that entry appears immediately next to the `g`, with no whitespace.
```
\begin{tabular}{c|cc}
  x   &\vline\hfill y   &\multicolumn{1}{|r}{z} \\ % row 1
  abc &def &\vline\hfill ghi                       % row 2
\end{tabular}

```

#### 8.23.3 `\cline`
Synopsis:
```
\cline{i-j}

```

In an `array` or `tabular` environment, draw a horizontal rule beginning in column i and ending in column j. The dash, `-`, must appear in the mandatory argument. To span a single column use the number twice, as with `\cline{2-2}`.
This example puts two horizontal lines between the first and second rows, one line in the first column only, and the other spanning the third and fourth columns. The two lines are side-by-side, at the same height.
```
\begin{tabular}{llrr}
  a &b &c &d \\ \cline{1-1} \cline{3-4}
  e &f &g &h
\end{tabular}

```

#### 8.23.4 `\hline`
Draw a horizontal line the width of the enclosing `tabular` or `array` environment. It’s most commonly used to draw a line at the top, bottom, and between the rows of a table.
In this example the top of the table has two horizontal rules, one above the other, that span both columns. The bottom of the table has a single rule spanning both columns. Because of the `\hline`, the `tabular` second row’s line ending double backslash `\\` is required.
```
\begin{tabular}{ll} \hline\hline
  Baseball   &Red Sox  \\
  Basketball &Celtics  \\ \hline
\end{tabular}

```

### 8.24 `thebibliography`
Synopsis:
```
\begin{thebibliography}{widest-label}
  \bibitem[label]{cite_key}
  ...
\end{thebibliography}

```

Produce a bibliography or reference list. There are two ways to produce bibliographic lists. This environment is suitable when you have only a few references and can maintain the list by hand. See [Using BibTeX](https://latexref.xyz/dev/latex2e.html#Using-BibTeX), for a more sophisticated approach.
This shows the environment with two entries.
```
This work is based on \cite{latexdps}.
Together they are \cite{latexdps, texbook}.
  ...
\begin{thebibliography}{9}
\bibitem{latexdps}
  Leslie Lamport.
  \textit{\LaTeX{}: a document preparation system}.
  Addison-Wesley, Reading, Massachusetts, 1993.
\bibitem{texbook}
  Donald Ervin Knuth.
  \textit{The \TeX book}.
  Addison-Wesley, Reading, Massachusetts, 1983.
\end{thebibliography}

```

This styles the first reference as ‘[1] Leslie ...’, and so that `... based on \cite{latexdps}` produces the matching ‘... based on [1]’. The second `\cite` produces ‘[1, 2]’. You must compile the document twice to resolve these references.
The mandatory argument widest-label is text that, when typeset, is as wide as the widest item label produced by the `\bibitem` commands. The tradition is to use `9` for bibliographies with less than 10 references, `99` for ones with less than 100, etc.
The bibliographic list is headed by a title such as ‘Bibliography’. To change it there are two cases. In the book and report classes, where the top level sectioning is `\chapter` and the default title is ‘Bibliography’, that title is in the macro `\bibname`. For article, where the class’s top level sectioning is `\section` and the default is ‘References’, the title is in macro `\refname`. Change it by redefining the command, as with `\renewcommand{\refname}{Cited references}`, after `\begin{document}`.
Language support packages such as `babel` will automatically redefine `\refname` or `\bibname` to fit the selected language.
See [`list`](https://latexref.xyz/dev/latex2e.html#list), for the list layout control parameters.
  * [`\bibitem`](https://latexref.xyz/dev/latex2e.html#g_t_005cbibitem)
  * [`\cite`](https://latexref.xyz/dev/latex2e.html#g_t_005ccite)
  * [`\nocite`](https://latexref.xyz/dev/latex2e.html#g_t_005cnocite)
  * [Using BibTeX](https://latexref.xyz/dev/latex2e.html#Using-BibTeX)

#### 8.24.1 `\bibitem`
Synopsis:
```
\bibitem{cite_key}

```

or
```
\bibitem[label]{cite_key}

```

Generate an entry labeled by default by a number generated using the `enumi` counter. The _citation key_ cite_key can be any string of letters, numbers, and punctuation symbols (but not comma).
See [`thebibliography`](https://latexref.xyz/dev/latex2e.html#thebibliography), for an example.
When provided, the optional label becomes the entry label and the `enumi` counter is not incremented. With this
```
\begin{thebibliography}
\bibitem[Lamport 1993]{latexdps}
  Leslie Lamport.
  \textit{\LaTeX{}: a document preparation system}.
  Addison-Wesley, Reading, Massachusetts, 1993.
\bibitem{texbook}
  Donald Ervin Knuth.
  \textit{The \TeX book}.
  Addison-Wesley, Reading, Massachusetts, 1983.
\end{thebibliography}

```

the first entry will be styled as ‘[Lamport 1993] Leslie ...’ (The amount of horizontal space that LaTeX leaves for the label depends on the widest-label argument of the `thebibliography` environment; see [`thebibliography`](https://latexref.xyz/dev/latex2e.html#thebibliography).) Similarly, `... based on \cite{latexdps}` will produce ‘... based on [Lamport 1994]’.
If you mix `\bibitem` entries having a label with those that do not then LaTeX will number the unlabelled ones sequentially. In the example above the `texbook` entry will appear as ‘[1] Donald ...’, despite that it is the second entry.
If you use the same cite_key twice then you get ‘LaTeX Warning: There were multiply-defined labels’.
Under the hood, LaTeX remembers the cite_key and label information because `\bibitem` writes it to the auxiliary file jobname.aux (see [Jobname](https://latexref.xyz/dev/latex2e.html#Jobname)). For instance, the above example causes the two `\bibcite{latexdps}{Lamport, 1993}` and `\bibcite{texbook}{1}` to appear in that file. The .aux file is read by the `\begin{document}` command and then the information is available for `\cite` commands. This explains why you need to run LaTeX twice to resolve references: once to write it out and once to read it in.
Because of this two-pass algorithm, when you add a `\bibitem` or change its cite_key you may get ‘LaTeX Warning: Label(s) may have changed. Rerun to get cross-references right’. Fix it by recompiling.
#### 8.24.2 `\cite`
Synopsis:
```
\cite{keys}

```

or
```
\cite[subcite]{keys}

```

Generate as output a citation to the references associated with keys. The mandatory keys is a citation key, or a comma-separated list of citation keys (see [`\bibitem`](https://latexref.xyz/dev/latex2e.html#g_t_005cbibitem)).
This
```
The ultimate source is \cite{texbook}.
  ...
\begin{thebibliography}
\bibitem{texbook}
  Donald Ervin Knuth.
  \textit{The \TeX book}.
  Addison-Wesley, Reading, Massachusetts, 1983.
\end{thebibliography}

```

produces output like ‘... source is [1]’. You can change the appearance of the citation and of the reference by using bibliography styles if you generate automatically the `thebibliography` environment. More information in [Using BibTeX](https://latexref.xyz/dev/latex2e.html#Using-BibTeX).
The optional argument subcite is appended to the citation. For example, `See 14.3 in \cite[p.~314]{texbook}` might produce ‘See 14.3 in [1, p. 314]’.
In addition to what appears in the output, `\cite` writes information to the auxiliary file jobname.aux (see [Jobname](https://latexref.xyz/dev/latex2e.html#Jobname)). For instance, `\cite{latexdps}` writes ‘\citation{latexdps}’ to that file. This information is used by BibTeX to include in your reference list only those works that you have actually cited; see [`\nocite`](https://latexref.xyz/dev/latex2e.html#g_t_005cnocite) also.
If keys is not in your bibliography information then you get ‘LaTeX Warning: There were undefined references’, and in the output the citation shows as a boldface question mark between square brackets. There are two possible causes. If you have mistyped something, as in `\cite{texbok}` then you need to correct the spelling. On the other hand, if you have just added or modified the bibliographic information and so changed the .aux file (see [`\bibitem`](https://latexref.xyz/dev/latex2e.html#g_t_005cbibitem)) then the fix may be to run LaTeX again.
#### 8.24.3 `\nocite`
Synopsis:
```
\nocite{keys}

```

Produces no output but writes keys to the auxiliary file jobname.aux (see [Jobname](https://latexref.xyz/dev/latex2e.html#Jobname)).
The mandatory argument keys is a comma-separated list of one or more citation keys (see [`\bibitem`](https://latexref.xyz/dev/latex2e.html#g_t_005cbibitem)). This information is used by BibTeX to include these works in your reference list even though you have not explicitly cited them (see [`\cite`](https://latexref.xyz/dev/latex2e.html#g_t_005ccite)).
#### 8.24.4 Using BibTeX
As described in `thebibliography` (see [`thebibliography`](https://latexref.xyz/dev/latex2e.html#thebibliography)), a sophisticated approach to managing bibliographies is provided by the BibTeX program. This is only an introduction; see the full documentation on CTAN (see [CTAN: The Comprehensive TeX Archive Network](https://latexref.xyz/dev/latex2e.html#CTAN)).
With BibTeX, you don’t use the `thebibliography` environment directly (see [`thebibliography`](https://latexref.xyz/dev/latex2e.html#thebibliography)). Instead, include these lines:
```
\bibliographystyle{bibstyle}
\bibliography{bibfile1, bibfile2, ...}

```

The bibstyle refers to a file bibstyle.bst, which defines how your citations will look. The standard bibstyle’s distributed with BibTeX are:

`alpha`

Labels are formed from name of author and year of publication. The bibliographic items are sorted alphabetically.

`plain`

Labels are integers. Sort the bibliographic items alphabetically.

`unsrt`

Like `plain`, but entries are in order of citation.

`abbrv`

Like `plain`, but more compact labels.
Many, many other BibTeX style files exist, tailored to the demands of various publications. See the CTAN topic <https://ctan.org/topic/bibtex-sty>.
The `\bibliography` command is what actually produces the bibliography. Its argument is a comma-separated list, referring to files named bibfile1.bib, bibfile2.bib, … These contain your database in BibTeX format. This shows a typical couple of entries in that format.
```
@book{texbook,
  title     = {The {{\TeX}}book},
  author    = {D.E. Knuth},
  isbn      = {0201134489},
  series    = {Computers \& typesetting},
  year      = {1983},
  publisher = {Addison-Wesley}
}
@book{sexbook,
    author    = {W.H. Masters and V.E. Johnson},
    title     = {Human Sexual Response},
    year      = {1966},
    publisher = {Bantam Books}
}

```

Only the bibliographic entries referred to via `\cite` and `\nocite` will be listed in the document’s bibliography. Thus you can keep all your sources together in one file, or a small number of files, and rely on BibTeX to include in this document only those that you used.
With BibTeX, the keys argument to `\nocite` can also be the single character ‘*’. This means to implicitly cite all entries from all given bibliographies.
  * [BibTeX error messages](https://latexref.xyz/dev/latex2e.html#BibTeX-error-messages)

#### 8.24.4.1 BibTeX error messages
If you forget to use `\bibliography` or `\bibliographystyle` in your document (or, less likely, any `\cite` or `\nocite` command), BibTeX will issue an error message. Because BibTeX can be used with any program, not just LaTeX, the error messages refer to the internal commands read by BibTeX (from the .aux file), rather than the user-level commands described above.
Here is a table showing internal commands mentioned in the BibTeX errors, and the corresponding user-level commands.

`\bibdata`

`\bibliography`

`\bibstyle`

`\bibliographystyle`

`\citation`

`\cite`, `\nocite`
For example, if your document has no `\bibliographystyle` command, BibTeX complains as follows:
```
I found no \bibstyle command---while reading file document.aux

```
