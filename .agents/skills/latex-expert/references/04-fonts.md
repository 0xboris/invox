# 04 Fonts

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 4.1 `fontenc` package
- 4.2 Font styles
- 4.3 Font sizes
- 4.4 Low-level font commands

## 4 Fonts
LaTeX comes with powerful font capacities. For one thing, its New Font Selection Scheme allows you to work easily with the font families in your document (for instance, see [Font styles](https://latexref.xyz/dev/latex2e.html#Font-styles)). And, LaTeX documents can use most fonts that are available today, including versions of Times Roman, Helvetica, Courier, etc. (Note, though, that many fonts do not have support for mathematics.)
The first typeface in the TeX world was the Computer Modern family, developed by Donald Knuth. It is the default for LaTeX documents and is still the most widely used. But changing to another font often only involves a few commands. For instance, putting the following in your preamble gives you a Palatino-like font, which is handsome and more readable online than many other fonts, while still allowing you to typeset mathematics. (This example is from Michael Sharpe, <https://math.ucsd.edu/~msharpe/RcntFnts.pdf>.)
```
\usepackage[osf]{newpxtext} % osf for text, not math
\usepackage{cabin} % sans serif
\usepackage[varqu,varl]{inconsolata} % sans serif typewriter
\usepackage[bigdelims,vvarbb]{newpxmath} % bb from STIX
\usepackage[cal=boondoxo]{mathalfa} % mathcal

```

In addition, the `xelatex` or `lualatex` engines allow you to use any fonts on your system that are in OpenType or TrueType format (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)).
The LaTeX Font Catalogue (<https://tug.org/FontCatalogue>) shows font sample graphics and copy-and-pasteable source to use many fonts, including many with support for mathematics. It aims to cover all Latin alphabet free fonts available for easy use with LaTeX.
More information is also available from the TeX Users Group, at <https://www.tug.org/fonts/>.
  * [`fontenc` package](https://latexref.xyz/dev/latex2e.html#fontenc-package)
  * [Font styles](https://latexref.xyz/dev/latex2e.html#Font-styles)
  * [Font sizes](https://latexref.xyz/dev/latex2e.html#Font-sizes)
  * [Low-level font commands](https://latexref.xyz/dev/latex2e.html#Low_002dlevel-font-commands)

### 4.1 `fontenc` package
Synopsis:
```
\usepackage[font_encoding]{fontenc}

```

or
```
\usepackage[font_encoding1, font_encoding2, ...]{fontenc}

```

Specify the font encodings. A font encoding is a mapping of the character codes to the font glyphs that are used to typeset your output.
This package only applies if you use the `pdflatex` engine (see [TeX engines](https://latexref.xyz/dev/latex2e.html#TeX-engines)). If you use the `xelatex` or `lualatex` engine then instead use the `fontspec` package.
TeX’s original font family, Computer Modern, has a limited character set. For instance, to make common accented characters you must use `\accent` (see [`\accent`](https://latexref.xyz/dev/latex2e.html#g_t_005caccent)) but this disables hyphenation. TeX users have agreed on a number of standards to access the larger sets of characters provided by modern fonts. If you are using `pdflatex` then put this in the preamble
```
\usepackage[T1]{fontenc}

```

gives you support for the most widespread European languages, including French, German, Italian, Polish, and others. In particular, if you have words with accented letters then LaTeX will hyphenate them and your output can be copied and pasted. (The optional second line allows you to directly enter accented characters into your source file.)
If you are using an encoding such as `T1` and the characters appear blurry or do not magnify well then your fonts may be bitmapped, sometimes called raster or Type 3. You want vector fonts. Use a package such as `lmodern` or `cm-super` to get a font that extends LaTeX’s default using vector fonts.
For each font_encoding given as an option but not already declared, this package loads the encoding definition files, named font_encodingenc.def. It also sets `\encodingdefault` to be the last encoding in the option list.
These are the common values for font_encoding:

`OT1`

The original 7-bit encoding for TeX. Limited to mostly English characters.

`OMS, OML`

Math symbols and math letters encoding.

`T1`

TeX text extended. Sometimes called the Cork encoding for the users group meeting where it was developed (1990). Gives access to most European accented characters. The most common option for this package.

`TS1`

Text Companion encoding.
LaTeX’s default is to load `OML`, `T1`, `OT1`, and then `OMS`, and set the default to `OT1`.
Even if you do not use accented letters, you may need to specify a font encoding if your font requires it.
If you use `T1` encoded fonts other than the default Computer Modern family then you may need to load the package that selects your fonts before loading fontenc, to prevent the system from loading any `T1` encoded fonts from the default.
The LaTeX team reserves encoding names starting with: ‘T’ for the standard text encodings with 256 characters, ‘TS’ for symbols that extend the corresponding T encodings, ‘X’ for test encodings, ‘M’ for standard math encodings with 256 characters, ‘A’ for special applications, ‘OT’ for standard text encodings with 128 characters, and ‘OM’ for standard math encodings with 128 characters (‘O’ stands for ‘obsolete’).
This package provides a number of commands, detailed below. Many of them are encoding-specific, so if you have defined a command that works for one encoding but the current encoding is different then the command is not in effect.
  * [`\DeclareFontEncoding`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareFontEncoding)
  * [`\DeclareTextAccent`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextAccent)
  * [`\DeclareTextAccentDefault`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextAccentDefault)
  * [`\DeclareTextCommand` & `\ProvideTextCommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextCommand-_0026-_005cProvideTextCommand)
  * [`\DeclareTextCommandDefault` & `\ProvideTextCommandDefault `](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextCommandDefault-_0026-_005cProvideTextCommandDefault)
  * [`\DeclareTextComposite`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextComposite)
  * [`\DeclareTextCompositeCommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextCompositeCommand)
  * [`\DeclareTextSymbol`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextSymbol)
  * [`\DeclareTextSymbolDefault`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextSymbolDefault)
  * [`\LastDeclaredEncoding`](https://latexref.xyz/dev/latex2e.html#g_t_005cLastDeclaredEncoding)
  * [`\UseTextSymbol` & `\UseTextAccent`](https://latexref.xyz/dev/latex2e.html#g_t_005cUseTextSymbol-_0026-_005cUseTextAccent)

#### 4.1.1 `\DeclareFontEncoding`
Synopsis:
```
\DeclareFontEncoding{encoding}{text-settings}{math-settings}

```

Declare the font encoding encoding. It also saves the value of encoding in `\LastDeclaredEncoding` (see [`\LastDeclaredEncoding`](https://latexref.xyz/dev/latex2e.html#g_t_005cLastDeclaredEncoding)).
The file t1enc.def contains this line (followed by many others).
```
\DeclareFontEncoding{T1}{}{}

```

The text-settings are the commands that LaTeX will run every time it switches from one encoding to another with the `\selectfont` and `\fontencoding` commands. The math-settings are the commands that LaTeX will use whenever the font is accessed as a math alphabet.
LaTeX ignores any space characters inside text-settings and math-settings, to prevent unintended spaces in the output.
If you invent an encoding you should pick a two or three letter name starting with ‘L’ for ‘local’, or ‘E’ for ‘experimental’.
Note that output encoding files may be read several times by LaTeX so using, e.g., `\newcommand` may cause an error. In addition, such files should contain `\ProvidesFile` line (see [Class and package commands](https://latexref.xyz/dev/latex2e.html#Class-and-package-commands)).
Note also that you should use the `\...Default` commands only in a package, not in the encoding definition files, since those files should only contain declarations specific to that encoding.
#### 4.1.2 `\DeclareTextAccent`
Synopsis:
```
\DeclareTextAccent{cmd}{encoding}{slot}

```

Define an accent, to be put on top of other glyphs, in the encoding encoding at the location slot.
A _slot_ is the number identifying a glyph within a font.
This line from t1enc.def declares that to make a circumflex accent as in `\^A`, the system will put the accent in slot 2 over the ‘A’ character, which is represented in ASCII as 65. (This holds unless there is a relevant `DeclareTextComposite` or `\DeclareTextCompositeCommand` declaration; see [`\DeclareTextComposite`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextComposite).)
```
\DeclareTextAccent{\^}{T1}{2}

```

If cmd has already been defined then `\DeclareTextAccent` does not give an error but it does log the redefinition in the transcript file.
#### 4.1.3 `\DeclareTextAccentDefault`
Synopsis:
```
\DeclareTextAccentDefault{\cmd}{encoding}

```

If there is an encoding-specific accent command \cmd but there is no associated `\DeclareTextAccent` for that encoding then this command will pick up the slack, by saying to use it as described for encoding.
For example, to make the encoding `OT1` be the default encoding for the accent `\"`, declare this.
```
\DeclareTextAccentDefault{\"}{OT1}

```

If you issue a `\"` when the current encoding does not have a definition for that accent then LaTeX will use the definition from `OT1`
That is, this command is equivalent to this call (see [`\UseTextSymbol` & `\UseTextAccent`](https://latexref.xyz/dev/latex2e.html#g_t_005cUseTextSymbol-_0026-_005cUseTextAccent)).
```
\DeclareTextCommandDefault[1]{\cmd}
   {\UseTextAccent{encoding}{\cmd}{#1}}

```

Note that `\DeclareTextAccentDefault` works for any one-argument fontenc command, not just the accent command.
#### 4.1.4 `\DeclareTextCommand` & `\ProvideTextCommand`
Synopsis, one of:
```
\DeclareTextCommand{\cmd}{encoding}{defn}
\DeclareTextCommand{\cmd}{encoding}[nargs]{defn}
\DeclareTextCommand{\cmd}{encoding}[nargs][optargdefault]{defn}

```

or one of:
```
\ProvideTextCommand{\cmd}{encoding}{defn}
\ProvideTextCommand{\cmd}{encoding}[nargs]{defn}
\ProvideTextCommand{\cmd}{encoding}[nargs][optargdefault]{defn}

```

Define the command `\cmd`, which will be specific to one encoding. The command name cmd must be preceded by a backslash, `\`. These commands can only appear in the preamble. Redefining \cmd does not cause an error. The defined command will be robust even if the code in defn is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
For example, the file t1enc.def contains this line.
```
\DeclareTextCommand{\textperthousand}{T1}{\%\char 24 }

```

With that, you can express parts per thousand.
```
\usepackage[T1]{fontenc}  % in preamble
  ...
Legal limit is \( 0.8 \)\textperthousand.

```

If you change the font encoding to `OT1` then you get an error like ‘LaTeX Error: Command \textperthousand unavailable in encoding OT1’.
The `\ProvideTextCommand` variant does the same, except that it does nothing if `\cmd` is already defined. The `\DeclareTextSymbol` command is faster than this one for simple slot-to-glyph association (see [`\DeclareTextSymbol`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextSymbol))
The optional nargs and optargdefault arguments play the same role here as in `\newcommand` (see [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand)). Briefly, nargs is an integer from 0 to 9 specifying the number of arguments that the defined command `\cmd` takes. This number includes any optional argument. Omitting this argument is the same as specifying 0, meaning that `\cmd` will have no arguments. And, if optargdefault is present then the first argument of `\cmd` is optional, with default value optargdefault (which may be the empty string). If optargdefault is not present then `\cmd` does not take an optional argument.
#### 4.1.5 `\DeclareTextCommandDefault` & `\ProvideTextCommandDefault `
Synopsis:
```
\DeclareTextCommandDefault{\cmd}{defn}

```

or:
```
\ProvideTextCommandDefault{\cmd}{defn}

```

Give a default definition for `\cmd`, for when that command is not defined in the encoding currently in force. This default should only use encodings known to be available.
This makes `\copyright` available.
```
\DeclareTextCommandDefault{\copyright}{\textcircled{c}}

```

It uses only an encoding (OMS) that is always available.
The `\DeclareTextCommandDefault` should not occur in the encoding definition files since those files should declare only commands for use when you select that encoding. It should instead be in a package.
As with the related non-default commands, the `\ProvideTextCommandDefault` has exactly the same behavior as `\DeclareTextCommandDefault` except that it does nothing if `\cmd` is already defined (see [`\DeclareTextCommand` & `\ProvideTextCommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextCommand-_0026-_005cProvideTextCommand)). So, packages can use it to provide fallbacks that other packages can improve upon.
#### 4.1.6 `\DeclareTextComposite`
Synopsis:
```
\DeclareTextComposite{\cmd}{encoding}{simple_object}{slot}

```

Access an accented glyph directly, that is, without having to put an accent over a separate character.
This line from t1enc.def means that `\^o` will cause LaTeX to typeset lowercase ‘o’ by taking the character directly from slot 224 in the font.
```
\DeclareTextComposite{\^}{T1}{o}{244}

```

See [`fontenc` package](https://latexref.xyz/dev/latex2e.html#fontenc-package), for a list of common encodings. The simple_object should be a single character or a single command. The slot argument is usually a positive integer represented in decimal (although octal or hexadecimal are possible). Normally \cmd has already been declared for this encoding, either with `\DeclareTextAccent` or with a one-argument `\DeclareTextCommand`. In t1enc.def, the above line follows the `\DeclareTextAccent{\^}{T1}{2}` command.
#### 4.1.7 `\DeclareTextCompositeCommand`
Synopsis:
```
\DeclareTextCompositeCommand{\cmd}{encoding}{arg}{code}

```

A more general version of `\DeclareTextComposite` that runs arbitrary code with `\cmd`.
This allows accents on ‘i’ to act like accents on dotless i, `\i`.
```
\DeclareTextCompositeCommand{\'}{OT1}{i}{\'\i}

```

See [`fontenc` package](https://latexref.xyz/dev/latex2e.html#fontenc-package), for a list of common encodings. Normally `\cmd` will have already been declared with `\DeclareTextAccent` or as a one argument `\DeclareTextCommand`.
#### 4.1.8 `\DeclareTextSymbol`
Synopsis:
```
\DeclareTextSymbol{\cmd}{encoding}{slot}

```

Define a symbol in the encoding encoding at the location slot. Symbols defined in this way are for use in text, not mathematics.
For example, this line from t1enc.def declares the number of the glyph to use for «, the left guillemet.
```
\DeclareTextSymbol{\guillemetleft}{T1}{19}

```

The command `\DeclareTextCommand{\guillemetleft}{T1}{\char 19}` has the same effect but is slower (see [`\DeclareTextCommand` & `\ProvideTextCommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareTextCommand-_0026-_005cProvideTextCommand)).
See [`fontenc` package](https://latexref.xyz/dev/latex2e.html#fontenc-package), for a list of common encodings. The slot can be specified in decimal, or octal (as in `'023`), or hexadecimal (as in `"13`), although decimal has the advantage that single quote or double quote could be redefined by another package.
If `\cmd` has already been defined then `\DeclareTextSymbol` does not give an error but it does log the redefinition in the transcript file.
#### 4.1.9 `\DeclareTextSymbolDefault`
Synopsis:
```
\DeclareTextSymbolDefault{\cmd}{encoding}

```

If there is an encoding-specific symbol command `\cmd` but there is no associated `\DeclareTextSymbol` for that encoding, then this command will pick up the slack, by saying to get the symbol as described for encoding.
For example, to declare that if the current encoding has no meaning for `\textdollar` then use the one from `OT1`, declare this.
```
\DeclareTextSymbolDefault{\textdollar}{OT1}

```

That is, this command is equivalent to this call (see [`\UseTextSymbol` & `\UseTextAccent`](https://latexref.xyz/dev/latex2e.html#g_t_005cUseTextSymbol-_0026-_005cUseTextAccent)).
```
\DeclareTextCommandDefault{\cmd}
   {\UseTextSymbol{encoding}{\cmd}}

```

Note that `\DeclareTextSymbolDefault` can be used to define a default for any zero-argument fontenc command.
#### 4.1.10 `\LastDeclaredEncoding`
Synopsis:
```
\LastDeclaredEncoding

```

Get the name of the most recently declared encoding. The `\DeclareFontEncoding` command stores the name so that it can be retrieved with this command (see [`\DeclareFontEncoding`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareFontEncoding)).
This relies on `\LastDeclaredEncoding` rather than give the name of the encoding explicitly.
```
\DeclareFontEncoding{JH1}{}{}
\DeclareTextAccent{\'}{\LastDeclaredEncoding}{0}

```

#### 4.1.11 `\UseTextSymbol` & `\UseTextAccent`
Synopsis:
```
\UseTextSymbol{encoding}{\cmd}

```

or:
```
\UseTextAccent{encoding}{\cmd}{text}

```

Use a symbol or accent not from the current encoding.
In general, to use a fontenc command in an encoding where it is not defined, and if the command has no arguments, then you can use it like this:
```
\UseTextSymbol{OT1}{\ss}

```

which is equivalent to this (note the outer braces form a group, so LaTeX reverts back to the prior encoding after the `\ss`):
```
{\fontencoding{OT1}\selectfont\ss}

```

Similarly, to use a fontenc command in an encoding where it is not defined, and if the command has one argument, you can use it like this:
```
\UseTextAccent{OT1}{\'}{a}

```

which is equivalent to this (again note the outer braces forming a group):
```
{fontencoding{OT1}\selectfont\'{\fontencoding{enc_in_use}\selectfont a}}

```

Here, enc_in_use is the encoding in force before this sequence of commands, so that ‘a’ is typeset using the current encoding and only the accent is taken from `OT1`.
### 4.2 Font styles
The following type style commands are supported by LaTeX.
In the table below the listed commands, the `\text...` commands, are used with an argument as in `\textit{text}`. This is the preferred form. But shown after it in parenthesis is the corresponding _declaration form_ , which is often useful. This form takes no arguments, as in `{\itshape text}`. The scope of the declaration form lasts until the next type style command or the end of the current group. In addition, each has an environment form such as `\begin{itshape}...\end{itshape}`, which we’ll describe further at the end of the section.
These commands, in any of the three forms, are cumulative; for instance you can get bold sans serif by saying either of `\sffamily\bfseries` or `\bfseries\sffamily`.
One advantage of these commands is that they automatically insert italic corrections if needed (see [`\/`](https://latexref.xyz/dev/latex2e.html#g_t_005c_002f)). Specifically, they insert the italic correction unless the following character is in the list `\nocorrlist`, which by default consists of period and comma. To suppress the automatic insertion of italic correction, use `\nocorr` at the start or end of the command argument, such as `\textit{\nocorr text}` or `\textsc{text \nocorr}`.

`\textrm (\rmfamily)`

Roman.

`\textit (\itshape)`

Italics.

`\textmd (\mdseries)`

Medium weight (default).

`\textbf (\bfseries)`

Boldface.

`\textup (\upshape)`

Upright (default).

`\textsl (\slshape)`

Slanted.

`\textsf (\sffamily)`

Sans serif.

`\textsc (\scshape)`

Small caps.

`\texttt (\ttfamily)`

Typewriter.

`\textnormal (\normalfont)`

Main document font.
Although it also changes fonts, the `\emph{text}` command is semantic, for text to be emphasized, and should not be used as a substitute for `\textit`. For example, `\emph{start text \emph{middle text} end text}` will result in the start text and end text in italics, but middle text will be in roman.
LaTeX also provides the following commands, which unconditionally switch to the given style, that is, are _not_ cumulative. They are used as declarations: `{\cmd...}` instead of `\cmd{...}`.
(The unconditional commands below are an older version of font switching. The earlier commands are an improvement in most circumstances. But sometimes an unconditional font switch is what is needed.)

`\bf`

Switch to bold face.

`\cal`

Switch to calligraphic letters for math.

`\it`

Italics.

`\rm`

Roman.

`\sc`

Small caps.

`\sf`

Sans serif.

`\sl`

Slanted (oblique).

`\tt`

Typewriter (monospace, fixed-width).
The `\em` command is the unconditional version of `\emph`.
The following commands are for use in math mode. They are not cumulative, so `\mathbf{\mathit{symbol}}` does not create a boldface and italic symbol; instead, it will just be in italics. This is because typically math symbols need consistent typographic treatment, regardless of the surrounding environment.

`\mathrm`

Roman, for use in math mode.

`\mathbf`

Boldface, for use in math mode.

`\mathsf`

Sans serif, for use in math mode.

`\mathtt`

Typewriter, for use in math mode.

`\mathit`

Italics, for use in math mode.

`\mathnormal`

For use in math mode, e.g., inside another type style declaration.

`\mathcal`

Calligraphic letters, for use in math mode.
These commands use the text fonts, but ignore spaces in their argument. If you need spaces, use the `\text...` font commands.
In addition, the command `\mathversion{bold}` can be used for switching to bold letters and symbols in formulas. `\mathversion{normal}` restores the default.
Finally, the command `\oldstylenums{numerals}` will typeset so-called “old-style” numerals, which have differing heights and depths (and sometimes widths) from the standard “lining” numerals, which all have the same height as uppercase letters. LaTeX’s default fonts support this, and will respect `\textbf` (but not other styles; there are no italic old-style numerals in Computer Modern). Many other fonts have old-style numerals also; sometimes package options are provided to make them the default. FAQ entry: <https://www.texfaq.org/FAQ-osf>.
### 4.3 Font sizes
The following standard type size commands are supported by LaTeX. The table shows the command name and the corresponding actual font size used (in points) with the ‘10pt’, ‘11pt’, and ‘12pt’ document size options, respectively (see [Document class options](https://latexref.xyz/dev/latex2e.html#Document-class-options)).
Command | `10pt` | `11pt` | `12pt`
---|---|---|---
`\tiny` | 5 | 6 | 6
`\scriptsize` | 7 | 8 | 8
`\footnotesize` | 8 | 9 | 10
`\small` | 9 | 10 | 10.95
`\normalsize` (default) | 10 | 10.95 | 12
`\large` | 12 | 12 | 14.4
`\Large` | 14.4 | 14.4 | 17.28
`\LARGE` | 17.28 | 17.28 | 20.74
`\huge` | 20.74 | 20.74 | 24.88
`\Huge` | 24.88 | 24.88 | 24.88
The commands are listed here in declaration (not environment) form, since that is how they are typically used. For example.
```
\begin{quotation} \small
  The Tao that can be named is not the eternal Tao.
\end{quotation}

```

Here, the scope of the `\small` lasts until the end of the `quotation` environment. It would also end at the next type style command or the end of the current group, so you could enclose it in curly braces `{\small This text is typeset in the small font.}`.
Trying to use these commands in math, as with `$\small mv^2/2$`, results in ‘LaTeX Font Warning: Command \small invalid in math mode’, and the font size doesn’t change. To work with a too-large formula, often the best option is to use the `displaymath` environment (see [Math formulas](https://latexref.xyz/dev/latex2e.html#Math-formulas)), or one of the environments from the `amsmath` package. For inline mathematics, such as in a table of formulas, an alternative is something like `{\small $mv^2/2$}`. (Sometimes `\scriptsize` and `\scriptstyle` are confused. Both change the font size, but the latter also changes a number of other aspects of how mathematics is typeset. See [Math styles](https://latexref.xyz/dev/latex2e.html#Math-styles).)
An _environment form_ of each of these commands is also defined; for instance, `\begin{tiny}...\end{tiny}`. However, in practice this form can easily lead to unwanted spaces at the beginning and/or end of the environment without careful consideration, so it’s generally less error-prone to stick to the declaration form.
(Aside: Technically, due to the way LaTeX defines `\begin` and `\end`, nearly every command that does not take an argument technically has an environment form. But in almost all cases, it would only cause confusion to use it. The reason for mentioning the environment form of the font size declarations specifically is that this particular use is not rare.)
### 4.4 Low-level font commands
These commands are primarily intended for writers of macros and packages. The commands listed here are only a subset of the available ones.

`\fontencoding{encoding}`

Select the font encoding, the encoding of the output font. There are a large number of valid encodings. The most common are `OT1`, Knuth’s original encoding for Computer Modern (the default), and `T1`, also known as the Cork encoding, which has support for the accented characters used by the most widespread European languages (German, French, Italian, Polish and others), which allows TeX to hyphenate words containing accented letters. For more, see <https://ctan.org/pkg/encguide>.

`\fontfamily{family}`

Select the font family. The web page <https://tug.org/FontCatalogue/> provides one way to browse through many of the fonts easily used with LaTeX. Here are examples of some common families.
`pag` | Avant Garde
---|---
`fvs` | Bitstream Vera Sans
`pbk` | Bookman
`bch` | Charter
`ccr` | Computer Concrete
`cmr` | Computer Modern
`cmss` | Computer Modern Sans Serif
`cmtt` | Computer Modern Typewriter
`pcr` | Courier
`phv` | Helvetica
`fi4` | Inconsolata
`lmr` | Latin Modern
`lmss` | Latin Modern Sans
`lmtt` | Latin Modern Typewriter
`pnc` | New Century Schoolbook
`ppl` | Palatino
`ptm` | Times
`uncl` | Uncial
`put` | Utopia
`pzc` | Zapf Chancery

`\fontseries{series}`

Select the font series. A _series_ combines a _weight_ and a _width_. Typically, a font supports only a few of the possible combinations. Some common combined series values include:
`m` | Medium (normal)
---|---
`b` | Bold
`c` | Condensed
`bc` | Bold condensed
`bx` | Bold extended
The possible values for weight, individually, are:
`ul` | Ultra light
---|---
`el` | Extra light
`l` | Light
`sl` | Semi light
`m` | Medium (normal)
`sb` | Semi bold
`b` | Bold
`eb` | Extra bold
`ub` | Ultra bold
The possible values for width, individually, are (the meaning and relationship of these terms varies with individual typefaces):
`uc` | Ultra condensed
---|---
`ec` | Extra condensed
`c` | Condensed
`sc` | Semi condensed
`m` | Medium
`sx` | Semi expanded
`x` | Expanded
`ex` | Extra expanded
`ux` | Ultra expanded
When forming the series string from the weight and width, drop the `m` that stands for medium weight or medium width, unless both weight and width are `m`, in which case use just one (‘`m`’).

`\fontshape{shape}`

Select font shape. Valid shapes are:
`n` | Upright (normal)
---|---
`it` | Italic
`sl` | Slanted (oblique)
`sc` | Small caps
`ui` | Upright italics
`ol` | Outline
The two last shapes are not available for most font families, and small caps are often missing as well.

`\fontsize{size}{skip}`

Set the font size and the line spacing. The unit of both parameters defaults to points (`pt`). The line spacing is the nominal vertical space between lines, baseline to baseline. It is stored in the parameter `\baselineskip`. The default `\baselineskip` for the Computer Modern typeface is 1.2 times the `\fontsize`. Changing `\baselineskip` directly is inadvisable since its value is reset every time a size change happens; instead use `\baselinestretch`. (see [`\baselineskip` & `\baselinestretch`](https://latexref.xyz/dev/latex2e.html#g_t_005cbaselineskip-_0026-_005cbaselinestretch)).

`\linespread{factor}`

Equivalent to `\renewcommand{\baselinestretch}{factor}`, and therefore must be followed by `\selectfont` to have any effect. Best specified in the preamble. See [`\baselineskip` & `\baselinestretch`](https://latexref.xyz/dev/latex2e.html#g_t_005cbaselineskip-_0026-_005cbaselinestretch), for using `setspace` package instead.

`\selectfont`

The effects of the font commands described above do not happen until `\selectfont` is called, as in `\fontfamily{familyname}\selectfont`. It is often useful to put this in a macro:
`\newcommand*{\myfont}{\fontfamily{familyname}\selectfont}`
(see [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand)).

`\usefont{enc}{family}{series}{shape}`

The same as invoking `\fontencoding`, `\fontfamily`, `\fontseries` and `\fontshape` with the given parameters, followed by `\selectfont`. For example:
```
\usefont{ot1}{cmr}{m}{n}

```
