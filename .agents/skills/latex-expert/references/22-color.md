# 22 Color

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 22.1 `color` package options
- 22.2 Color models
- 22.3 Commands for color

## 22 Color
You can add color to text, rules, etc. You can also have color in a box or on an entire page and write text on top of it.
Color support comes as an additional package. So put `\usepackage{color}` in your document preamble to use the commands described here.
Many other packages also supplement LaTeX’s color abilities. Particularly worth mentioning is xcolor, which is widely used and significantly extends the capabilities described here, including adding ‘HTML’ and ‘Hsb’ color models.
  * [`color` package options](https://latexref.xyz/dev/latex2e.html#Color-package-options)
  * [Color models](https://latexref.xyz/dev/latex2e.html#Color-models)
  * [Commands for color](https://latexref.xyz/dev/latex2e.html#Commands-for-color)

### 22.1 `color` package options
Synopsis (must be in the document preamble):
```
\usepackage[comma-separated option list]{color}

```

When you load the color package there are two kinds of available options.
The first specifies the _printer driver_. LaTeX doesn’t contain information about different output systems but instead depends on information stored in a file. Normally you should not specify the driver option in the document, and instead rely on your system’s default. One advantage of this is that it makes the document portable across systems. For completeness we include a list of the drivers. The currently relevant ones are: dvipdfmx, dvips, dvisvgm, luatex, pdftex, xetex. The two xdvi and oztex are essentially aliases for dvips (and xdvi is monochrome). Ones that should not be used for new systems are: dvipdf, dvipdfm, dviwin, dvipsone, emtex, pctexps, pctexwin, pctexhp, pctex32, truetex, tcidvi, vtex (and dviwindo is an alias for dvipsone).
The second kind of options, beyond the drivers, are below.

`monochrome`

Disable the color commands, so that they do not generate errors but do not generate color either.

`dvipsnames`

Make available a list of 68 color names that are often used, particularly in legacy documents. These color names were originally provided by the dvips driver, giving the option name.

`nodvipsnames`

Do not load that list of color names, saving LaTeX a tiny amount of memory space.
### 22.2 Color models
A _color model_ is a way of representing colors. LaTeX’s capabilities depend on the printer driver. However, the pdftex, xetex, and luatex printer drivers are today by far the most commonly used. The models below work for those drivers. All but one of these is also supported by essentially all other printer drivers used today.
Note that color combination can be additive or subtractive. Additive mixes colors of light, so that for instance combining full intensities of red, green, and blue produces white. Subtractive mixes pigments, such as with inks, so that combining full intensity of cyan, magenta, and yellow makes black.

`cmyk`

A comma-separated list with four real numbers between 0 and 1, inclusive. The first number is the intensity of cyan, the second is magenta, and the others are yellow and black. A number value of 0 means minimal intensity, while a 1 is for full intensity. This model is often used in color printing. It is a subtractive model.

`gray`

A single real number between 0 and 1, inclusive. The colors are shades of grey. The number 0 produces black while 1 gives white.

`rgb`

A comma-separated list with three real numbers between 0 and 1, inclusive. The first number is the intensity of the red component, the second is green, and the third the blue. A number value of 0 means that none of that component is added in, while a 1 means full intensity. This is an additive model.

`RGB`

(pdftex, xetex, luatex drivers) A comma-separated list with three integers between 0 and 255, inclusive. This model is a convenience for using `rgb` since outside of LaTeX colors are often described in a red-green-blue model using numbers in this range. The values entered here are converted to the `rgb` model by dividing by 255.

`named`

Colors are accessed by name, such as ‘PrussianBlue’. The list of names depends on the driver, but all support the names ‘black’, ‘blue’, ‘cyan’, ‘green’, ‘magenta’, ‘red’, ‘white’, and ‘yellow’ (See the `dvipsnames` option in [`color` package options](https://latexref.xyz/dev/latex2e.html#Color-package-options)).
### 22.3 Commands for color
These are the commands available with the color package.
  * [Define colors](https://latexref.xyz/dev/latex2e.html#Define-colors)
  * [Colored text](https://latexref.xyz/dev/latex2e.html#Colored-text)
  * [Colored boxes](https://latexref.xyz/dev/latex2e.html#Colored-boxes)
  * [Colored pages](https://latexref.xyz/dev/latex2e.html#Colored-pages)

#### 22.3.1 Define colors
Synopsis:
```
\definecolor{name}{model}{specification}

```

Give the name name to the color. For example, after this
```
\definecolor{silver}{rgb}{0.75,0.75,0.74}

```

you can use that color name with `Hi ho, \textcolor{silver}{Silver}!`.
This example gives the color a more abstract name, so it could change and not be misleading.
```
\definecolor{logocolor}{RGB}{145,92,131}    % RGB needs pdflatex
\newcommand{\logo}{\textcolor{logocolor}{Bob's Big Bagels}}

```

Often a document’s colors are defined in the preamble, or in the class or style, rather than in the document body.
#### 22.3.2 Colored text
Synopses:
```
\textcolor{name}{...}
\textcolor[color model]{color specification}{...}

```

or
```
\color{name}
\color[color model]{color specification}

```

The affected text gets the color. This line
```
\textcolor{magenta}{My name is Ozymandias, King of Kings;}
Look on my works, ye Mighty, and despair!

```

causes the first half to be in magenta while the rest is in black. You can use a color declared with `\definecolor` in exactly the same way that we just used the builtin color ‘magenta’.
```
\definecolor{MidlifeCrisisRed}{rgb}{1.0,0.11,0.0}
I'm thinking about getting a \textcolor{MidlifeCrisisRed}{sports car}.

```

The two `\textcolor` and `\color` differ in that the first is a command form, enclosing the text to be colored as an argument. Often this form is more convenient, or at least more explicit. The second form is a declaration, as in `The moon is made of {\color{green} green} cheese`, so it is in effect until the end of the current group or environment. This is sometimes useful when writing macros or as below where it colors everything inside the `center` environment, including the vertical and horizontal lines.
```
\begin{center} \color{blue}
  \begin{tabular}{l|r}
    UL &UR \\ \hline
    LL &LR
  \end{tabular}
\end{center}

```

You can use color in equations. A document might have this definition in the preamble
```
\definecolor{highlightcolor}{RGB}{225,15,0}

```

and then contain this equation.
```
\begin{equation}
  \int_a^b \textcolor{highlightcolor}{f'(x)}\,dx=f(b)-f(a)
\end{equation}

```

Typically the colors used in a document are declared in a class or style but sometimes you want a one-off. Those are the second forms in the synopses.
```
Colors of \textcolor[rgb]{0.33,0.14,0.47}{Purple} and
{\color[rgb]{0.72,0.60,0.37}Gold} for the team.

```

The format of color specification depends on the color model (see [Color models](https://latexref.xyz/dev/latex2e.html#Color-models)). For instance, while `rgb` takes three numbers, `gray` takes only one.
```
The selection was \textcolor[gray]{0.5}{grayed out}.

```

Colors inside colors do not combine. Thus
```
\textcolor{green}{kind of \textcolor{blue}{blue}}

```

has a final word that is blue, not a combination of blue and green.
#### 22.3.3 Colored boxes
Synopses:
```
\colorbox{name}{...}
\colorbox[model name]{box background color}{...}

```

or
```
\fcolorbox{frame color}{box background color}{...}
\fcolorbox[model name]{frame color}{box background color}{...}

```

Make a box with the stated background color. The `\fcolorbox` command puts a frame around the box. For instance this
```
Name:~\colorbox{cyan}{\makebox[5cm][l]{\strut}}

```

makes a cyan-colored box that is five centimeters long and gets its depth and height from the `\strut` (so the depth is `-.3\baselineskip` and the height is `\baselineskip`). This puts white text on a blue background.
```
\colorbox{blue}{\textcolor{white}{Welcome to the machine.}}

```

The `\fcolorbox` commands use the same parameters as `\fbox` (see [`\fbox` & `\framebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cfbox-_0026-_005cframebox)), `\fboxrule` and `\fboxsep`, to set the thickness of the rule and the boundary between the box interior and the surrounding rule. LaTeX’s defaults are `0.4pt` and `3pt`, respectively.
This example changes the thickness of the border to 0.8 points. Note that it is surrounded by curly braces so that the change ends at the end of the second line.
```
{\setlength{\fboxrule}{0.8pt}
\fcolorbox{black}{red}{Under no circumstances turn this knob.}}

```

#### 22.3.4 Colored pages
Synopses:
```
\pagecolor{name}
\pagecolor[color model]{color specification}
\nopagecolor

```

The first two set the background of the page, and all subsequent pages, to the color. For an explanation of the specification in the second form see [Colored text](https://latexref.xyz/dev/latex2e.html#Colored-text). The third returns the background to normal, which is a transparent background. (If that is not supported use `\pagecolor{white}`, although that will make a white background rather than the default transparent background.)
```
 ...
\pagecolor{cyan}
 ...
\nopagecolor

```
