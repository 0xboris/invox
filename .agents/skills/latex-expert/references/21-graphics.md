# 21 Graphics

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 21.1 `graphics` package options
- 21.2 `graphics` package configuration
- 21.3 Commands for graphics

## 21 Graphics
You can use graphics such as PNG or PDF files in your LaTeX document. You need an additional package, which comes standard with LaTeX. This example is the short how-to.
```
\include{graphicx}  % goes in the preamble
  ...
\includegraphics[width=0.5\linewidth]{plot.pdf}

```

To use the commands described here your document preamble must contain either `\usepackage{graphicx}` or `\usepackage{graphics}`. Most of the time, `graphicx` is the better choice.
Graphics come in two main types, raster and vector. LaTeX can use both. In raster graphics the file contains an entry for each location in an array, describing what color it is. An example is a photograph in JPG format. In vector graphics, the file contains a list of instructions such as ‘draw a circle with this radius and that center’. An example is a line drawing produced by the Asymptote program, in PDF format. Generally vector graphics are more useful because you can rescale their size without pixelation or other problems, and because they often have a smaller size.
There are systems particularly well-suited to make graphics for a LaTeX document. For example, these allow you to use the same fonts as in your document. LaTeX comes with a `picture` environment (see [`picture`](https://latexref.xyz/dev/latex2e.html#picture)) that has simple capabilities. Besides that, there are other ways to include the graphic-making commands in the document. Two such systems are the PSTricks and TikZ packages. There are also systems external to LaTeX, that generate a graphic that you include using the commands of this chapter. Two that use a programming language are Asymptote and MetaPost. One that uses a graphical interface is Xfig. Full description of these systems is outside the scope of this document; see their documentation on CTAN.
  * [`graphics` package options](https://latexref.xyz/dev/latex2e.html#Graphics-package-options)
  * [`graphics` package configuration](https://latexref.xyz/dev/latex2e.html#Graphics-package-configuration)
  * [Commands for graphics](https://latexref.xyz/dev/latex2e.html#Commands-for-graphics)

### 21.1 `graphics` package options
Synopsis (must be in the document preamble):
```
\usepackage[comma-separated option list]{graphics}

```

or
```
\usepackage[comma-separated option list]{graphicx}

```

The `graphicx` package has a format for optional arguments to the `\includegraphics` command that is convenient (it is the key-value format), so it is the better choice for new documents. When you load the `graphics` or `graphicx` package with `\usepackage` there are two kinds of available options.
The first is that LaTeX does not contain information about different output systems but instead depends on information stored in a _printer driver_ file. Normally you should not specify the driver option in the document, and instead rely on your system’s default. One advantage of this is that it makes the document portable across systems.
For completeness here is a list of the drivers. The currently relevant ones are: dvipdfmx, dvips, dvisvgm, luatex, pdftex, xetex. The two xdvi and oztex are essentially aliases for dvips (and xdvi is monochrome). Ones that should not be used for new systems are: dvipdf, dvipdfm, dviwin, dvipsone, emtex, pctexps, pctexwin, pctexhp, pctex32, truetex, tcidvi, vtex (and dviwindo is an alias for dvipsone). These are stored in files with a .def extension, such as pdftex.def.
The second kind of options are below.

`demo`

Instead of an image file, LaTeX puts in a 150 pt by 100 pt rectangle (unless another size is specified in the `\includegraphics` command).

`draft`

For each graphic file, it is not shown but instead its file name is printed in a box of the correct size. In order to determine the size, the file must be present.

`final`

(Default) Override any previous `draft` option, so that the document shows the contents of the graphic files.

`hiderotate`

Do not show rotated text. (This allows for the possibility that a previewer does not have the capability to rotate text.)

`hidescale`

Do not show scaled text. (This allows for the possibility that a previewer does not have the capability to scale.)

`hiresbb`

In a PS or EPS file the graphic size may be specified in two ways. The `%%BoundingBox` lines describe the graphic size using integer multiples of a PostScript point, that is, integer multiples of 1/72 inch. A later addition to the PostScript language allows decimal multiples, such as 1.23, in `%%HiResBoundingBox` lines. This option has LaTeX to read the size from the latter.
### 21.2 `graphics` package configuration
These commands configure the way LaTeX searches the file system for the graphic.
The behavior of file system search code is necessarily platform dependent. In this document we cover GNU/Linux, Macintosh, and Windows, as those systems are typically configured. For other situations consult the documentation in grfguide.pdf, or the LaTeX source, or your TeX distribution’s documentation.
  * [`\graphicspath`](https://latexref.xyz/dev/latex2e.html#g_t_005cgraphicspath)
  * [`\DeclareGraphicsExtensions`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareGraphicsExtensions)
  * [`\DeclareGraphicsRule`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareGraphicsRule)

#### 21.2.1 `\graphicspath`
Synopsis:
```
\graphicspath{list of directories inside curly braces}

```

Declare a list of directories to search for graphics files. This allows you to later say something like `\includegraphics{lion.png}` instead of having to give its path.
LaTeX always looks for graphic files first in the current directory (and the output directory, if specified; see [output directory](https://latexref.xyz/dev/latex2e.html#output-directory)). The declaration below tells the system to then look in the subdirectory pix, and then ../pix.
```
\usepackage{graphicx}   % or graphics; put in preamble
  ...
\graphicspath{ {pix/} {../pix/} }

```

The `\graphicspath` declaration is optional. If you don’t include it then LaTeX’s default is to search all of the places that it usually looks for a file (it uses LaTeX’s `\input@path`). In particular, in this case one of the places it looks is the current directory.
Enclose each directory name in curly braces; for example, above it says ‘`{pix}`’. Do this even if there is only one directory. Each directory name must end in a forward slash, /. This is true even on Windows, where good practice is to use forward slashes for all the directory separators since it makes the document portable to other platforms. If you have spaces in your directory name then use double quotes, as with `{"my docs/"}`. Getting one of these rules wrong will cause LaTeX to report `Error: File `filename' not found`.
Basically, the algorithm is that with this example, after looking in the current directory,
```
\graphicspath{ {pix/} {../pix/} }
...
\usepackage{lion.png}

```

for each of the listed directories, LaTeX concatenates it with the filename and searches for the result, checking for pix/lion.png and then ../pix/lion.png. This algorithm means that the `\graphicspath` command does not recursively search subdirectories: if you issue `\graphicspath{{a/}}` and the graphic is in a/b/lion.png then LaTeX will not find it. It also means that you can use absolute paths such as `\graphicspath{{/home/jim/logos/}}` or `\graphicspath{{C:/Users/Albert/Pictures/}}`. However, using these means that the document is not portable. (You could preserve portability by adjusting your TeX system settings configuration file parameter `TEXINPUTS`; see the documentation of your system.)
You can use `\graphicspath` anywhere in the document. You can use it more than once. Show its value with `\makeatletter\typeout{\Ginput@path}\makeatother`.
The directories are taken with respect to the base file. That is, suppose that you are working on a document based on book/book.tex and it contains `\include{chapters/chap1}`. If in chap1.tex you put `\graphicspath{{plots/}}` then LaTeX will not search for graphics in book/chapters/plots, but instead in book/plots.
#### 21.2.2 `\DeclareGraphicsExtensions`
Synopses:
```
\DeclareGraphicsExtensions{comma-separated list of file extensions}

```

Declare the filename extensions to try. This allows you to specify the order in which to choose graphic formats when you include graphic files by giving the filename without the extension, as in `\includegraphics{functionplot}`.
In this example, LaTeX will find files in the PNG format before PDF files.
```
\DeclareGraphicsExtensions{.png,PNG,.pdf,.PDF}
  ...
\includegraphics{lion}   % will find lion.png before lion.pdf

```

Because the filename lion does not have a period, LaTeX uses the extension list. For each directory in the graphics path (see [`\graphicspath`](https://latexref.xyz/dev/latex2e.html#g_t_005cgraphicspath)), LaTeX will try the extensions in the order given. If it does not find such a file after trying all the directories and extensions then it reports ‘! LaTeX Error: File `lion' not found’. Note that you must include the periods at the start of the extensions.
Because GNU/Linux and Macintosh filenames are case sensitive, the list of file extensions is case sensitive on those platforms. The Windows platform is not case sensitive.
You are not required to include `\DeclareGraphicsExtensions` in your document; the printer driver has a sensible default. For example, the most recent pdftex.def has this extension list.
```
.pdf,.png,.jpg,.mps,.jpeg,.jbig2,.jb2,.PDF,.PNG,.JPG,.JPEG,.JBIG2,.JB2

```

To change the order, use the `grfext` package.
You can use this command anywhere in the document. You can use it more than once. Show its value with `\makeatletter\typeout{\Gin@extensions}\makeatother`.
#### 21.2.3 `\DeclareGraphicsRule`
Synopsis:
```
\DeclareGraphicsRule{extension}{type}{size-file extension}{command}

```

Declare how to handle graphic files whose names end in extension.
This example declares that all files with names of the form filename-without-dot.mps will be treated as output from MetaPost, meaning that the printer driver will use its MetaPost-handling code to input the file.
```
\DeclareGraphicsRule{.mps}{mps}{.mps}{}

```

This
```
\DeclareGraphicsRule{*}{mps}{*}{}

```

tells LaTeX that it should handle as MetaPost output any file with an extension not covered by another rule, so it covers filename.1, filename.2, etc.
This describes the four arguments.

extension

The file extension to which this rule applies. The extension is anything after and including the first dot in the filename. Use the Kleene star, `*`, to denote the default behavior for all undeclared extensions.

type

The type of file involved. This type is a string that must be defined in the printer driver. For instance, files with extensions .ps, .eps, or .ps.gz may all be classed as type `eps`. All files of the same type will be input with the same internal command by the printer driver. For example, the file types that pdftex recognizes are: `jpg`, `jbig2`, `mps`, `pdf`, `png`, `tif`.

size-file extension

The extension of the file to be read to determine the size of the graphic, if there is such a file. It may be the same as extension but it may be different.
As an example, consider a PostScript graphic. To make it smaller, it might be compressed into a .ps.gz file. Compressed files are not easily read by LaTeX so you can put the bounding box information in a separate file. If size-file extension is empty then you must specify size information in the arguments of `\includegraphics`.
If the driver file has a procedure for reading size files for `type` then that will be used, otherwise it will use the procedure for reading .eps files. (Thus you may specify the size of bitmap files in a file with a PostScript style `%%BoundingBox` line if no other format is available.)

command

A command that will be applied to the file. This is often left empty. This command must start with a single backward quote. Thus, `\DeclareGraphicsRule{.eps.gz}{eps}{.eps.bb}{`gunzip -c #1}` specifies that any file with the extension .eps.gz should be treated as an `eps` file, with the BoundingBox information stored in the file with extension .eps.bb, and that the command `gunzip -c` will run on your platform to decompresses the file.
Such a command is specific to your platform. In addition, your TeX system must allow you to run external commands; as a security measure modern systems restrict running commands unless you explicitly allow it. See the documentation for your TeX distribution.
### 21.3 Commands for graphics
These are the commands available with the `graphics` and `graphicx` packages.
  * [`\includegraphics`](https://latexref.xyz/dev/latex2e.html#g_t_005cincludegraphics)
  * [`\rotatebox`](https://latexref.xyz/dev/latex2e.html#g_t_005crotatebox)
  * [`\scalebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cscalebox)
  * [`\resizebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cresizebox)

#### 21.3.1 `\includegraphics`
Synopses for `graphics` package:
```
\includegraphics{filename}
\includegraphics[urx,ury]{filename}
\includegraphics[llx,lly][urx,ury]{filename}
\includegraphics*{filename}
\includegraphics*[urx,ury]{filename}
\includegraphics*[llx,lly][urx,ury]{filename}

```

Synopses for `graphicx` package:
```
\includegraphics{filename}
\includegraphics[key-value list]{filename}
\includegraphics*{filename}
\includegraphics*[key-value list]{filename}

```

Include a graphics file. The starred form `\includegraphics*` will clip the graphic to the size specified, while for the unstarred form any part of the graphic that is outside the box of the specified size will over-print the surrounding area.
This
```
\usepackage{graphicx}  % in preamble
  ...
\begin{center}
  \includegraphics{plot.pdf}
\end{center}

```

will incorporate into the document the graphic in plot.pdf, centered and at its nominal size. You can also give a path to the file, as with `\includegraphics{graphics/plot.pdf}`. To specify a list of locations to search for the file, see [`\graphicspath`](https://latexref.xyz/dev/latex2e.html#g_t_005cgraphicspath).
If your filename includes spaces then put it in double quotes. An example is `\includegraphics{"sister picture.jpg"}`.
The `\includegraphics{filename}` command decides on the type of graphic by splitting filename on the first dot. You can instead use filename with no dot, as in `\includegraphics{turing}`, and then LaTeX tries a sequence of extensions such as `.png` and `.pdf` until it finds a file with that extension (see [`\DeclareGraphicsExtensions`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareGraphicsExtensions)).
If your file name contains dots before the extension then you can hide them with curly braces, as in `\includegraphics{{plot.2018.03.12.a}.pdf}`. Or, if you use the `graphicx` package then you can use the options `type` and `ext`; see below. This and other filename issues are also handled with the package grffile.
This example puts a graphic in a `figure` environment so LaTeX can move it to the next page if fitting it on the current page is awkward (see [`figure`](https://latexref.xyz/dev/latex2e.html#figure)).
```
\begin{figure}
  \centering
  \includegraphics[width=3cm]{lungxray.jpg}
  \caption{The evidence is overwhelming: don't smoke.}  \label{fig:xray}
\end{figure}

```

This places a graphic that will not float, so it is sure to appear at this point in the document even if makes LaTeX stretch the text or resort to blank areas on the page. It will be centered and will have a caption.
```
\usepackage{caption}  % in preamble
  ...
\begin{center}
  \includegraphics{pix/nix.png}
  \captionof{figure}{The spirit of the night} \label{pix:nix} % optional
\end{center}

```

This example puts a box with a graphic side by side with one having text, with the two vertically centered.
```
\newcommand*{\vcenteredhbox}[1]{\begin{tabular}{@{}c@{}}#1\end{tabular}}
  ...
\begin{center}
  \vcenteredhbox{\includegraphics[width=0.4\textwidth]{plot}}
  \hspace{1em}
  \vcenteredhbox{\begin{minipage}{0.4\textwidth}
                   \begin{displaymath}
                     f(x)=x\cdot \sin (1/x)
                   \end{displaymath}
                 \end{minipage}}
\end{center}

```

If you use the `graphics` package then the only options involve the size of the graphic (but see [`\rotatebox`](https://latexref.xyz/dev/latex2e.html#g_t_005crotatebox) and [`\scalebox`](https://latexref.xyz/dev/latex2e.html#g_t_005cscalebox)). When one optional argument is present then it is `[urx,ury]` and it gives the coordinates of the top right corner of the image, as a pair of TeX dimensions (see [Units of length](https://latexref.xyz/dev/latex2e.html#Units-of-length)). If the units are omitted they default to `bp`. In this case, the lower left corner of the image is assumed to be at (0,0). If two optional arguments are present then the leading one is `[llx,lly]`, specifying the coordinates of the image’s lower left. Thus, `\includegraphics[1in,0.618in]{...}` calls for the graphic to be placed so it is 1 inch wide and 0.618 inches tall and so its origin is at (0,0).
The `graphicx` package gives you many more options. Specify them in a key-value form, as here.
```
\begin{center}
  \includegraphics[width=1in,angle=90]{lion}
  \hspace{2em}
  \includegraphics[angle=90,width=1in]{lion}
\end{center}

```

The options are read left-to-right. So the first graphic above is made one inch wide and then rotated, while the second is rotated and then made one inch wide. Thus, unless the graphic is perfectly square, the two will end with different widths and heights.
There are many options. The primary ones are listed first.
Note that a graphic is placed by LaTeX into a box, which is traditionally referred to as its _bounding box_ (distinct from the PostScript BoundingBox described below). The graphic’s printed area may go beyond this box, or sit inside this box, but when LaTeX makes up a page it puts together boxes and this is the box allocated for the graphic.

`width`

The graphic will be shown so its bounding box is this width. An example is `\includegraphics[width=1in]{plot}`. You can use the standard TeX dimensions (see [Units of length](https://latexref.xyz/dev/latex2e.html#Units-of-length)) and also convenient is `\linewidth`, or in a two-column document, `\columnwidth` (see [Page layout parameters](https://latexref.xyz/dev/latex2e.html#Page-layout-parameters)). An example is that by using the calc package you can make the graphic be 1 cm narrower than the width of the text with `\includegraphics[width=\linewidth-1.0cm]{hefferon.jpg}`.

`height`

The graphic will be shown so its bounding box is this height. You can use the standard TeX dimensions (see [Units of length](https://latexref.xyz/dev/latex2e.html#Units-of-length)), and also convenient are `\pageheight` and `\textheight` (see [Page layout parameters](https://latexref.xyz/dev/latex2e.html#Page-layout-parameters)). For instance, the command `\includegraphics[height=0.25\textheight]{godel}` will make the graphic a quarter of the height of the text area.

`totalheight`

The graphic will be shown so its bounding box has this height plus depth. This differs from the height if the graphic was rotated. For instance, if it has been rotated by -90 then it will have zero height but a large depth.

`keepaspectratio`

If set to `true`, or just specified as here
```
\includegraphics[...,keepaspectratio,...]{...}

```

and you give as options both `width` and `height` (or `totalheight`), then LaTeX will make the graphic is as large as possible without distortion. That is, LaTeX will ensure that neither is the graphic wider than `width` nor taller than `height` (or `totalheight`).

`scale`

Factor by which to scale the graphic. To make a graphic twice its nominal size, enter `\includegraphics[scale=2.0]{...}`. This number may be any value; a number between 0 and 1 will shrink the graphic and a negative number will reflect it.

`angle`

Rotate the graphic. The angle is taken in degrees and counterclockwise. The graphic is rotated about its `origin`; see that option. For a complete description of how rotated material is typeset, see [`\rotatebox`](https://latexref.xyz/dev/latex2e.html#g_t_005crotatebox).

`origin`

The point of the graphic about which the rotation happens. Possible values are any string containing one or two of: `l` for left, `r` for right, `b` for bottom, `c` for center, `t` for top, and `B` for baseline. Thus, entering the command `\includegraphics[angle=180,origin=c]{moon}` will turn the picture upside down about that picture’s center, while the command `\includegraphics[angle=180,origin=lB]{LeBateau}` will turn its picture upside down about its left baseline. (The character `c` gives the horizontal center in `bc` or `tc`, but gives the vertical center in `lc` or `rc`.) The default is `lB`.
To rotate about an arbitrary point, see [`\rotatebox`](https://latexref.xyz/dev/latex2e.html#g_t_005crotatebox).
These are lesser-used options.

`viewport`

Pick out a subregion of the graphic to show. Takes four arguments, separated by spaces and given in TeX dimensions, as with `\includegraphics[.., viewport=0in 0in 1in 0.618in]{...}`. When the unit is omitted, the dimensions default to big points, `bp`. They are taken relative to the origin specified by the bounding box. See also the `trim` option.

`trim`

Gives parts of the graphic to not show. Takes four arguments, separated by spaces, that are given in TeX dimensions, as with `\includegraphics[.., trim= 0in 0.1in 0.2in 0.3in, ...]{...}`. These give the amounts of the graphic not to show, that is, LaTeX will crop the picture by 0 inches on the left, 0.1 inches on the bottom, 0.2 inches on the right, and 0.3 inches on the top. See also the `viewport` option.

`clip`

If set to `true`, or just specified as here
```
\includegraphics[...,clip,...]{...}

```

then the graphic is cropped to the bounding box. This is the same as using the starred form of the command, `\includegraphics*[...]{...}`.

`page`

Give the page number of a multi-page PDF file. The default is `page=1`.

`pagebox`

Specifies which bounding box to use for PDF files from among `mediabox`, `cropbox`, `bleedbox`, `trimbox`, or `artbox`. PDF files do not have the BoundingBox that PostScript files have, but may specify up to four predefined rectangles. The MediaBox gives the boundaries of the physical medium. The CropBox is the region to which the contents of the page are to be clipped when displayed. The BleedBox is the region to which the contents of the page should be clipped in production. The TrimBox is the intended dimensions of the finished page. The ArtBox is the extent of the page’s meaningful content. The driver will set the image size based on CropBox if present, otherwise it will not use one of the others, with a driver-defined order of preference. MediaBox is always present.

`interpolate`

Enable or disable interpolation of raster images by the viewer. Can be set with `interpolate=true` or just specified as here.
```
\includegraphics[...,interpolate,...]{...}

```

`quiet`

Do not write information to the log. You can set it with `quiet=true` or just specified it with `\includegraphics[...,quiet,...]{...}`,

`draft`

If you set it with `draft=true` or just specify it with
```
\includegraphics[...,draft,...]{...}

```

then the graphic will not appear in the document, possibly saving color printer ink. Instead, LaTeX will put an empty box of the correct size with the filename printed in it.
These options address the bounding box for Encapsulated PostScript graphic files, which have a size specified with a line `%%BoundingBox` that appears in the file. It has four values, giving the lower _x_ coordinate, lower _y_ coordinate, upper _x_ coordinate, and upper _y_ coordinate. The units are PostScript points, equivalent to TeX’s big points, 1/72 inch. For example, if an .eps file has the line `%%BoundingBox 10 20 40 80` then its natural size is 30/72 inch wide by 60/72 inch tall.

`bb`

Specify the bounding box of the displayed region. The argument is four dimensions separated by spaces, as with `\includegraphics[.., bb= 0in 0in 1in 0.618in]{...}`. Usually `\includegraphics` reads the BoundingBox numbers from the EPS file automatically, so this option is only useful if the bounding box is missing from that file or if you want to change it.

`bbllx, bblly, bburx, bbury`

Set the bounding box. These four are obsolete, but are retained for compatibility with old packages.

`natwidth, natheight`

An alternative for `bb`. Setting
```
\includegraphics[...,natwidth=1in,natheight=0.618in,...]{...}

```

is the same as setting `bb=0 0 1in 0.618in`.

`hiresbb`

If set to `true`, or just specified as with
```
\includegraphics[...,hiresbb,...]{...}

```

then LaTeX will look for `%%HiResBoundingBox` lines instead of `%%BoundingBox` lines. (The `BoundingBox` lines use only natural numbers while the `HiResBoundingBox` lines use decimals; both use units equivalent to TeX’s big points, 1/72 inch.) To override a prior setting of `true`, you can set it to `false`.
These following options allow a user to override LaTeX’s method of choosing the graphic type based on the filename extension. An example is that `\includegraphics[type=png,ext=.xyz,read=.xyz]{lion}` will read the file lion.xyz as though it were lion.png. For more on these, see [`\DeclareGraphicsRule`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareGraphicsRule).

`type`

Specify the graphics type.

`ext`

Specify the graphics extension. Only use this in conjunction with the option `type`.

`read`

Specify the file extension of the read file. Only use this in conjunction with the option `type`.

`command`

Specify a command to be applied to this file. Only use this in conjunction with the option `type`. See [Command line options](https://latexref.xyz/dev/latex2e.html#Command-line-options), for a discussion of enabling the `\write18` functionality to run external commands.
#### 21.3.2 `\rotatebox`
Synopsis if you use the `graphics` package:
```
\rotatebox{angle}{material}

```

Synopses if you use the `graphicx` package:
```
\rotatebox{angle}{material}
\rotatebox[key-value list]{angle}{material}

```

Put material in a box and rotate it angle degrees counterclockwise.
This example rotates the table column heads forty-five degrees.
```
\begin{tabular}{ll}
  \rotatebox{45}{Character} &\rotatebox{45}{NATO phonetic}   \\
  A                         &AL-FAH  \\
  B                         &BRAH-VOH
\end{tabular}

```

The material can be anything that goes in a box, including a graphic.
```
  \rotatebox[origin=c]{45}{\includegraphics[width=1in]{lion}}

```

To place the rotated material, the first step is that LaTeX sets material in a box, with a reference point on the left baseline. The second step is the rotation, by default about the reference point. The third step is that LaTeX computes a box to bound the rotated material. Fourth, LaTeX moves this box horizontally so that the left edge of this new bounding box coincides with the left edge of the box from the first step (they need not coincide vertically). This new bounding box, in its new position, is what LaTeX uses as the box when typesetting this material.
If you use the `graphics` package then the rotation is about the reference point of the box. If you use the `graphicx` package then these are the options that can go in the key-value list, but note that you can get the same effect without needing this package, except for the `x` and `y` options (see [`\includegraphics`](https://latexref.xyz/dev/latex2e.html#g_t_005cincludegraphics)).

`origin`

The point of the material’s box about which the rotation happens. Possible value is any string containing one or two of: `l` for left, `r` for right, `b` for bottom, `c` for center, `t` for top, and `B` for baseline. Thus, the first line here
```
\rotatebox[origin=c]{180}{moon}
\rotatebox[origin=lB]{180}{LeBateau}

```

will turn the picture upside down from the center while the second will turn its picture upside down about its left baseline. (The character `c` gives the horizontal center in `bc` or `tc` but gives the vertical center in `lc` or `rc`, and gives both in `c`.) The default is `lB`.

`x, y`

Specify an arbitrary point of rotation with `\rotatebox[x=TeX dimension,y=TeX dimension]{...}` (see [Units of length](https://latexref.xyz/dev/latex2e.html#Units-of-length)). These give the offset from the box’s reference point.

`units`

This key allows you to change the default of degrees counterclockwise. Setting `units=-360` changes the direction to degrees clockwise and setting `units=6.283185` changes to radians counterclockwise.
#### 21.3.3 `\scalebox`
Synopses:
```
\scalebox{horizontal factor}{material}
\scalebox{horizontal factor}[vertical factor]{material}
\reflectbox{material}

```

Scale the material.
This example halves the size, both horizontally and vertically, of the first text and doubles the size of the second.
```
\scalebox{0.5}{DRINK ME} and \scalebox{2.0}{Eat Me}

```

If you do not specify the optional vertical factor then it defaults to the same value as the horizontal factor.
You can use this command to resize a graphic, as here.
```
\scalebox{0.5}{\includegraphics{lion}}

```

If you use the `graphicx` package then you can accomplish the same thing with optional arguments to `\includegraphics` (see [`\includegraphics`](https://latexref.xyz/dev/latex2e.html#g_t_005cincludegraphics)).
The `\reflectbox` command abbreviates `\scalebox{-1}[1]{material}`. Thus, `Able was I\reflectbox{Able was I}` will show the phrase ‘Able was I’ immediately followed by its mirror reflection against a vertical axis.
#### 21.3.4 `\resizebox`
Synopses:
```
\resizebox{horizontal length}{vertical length}{material}
\resizebox*{horizontal length}{vertical length}{material}

```

Given a size, such as `3cm`, transform material to make it that size. If either horizontal length or vertical length is an exclamation point `!` then the other argument is used to determine a scale factor for both directions.
This example makes the graphic be a half inch wide and scales it vertically by the same factor to keep it from being distorted.
```
\resizebox{0.5in}{!}{\includegraphics{lion}}

```

The unstarred form `\resizebox` takes vertical length to be the box’s height while the starred form `\resizebox*` takes it to be height+depth. For instance, make the text have a height+depth of a quarter-inch with `\resizebox*{!}{0.25in}{\parbox{3.5in}{This box has both height and depth.}}`.
You can use `\depth`, `\height`, `\totalheight`, and `\width` to refer to the original size of the box. Thus, make the text two inches wide but keep the original height with `\resizebox{2in}{\height}{Two inches}`.
