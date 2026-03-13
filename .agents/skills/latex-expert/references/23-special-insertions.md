# 23 Special insertions

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 23.1 Printing special characters
- 23.2 Upper and lower case
- 23.3 Symbols by font position
- 23.4 Text symbols
- 23.5 Accents
- 23.6 Additional Latin letters
- 23.7 `inputenc` package
- 23.8 `\rule`
- 23.9 `\today`

## 23 Special insertions
LaTeX provides commands for inserting characters that have a special meaning do not correspond to simple characters you can type.
  * [Printing special characters](https://latexref.xyz/dev/latex2e.html#Printing-special-characters)
  * [Upper and lower case](https://latexref.xyz/dev/latex2e.html#Upper-and-lower-case)
  * [Symbols by font position](https://latexref.xyz/dev/latex2e.html#Symbols-by-font-position)
  * [Text symbols](https://latexref.xyz/dev/latex2e.html#Text-symbols)
  * [Accents](https://latexref.xyz/dev/latex2e.html#Accents)
  * [Additional Latin letters](https://latexref.xyz/dev/latex2e.html#Additional-Latin-letters)
  * [`inputenc` package](https://latexref.xyz/dev/latex2e.html#inputenc-package)
  * [`\rule`](https://latexref.xyz/dev/latex2e.html#g_t_005crule)
  * [`\today`](https://latexref.xyz/dev/latex2e.html#g_t_005ctoday)

### 23.1 Printing special characters
LaTeX sets aside a few characters for special purposes; they are called reserved characters or special characters. Here they are:
```
# $ % & { } _ ~ ^ \

```

The meaning of all the special characters is given elsewhere in this manual (see [Reserved characters](https://latexref.xyz/dev/latex2e.html#Reserved-characters)).
If you want a reserved character to be printed as itself, in the text body font, for all but the final three characters in that list simply put a `\` in front of the character. Thus, typing `\$1.23` will produce `$1.23` in your output.
As to the last three characters, to get a tilde in the text body font use `\~{}` (omitting the curly braces would result in the next character receiving a tilde accent). Similarly, to get a text body font circumflex use `\^{}`. To get a backslash in the font of the text body, enter `\textbackslash{}`.
To produce the reserved characters in a typewriter font, use `\verb!!` as below (the `\newline` in the example is there only to split the lines in the output).
```
\begin{center}
  \# \$ \% \& \{ \} \_ \~{} \^{} \textbackslash \newline
  \verb!# $ % & { } _ ~ ^ \!
\end{center}

```

### 23.2 Upper and lower case
Synopsis:
```
\uppercase{text}
\lowercase{text}
\MakeUppercase{text}
\MakeLowercase{text}

```

Change the case of characters. The TeX primitive commands `\uppercase` and `\lowercase` are set up by default to work only with the 26 letters a–z and A–Z. The LaTeX commands `\MakeUppercase` and `\MakeLowercase` commands also change characters accessed by commands such as `\ae` or `\aa`. The commands `\MakeUppercase` and `\MakeLowercase` are robust but they have moving arguments (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
These commands do not change the case of letters used in the name of a command within text. But they do change the case of every other Latin letter inside the argument text. Thus, `\MakeUppercase{Let $y=f(x)$`} produces ‘LET Y=F(X)’. Another example is that the name of an environment will be changed, so that `\MakeUppercase{\begin{tabular} ... \end{tabular}}` will produce an error because the first half is changed to `\begin{TABULAR}`.
LaTeX uses the same fixed table for changing case throughout a document, The table used is designed for the font encoding T1; this works well with the standard TeX fonts for all Latin alphabets but will cause problems when using other alphabets.
To change the case of text that results from a macro inside text you need to do expansion. Here the `\Schoolname` produces ‘COLLEGE OF MATHEMATICS’.
```
\newcommand{\schoolname}{College of Mathematics}
\newcommand{\Schoolname}{\expandafter\MakeUppercase
                           \expandafter{\schoolname}}

```

The `textcase` package brings some of the missing feature of the standard LaTeX commands `\MakeUppercase` and `\MakeLowerCase`.
To uppercase only the first letter of words, you can use the package `mfirstuc`.
Handling all the casing rules specified by Unicode, e.g., for non-Latin scripts, is a much bigger job than anything envisioned in the original TeX and LaTeX. It has been implemented in the `expl3` package as of 2020. The article “Case changing: From TeX primitives to the Unicode algorithm”, (Joseph Wright, TUGboat 41:1, <https://tug.org/TUGboat/tb41-1/tb127wright-case.pdf>), gives a good overview of the topic, past and present.
### 23.3 Symbols by font position
You can access any character of the current font using its number with the `\symbol` command. For example, the visible space character used in the `\verb*` command has the code decimal 32 in the standard Computer Modern typewriter font, so it can be typed as `\symbol{32}`.
You can also specify numbers in octal (base 8) by using a `'` prefix, or hexadecimal (base 16) with a `"` prefix, so the visible space at 32 decimal could also be written as `\symbol{'40}` or `\symbol{"20}`.
### 23.4 Text symbols
LaTeX provides commands to generate a number of non-letter symbols in running text. Some of these, especially the more obscure ones, are not available in OT1. As of the LaTeX February 2020 release, all symbols are available by default; before that, it was necessary to use the `textcomp` package for some (technically, those in the `TS1` font encoding).

`\copyright`

`\textcopyright`

© The copyright symbol.

`\dag`

† The dagger symbol (in text).

`\ddag`

‡ The double dagger symbol (in text).

`\LaTeX`

The LaTeX logo.

`\LaTeXe`

The LaTeX2e logo.

`\guillemetleft («)`

`\guillemetright (»)`

`\guillemotleft («)`

`\guillemotright (»)`

`\guilsinglleft (‹)`

`\guilsinglright (›)`

«, », ‹, › Double and single angle quotation marks, commonly used in French. The commands `@guillemotleft` and `@guillemotright` are synonyms for `@guillemet...`; these are misspellings inherited from Adobe. (Guillemots are seabirds; guillemets are French quotes.)

`\ldots`

`\textellipsis`

`\dots`

… An ellipsis (three dots at the baseline): `\ldots` and `\dots` also work in math mode (see [Dots, horizontal or vertical](https://latexref.xyz/dev/latex2e.html#Dots)). See that math mode ellipsis description for additional general information.

`\lq`

‘ Left (opening) quote.

`\P`

`\textparagraph`

¶ Paragraph sign (pilcrow).

`\pounds`

`\textsterling`

£ English pounds sterling.

`\quotedblbase („)`

`\quotesinglbase (‚)`

„ and ‚ Double and single quotation marks on the baseline.

`\rq`

’ Right (closing) quote.

`\S`

`\textsection`

§ Section sign.

`\TeX`

The TeX logo.

`\textasciicircum`

^ ASCII circumflex.

`\textasciitilde`

~ ASCII tilde.

`\textasteriskcentered`

* Centered asterisk.

`\textbackslash`

\ Backslash. However, `\texttt{\textbackslash}` produces a roman (not typewriter) backslash by default; for a typewriter backslash, it is necessary to use the T1 (or other non-default) font encoding, as in:
```
\usepackage[T1]{fontenc}

```

`\textbar`

| Vertical bar.

`\textbardbl`

⏸ Double vertical bar.

`\textbigcircle`

◯, Big circle symbol.

`\textbraceleft`

{ Left brace. See remarks at `\textbackslash` above about making `\texttt{\textbraceleft}` produce a typewriter brace.

`\textbraceright`

} Right brace. See remarks at `\textbackslash` above about making `\texttt{\textbraceright}` produce a typewriter brace.

`\textbullet`

• Bullet.

`\textcircled{letter}`

Ⓐ, Circle around letter.

`\textcompwordmark`

`\textcapitalcompwordmark`

`\textascendercompwordmark`

Used to separate letters that would normally ligature. For example, `f\textcompwordmark i` produces ‘fi’ without a ligature. This is most useful in non-English languages. The `\textcapitalcompwordmark` form has the cap height of the font while the `\textascendercompwordmark` form has the ascender height.

`\textdagger`

† Dagger.

`\textdaggerdbl`

‡ Double dagger.

`\textdollar (or `\$`)`

$ Dollar sign.

`\textemdash (or `---`)`

— Em-dash. Used for punctuation, usually similar to commas or parentheses, as in ‘`The playoffs---if you're lucky enough to make the playoffs---are more like a sprint.`’ Conventions for spacing around em-dashes vary widely.

`\textendash (or `--`)`

– En-dash. Used for ranges, as in ‘`see pages 12--14`’.

`\texteuro`

The Euro currency symbol: €.
For an alternative glyph design, try the `eurosym` package; also, most fonts nowadays come with their own Euro symbol (Unicode U+20AC).

`\textexclamdown (or `!``)`

¡ Upside down exclamation point.

`\textfiguredash`

Dash used between numerals, Unicode U+2012. Defined in the June 2021 release of LaTeX. When used in pdfTeX, approximated by an en-dash; with a Unicode engine, either typesets the glyph if available in the current font, or writes the usual “Missing character” warning to the log file.

`\textgreater`

> Greater than symbol.

`\texthorizontalbar`

Horizontal bar character, Unicode U+2015. Defined in the June 2021 release of LaTeX. Behavior as with `\textfiguredash` above; the pdfTeX approximation is an em-dash.

`\textless`

< Less than symbol.

`\textleftarrow`

←, Left arrow.

`\textnonbreakinghyphen`

Non-breaking hyphen character, Unicode U+2011. Defined in the June 2021 release of LaTeX. Behavior as with `\textfiguredash` above; the pdfTeX approximation is a regular ASCII hyphen (with breaks disallowed after).

`\textordfeminine`

`\textordmasculine`

ª, º Feminine and masculine ordinal symbols.

`\textperiodcentered`

· Centered period.

`\textquestiondown (or `?``)`

¿ Upside down question mark.

`\textquotedblleft (or ````)`

“ Double left quote.

`\textquotedblright (or `''`)`

” Double right quote.

`\textquoteleft (or ```)`

‘ Single left quote.

`\textquoteright (or `'`)`

’ Single right quote.

`\textquotesingle`

', Straight single quote. (From TS1 encoding.)

`\textquotestraightbase`

`\textquotestraightdblbase`

Single and double straight quotes on the baseline.

`\textregistered`

® Registered symbol.

`\textrightarrow`

→, Right arrow.

`\textthreequartersemdash`

﹘, “Three-quarters” em-dash, between en-dash and em-dash.

`\texttrademark`

™ Trademark symbol.

`\texttwelveudash`

﹘, “Two-thirds” em-dash, between en-dash and em-dash.

`\textunderscore`

_ Underscore.

`\textvisiblespace`

␣, Visible space symbol.
### 23.5 Accents
LaTeX has wide support for many of the world’s scripts and languages, provided through the core `babel` package, which supports pdfLaTeX, XeLaTeX and LuaLaTeX. The `polyglossia` package provides similar support with the latter two engines.
This section does not cover that support. It only lists the core LaTeX commands for creating accented characters. The `\capital...` commands shown here produce alternative forms for use with capital letters. These are not available with OT1.
Below, to make them easier to find, the accents are all illustrated with lowercase ‘o’.
Note that `\i` produces a dotless i,  and `\j` produces a dotless j. These are often used in place of their dotted counterparts when they are accented.

`\"`

`\capitaldieresis`

ö Umlaut (dieresis).

`\'`

`\capitalacute`

ó Acute accent.

`\.`

ȯ Dot accent.

`\=`

`\capitalmacron`

ō Macron (overbar) accent.

`\^`

`\capitalcircumflex`

ô Circumflex (hat) accent.

`\``

`\capitalgrave`

ò Grave accent.

`\~`

`\capitaltilde`

ñ Tilde accent.

`\b`

o̲ Bar accent underneath.
Related to this, `\underbar{text}` produces a bar under text. The argument is always processed in LR mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). The bar is always a fixed position under the baseline, thus crossing through descenders. See also `\underline` in [Over- and Underlining](https://latexref.xyz/dev/latex2e.html#Over_002d-and-Underlining).

`\c`

`\capitalcedilla`

ç Cedilla accent underneath.

`\d`

`\capitaldotaccent`

ọ Dot accent underneath.

`\H`

`\capitalhungarumlaut`

ő Long Hungarian umlaut accent.

`\k`

`\capitalogonek`

ǫ Ogonek. Not available in the OT1 encoding.

`\r`

`\capitalring`

o̊ Ring accent.

`\t`

`\capitaltie`

`\newtie`

`\capitalnewtie`

Tie-after accent (used for transliterating from Cyrillic, such as in the ALA-LC romanization). It expects that the argument has two characters. The `\newtie` form is centered in its box.

`\u`

`\capitalbreve`

ŏ Breve accent.

`\v`

`\capitalcaron`

ǒ Háček (check, caron) accent.
  * [`\accent`](https://latexref.xyz/dev/latex2e.html#g_t_005caccent)

#### 23.5.1 `\accent`
Synopsis:
```
\accent number character

```

A TeX primitive command used to generate accented characters from accent marks and letters. The accent mark is selected by number, a numeric argument, followed by a space and then a character argument to construct the accented character in the current font.
These are accented ‘e’ characters.
```
\accent18 e
\accent20 e
\accent21 e
\accent22 e
\accent23 e

```

The first is a grave, the second a caron, the third a breve, the fourth a macron, and the fifth a ring above.
The position of the accent is determined by the font designer and so the outcome of `\accent` use may differ between fonts. In LaTeX it is desirable to have glyphs for accented characters rather than building them using `\accent`. Using glyphs that already contain the accented characters (as in T1 encoding) allows correct hyphenation whereas `\accent` disables hyphenation (specifically with OT1 font encoding where accented glyphs are absent).
There can be an optional font change between number and character. Note also that this command sets the `\spacefactor` to 1000 (see [`\spacefactor`](https://latexref.xyz/dev/latex2e.html#g_t_005cspacefactor)).
An unavoidable characteristic of some Cyrillic letters and the majority of accented Cyrillic letters is that they must be assembled from multiple elements (accents, modifiers, etc.) while `\accent` provides for a single accent mark and a single letter combination. There are also cases where accents must appear between letters that \accent does not support. Still other cases exist where the letters I and J have dots above their lowercase counterparts that conflict with dotted accent marks. The use of `\accent` in these cases will not work as it cannot analyze upper/lower case.
### 23.6 Additional Latin letters
Here are the basic LaTeX commands for inserting letters beyond A–Z that extend the Latin alphabet, used primarily in languages other than English.

`\aa`

`\AA`

å and Å.

`\ae`

`\AE`

æ and Æ.

`\dh`

`\DH`

Icelandic letter eth: ð and Ð. Not available with OT1 encoding, you need the fontenc package to select an alternate font encoding, such as T1.

`\dj`

`\DJ`

Crossed d and D, a.k.a. capital and small letter d with stroke. Not available with OT1 encoding, you need the fontenc package to select an alternate font encoding, such as T1.

`\ij`

`\IJ`

ij and IJ (except somewhat closer together than appears here).

`\l`

`\L`

ł and Ł.

`\ng`

`\NG`

Lappish letter eng, also used in phonetics.

`\o`

`\O`

ø and Ø.

`\oe`

`\OE`

œ and Œ.

`\ss`

`\SS`

ß and SS.

`\th`

`\TH`

Icelandic letter thorn: þ and Þ. Not available with OT1 encoding, you need the fontenc package to select an alternate font encoding, such as T1.
### 23.7 `inputenc` package
Synopsis:
```
\usepackage[encoding-name]{inputenc}

```

Declare the input file’s text encoding to be encoding-name. (For basic background, see [Input encodings](https://latexref.xyz/dev/latex2e.html#Input-encodings)). The default, if this package is not loaded, is UTF-8. Technically, specifying the encoding name is optional, but in practice it is not useful to omit it.
The `inputenc` package tells LaTeX what encoding is used. For instance, the following command explicitly says that the input file is UTF-8 (note the lack of a dash).
```
\usepackage[utf8]{inputenc}

```

The most common values for encoding-name are: `ascii`, `latin1`, `latin2`, `latin3`, `latin4`, `latin5`, `latin9`, `latin10`, `utf8`.
Caution: use `inputenc` only with the pdfTeX engine (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)); with `xelatex` or `lualatex`, declaring a non-UTF-8 encoding with `inputenc`, such as `latin1`, will get the error `inputenc is not designed for xetex or luatex`.
An `inputenc` package error such as `Invalid UTF-8 byte "96` means that some of the material in the input file does not follow the encoding scheme. Often these errors come from copying material from a document that uses a different encoding than the input file. The simplest solution is often to replace the non-UTF-8 character with a UTF-8 or LaTeX equivalent.
If you need to process a non-UTF-8 document with LuaTeX, you can use the `luainputenc` package (<https://ctan.org/pkg/luainputenc>). With XeTeX, the `\XeTeXinputencoding` and `\XeTeXdefaultencoding` primitives can be used (for an explanation and examples, see <https://tex.stackexchange.com/questions/324948>).
It’s also possible to re-encode a document from an 8-bit encoding to UTF-8 outside of TeX, using system utilities. For example, ‘recode latin1..utf8’ or ‘iconv -f latin1 -t utf8’.
In a few documents, such as a collection of journal articles from a variety of authors, changing the encoding in mid-document may be necessary. You can use the command `\inputencoding{encoding-name}` for this.
### 23.8 `\rule`
Synopsis, one of:
```
\rule{width}{thickness}
\rule[raise]{width}{thickness}

```

Produce a _rule_ , a filled-in rectangle.
This example produces a rectangular blob, sometimes called a Halmos symbol, or just “qed”, often used to mark the end of a proof:
```
\newcommand{\qedsymbol}{\rule{0.4em}{2ex}}

```

The `amsthm` package includes this command, with a somewhat different-looking symbol.
The mandatory arguments give the horizontal width and vertical thickness of the rectangle. They are rigid lengths (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). The optional argument raise is also a rigid length, and tells LaTeX how much to raise the rule above the baseline, or lower it if the length is negative.
This produces a line, a rectangle that is wide but not tall.
```
\noindent\rule{\textwidth}{0.4pt}

```

The line is the width of the page and 0.4 points tall. This line thickness is common in LaTeX.
A rule that has zero width, or zero thickness, will not show up in the output, but can cause LaTeX to change the output around it. See [`\strut`](https://latexref.xyz/dev/latex2e.html#g_t_005cstrut), for examples.
### 23.9 `\today`
Synopsis:
```
\today

```

Produce today’s date in the format ‘month dd, yyyy’. An example of a date in that format is ‘July 4, 1976’.
Multilingual packages such as `babel` or `polyglossia`, or classes such as lettre, will localize `\today`. For example, the following will output ‘4 juillet 1976’:
```
\year=1976 \month=7 \day=4
\documentclass{minimal}
\usepackage[french]{babel}
\begin{document}
\today
\end{document}

```

`\today` uses the counters `\day`, `\month`, and `\year` (see [`\day` & `\month` & `\year`](https://latexref.xyz/dev/latex2e.html#g_t_005cday-_0026-_005cmonth-_0026-_005cyear)).
A number of package on CTAN work with dates. One is `datetime` package which can produce a wide variety of date formats, including ISO standards.
The date is not updated as the LaTeX process runs, so in principle the date could be incorrect by the time the program finishes.
