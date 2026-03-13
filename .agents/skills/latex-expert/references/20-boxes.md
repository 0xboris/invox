# 20 Boxes

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 20.1 `\mbox` & `\makebox`
- 20.2 `\fbox` & `\framebox`
- 20.3 `\parbox`
- 20.4 `\raisebox`
- 20.5 `\sbox` & `\savebox`
- 20.6 `lrbox`
- 20.7 `\usebox`

## 20 Boxes
At its core, LaTeX puts things in boxes and then puts the boxes on a page. So these commands are central.
There are many packages on CTAN that are useful for manipulating boxes. One useful adjunct to the commands here is `adjustbox`.
  * [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox)
  * [`\fbox` & `\framebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cfbox-_0026-_005cframebox)
  * [`\parbox`](https://latexref.xyz/dev/latex2e.html#g_t_005cparbox)
  * [`\raisebox`](https://latexref.xyz/dev/latex2e.html#g_t_005craisebox)
  * [`\sbox` & `\savebox`](https://latexref.xyz/dev/latex2e.html#g_t_005csbox-_0026-_005csavebox)
  * [`lrbox`](https://latexref.xyz/dev/latex2e.html#lrbox)
  * [`\usebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cusebox)

### 20.1 `\mbox` & `\makebox`
Synopsis, one of:
```
\mbox{text}
\makebox{text}
\makebox[width]{text}
\makebox[width][position]{text}

```

Create a box, a container for material. The text is typeset in LR mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)) so it is not broken into lines. The `\mbox` command is robust, while `\makebox` is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
Because `text` is not broken into lines, you can use `\mbox` to prevent hyphenation. In this example, LaTeX will not hyphenate the tank name, ‘T-34’.
```
The soviet tank \mbox{T-34} is a symbol of victory against nazism.

```

The first two command invocations shown, `\mbox` and `\makebox`, are roughly the same. They create a box just wide enough to contain the text. (They are like plain TeX’s `\hbox`.)
In the third version the optional argument width specifies the width of the box. Note that the space occupied by the text need not equal the width of the box. For one thing, text can be too small; this creates a full-line box:
```
\makebox[\linewidth]{Chapter Exam}

```

with ‘Chapter Exam’ centered. But text can also be too wide for width. See the example below of zero-width boxes.
In the width argument you can use the following lengths that refer to the dimension of the box that LaTeX gets on typesetting text: `\depth`, `\height`, `\width`, `\totalheight` (this is the box’s height plus its depth). For example, to make a box with the text stretched to double the natural size you can say this.
```
\makebox[2\width]{Get a stretcher}

```

For the fourth command synopsis version the optional argument position gives position of the text within the box. It may take the following values:

`c`

The text is centered (default).

`l`

The text is flush left.

`r`

Flush right.

`s`

Stretch the interword space in text across the entire width. The text must contain stretchable space for this to work. For instance, this could head a press release: `\noindent\makebox[\textwidth][s]{\large\hfil IMMEDIATE\hfil RELEASE\hfil}`
A common use of `\makebox` is to make zero-width text boxes. This puts the value of the quiz questions to the left of those questions.
```
\newcommand{\pts}[1]{\makebox[0em][r]{#1 points\hspace*{1em}}}
\pts{10}What is the air-speed velocity of an unladen swallow?

\pts{90}An African or European swallow?

```

The right edge of the output ‘10 points ’ (note the ending space after ‘points’) will be just before the ‘What’. You can use `\makebox` similarly when making graphics, such as in `TikZ` or `Asymptote`, where you put the edge of the text at a known location, regardless of the length of that text.
For boxes with frames see [`\fbox` & `\framebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cfbox-_0026-_005cframebox). For colors see [Colored boxes](https://latexref.xyz/dev/latex2e.html#Colored-boxes).
There is a related version of `\makebox` that is used within the `picture` environment, where the length is given in terms of `\unitlength` (see [`\makebox` (picture)](https://latexref.xyz/dev/latex2e.html#g_t_005cmakebox-_0028picture_0029)).
As text is typeset in LR mode, neither a double backslash `\\` nor `\par` will give you a new line; for instance `\makebox{abc def \\ ghi}` outputs ‘abc defghi’ while `\makebox{abc def \par ghi}` outputs ‘abc def ghi’, both on a single line. To get multiple lines see [`\parbox`](https://latexref.xyz/dev/latex2e.html#g_t_005cparbox) and [`minipage`](https://latexref.xyz/dev/latex2e.html#minipage).
### 20.2 `\fbox` & `\framebox`
Synopses, one of:
```
\fbox{text}
\framebox{text}
\framebox[width]{text}
\framebox[width][position]{text}

```

Create a box with an enclosing frame, four rules surrounding the text. These commands are the same as `\mbox` and `\makebox` except for the frame (see [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox)). The `\fbox` command is robust, the `\framebox` command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
```
\fbox{Warning! No work shown, no credit given.}

```

LaTeX puts the text into a box, the text cannot be hyphenated. Around that box, separated from it by a small gap, are four rules making a frame.
The first two command invocations, `\fbox{...}` and `\framebox{...}`, are roughly the same. As to the third and fourth invocations, the optional arguments allow you to specify the box width as width and the position of the text inside that box as position. See [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox), for the full description but here is an example creating an empty box that is 1/4in wide.
```
\setlength{\fboxsep}{0pt}\framebox[0.25in]{\strut}}

```

The `\strut` ensures a total height of `\baselineskip` (see [`\strut`](https://latexref.xyz/dev/latex2e.html#g_t_005cstrut)).
These parameters determine the frame layout.

`\fboxrule`

The thickness of the rules around the enclosed box. The default is 0.2pt. Change it with a command such as `\setlength{\fboxrule}{0.8pt}` (see [`\setlength`](https://latexref.xyz/dev/latex2e.html#g_t_005csetlength)).

`\fboxsep`

The distance from the frame to the enclosed box. The default is 3pt. Change it with a command such as `\setlength{\fboxsep}{0pt}` (see [`\setlength`](https://latexref.xyz/dev/latex2e.html#g_t_005csetlength)). Setting it to 0pt is useful sometimes: this will put a frame around the picture with no white border.
```
{\setlength{\fboxsep}{0pt}%
 \framebox{%
   \includegraphics[width=0.5\textwidth]{prudence.jpg}}}

```

The extra curly braces keep the effect of the `\setlength` local.
As with `\mbox` and `\makebox`, LaTeX will not break lines in text. But this example has LaTeX break lines to make a paragraph, and then frame the result.
```
\framebox{%
  \begin{minipage}{0.6\linewidth}
    My dear, here we must run as fast as we can, just to stay in place.
    And if you wish to go anywhere you must run twice as fast as that.
  \end{minipage}}

```

See [Colored boxes](https://latexref.xyz/dev/latex2e.html#Colored-boxes), for colors other than black and white.
The `picture` environment has a version of the `\framebox` command where the units depend on `picture`’s `\unitlength` (see [`\framebox` (picture)](https://latexref.xyz/dev/latex2e.html#g_t_005cframebox-_0028picture_0029)).
### 20.3 `\parbox`
Synopses, one of:
```
\parbox{width}{contents}
\parbox[position]{width}{contents}
\parbox[position][height]{width}{contents}
\parbox[position][height][inner-pos]{width}{contents}

```

Produce a box of text that is width wide. Use this command to make a box of small pieces of text, of a single paragraph. This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
```
\begin{picture}(0,0)
  ...
  \put(1,2){\parbox{1.75in}{\raggedright Because the graph is a line on
                         this semilog paper, the relationship is
                         exponential.}}
\end{picture}

```

The contents are processed in a text mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)) so LaTeX will break lines to make a paragraph. But it won’t make multiple paragraphs; for that, use a `minipage` environment (see [`minipage`](https://latexref.xyz/dev/latex2e.html#minipage)).
The options for `\parbox` (except for contents) are the same as those for `minipage`. For convenience a summary of the options is here but see [`minipage`](https://latexref.xyz/dev/latex2e.html#minipage) for a complete description.
There are two required arguments. The width is a rigid length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). It sets the width of the box into which LaTeX typesets contents. The contents is the text that is placed in that box. It should not have any paragraph-making components.
There are three optional arguments, position, height, and inner-pos. The position gives the vertical alignment of the _parbox_ with respect to the surrounding material. The supported values are `c` or `m` to make the vertical center of the parbox lines up with the center of the adjacent text line (this is the default), or `t` to match the top line of the parbox with the baseline of the surrounding material, or `b` to match the bottom line.
The optional argument height overrides the natural height of the box.
The optional argument inner-pos controls the placement of content inside the `parbox`. Its default is the value of position. Its possible values are: `t` to put the content at the top of the box, `c` to put it in the vertical center, `b` to put it at the bottom of the box, and `s` to stretch it out vertically (for this, the text must contain vertically stretchable space).
### 20.4 `\raisebox`
Synopsis, one of:
```
\raisebox{distance}{text}
\raisebox{distance}[height]{text}
\raisebox{distance}[height][depth]{text}

```

Raise or lower text. This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
This example makes a command for denoting the restriction of a function by lowering the vertical bar symbol.
```
\newcommand*\restricted[1]{\raisebox{-.5ex}{$|$}_{#1}}
$f\restricted{A}$

```

The first mandatory argument distance specifies how far to raise the second mandatory argument text. This is a rigid length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). If it is negative then it lowers text. The text is processed in LR mode so it cannot contain line breaks (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)).
The optional arguments height and depth are dimensions. If they are specified, they override the natural height and depth of the box LaTeX gets by typesetting text.
In the arguments distance, height, and depth you can use the following lengths that refer to the dimension of the box that LaTeX gets on typesetting text: `\depth`, `\height`, `\width`, `\totalheight` (this is the box’s height plus its depth).
This will align two graphics on their top (see [Graphics](https://latexref.xyz/dev/latex2e.html#Graphics)).
```
\usepackage{graphicx,calc}  % in preamble
   ...
\begin{center}
  \raisebox{1ex-\height}{%
    \includegraphics[width=0.4\linewidth]{lion.png}}
  \qquad
  \raisebox{1ex-\height}{%
    \includegraphics[width=0.4\linewidth]{meta.png}}
\end{center}

```

The first `\height` is the height of lion.png while the second is the height of meta.png.
### 20.5 `\sbox` & `\savebox`
Synopsis, one of:
```
\sbox{box-cmd}{text}
\savebox{box-cmd}{text}
\savebox{box-cmd}[width]{text}
\savebox{box-cmd}[width][pos]{text}

```

Typeset text just as with `\makebox` (see [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox)) except that LaTeX does not output it but instead saves it in a box register referred to by a variable named box-cmd. The variable name box-cmd begins with a backslash, `\`. You must have previously allocated the box register box-cmd with `\newsavebox` (see [`\newsavebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewsavebox)). The `\sbox` command is robust while `\savebox` is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
This creates and uses a box register.
```
\newsavebox{\fullname}
\sbox{\fullname}{John Jacob Jingleheimer Schmidt}
  ...
\usebox{\fullname}! His name is my name, too!
Whenever we go out, the people always shout!
There goes \usebox{\fullname}!  Ya da da da da da da.

```

One advantage of using and reusing a box register over a `\newcommand` macro variable is efficiency, that LaTeX need not repeatedly retypeset the contents. See the example below.
The first two command invocations shown above, `\sbox{box-cmd}{text}` and `\savebox{box-cmd}{text}`, are roughly the same. As to the third and fourth, the optional arguments allow you to specify the box width as width, and the position of the text inside that box as position. See [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox), for the full description.
In the `\sbox` and `\savebox` commands the text is typeset in LR mode so it does not have line breaks (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). If you use these then LaTeX doesn’t give you an error but it ignores what you want: if you enter `\sbox{\newreg}{test \\ test}` and `\usebox{\newreg}` then you get ‘testtest’, while if you enter `\sbox{\newreg}{test \par test}` and `\usebox{\newreg}` then you get ‘test test’, but no error or warning. To fix this use a `\parbox` or `minipage` as here.
```
\newsavebox{\areg}
\savebox{\areg}{%
  \begin{minipage}{\linewidth}
    \begin{enumerate}
      \item First item
      \item Second item
    \end{enumerate}
  \end{minipage}}
  ...
\usebox{\areg}

```

As an example of the efficiency of reusing a register’s contents, this puts the same picture on each page of the document by putting it in the header. LaTeX only typesets it once.
```
\usepackage{graphicx}  % all this in the preamble
\newsavebox{\sealreg}
\savebox{\sealreg}{%
  \setlength{\unitlength}{1in}%
  \begin{picture}(0,0)%
     \put(1.5,-2.5){%
       \begin{tabular}{c}
          \includegraphics[height=2in]{companylogo.png} \\
          Office of the President
       \end{tabular}}
  \end{picture}%
}
\markright{\usebox{\sealreg}}
\pagestyle{headings}

```

The `picture` environment is good for fine-tuning the placement.
If the register `\noreg` has not already been defined then you get something like ‘Undefined control sequence. <argument> \noreg’.
### 20.6 `lrbox`
Synopsis:
```
\begin{lrbox}{box-cmd}
  text
\end{lrbox}

```

This is the environment form of the `\sbox` and `\savebox` commands, and is equivalent to them. See [`\sbox` & `\savebox`](https://latexref.xyz/dev/latex2e.html#g_t_005csbox-_0026-_005csavebox), for the full description.
The text inside the environment is saved in the box register referred to by variable `box-cmd`. The variable name box-cmd must begin with a backslash, `\`. You must allocate this box register in advance with `\newsavebox` (see [`\newsavebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewsavebox)). In this example the environment is convenient for entering the `tabular`.
```
\newsavebox{\jhreg}
\begin{lrbox}{\jhreg}
  \begin{tabular}{c}
    \includegraphics[height=1in]{jh.png} \\
    Jim Hef{}feron
  \end{tabular}
\end{lrbox}
  ...
\usebox{\jhreg}

```

### 20.7 `\usebox`
Synopsis:
```
\usebox{box-cmd}

```

Produce the box most recently saved in the box register box-cmd by the commands `\sbox` or `\savebox`, or the `lrbox` environment. For more information and examples, see [`\sbox` & `\savebox`](https://latexref.xyz/dev/latex2e.html#g_t_005csbox-_0026-_005csavebox). (Note that the variable name box-cmd starts with a backslash, `\`.) This command is robust (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
