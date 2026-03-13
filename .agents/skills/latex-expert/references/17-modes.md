# 17 Modes

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 17.1 `\ensuremath`

## 17 Modes
As LaTeX processes your document, at any point it is in one of six modes. They fall into three categories of two each, the horizontal modes, the math modes, and the vertical modes. Some commands only work in one mode or another (in particular, many commands only work in one of the math modes), and error messages will refer to these.
  * _Paragraph mode_ (in plain TeX this is called _horizontal mode_) is what LaTeX is in when processing ordinary text. It breaks the input text into lines and finds the positions of line breaks, so that in vertical mode page breaks can be done. This is the mode LaTeX is in most of the time.
_LR mode_ (for left-to-right mode; in plain TeX this is called _restricted horizontal mode_) is in effect when LaTeX starts making a box with an `\mbox` command. As in paragraph mode, LaTeX’s output is a string of words with spaces between them. Unlike in paragraph mode, in LR mode LaTeX never starts a new line, it just keeps going from left to right. (Although LaTeX will not complain that the LR box is too long, when it is finished and next tries to put that box into a line, it might well complain that the finished LR box won’t fit there.)
  * _Math mode_ is when LaTeX is generating an inline mathematical formula.
_Display math mode_ is when LaTeX is generating a displayed mathematical formula. (Displayed formulas differ somewhat from inline ones. One example is that the placement of the subscript on `\int` differs in the two situations.)
  * _Vertical mode_ is when LaTeX is building the list of lines and other material making the output page, which comprises insertion of page breaks. This is the mode LaTeX is in when it starts a document.
_Internal vertical mode_ is in effect when LaTeX starts making a `\vbox`. It has not such thing as page breaks, and as such is the vertical analogue of LR mode.

For instance, if you begin a LaTeX article with ‘Let \\( x \\) be ...’ then these are the modes: first LaTeX starts every document in vertical mode, then it reads the ‘L’ and switches to paragraph mode, then the next switch happens at the ‘\\(’ where LaTeX changes to math mode, and then when it leaves the formula it pops back to paragraph mode.
Paragraph mode has two subcases. If you use a `\parbox` command or a `minipage` then LaTeX is put into paragraph mode. But it will not put a page break here. Inside one of these boxes, called a _parbox_ , LaTeX is in _inner paragraph mode_. Its more usual situation, where it can put page breaks, is _outer paragraph mode_ (see [Page breaking](https://latexref.xyz/dev/latex2e.html#Page-breaking)).
  * [`\ensuremath`](https://latexref.xyz/dev/latex2e.html#g_t_005censuremath)

### 17.1 `\ensuremath`
Synopsis:
```
\ensuremath{formula}

```

Ensure that formula is typeset in math mode.
For instance, you can redefine commands that ordinarily can be used only in math mode, so that they can be used both in math and in plain text.
```
\newcommand{\dx}{\ensuremath{dx}}
In $\int f(x)\, \dx$, the \dx{} is an infinitesimal.

```

Caution: the `\ensuremath` command is useful but not a panacea.
```
\newcommand{\alf}{\ensuremath{\alpha}}
You get an alpha in text mode: \alf.
But compare the correct spacing in $\alf+\alf$ with that in \alf+\alf.

```

Best is to typeset math things in a math mode.
