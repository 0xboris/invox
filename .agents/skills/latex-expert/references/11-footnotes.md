# 11 Footnotes

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 11.1 `\footnote`
- 11.2 `\footnotemark`
- 11.3 `\footnotetext`
- 11.4 Footnotes in section headings
- 11.5 Footnotes in a table
- 11.6 Footnotes of footnotes

## 11 Footnotes
Place a footnote at the bottom of the current page, as here.
```
Noël Coward quipped that having to read a footnote is like having
to go downstairs to answer the door, while in the midst of making
love.\footnote{%
  I wouldn't know, I don't read footnotes.}

```

You can put multiple footnotes on a page. If the footnote text becomes too long then it will flow to the next page.
You can also produce footnotes by combining the `\footnotemark` and the `\footnotetext` commands, which is useful in special circumstances.
To make bibliographic references come out as footnotes you need to include a bibliographic style with that behavior (see [Using BibTeX](https://latexref.xyz/dev/latex2e.html#Using-BibTeX)).
  * [`\footnote`](https://latexref.xyz/dev/latex2e.html#g_t_005cfootnote)
  * [`\footnotemark`](https://latexref.xyz/dev/latex2e.html#g_t_005cfootnotemark)
  * [`\footnotetext`](https://latexref.xyz/dev/latex2e.html#g_t_005cfootnotetext)
  * [Footnotes in section headings](https://latexref.xyz/dev/latex2e.html#Footnotes-in-section-headings)
  * [Footnotes in a table](https://latexref.xyz/dev/latex2e.html#Footnotes-in-a-table)
  * [Footnotes of footnotes](https://latexref.xyz/dev/latex2e.html#Footnotes-of-footnotes)

### 11.1 `\footnote`
Synopsis, one of:
```
\footnote{text}
\footnote[number]{text}

```

Place a footnote text at the bottom of the current page, with a footnote marker at the current position in the text.
```
There are over a thousand footnotes in Gibbon's
\textit{Decline and Fall of the Roman Empire}.\footnote{%
  After reading an early version with endnotes David Hume complained,
  ``One is also plagued with his Notes, according to the present Method
  of printing the Book'' and suggested that they ``only to be printed
  at the Margin or the Bottom of the Page.''}

```

The optional argument number allows you to specify the number of the footnote. If you use this then LaTeX does not increment the `footnote` counter.
By default, LaTeX uses arabic numbers as footnote markers. Change this with something like `\renewcommand{\thefootnote}{\fnsymbol{footnote}}`, which uses a sequence of symbols (see [`\alph \Alph \arabic \roman \Roman \fnsymbol`: Printing counters](https://latexref.xyz/dev/latex2e.html#g_t_005calph-_005cAlph-_005carabic-_005croman-_005cRoman-_005cfnsymbol)). To make this change global put that in the preamble. If you make the change local then you may want to reset the counter with `\setcounter{footnote}{0}`.
LaTeX determines the spacing of footnotes with two parameters.

`\footnoterule`

Produces the rule separating the main text on a page from the page’s footnotes. Default dimensions in the standard document classes (except `slides`, where it does not appear) are: vertical thickness of `0.4pt`, and horizontal size of `0.4\columnwidth` long. Change the rule with something like this.
```
% \footnoterule is expanded in vertical mode, thus \kern
% commands ensure that no vertical space is created,
% and the rule is separated vertically with 2pt
% above the note text.
\renewcommand*{\footnoterule}{%
  \kern -3pt                         % This -3 is negative
  \hrule width \textwidth height 1pt % of the sum of this 1
  \kern 2pt}                         % and this 2

```

`\footnotesep`

The height of the strut placed at the beginning of the footnote (see [`\strut`](https://latexref.xyz/dev/latex2e.html#g_t_005cstrut)). By default, this is set to the normal strut for `\footnotesize` fonts (see [Font sizes](https://latexref.xyz/dev/latex2e.html#Font-sizes)), therefore there is no extra space between footnotes. This is ‘6.65pt’ for ‘10pt’, ‘7.7pt’ for ‘11pt’, and ‘8.4pt’ for ‘12pt’. Change it as with `\setlength{\footnotesep}{11pt}`.
The `\footnote` command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
LaTeX’s default puts many restrictions on where you can use a `\footnote`; for instance, you cannot use it in an argument to a sectioning command such as `\chapter` (it can only be used in outer paragraph mode; see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). There are some workarounds; see following sections.
In a `minipage` environment the `\footnote` command uses the `mpfootnote` counter instead of the `footnote` counter, so they are numbered independently. They are shown at the bottom of the environment, not at the bottom of the page. And by default they are shown alphabetically. See [`minipage`](https://latexref.xyz/dev/latex2e.html#minipage) and [Footnotes in a table](https://latexref.xyz/dev/latex2e.html#Footnotes-in-a-table).
### 11.2 `\footnotemark`
Synopsis, one of:
```
\footnotemark
\footnotemark[number]

```

Put the current footnote mark in the text. To specify associated text for the footnote see [`\footnotetext`](https://latexref.xyz/dev/latex2e.html#g_t_005cfootnotetext). The optional argument number causes the command to use that number to determine the footnote mark. This command can be used in inner paragraph mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)).
If you use `\footnotemark` without the optional argument then it increments the `footnote` counter, but if you use the optional number then it does not. The next example produces several consecutive footnote markers referring to the same footnote.
```
The first theorem\footnote{Due to Gauss.}
and the second theorem\footnotemark[\value{footnote}]
and the third theorem.\footnotemark[\value{footnote}]

```

If there are intervening footnotes then you must remember the value of the number of the common mark. This example gives the same institutional affiliation to both the first and third authors (`\thanks` is a version of `\footnote`), by explicitly specifying the number of the footnote (‘1’).
```
\title{A Treatise on the Binomial Theorem}
\author{J Moriarty\thanks{University of Leeds}
  \and A C Doyle\thanks{Durham University}
  \and S Holmes\footnotemark[1]}
\begin{document}
\maketitle

```

This example accomplishes the same by using the package `cleveref`.
```
\usepackage{cleveref}[2012/02/15]   % in preamble
\crefformat{footnote}{#2\footnotemark[#1]#3}
...
The theorem is from Evers.\footnote{\label{fn:TE}Tinker, Evers, 1994.}
The corollary is from Chance.\footnote{Evers, Chance, 1990.}
But the key lemma is from Tinker.\cref{fn:TE}

```

It will work with the package `hyperref`.
This uses a counter to remember the footnote number. The third sentence is followed by the same footnote marker as the first.
```
\newcounter{footnoteValueSaver}
All babies are illogical.\footnote{%
  Lewis Carroll.}\setcounter{footnoteValueSaver}{\value{footnote}}
Nobody is despised who can manage a crocodile.\footnote{%
  Captain Hook.}
Illogical persons are despised.\footnotemark[\value{footnoteValueSaver}]
Therefore, anyone who can manage a crocodile is not a baby.

```

### 11.3 `\footnotetext`
Synopsis, one of:
```
\footnotetext{text}
\footnotetext[number]{text}

```

Place text at the bottom of the page as a footnote. It pairs with `\footnotemark` (see [`\footnotemark`](https://latexref.xyz/dev/latex2e.html#g_t_005cfootnotemark)) and can come anywhere after that command, but must appear in outer paragraph mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). The optional argument number changes the number of the footnote mark.
See [`\footnotemark`](https://latexref.xyz/dev/latex2e.html#g_t_005cfootnotemark) and [Footnotes in a table](https://latexref.xyz/dev/latex2e.html#Footnotes-in-a-table), for usage examples.
### 11.4 Footnotes in section headings
Putting a footnote in a section heading, as in:
```
\section{Full sets\protect\footnote{This material due to ...}}

```

causes the footnote to appear at the bottom of the page where the section starts, as usual, but also at the bottom of the table of contents, where it is not likely to be desired. The simplest way to have it not appear on the table of contents is to use the optional argument to `\section`.
```
\section[Please]{Please\footnote{%
  Don't footnote in chapter and section headers!}}

```

No `\protect` is needed in front of `\footnote` here because what gets moved to the table of contents is the optional argument.
### 11.5 Footnotes in a table
Inside a `tabular` or `array` environment the `\footnote` command does not work; there is a footnote mark in the table cell but the footnote text does not appear. The solution is to use a `minipage` environment as here (see [`minipage`](https://latexref.xyz/dev/latex2e.html#minipage)).
```
\begin{center}
  \begin{minipage}{\textwidth} \centering
     \begin{tabular}{l|l}
       \textsc{Ship}           &\textsc{Book} \\ \hline
       \textit{HMS Sophie}     &Master and Commander  \\
       \textit{HMS Polychrest} &Post Captain  \\
       \textit{HMS Lively}     &Post Captain \\
       \textit{HMS Surprise}   &A number of books\footnote{%
                                  Starting with \textit{HMS Surprise}.}
     \end{tabular}
  \end{minipage}
\end{center}

```

Inside a `minipage`, footnote marks are lowercase letters. Change that with something like `\renewcommand{\thempfootnote}{\arabic{mpfootnote}}` (see [`\alph \Alph \arabic \roman \Roman \fnsymbol`: Printing counters](https://latexref.xyz/dev/latex2e.html#g_t_005calph-_005cAlph-_005carabic-_005croman-_005cRoman-_005cfnsymbol)).
The footnotes in the prior example appear at the bottom of the `minipage`. To have them appear at the bottom of the main page, as part of the regular footnote sequence, use the `\footnotemark` and `\footnotetext` pair and make a new counter.
```
\newcounter{mpFootnoteValueSaver}
\begin{center}
  \begin{minipage}{\textwidth}
    \setcounter{mpFootnoteValueSaver}{\value{footnote}} \centering
     \begin{tabular}{l|l}
       \textsc{Woman}             &\textsc{Relationship} \\ \hline
       Mona                       &Attached\footnotemark  \\
       Diana Villiers             &Eventual wife  \\
       Christine Hatherleigh Wood &Fiance\footnotemark
     \end{tabular}
  \end{minipage}%  percent sign keeps footnote text close to minipage
  \stepcounter{mpFootnoteValueSaver}%
    \footnotetext[\value{mpFootnoteValueSaver}]{%
      Little is known other than her death.}%
  \stepcounter{mpFootnoteValueSaver}%
    \footnotetext[\value{mpFootnoteValueSaver}]{%
      Relationship is unresolved.}
\end{center}

```

For a floating `table` environment (see [`table`](https://latexref.xyz/dev/latex2e.html#table)), use the `tablefootnote` package.
```
\usepackage{tablefootnote}  % in preamble
   ...
\begin{table}
  \centering
     \begin{tabular}{l|l}
     \textsc{Date}  &\textsc{Campaign} \\ \hline
     1862           &Fort Donelson \\
     1863           &Vicksburg     \\
     1865           &Army of Northern Virginia\tablefootnote{%
                      Ending the war.}
     \end{tabular}
    \caption{Forces captured by US Grant}
\end{table}

```

The footnote appears at the page bottom and is numbered in sequence with other footnotes.
### 11.6 Footnotes of footnotes
Particularly in the humanities, authors can have multiple classes of footnotes, including having footnotes of footnotes. The package `bigfoot` extends LaTeX’s default footnote mechanism in many ways, including allow these two, as in this example.
```
\usepackage{bigfoot}    % in preamble
\DeclareNewFootnote{Default}
\DeclareNewFootnote{from}[alph]   % create class \footnotefrom{}
 ...
The third theorem is a partial converse of the
second.\footnotefrom{%
  Noted in Wilson.\footnote{Second edition only.}}

```
