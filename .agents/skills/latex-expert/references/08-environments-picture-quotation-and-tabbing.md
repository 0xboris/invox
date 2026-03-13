# 08 Environments: Picture, quotation, quote, and tabbing

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 8.19 `picture`
- 8.20 `quotation` & `quote`
- 8.21 `tabbing`

### 8.19 `picture`
Synopses:
```
\begin{picture}(width,height)
   picture command
\end{picture}

```

or
```
\begin{picture}(width,height)(xoffset,yoffset)
  picture command
\end{picture}

```

Where there may be any number of picture command’s.
An environment to create simple pictures containing lines, arrows, boxes, circles, and text. This environment is not obsolete, but new documents typically use much more powerful graphics creation systems, such as TikZ, PSTricks, MetaPost, or Asymptote. None of these are covered in this document; see CTAN.
To start, here’s an example showing the parallelogram law for adding vectors.
```
\setlength{\unitlength}{1cm}
\begin{picture}(6,6)      % picture box will be 6cm wide by 6cm tall
  \put(0,0){\vector(2,1){4}}  % for every 2 over this vector goes 1 up
    \put(2,1){\makebox(0,0)[l]{\ first leg}}
  \put(4,2){\vector(1,2){2}}
    \put(5,4){\makebox(0,0)[l]{\ second leg}}
  \put(0,0){\vector(1,1){6}}
    \put(3,3){\makebox(0,0)[r]{sum\ }}
\end{picture}

```

The `picture` environment has one required argument, a pair of positive real numbers (width,height). Multiply these by the value `\unitlength` to get the nominal size of the output, i.e. the space that LaTeX reserves on the output page. This nominal size need not be how large the picture really is; LaTeX will draw things from the picture outside the picture’s box.
This environment also has an optional argument (xoffset,yoffset). It is used to shift the origin. Unlike most optional arguments, this one is not contained in square brackets. As with the required argument, it consists of a pair of two real numbers, but these may also be negative or null. Multiply these by `\unitlength` to get the coordinates of the point at the lower-left corner of the picture.
For example, if `\unitlength` has been set to `1mm`, the command
```
\begin{picture}(100,200)(10,20)

```

produces a box of width 100 millimeters and height 200 millimeters. The picture’s origin is the point (10mm,20mm) and so the lower-left corner is there, and the upper-right corner is at (110mm,220mm). When you first draw a picture you typically omit the optional argument, leaving the origin at the lower-left corner. If you then want to modify your picture by shifting everything, you can just add the appropriate optional argument.
Each picture command tells LaTeX where to put something by providing its position. A _position_ is a pair such as `(2.4,-5)` giving the x- and y-coordinates. A _coordinate_ is a not a length, it is a real number (it may have a decimal point or a minus sign). It specifies a length in multiples of the unit length `\unitlength`, so if `\unitlength` has been set to `1cm`, then the coordinate `2.54` specifies a length of 2.54 centimeters.
LaTeX’s default for `\unitlength` is `1pt`. It is a rigid length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). Change it with the `\setlength` command (see [`\setlength`](https://latexref.xyz/dev/latex2e.html#g_t_005csetlength)). Make this change only outside of a `picture` environment.
The `picture` environment supports using standard arithmetic expressions as well as numbers.
Coordinates are given with respect to an origin, which is by default at the lower-left corner of the picture. Note that when a position appears as an argument, as with `\put(1,2){...}`, it is not enclosed in braces since the parentheses serve to delimit the argument. Also, unlike in some computer graphics systems, larger y-coordinates are further up the page, for example, _y = 1_ is _above_ _y = 0_.
There are four ways to put things in a picture: `\put`, `\multiput`, `\qbezier`, and `\graphpaper`. The most often used is `\put`. This
```
\put(11.3,-0.3){...}

```

places the object with its reference point at coordinates _(11.3,-0.3)_. The reference points for various objects will be described below.  The `\put` command creates an _LR box_ (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)). Anything that can go in an `\mbox` (see [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox)) can go in the text argument of the `\put` command. The reference point will be the lower left corner of the box. In this picture
```
\setlength{\unitlength}{1cm}
...\begin{picture}(1,1)
  \put(0,0){\line(1,0){1}}
  \put(0,0){\line(1,1){1}}
\end{picture}

```

the three dots are just slightly left of the point of the angle formed by the two lines. (Also, `\line(1,1){1}` does not call for a line of length one; rather the line has a change in the x coordinate of 1.)
The `\multiput`, `qbezier`, and `graphpaper` commands are described below.
You can also use this environment to place arbitrary material at an exact location. For example:
```
\usepackage{color,graphicx}  % in preamble
  ...
\begin{center}
\setlength{\unitlength}{\textwidth}
\begin{picture}(1,1)      % leave space, \textwidth wide and tall
  \put(0,0){\includegraphics[width=\textwidth]{desertedisland.jpg}}
  \put(0.25,0.35){\textcolor{red}{X Treasure here}}
\end{picture}
\end{center}

```

The red X will be precisely a quarter of the `\textwidth` from the left margin, and `0.35\textwidth` up from the bottom of the picture. Another example of this usage is to put similar code in the page header to get repeat material on each of a document’s pages.
  * [`\put`](https://latexref.xyz/dev/latex2e.html#g_t_005cput)
  * [`\multiput`](https://latexref.xyz/dev/latex2e.html#g_t_005cmultiput)
  * [`\qbezier`](https://latexref.xyz/dev/latex2e.html#g_t_005cqbezier)
  * [`\graphpaper`](https://latexref.xyz/dev/latex2e.html#g_t_005cgraphpaper)
  * [`\line`](https://latexref.xyz/dev/latex2e.html#g_t_005cline)
  * [`\linethickness`](https://latexref.xyz/dev/latex2e.html#g_t_005clinethickness)
  * [`\thinlines`](https://latexref.xyz/dev/latex2e.html#g_t_005cthinlines)
  * [`\thicklines`](https://latexref.xyz/dev/latex2e.html#g_t_005cthicklines)
  * [`\circle`](https://latexref.xyz/dev/latex2e.html#g_t_005ccircle)
  * [`\oval`](https://latexref.xyz/dev/latex2e.html#g_t_005coval)
  * [`\shortstack`](https://latexref.xyz/dev/latex2e.html#g_t_005cshortstack)
  * [`\vector`](https://latexref.xyz/dev/latex2e.html#g_t_005cvector)
  * [`\makebox` (picture)](https://latexref.xyz/dev/latex2e.html#g_t_005cmakebox-_0028picture_0029)
  * [`\framebox` (picture)](https://latexref.xyz/dev/latex2e.html#g_t_005cframebox-_0028picture_0029)
  * [`\frame`](https://latexref.xyz/dev/latex2e.html#g_t_005cframe)
  * [`\dashbox`](https://latexref.xyz/dev/latex2e.html#g_t_005cdashbox)

#### 8.19.1 `\put`
Synopsis:
```
\put(xcoord,ycoord){content}

```

Place content at the coordinate (xcoord,ycoord). See the discussion of coordinates and `\unitlength` in [`picture`](https://latexref.xyz/dev/latex2e.html#picture). The content is processed in LR mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)) so it cannot contain line breaks.
This includes the text into the `picture`.
```
\put(4.5,2.5){Apply the \textit{unpoke} move}

```

The reference point, the location (4.5,2.5), is the lower left of the text, at the bottom left of the ‘A’.
#### 8.19.2 `\multiput`
Synopsis:
```
\multiput(x,y)(delta_x,delta_y){num-copies}{obj}

```

Copy obj a total of num-copies times, with an increment of delta_x,delta_y. The obj first appears at position _(x,y)_ , then at _(x+\delta_x,y+\delta_y)_ , and so on.
This draws a simple grid with every fifth line in bold (see also [`\graphpaper`](https://latexref.xyz/dev/latex2e.html#g_t_005cgraphpaper)).
```
\begin{picture}(10,10)
  \linethickness{0.05mm}
  \multiput(0,0)(1,0){10}{\line(0,1){10}}
  \multiput(0,0)(0,1){10}{\line(1,0){10}}
  \linethickness{0.5mm}
  \multiput(0,0)(5,0){3}{\line(0,1){10}}
  \multiput(0,0)(0,5){3}{\line(1,0){10}}
\end{picture}

```

#### 8.19.3 `\qbezier`
Synopsis:
```
\qbezier(x1,y1)(x2,y2)(x3,y3)
\qbezier[num](x1,y1)(x2,y2)(x3,y3)

```

Draw a quadratic Bezier curve whose control points are given by the three required arguments `(x1,y1)`, `(x2,y2)`, and `(x3,y3)`. That is, the curve runs from (x1,y1) to (x3,y3), is quadratic, and is such that the tangent line at (x1,y1) passes through (x2,y2), as does the tangent line at (x3,y3).
This draws a curve from the coordinate (1,1) to (1,0).
```
\qbezier(1,1)(1.25,0.75)(1,0)

```

The curve’s tangent line at (1,1) contains (1.25,0.75), as does the curve’s tangent line at (1,0).
The optional argument num gives the number of calculated intermediate points. The default is to draw a smooth curve whose maximum number of points is `\qbeziermax` (change this value with `\renewcommand`).
This draws a rectangle with a wavy top, using `\qbezier` for that curve.
```
\begin{picture}(8,4)
  \put(0,0){\vector(1,0){8}}  % x axis
  \put(0,0){\vector(0,1){4}}  % y axis
  \put(2,0){\line(0,1){3}}       % left side
  \put(4,0){\line(0,1){3.5}}     % right side
  \qbezier(2,3)(2.5,2.9)(3,3.25)
    \qbezier(3,3.25)(3.5,3.6)(4,3.5)
  \thicklines                 % below here, lines are twice as thick
  \put(2,3){\line(4,1){2}}
  \put(4.5,2.5){\framebox{Trapezoidal Rule}}
\end{picture}

```

#### 8.19.4 `\graphpaper`
Synopsis:
```
\graphpaper(x_init,y_init)(x_dimen,y_dimen)
\graphpaper[spacing](x_init,y_init)(x_dimen,y_dimen)

```

Draw a coordinate grid. Requires the `graphpap` package. The grid’s origin is `(x_init,y_init)`. Grid lines come every spacing units (the default is 10). The grid extends x_dimen units to the right and y_dimen units up. All arguments must be positive integers.
This make a grid with seven vertical lines and eleven horizontal lines.
```
\usepackage{graphpap}    % in preamble
  ...
\begin{picture}(6,20)    % in document body
  \graphpaper[2](0,0)(12,20)
\end{picture}

```

The lines are numbered every ten units.
#### 8.19.5 `\line`
Synopsis:
```
\line(x_run,y_rise){travel}

```

Draw a line. It slopes such that it vertically rises y_rise for every horizontal x_run. The travel is the total horizontal change—it is not the length of the vector, it is the change in _x_. In the special case of vertical lines, where (x_run,y_rise)=(0,1), the travel gives the change in _y_.
This draws a line starting at coordinates (1,3).
```
\put(1,3){\line(2,5){4}}

```

For every over 2, this line will go up 5. Because travel specifies that this goes over 4, it must go up 10. Thus its endpoint is _(1,3)+(4,10)=(5,13)_. In particular, note that _travel =4_ is not the length of the line, it is the change in _x_.
The arguments x_run and y_rise are integers that can be positive, negative, or zero. (If both are 0 then LaTeX treats the second as 1.) With `\put(x_init,y_init){\line(x_run,y_rise){travel}}`, if x_run is negative then the line’s ending point has a first coordinate that is less than x_init. If y_rise is negative then the line’s ending point has a second coordinate that is less than y_init.
If travel is negative then you get `LaTeX Error: Bad \line or \vector argument.`
Standard LaTeX can only draw lines with a limited range of slopes because these lines are made by putting together line segments from pre-made fonts. The two numbers x_run and y_rise must have integer values from −6 through 6. Also, they must be relatively prime, so that (x_run,y_rise) can be (2,1) but not (4,2) (if you choose the latter then instead of lines you get sequences of arrowheads; the solution is to switch to the former). To get lines of arbitrary slope and plenty of other shapes in a system like `picture`, see the package `pict2e` (<https://ctan.org/pkg/pict2e>). Another solution is to use a full-featured graphics system such as TikZ, PSTricks, MetaPost, or Asymptote.
#### 8.19.6 `\linethickness`
Synopsis:
```
\linethickness{dim}

```

Declares the thickness of subsequent horizontal and vertical lines in a picture to be dim, which must be a positive length (see [Lengths](https://latexref.xyz/dev/latex2e.html#Lengths)). It differs from `\thinlines` and `\thicklines` in that it does not affect the thickness of slanted lines, circles, or ovals (see [`\oval`](https://latexref.xyz/dev/latex2e.html#g_t_005coval)).
#### 8.19.7 `\thinlines`
Declaration to set the thickness of subsequent lines, circles, and ovals in a picture environment to be 0.4pt. This is the default thickness, so this command is unnecessary unless the thickness has been changed with either [`\linethickness`](https://latexref.xyz/dev/latex2e.html#g_t_005clinethickness) or [`\thicklines`](https://latexref.xyz/dev/latex2e.html#g_t_005cthicklines).
#### 8.19.8 `\thicklines`
Declaration to set the thickness of subsequent lines, circles, and ovals in a picture environment to be 0.8pt. See also [`\linethickness`](https://latexref.xyz/dev/latex2e.html#g_t_005clinethickness) and [`\thinlines`](https://latexref.xyz/dev/latex2e.html#g_t_005cthinlines). This command is illustrated in the Trapezoidal Rule example of [`\qbezier`](https://latexref.xyz/dev/latex2e.html#g_t_005cqbezier).
#### 8.19.9 `\circle`
Synopsis:
```
\circle{diameter}
\circle*{diameter}

```

Produces a circle with a diameter as close as possible to the specified one. The `*` form produces a filled-in circle.
This draws a circle of radius 6, centered at `(5,7)`.
```
\put(5,7){\circle{6}}

```

The available radii for `\circle` are, in points, the even numbers from 2 to 20, inclusive. For `\circle*` they are all the integers from 1 to 15.
#### 8.19.10 `\oval`
Synopsis:
```
\oval(width,height)
\oval(width,height)[portion]

```

Produce a rectangle with rounded corners, hereinafter referred to as an _oval_. The optional argument portion allows you to produce only half or a quarter of the oval. For half an oval take portion to be one of these.

`t`

top half

`b`

bottom half

`r`

right half

`l`

left half
Produce only one quarter of the oval by setting portion to `tr`, `br`, `bl`, or `tl`.
This draws the top half of an oval that is 3 wide and 7 tall.
```
\put(5,7){\oval(3,7)[t]}

```

The (5,7) is the center of the entire oval, not just the center of the top half.
These shapes are not ellipses. They are rectangles whose corners are made with quarter circles. These circles have a maximum radius of 20pt (see [`\circle`](https://latexref.xyz/dev/latex2e.html#g_t_005ccircle) for the sizes). Thus large ovals are just frames with a small amount of corner rounding.
#### 8.19.11 `\shortstack`
Synopsis:
```
\shortstack[position]{line 1 \\ ... }

```

Produce a vertical stack of objects.
This labels the _y_ axis by writing the word ‘ _y_ ’ above the word ‘axis’.
```
\setlength{\unitlength}{1cm}
\begin{picture}(5,2.5)(-0.75,0)
   \put(0,0){\vector(1,0){4}}   % x axis
   \put(0,0){\vector(0,1){2}}   % y
   \put(-0.2,2){\makebox(0,0)[r]{\shortstack[r]{$y$\\ axis}}}
\end{picture}

```

For a short stack, the reference point is the lower left of the stack. In the above example the `\makebox` (see [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox)) puts the stack flush right in a zero width box so in total the short stack sits slightly to the left of the _y_ axis.
The valid positions are:

`r`

Make objects flush right

`l`

Make objects flush left

`c`

Center objects (default)
Separate objects into lines with `\\`. These stacks are short in that, unlike in a `tabular` or `array` environment, here the rows are not spaced out to be of even baseline skips. Thus, in `\shortstack{X\\o\\o\\X}` the first and last rows are taller than the middle two, and therefore the baseline skip between the two middle rows is smaller than that between the third and last row. You can adjust row heights and depths either by putting in the usual interline spacing with `\shortstack{X\\ \strut o\\o\\X}` (see [`\strut`](https://latexref.xyz/dev/latex2e.html#g_t_005cstrut)), or explicitly, via an zero-width box `\shortstack{X \\ \rule{0pt}{12pt} o\\o\\X}` or by using `\\`’s optional argument `\shortstack{X\\[2pt] o\\o\\X}`.
The `\shortstack` command is also available outside the `picture` environment.
#### 8.19.12 `\vector`
Synopsis:
```
\vector(x_run,y_rise){travel}

```

Draw a line ending in an arrow. The slope of that line is: it vertically rises y_rise for every horizontal x_run. The travel is the total horizontal change—it is not the length of the vector, it is the change in _x_. In the special case of vertical vectors, if (x_run,y_rise)=(0,1), then travel gives the change in _y_.
For an example see [`picture`](https://latexref.xyz/dev/latex2e.html#picture).
For elaboration on x_run and y_rise see [`\line`](https://latexref.xyz/dev/latex2e.html#g_t_005cline). As there, the values of x_run and y_rise are limited. For `\vector` you must chooses integers between −4 and 4, inclusive. Also, the two you choose must be relatively prime. Thus, `\vector(2,1){4}` is acceptable but `\vector(4,2){4}` is not (if you use the latter then you get a sequence of arrowheads).
#### 8.19.13 `\makebox` (picture)
Synopsis:
```
\makebox(rec-width,rec-height){text}
\makebox(rec-width,rec-height)[position]{text}

```

Make a box to hold text. This command fits with the `picture` environment, although you can use it outside of there, because rec-width and rec-height are numbers specifying distances in terms of the `\unitlength` (see [`picture`](https://latexref.xyz/dev/latex2e.html#picture)). This command is similar to the normal `\makebox` command (see [`\mbox` & `\makebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cmbox-_0026-_005cmakebox)) except here that you must specify the width and height. This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
This makes a box of length 3.5 times `\unitlength` and height 4 times `\unitlength`.
```
\put(1,2){\makebox(3.5,4){...}}

```

The optional argument `position` specifies where in the box the text appears. The default is to center it, both horizontally and vertically. To place it somewhere else, use a string with one or two of these letters.

`t`

Puts text the top of the box.

`b`

Put text at the bottom.

`l`

Put text on the left.

`r`

Put text on the right.
#### 8.19.14 `\framebox` (picture)
Synopsis:
```
\framebox(rec-width,rec-height){text}
\framebox(rec-width,rec-height)[position]{text}

```

This is the same as [`\makebox` (picture)](https://latexref.xyz/dev/latex2e.html#g_t_005cmakebox-_0028picture_0029) except that it puts a frame around the outside of the box that it creates. The reference point is the bottom left corner of the frame. This command fits with the `picture` environment, although you can use it outside of there, because lengths are numbers specifying the distance in terms of the `\unitlength` (see [`picture`](https://latexref.xyz/dev/latex2e.html#picture)). This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
This example creates a frame 2.5 inches by 3 inches and puts the text in the center.
```
\setlength{\unitlength}{1in}
\framebox(2.5,3){test text}

```

The required arguments are that the rectangle has overall width rect-width units and height rect-height units.
The optional argument position specifies the position of text; see [`\makebox` (picture)](https://latexref.xyz/dev/latex2e.html#g_t_005cmakebox-_0028picture_0029) for the values that it can take.
The rule has thickness `\fboxrule` and there is a blank space `\fboxsep` between the frame and the contents of the box.
For this command, you must specify the width and height. If you want to just put a frame around some contents whose dimension is determined in some other way then either use `\fbox` (see [`\fbox` & `\framebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cfbox-_0026-_005cframebox)) or `\frame` (see [`\frame`](https://latexref.xyz/dev/latex2e.html#g_t_005cframe)).
#### 8.19.15 `\frame`
Synopsis:
```
\frame{contents}

```

Puts a rectangular frame around contents. The reference point is the bottom left corner of the frame. In contrast to `\framebox` (see [`\framebox` (picture)](https://latexref.xyz/dev/latex2e.html#g_t_005cframebox-_0028picture_0029)), this command puts no extra space between the frame and the object. It is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
#### 8.19.16 `\dashbox`
Synopsis:
```
\dashbox{dash-len}(rect-width,rect-height){text}
\dashbox{dash-len}(rect-width,rect-height)[position]{text}

```

Create a dashed rectangle around text. This command fits with the `picture` environment, although you can use it outside of there, because lengths are numbers specifying the distance in terms of the `\unitlength` (see [`picture`](https://latexref.xyz/dev/latex2e.html#picture)).
The required arguments are: dashes are dash-len units long, with the same length gap, and the rectangle has overall width rect-width units and height rect-height units.
The optional argument position specifies the position of text; see [`\makebox` (picture)](https://latexref.xyz/dev/latex2e.html#g_t_005cmakebox-_0028picture_0029) for the values that it can take.
This shows that you can use non-integer value for dash-len.
```
\put(0,0){\dashbox{0.1}(5,0.5){My hovercraft is full of eels.}}

```

Each dash will be `0.1\unitlength` long, the box’s width is `5\unitlength` and its height is `0.5\unitlength`.
As in that example, a dashed box looks best when rect-width and rect-height are multiples of the dash-len.
### 8.20 `quotation` & `quote`
Synopsis:
```
\begin{quotation}
  text
\end{quotation}

```

or
```
\begin{quote}
  text
\end{quote}

```

Include a quotation. Both environments indent margins on both sides by `\leftmargin` (i.e., `\rightmargin` is set to `\leftmargin`), and the text is right-justified.
They differ in how they treat paragraphs:
  * In the `quotation` environment, paragraphs are indented by 1.5em and the space between paragraphs is small, `0pt plus 1pt`.
  * In the `quote` environment, paragraphs are not indented and the vertical space between paragraphs is the rubber length `\parsep` (see [`list`](https://latexref.xyz/dev/latex2e.html#list)).

Here is a quotation using the `quote` environment:
```
\begin{quote} \small\it
  Four score and seven years ago \ldots\

  Now we are engaged \ldots

  But, in a larger sense, \ldots

  \hspace{1em plus 1fill}---Abraham Lincoln
\end{quote}

```

Because it uses `quote`, there will be `\parsep` space between each paragraph, and the paragraphs won’t be indented. If we had used `quotation`, each paragraph would be indented and there would be only that small amount of stretch between paragraphs.
The `quote` and `quotation` environments are implemented as lists (see [`list`](https://latexref.xyz/dev/latex2e.html#list)). The `csquotes` and `quoting` packages provide additional functionality and parameters.
### 8.21 `tabbing`
Synopsis:
```
\begin{tabbing}
row1-col1 \= row1-col2 ...  \\
row2-col1 \> row2-col2 ...  \\
...
\end{tabbing}

```

Align text in columns, by setting tab stops and tabbing to them much as is done on a typewriter. This environment is less often used than the `tabular` (see [`tabular`](https://latexref.xyz/dev/latex2e.html#tabular)) and `array` (see [`array`](https://latexref.xyz/dev/latex2e.html#array)) environments, because in those the width of each column need not be known in advance.
  * [`tabbing` first example](https://latexref.xyz/dev/latex2e.html#tabbing-example)
  * [`tabbing` commands](https://latexref.xyz/dev/latex2e.html#tabbing-commands)
  * [`tabbing` complex examples](https://latexref.xyz/dev/latex2e.html#tabbing-complex-examples)

#### 8.21.1 `tabbing` first example
This first example sets the tab stops to explicit widths in the first line, which is ended by a `\kill` command to avoid typesetting anything (described further below):
```
\begin{tabbing}
\hspace{1.2in}\=\hspace{1in}\=\kill
Ship                \>Guns             \>Year    \\
\textit{Sophie}     \>14               \>1800    \\
\textit{Polychrest} \>24               \>1803    \\
\textit{Lively}     \>38               \>1804    \\
\textit{Surprise}   \>28               \>1805    \\
\end{tabbing}

```

The `tabbing` environment contains a sequence of _tabbed rows_. The first tabbed row begins immediately after `\begin{tabbing}` and each row ends with `\\` or `\kill`. The last row may omit the `\\` and end at the `\end{tabbing}`.
Both the `tabbing` environment and the more widely-used `tabular` environment put text in columns. The most important distinction is that in `tabular` the width of columns is determined automatically by LaTeX, while in `tabbing` the user sets the tab stops. Another distinction is that `tabular` generates a box that cannot be broken, but `tabbing` can be broken across pages. Finally, while `tabular` can be used in any mode, `tabbing` can be used only in paragraph mode and it always starts a new paragraph, without indentation.
As shown in the example above, there is no need to use the starred form of the `\hspace` command (see [`\hspace`](https://latexref.xyz/dev/latex2e.html#g_t_005chspace)) at the beginning of a tabbed row. The right margin of the `tabbing` environment is the end of line, so that the width of the environment is `\linewidth`.
#### 8.21.2 `tabbing` commands
The best overall description of the tabbing environment commands we know is in Leslie Lamport’s original reference manual, section C.10.1 of \LaTeX: A Document Preparation System. A summary of the commands follows.
In general, at any point the `tabbing` environment has a _current tab stop pattern_ : a sequence of _n > 0_ tab stops, numbered 0, 1, etc. Each tab stop creates a corresponding column. Tab stop 0 is always the left margin, defined by the enclosing environment. Tab stop number i is set if it is assigned a horizontal position on the page. Tab stop number i can only be set if all the stops 0, …, _i-1_ have already been set; normally later stops are to the right of earlier ones.
By default any text typeset in a `tabbing` environment is typeset ragged right and left-aligned on the current tab stop. Typesetting is done in LR mode (see [Modes](https://latexref.xyz/dev/latex2e.html#Modes)).
The following commands can be used inside a `tabbing` environment. They are all fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).

`\\ (tabbing)`

End a tabbed line and typeset it.

`\= (tabbing)`

Set a tab stop at the current position.

`\> (tabbing)`

Advance to the next tab stop.

`\+ (tabbing)`

Move the left margin of the next and all the following commands one tab stop to the right, beginning a tabbed line if necessary.

`\< (tabbing)`

Put following text to the left of the local margin (without changing the margin). Can only be used at the start of a line, and a preceding line must have used `\+`.

`\- (tabbing)`

Move the left margin of the next and all following lines one tab stop to the left (undoing one `\+`). Does not change the current line.

`\' (tabbing)`

Move everything in the current column so far, i.e., everything from the most recent `\>`, `\<`, `\'`, `\\`, or `\kill` command, to the previous column and aligned to the right, flush against the current column’s tab stop.

`\` (tabbing)`

Move all the text following, up to the `\\` or `\end{tabbing}` command that ends the line, to the right margin of the `tabbing` environment. There must be no `\>` or `\'` command between the `\`` and the `\\` or `\end{tabbing}` command that ends the line.
This allows you to put text flush right against any tab stop, including tab stop 0. However, it can’t move text to the right of the last column because there’s no tab stop there.

`\a (tabbing)`

In a `tabbing` environment, the commands `\=`, `\'` and `\`` do not produce accents as usual (see [Accents](https://latexref.xyz/dev/latex2e.html#Accents)). Instead, use the commands `\a=`, `\a'` and `\a``.

`\kill (tabbing)`

Sets tab stops without producing text. Works just like `\\` except that it throws away the current line instead of producing output for it. Any `\=`, `\+` or `\-` commands in that line remain in effect.

`\poptabs`

Restores the tab stop positions saved by the last `\pushtabs`.

`\pushtabs`

Saves all current tab stop positions. Useful for temporarily changing tab stop positions in the middle of a `tabbing` environment.

`\tabbingsep`

Distance of the text moved by `\'` to left of current tab stop; its default value is `\labelsep` (see [list labelsep](https://latexref.xyz/dev/latex2e.html#list-labelsep)).
#### 8.21.3 `tabbing` complex examples
Here is a simple example using the (rather confusing) `\<` command, along with `\+` and `\-`:
```
\begin{tabbing}
\hspace{1in}\=\hspace{1in}\=\kill
\+ \> A   \\ % change left margin to second tab stop
\< left   \\ % but typeset "left" at first tab stop
\- B \> C \\ % return to normal left margin on next line
D \> E    \\
\end{tabbing}

```

The output looks like this (except not in typewriter):
```
      A
left
      B   C
D     E

```

This last example typesets a Pascal function (in typewriter), defining new tab stops and using `\+` and `\-` for the different indentation levels:
```
{\tt \frenchspacing \begin{tabbing}
function \= fact(n : integer) : integer;\\
         \> begin \= \+ \\
               \> if \= n > 1 then \+ \\
                        fact := n * fact(n-1) \- \\
                  else \+ \\
                        fact := 1; \-\- \\
            end;\\
\end{tabbing}
}

```

The output looks like this:
```
function fact(n : integer) : integer;
         begin
               if n > 1 then
                  fact := n * fact(n-1);
               else
                  fact := 1;
         end;

```

This example is just for illustration of the environment. To actually typeset computer code in typewriter like this, a verbatim environment (see [`verbatim`](https://latexref.xyz/dev/latex2e.html#verbatim)) would normally be best. For pretty-printed (not typewriter) code, there are quite a few packages, including `algorithm2e`, `fancyvrb`, `listings`, and `minted`.
