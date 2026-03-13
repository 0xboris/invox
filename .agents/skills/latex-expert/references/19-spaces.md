# 19 Spaces

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 19.1 `\enspace` & `\quad` & `\qquad`
- 19.2 `\hspace`
- 19.3 `\hfill`
- 19.4 `\hss`
- 19.5 `\spacefactor`
- 19.6 Backslash-space, `\ `
- 19.7 `~`, `\nobreakspace`
- 19.8 `\thinspace` & `\negthinspace`
- 19.9 `\/`
- 19.10 `\hrulefill` & `\dotfill`
- 19.11 `\bigskip` & `\medskip` & `\smallskip`
- 19.12 `\bigbreak` & `\medbreak` & `\smallbreak`
- 19.13 `\strut`
- 19.14 `\vspace`
- 19.15 `\vfill`
- 19.16 `\addvspace`

## 19 Spaces
LaTeX has many ways to produce white space, or filled space. Some of these are best suited to mathematical text; for these see [Spacing in math mode](https://latexref.xyz/dev/latex2e.html#Spacing-in-math-mode).
  * [`\enspace` & `\quad` & `\qquad`](https://latexref.xyz/dev/latex2e.html#g_t_005censpace-_0026-_005cquad-_0026-_005cqquad)
  * [`\hspace`](https://latexref.xyz/dev/latex2e.html#g_t_005chspace)
  * [`\hfill`](https://latexref.xyz/dev/latex2e.html#g_t_005chfill)
  * [`\hss`](https://latexref.xyz/dev/latex2e.html#g_t_005chss)
  * [`\spacefactor`](https://latexref.xyz/dev/latex2e.html#g_t_005cspacefactor)
  * [Backslash-space, `\ `](https://latexref.xyz/dev/latex2e.html#g_t_005c_0028SPACE_0029)
  * [`~`, `\nobreakspace`](https://latexref.xyz/dev/latex2e.html#g_t_007e)
  * [`\thinspace` & `\negthinspace`](https://latexref.xyz/dev/latex2e.html#g_t_005cthinspace-_0026-_005cnegthinspace)
  * [`\/`](https://latexref.xyz/dev/latex2e.html#g_t_005c_002f)
  * [`\hrulefill` & `\dotfill`](https://latexref.xyz/dev/latex2e.html#g_t_005chrulefill-_0026-_005cdotfill)
  * [`\bigskip` & `\medskip` & `\smallskip`](https://latexref.xyz/dev/latex2e.html#g_t_005cbigskip-_0026-_005cmedskip-_0026-_005csmallskip)
  * [`\bigbreak` & `\medbreak` & `\smallbreak`](https://latexref.xyz/dev/latex2e.html#g_t_005cbigbreak-_0026-_005cmedbreak-_0026-_005csmallbreak)
  * [`\strut`](https://latexref.xyz/dev/latex2e.html#g_t_005cstrut)
  * [`\vspace`](https://latexref.xyz/dev/latex2e.html#g_t_005cvspace)
  * [`\vfill`](https://latexref.xyz/dev/latex2e.html#g_t_005cvfill)
  * [`\addvspace`](https://latexref.xyz/dev/latex2e.html#g_t_005caddvspace)

### 19.1 `\enspace` & `\quad` & `\qquad`
Synopsis, one of:
```
\enspace
\quad
\qquad

```

Insert a horizontal space of 1/2em, 1em, or 2em. The em is a length defined by a font designer, often thought of as being the width of a capital M. One advantage of describing space in ems is that it can be more portable across documents than an absolute measurement such as points (see [Lengths/em](https://latexref.xyz/dev/latex2e.html#Lengths_002fem)).
This puts a suitable gap between two graphics.
```
\begin{center}
  \includegraphics{womensmile.png}%
  \qquad\includegraphics{mensmile.png}
\end{center}

```

See [Spacing in math mode](https://latexref.xyz/dev/latex2e.html#Spacing-in-math-mode), for `\quad` and `\qquad`. These are lengths from centuries of typesetting and so may be a better choice in many circumstances than arbitrary lengths, such as you get with `\hspace`.
### 19.2 `\hspace`
Synopsis, one of:
```
\hspace{length}
\hspace*{length}

```

Insert the amount length of horizontal space. The length can be positive, negative, or zero; adding a negative amount of space is like backspacing. It is a rubber length, that is, it may contain a `plus` or `minus` component, or both (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). Because the space is stretchable and shrinkable, it is sometimes called _glue_.
This makes a line with ‘Name:’ an inch from the right margin.
```
\noindent\makebox[\linewidth][r]{Name:\hspace{1in}}

```

The `*`-form inserts horizontal space that is non-discardable. More precisely, when TeX breaks a paragraph into lines any white space—glues and kerns—that come at a line break are discarded. The `*`-form avoids that (technically, it adds a non-discardable invisible item in front of the space).
In this example
```
\parbox{0.8\linewidth}{%
  Fill in each blank: Four \hspace*{1in} and seven years ago our
  fathers brought forth on this continent, a new \hspace*{1in},
  conceived in \hspace*{1in}, and dedicated to the proposition
  that all men are created \hspace*{1in}.}

```

the 1 inch blank following ‘conceived in’ falls at the start of a line. If you erase the `*` then LaTeX discards the blank.
Here, the `\hspace` separates the three graphics.
```
\begin{center}
  \includegraphics{lion.png}%   comment keeps out extra space
  \hspace{1cm minus 0.25cm}\includegraphics{tiger.png}%
  \hspace{1cm minus 0.25cm}\includegraphics{bear.png}
\end{center}

```

Because the argument to each `\hspace` has `minus 0.25cm`, each can shrink a little if the three figures are too wide. But each space won’t shrink more than 0.25cm (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)).
### 19.3 `\hfill`
Synopsis:
```
\hfill

```

Produce a rubber length which has no natural space but that can stretch horizontally as far as needed (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)).
This creates a one-line paragraph with ‘Name:’ on the left side of the page and ‘Quiz One’ on the right.
```
\noindent Name:\hfill Quiz One

```

The `\hfill` command is equivalent to `\hspace{\fill}` and so the space can be discarded at line breaks. To avoid that instead use `\hspace*{\fill}` (see [`\hspace`](https://latexref.xyz/dev/latex2e.html#g_t_005chspace)).
Here the graphs are evenly spaced in the middle of the figure.
```
\newcommand*{\vcenteredhbox}[1]{\begin{tabular}{@{}c@{}}#1\end{tabular}}
  ...
\begin{figure}
  \hspace*{\fill}%
  \vcenteredhbox{\includegraphics{graph0.png}}%
    \hfill\vcenteredhbox{\includegraphics{graph1.png}}%
  \hspace*{\fill}%
  \caption{Comparison of two graphs} \label{fig:twographs}
\end{figure}

```

Note the `\hspace*`’s where the space could otherwise be dropped.
### 19.4 `\hss`
Synopsis:
```
\hss

```

Produce a horizontal space that is infinitely shrinkable as well as infinitely stretchable (this command is a TeX primitive). LaTeX authors should reach first for the `\makebox` command to get the effects of `\hss` (see [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox)).
Here, the first line’s `\hss` makes the Z stick out to the right, overwriting the Y. In the second line the Z sticks out to the left, overwriting the X.
```
X\hbox to 0pt{Z\hss}Y
X\hbox to 0pt{\hss Z}Y

```

Without the `\hss` you get something like ‘Overfull \hbox (6.11111pt too wide) detected at line 20’.
### 19.5 `\spacefactor`
Synopsis:
```
\spacefactor=integer

```

Influence LaTeX’s stretching and shrinking of glue. Few user-level documents need to use this.
While LaTeX is laying out the material, it may stretch or shrink the gaps between words. (This space is not a character; it is called the _interword glue_ ; see [`\hspace`](https://latexref.xyz/dev/latex2e.html#g_t_005chspace)). The `\spacefactor` parameter (a TeX primitive) allows you to, for instance, have the space after a period stretch more than the space after a word-ending letter.
After LaTeX places each character, or rule or other box, it sets a parameter called the _space factor_. If the next thing in the input is a space then this parameter affects how much stretching or shrinking can happen. A space factor that is larger than the normal value means that the glue can stretch more and shrink less. Normally, the space factor is 1000. This value is in effect following most characters, and any non-character box or math formula. But it is 3000 after a period, exclamation mark, or question mark, 2000 after a colon, 1500 after a semicolon, 1250 after a comma, and 0 after a right parenthesis or bracket, or closing double quote or single quote. Finally, it is 999 after a capital letter.
If the space factor f is 1000 then the glue gap will be the font’s normal space value (for Computer Modern Roman 10 point this is 3.3333pt). Otherwise, if the space factor f is greater than 2000 then TeX adds the font’s extra space value (for Computer Modern Roman 10 point this is 1.11111pt), and then the font’s normal stretch value is multiplied by _f /1000_ and the normal shrink value is multiplied by _1000/f_ (for Computer Modern Roman 10 point these are 1.66666 and 1.11111pt).
For example, consider the period ending ‘A man's best friend is his dog.’. After it, TeX puts in a fixed extra space, and also allows the glue to stretch 3 times as much and shrink 1/3 as much, as the glue after `friend` or any of the other words, since they are not followed by punctuation.
The rules for space factors are even more complex because they play additional roles. In practice, there are two consequences. First, if a period or other punctuation is followed by a right parenthesis or bracket, or right single or double quote then the spacing effect of that period carries through those characters (that is, the following glue will have increased stretch and shrink). Second, if punctuation comes after a capital letter then the normal effect of the period is does not happen, so you get an ordinary space. This second case also affects abbreviations that do not end in a capital letter (see [`\@`](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040)).
You can only use `\spacefactor` in paragraph mode or LR mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). You can see the current value with `\the\spacefactor` or `\showthe\spacefactor`.
Finally, not especially related to `\spacefactor` itself: if you get errors like ‘You can't use `\spacefactor' in vertical mode’, or ‘You can't use `\spacefactor' in math mode.’, or ‘Improper \spacefactor’ then you have probably tried to redefine an internal command. See [`\makeatletter` & `\makeatother`](https://latexref.xyz/dev/latex2e.html#g_t_005cmakeatletter-_0026-_005cmakeatother).
  * [`\@`](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040)
  * [`\frenchspacing` & `\nonfrenchspacing`](https://latexref.xyz/dev/latex2e.html#g_t_005cfrenchspacing-_0026-_005cnonfrenchspacing)
  * [`\normalsfcodes`](https://latexref.xyz/dev/latex2e.html#g_t_005cnormalsfcodes)

#### 19.5.1 `\@`
Synopsis:
```
capital-letter\@.

```

Treat a following period (or other punctuation) as sentence-ending. By default, LaTeX thinks that a period ends an abbreviation if the period comes after a capital letter, and otherwise thinks the period ends the sentence.
This example shows the two cases to remember.
```
The songs \textit{Red Guitar}, etc.\ are by Loudon Wainwright~III\@.

```

The first period ends the abbreviation ‘etc.’ but not the sentence. The backslash-space, `\ `, produces a mid-sentence space. The second period ends the sentence, despite it being preceded by a capital letter. We tell LaTeX that it ends the sentence by putting `\@` before it.
So: if you have a capital letter followed by a period that ends the sentence, then put `\@` before the period. This holds even if there is an intervening right parenthesis or bracket, or right single or double quote, because the spacing effect of that period carries through those characters. For example, this
```
Use the \textit{Instructional Practices Guide},
(a book by the MAA)\@.

```

will have correct inter-sentence spacing after the period.
The `\@` command is only for text modes. If you use it outside of a text mode then you get the error ‘You can't use `\spacefactor' in vertical mode’ (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)).
All the above applies equally to question marks and exclamation points as periods, since all are sentence-ending punctuation, and LaTeX increases the space after each in the same way, when they end a sentence. LaTeX also increases spacing after colon, semicolon, and comma characters (see [`\spacefactor`](https://latexref.xyz/dev/latex2e.html#g_t_005cspacefactor)).
In contrast: the converse case is a period (or other punctuation) that does not end a sentence. For that case, follow the period with a backslash-space, (`\ `), or a tie, (`~`), or `\@`. Examples are `Nat.\ Acad.\ Science`, and `Mr.~Bean`, and `(manure, etc.\@) for sale` (note in the last one that the `\@` comes after the period but before the closing parenthesis).
#### 19.5.2 `\frenchspacing` & `\nonfrenchspacing`
Synopsis, one of:
```
\frenchspacing
\nonfrenchspacing

```

`\frenchspacing` causes LaTeX to make spacing after all punctuation, including periods, be the same as the space between words in the middle of a sentence. `\nonfrenchspacing` switches back to the default handling in which spacing after most punctuation stretches or shrinks differently than a word space (see [`\spacefactor`](https://latexref.xyz/dev/latex2e.html#g_t_005cspacefactor)).
In American English, the typesetting tradition is to adjust, typically increasing, the space after punctuation more than the space between words that are in the middle of a sentence. Declaring `\frenchspacing` (the command is inherited from plain TeX) switches to the tradition that all spaces are treated equally.
If your LaTeX document specifies the language being used, for example with the `babel` package, the necessary settings should be taken care of for you.
#### 19.5.3 `\normalsfcodes`
Synopsis:
```
\normalsfcodes

```

Reset the LaTeX space factors to the default values (see [`\spacefactor`](https://latexref.xyz/dev/latex2e.html#g_t_005cspacefactor)).
### 19.6 Backslash-space, `\ `
This section refers to the command consisting of two characters, a backslash followed by a space. Synopsis:
```
\

```

Produce a space. By default it produces white space of length 3.33333pt plus 1.66666pt minus 1.11111pt.
When you type one or more blanks between words, LaTeX produces whitespace that is different than an explicit space. This illustrates:
```
\begin{tabular}{rl}
One blank:& makes some space \\
Three blanks:&   in a row \\
Three spaces:&\ \ \ in a row \\
\end{tabular}

```

On the first line LaTeX puts some space after the colon. On the second line LaTeX collapses the three blanks to output one whitespace, so you end with the same space after the colon as in the first line. LaTeX would similarly collapse them to a single whitespace if one, two or all of the three blanks were replaced by a tab, or by a newline. However, the bottom line asks for three spaces so the white area is wider. That is, the backslash-space command creates a fixed amount of horizontal space. (Note that you can define a horizontal space of any width at all with `\hspace`; see [`\hspace`](https://latexref.xyz/dev/latex2e.html#g_t_005chspace).)
The backslash-space command has two main uses. It is often used after control sequences to keep them from gobbling the blank that follows, as after `\TeX` in `\TeX\ (or \LaTeX)`. (But using curly braces has the advantage of still working whether the next character is a blank or any other non-letter, as in `\TeX{} (or \LaTeX{})` in which `{}` can be added after `\LaTeX` as well as after `\TeX`.) The other common use is that it marks a period as ending an abbreviation instead of ending a sentence, as in `Prof.\ Smith` or `Jones et al.\ (1993)` (see [`\@`](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040)).
Under normal circumstances, `\``TAB` and `\``NEWLINE` are equivalent to backslash-space, `\ `.
In order to allow source code indentation, under normal circumstances, TeX ignores leading blanks in a line. So the following prints ‘one word’:
```
one
 word

```

where the white space between ‘one’ and ‘word’ is produced by the newline after ‘one’, not by the space before ‘word’.
### 19.7 `~`, `\nobreakspace`
Synopsis:
```
before~after

```

The _tie_ character, `~`, produces a space between before and after at which the line will not be broken. By default the white space has length 3.33333pt plus 1.66666pt minus 1.11111pt (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). The command `\nobreakspace` and the Unicode input character U+00A0 (also in many 8-bit encodings) are synonyms.
Note that the word ‘tie’ has this meaning in the TeX/Texinfo community; this differs from the typographic term “tie”, which is a diacritic in the shape of an arc, called a “tie-after” accent in The TeXbook.
Here LaTeX will not break the line between the final two words:
```
Thanks to Prof.~Lerman.

```

In addition, despite the period, LaTeX does not use the end-of-sentence spacing (see [`\@`](https://latexref.xyz/dev/latex2e.html#g_t_005c_0040)).
Ties prevent a line break where that could cause confusion. They also still allow hyphenation (of either of the tied words), so they are generally preferable to putting consecutive words in an `\mbox` (see [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox)).
Exactly where ties should be used is something of a matter of taste, sometimes alarmingly dogmatic taste, among readers. Nevertheless, here are some usage models, many of them from The TeXbook.
  * Between an enumerator label and number, such as in references: `Chapter~12`, or `Theorem~\ref{th:Wilsons}`, or `Figure~\ref{fig:KGraph}`.
  * When cases are enumerated inline: `(b)~Show that $f(x)$ is (1)~continuous, and (2)~bounded`.
  * Between a number and its unit: `$745.7.8$~watts` (the `siunitx` package has a special facility for this) or `144~eggs`. This includes between a month and day number in a date: `October~12` or `12~Oct`. In general, in any expressions where numbers and abbreviations or symbols are separated by a space: `AD~565`, or `2:50~pm`, or `Boeing~747`, or `268~Plains Road`, or `\$$1.4$~billion`. Other common choices here are a thin space (see [`\thinspace` & `\negthinspace`](https://latexref.xyz/dev/latex2e.html#g_t_005cthinspace-_0026-_005cnegthinspace)) and no space at all.
  * When mathematical phrases are rendered in words: `equals~$n$`, or `less than~$\epsilon$`, or `given~$X$`, or `modulo~$p^e$ for all large~$n$` (but compare `is~$15$` with `is $15$~times the height`). Between mathematical symbols in apposition with nouns: `dimension~$d$` or `function~$f(x)$` (but compare `with length $l$~or more`). When a symbol is a tightly bound object of a preposition: `of~$x$`, or `from $0$ to~$1$`, or `in common with~$m$`.
  * Between symbols in series: `$1$,~$2$, or~$3$` or `$1$,~$2$, \ldots,~$n$`.
  * Between a person’s given names and between multiple surnames: `Donald~E. Knuth`, or `Luis~I. Trabb~Pardo`, or `Charles~XII`—but you must give TeX places to break the line so you might do `Charles Louis Xavier~Joseph de~la Vall\'ee~Poussin`.

### 19.8 `\thinspace` & `\negthinspace`
Synopsis, one of:
```
\thinspace
\negthinspace

```

These produce unbreakable and unstretchable spaces of 1/6em and -1/6em, respectively. These are the text mode equivalents of `\,` and `\!` (see [Spacing in math mode/\thinspace](https://latexref.xyz/dev/latex2e.html#Spacing-in-math-mode_002f_005cthinspace)).
You can use `\,` as a synonym for `\thinspace` in text mode.
One common use of `\thinspace` is as the space between nested quotes:
```
Killick replied, ``I heard the Captain say, `Ahoy there.'\thinspace''

```

Another use is that some style guides call for a `\thinspace` between an ellipsis and a sentence ending period (other style guides, think the three dots and/or four dots are plenty). Another style-specific use is between initials, as in `D.\thinspace E.\ Knuth`.
LaTeX provides a variety of similar spacing commands for math mode (see [Spacing in math mode](https://latexref.xyz/dev/latex2e.html#Spacing-in-math-mode)). With the `amsmath` package, or as of the 2020-10-01 LaTeX release, they can be used in text mode as well as math mode, including `\!` for `\negthinspace`; but otherwise, they are available only in math mode.
### 19.9 `\/`
Synopsis:
```
before-character\/after-character

```

Insert an _italic correction_ , a small space defined by the font designer for each character (possibly zero), to avoid the character colliding with whatever follows. When you use `\/`, LaTeX takes the correction from the font metric file, scales it by any scaling that has been applied to the font, and then inserts that much horizontal space.
Here, were it not for the `\/`, the before-character italic f would hit the after-character roman H
```
\newcommand{\companylogo}{{\it f}\/H}

```

because the italic letter f leans far to the right.
If after-character is a period or comma then don’t insert an italic correction since those punctuation symbols are so low to the baseline already. However, with semicolons or colons, as well as with normal letters, the italic correction can help. It is typically used between a switch from italic or slanted fonts to an upright font.
When you use commands such as `\emph` and `\textit` and `\textsl` to change fonts, LaTeX automatically inserts the italic correction when needed (see [Font styles](https://latexref.xyz/dev/latex2e.html#Font-styles)). However, declarations such as `\em` and `\itshape` and `\slshape` do not automatically insert italic corrections.
Upright characters can also have an italic correction. An example where this is needed is the name `pdf\/\TeX`. However, most upright characters have a zero italic correction. Some font creators do not include italic correction values even for italic fonts.
Technically, LaTeX uses another font-specific value, the so-called _slant parameter_ (namely `\fontdimen1`), to determine whether to possibly insert an italic correction, rather than tying the action to particular font commands.
There is no concept of italic correction in math mode; math spacing is done in a different way.
### 19.10 `\hrulefill` & `\dotfill`
Synopsis, one of:
```
\hrulefill
\dotfill

```

Produce an infinite horizontal rubber length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)) that LaTeX fills with a rule (that is, a line) or with dots, instead of white space.
This outputs a line 2 inches long.
```
Name:~\makebox[2in]{\hrulefill}

```

This example, when placed between blank lines, creates a paragraph that is left and right justified and where the middle is filled with evenly spaced dots.
```
\noindent John Aubrey, RN \dotfill{} Melbury Lodge

```

To make the rule or dots go to the line’s end use `\null` at the start or end.
To change the rule’s thickness, copy the definition and adjust it, as here
```
\renewcommand{\hrulefill}{%
  \leavevmode\leaders\hrule height 1pt\hfill\kern0pt }

```

which changes the default thickness of 0.4pt to 1pt. Similarly, adjust the dot spacing as with
```
\renewcommand{\dotfill}{%
  \leavevmode\cleaders\hbox to 1.00em{\hss .\hss }\hfill\kern0pt }

```

which changes the default length of 0.33em to 1.00em.
This example produces a line for a signature.
```
\begin{minipage}{4cm}
  \centering
  \hrulefill\\
  Signed
\end{minipage}

```

The line is 4cm long.
### 19.11 `\bigskip` & `\medskip` & `\smallskip`
Synopsis, one of:
```
\bigskip
\medskip
\smallskip

```

Produce an amount of vertical space, large or medium-sized or small. These commands are fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
Here the skip suggests the passage of time (from _The Golden Ocean_ by O’Brian).
```
Mr Saumarez would have something rude to say to him, no doubt: he
was at home again, and it was delightful.

\bigskip
``A hundred and fifty-seven miles and one third, in twenty-four hours,''
said Peter.

```

Each command is associated with a length defined in the document class file.

`\bigskip`

The same as `\vspace{\bigskipamount}`, ordinarily about one line space, with stretch and shrink. The default for the `book` and `article` classes is `12pt plus 4pt minus 4pt`.

`\medskip`

The same as `\vspace{\medskipamount}`, ordinarily about half of a line space, with stretch and shrink. The default for the `book` and `article` classes is `6pt plus 2pt minus 2pt`.

`\smallskip`

The same as `\vspace{\smallskipamount}`, ordinarily about a quarter of a line space, with stretch and shrink. The default for the `book` and `article` classes is `3pt plus 1pt minus 1pt`.
Because each command is a `\vspace`, if you use it in mid-paragraph then it will insert its vertical space between the line in which you use it and the next line, not necessarily at the place that you use it. So these are best between paragraphs.
The commands `\bigbreak`, `\medbreak`, and `\smallbreak` are similar but also suggest to LaTeX that this is a good place to put a page break (see [`\bigbreak` & `\medbreak` & `\smallbreak`](https://latexref.xyz/dev/latex2e.html#g_t_005cbigbreak-_0026-_005cmedbreak-_0026-_005csmallbreak).
### 19.12 `\bigbreak` & `\medbreak` & `\smallbreak`
Synopsis, one of:
```
\bigbreak
\medbreak
\smallbreak

```

Produce a vertical space that is big or medium-sized or small, and suggest to LaTeX that this is a good place to break the page. (The associated penalties are respectively −200, −100, and −50.)
See [`\bigskip` & `\medskip` & `\smallskip`](https://latexref.xyz/dev/latex2e.html#g_t_005cbigskip-_0026-_005cmedskip-_0026-_005csmallskip), for more. These commands produce the same vertical space but differ in that they also remove a preceding vertical space if it is less than what they would insert (as with `\addvspace`). In addition, they terminate a paragraph where they are used: this example
```
abc\bigbreak def ghi

jkl mno pqr

```

will output three paragraphs, the first ending in ‘abc’ and the second starting, after an extra vertical space and a paragraph indent, with ‘def’.
### 19.13 `\strut`
Synopsis:
```
\strut

```

Ensure that the current line has height at least `0.7\baselineskip` and depth at least `0.3\baselineskip`. Essentially, LaTeX inserts into the line a rectangle having zero width, `\rule[-0.3\baselineskip]{0pt}{\baselineskip}` (see [`\rule`](https://latexref.xyz/dev/latex2e.html#g_t_005crule)). The `\baselineskip` changes with the current font or fontsize.
In this example the `\strut` keeps the box inside the frame from having zero height.
```
\setlength{\fboxsep}{0pt}\framebox[2in]{\strut}

```

This example has four lists. In the first there is a much bigger gap between items 2 and 3 than there is between items 1 and 2. The second list fixes that with a `\strut` at the end of its first item’s second line.
```
\setlength{\fboxsep}{0pt}
\noindent\begin{minipage}[t]{0.2\linewidth}
\begin{enumerate}
  \item \parbox[t]{15pt}{test \\ test}
  \item test
  \item test
\end{enumerate}
\end{minipage}%
\begin{minipage}[t]{0.2\linewidth}
\begin{enumerate}
  \item \parbox[t]{15pt}{test \\ test\strut}
  \item test
  \item test
\end{enumerate}
\end{minipage}%
\begin{minipage}[t]{0.2\linewidth}
\begin{enumerate}
  \item \fbox{\parbox[t]{15pt}{test \\ test}}
  \item \fbox{test}
  \item \fbox{test}
\end{enumerate}
\end{minipage}%
\begin{minipage}[t]{0.2\linewidth}
\begin{enumerate}
  \item \fbox{\parbox[t]{15pt}{test \\ test\strut}}
  \item \fbox{test}
  \item \fbox{test}
\end{enumerate}
\end{minipage}%

```

The final two lists use `\fbox` to show what’s happening. The first item `\parbox` of the third list goes only to the bottom of its second ‘test’, which happens not have any characters that descend below the baseline. The fourth list adds the strut that gives the needed extra below-baseline space.
The `\strut` command is often useful in graphics, such as in `TikZ` or `Asymptote`. For instance, you may have a command such as `\graphnode{node-name}` that fits a circle around node-name. However, unless you are careful the node-name’s ‘x’ and ‘y’ will produce different-diameter circles because the characters are different sizes. A careful `\graphnode` might insert a `\strut`, then node-name, and then draw the circle.
The general approach of using a zero width `\rule` is useful in many circumstances. In this table, the zero-width rule keeps the top of the first integral from hitting the `\hline`. Similarly, the second rule keeps the second integral from hitting the first.
```
\begin{tabular}{rl}
  \textsc{Integral}   &\textsc{Value}           \\
  \hline
  $\int_0^x t\, dt$   &$x^2/2$  \rule{0em}{2.5ex} \\
  $\int_0^x t^2\, dt$ &$x^3/3$  \rule{0em}{2.5ex}
\end{tabular}

```

(Although the line-ending double backslash command has an available optional argument to change the corresponding baseline skip, that won’t solve this issue. Changing the first double backslash to something like `\\[2.5ex]` will put more room between the header line and the `\hline` rule, and the integral would still hit the rule.)
### 19.14 `\vspace`
Synopsis, one of:
```
\vspace{length}
\vspace*{length}

```

Add the vertical space length. The length can be positive, negative, or zero. It is a rubber length—it may contain a `plus` or `minus` component (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)).
This puts space between the two paragraphs.
```
And I slept.

\vspace{1ex plus 0.5ex}
The new day dawned cold.

```

(See [`\bigskip` & `\medskip` & `\smallskip`](https://latexref.xyz/dev/latex2e.html#g_t_005cbigskip-_0026-_005cmedskip-_0026-_005csmallskip), for common inter-paragraph spaces.)
The `*`-form inserts vertical space that is non-discardable. More precisely, LaTeX discards vertical space at a page break and the `*`-form causes the space to stay. This example leaves space between the two questions.
```
Question: Find the integral of \( 5x^4+5 \).

\vspace*{2cm plus 0.5cm}
Question: Find the derivative of \( x^5+5x+9 \).

```

That space will be present even if the page break happens to fall between the questions.
If you use `\vspace` in the middle of a paragraph (i.e., in horizontal mode) then the space is inserted after the line containing the `\vspace` command; it does not start a new paragraph at the `\vspace` command.
In this example the two questions will be evenly spaced vertically on the page, with at least one inch of space below each.
```
\begin{document}
1) Who put the bomp in the bomp bah bomp bah bomp?
\vspace{1in plus 1fill}

2) Who put the ram in the rama lama ding dong?
\vspace{1in plus 1fill}
\end{document}

```

### 19.15 `\vfill`
Synopsis:
```
\vfill

```

End the current paragraph and insert a vertical rubber length that is infinite, so it can stretch or shrink as far as needed (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)).
It is often used in the same way as `\vspace{\fill}`, except that `\vfill` ends the current paragraph whereas `\vspace{\fill}` adds the infinite vertical space below its line, irrespective of the paragraph structure. In both cases that space will disappear at a page boundary; to circumvent this see the starred option in [`\vspace`](https://latexref.xyz/dev/latex2e.html#g_t_005cvspace).
In this example the page is filled, so the top and bottom lines contain the text ‘Lost Dog!’ and the second ‘Lost Dog!’ is exactly halfway between them.
```
\begin{document}
Lost Dog!
\vfill
Lost Dog!  % perfectly in the middle
\vfill
Lost Dog!
\end{document}

```

### 19.16 `\addvspace`
Synopsis:
```
\addvspace{vert-length}

```

Add a vertical space of vert-length. However, if there are two or more `\addvspace`’s in a sequence then together they only add the space needed to make the natural length equal to the maximum of the vert-length’s in that sequence. This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)). The vert-length is a rubber length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)).
This example illustrates. The `picture` draws a scale over which to rules are placed. In a standard LaTeX article the length `\baselineskip` is 12pt. As shown by the scale, the two rules are 22pt apart: the sum of the `\baselineskip` and the 10pt from the first `\addvspace`.
```
\documentclass{article}
\usepackage{color}
\begin{document}
\setlength{\unitlength}{2pt}%
\noindent\begin{picture}(0,0)%
  \multiput(0,0)(0,-1){25}{{\color{blue}\line(1,0){1}}}
  \multiput(0,0)(0,-5){6}{{\color{red}\line(1,0){2}}}
\end{picture}%
\rule{0.25\linewidth}{0.1pt}%
\par\addvspace{10pt}% \addvspace{20pt}%
\par\noindent\rule{0.25\linewidth}{0.1pt}%
\end{document}

```

Now uncomment the second `\addvspace`. It does not make the gap 20pt longer; instead the gap is the sum of `\baselineskip` and 20pt. So `\addvspace` in a sense does the opposite of its name—it makes sure that multiple vertical spaces do not accumulate, but instead that only the largest one is used.
LaTeX uses this command to adjust the vertical space above or below an environment that starts a new paragraph. For instance, a `theorem` environment begins and ends with `\addvspace` so that two consecutive `theorem`’s are separated by one vertical space, not two.
A error ‘Something's wrong--perhaps a missing \item’ pointing to an `\addvspace` means that you were not in vertical mode when you hit this command. One way to change that is to precede `\addvspace` with a `\par` command (see [`\par`](https://latexref.xyz/dev/latex2e.html#g_t_005cpar)), as in the above example.
