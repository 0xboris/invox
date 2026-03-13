# 13 Counters

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 13.1 `\alph \Alph \arabic \roman \Roman \fnsymbol`: Printing counters
- 13.2 `\usecounter`
- 13.3 `\value`
- 13.4 `\setcounter`
- 13.5 `\addtocounter`
- 13.6 `\refstepcounter`
- 13.7 `\stepcounter`
- 13.8 `\day` & `\month` & `\year`

## 13 Counters
Everything LaTeX numbers for you has a counter associated with it. The name of the counter is often the same as the name of the environment or command associated with the number, except that the counter’s name has no backslash `\`. Thus, associated with the `\chapter` command is the `chapter` counter that keeps track of the chapter number.
Below is a list of the counters used in LaTeX’s standard document classes to control numbering.
```
part            paragraph       figure          enumi
chapter         subparagraph    table           enumii
section         page            footnote        enumiii
subsection      equation        mpfootnote      enumiv
subsubsection

```

The `mpfootnote` counter is used by the `\footnote` command inside of a minipage (see [`minipage`](https://latexref.xyz/dev/latex2e.html#minipage)). The counters `enumi` through `enumiv` are used in the `enumerate` environment, for up to four levels of nesting (see [`enumerate`](https://latexref.xyz/dev/latex2e.html#enumerate)).
Counters can have any integer value but they are typically positive.
New counters are created with `\newcounter`. See [`\newcounter`: Allocating a counter](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcounter).
  * [`\alph \Alph \arabic \roman \Roman \fnsymbol`: Printing counters](https://latexref.xyz/dev/latex2e.html#g_t_005calph-_005cAlph-_005carabic-_005croman-_005cRoman-_005cfnsymbol)
  * [`\usecounter`](https://latexref.xyz/dev/latex2e.html#g_t_005cusecounter)
  * [`\value`](https://latexref.xyz/dev/latex2e.html#g_t_005cvalue)
  * [`\setcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005csetcounter)
  * [`\addtocounter`](https://latexref.xyz/dev/latex2e.html#g_t_005caddtocounter)
  * [`\refstepcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005crefstepcounter)
  * [`\stepcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005cstepcounter)
  * [`\day` & `\month` & `\year`](https://latexref.xyz/dev/latex2e.html#g_t_005cday-_0026-_005cmonth-_0026-_005cyear)

### 13.1 `\alph \Alph \arabic \roman \Roman \fnsymbol`: Printing counters
Print the value of a counter, in a specified style. For instance, if the counter counter has the value 1 then a `\alph{counter}` in your source will result in a lowercase letter a appearing in the output.
All of these commands take a single counter as an argument, for instance, `\alph{enumi}`. Note that the counter name does not start with a backslash.

`\alph{counter}`

Print the value of counter in lowercase letters: ‘a’, ‘b’, ... If the counter’s value is less than 1 or more than 26 then you get ‘LaTeX Error: Counter too large.’

`\Alph{counter}`

Print in uppercase letters: ‘A’, ‘B’, ... If the counter’s value is less than 1 or more than 26 then you get ‘LaTeX Error: Counter too large.’

`\arabic{counter}`

Print in Arabic numbers such as ‘5’ or ‘-2’.

`\roman{counter}`

Print in lowercase roman numerals: ‘i’, ‘ii’, ... If the counter’s value is less than 1 then you get no warning or error but LaTeX does not print anything in the output.

`\Roman{counter}`

Print in uppercase roman numerals: ‘I’, ‘II’, ... If the counter’s value is less than 1 then you get no warning or error but LaTeX does not print anything in the output.

`\fnsymbol{counter}`

Prints the value of counter using a sequence of nine symbols that are traditionally used for labeling footnotes. The value of counter should be between 1 and 9, inclusive. If the counter’s value is less than 0 or more than 9 then you get ‘LaTeX Error: Counter too large’, while if it is 0 then you get no error or warning but LaTeX does not output anything.
Here are the symbols:
Number | Name | Command | Symbol
---|---|---|---
1 | asterisk | `\ast` | *
2 | dagger | `\dagger` | †
3 | ddagger | `\ddagger` | ‡
4 | section-sign | `\S` | §
5 | paragraph-sign | `\P` | ¶
6 | double-vert | `\parallel` | ‖
7 | double-asterisk | `\ast\ast` | **
8 | double-dagger | `\dagger\dagger` | ††
9 | double-ddagger | `\ddagger\ddagger` | ‡‡
### 13.2 `\usecounter`
Synopsis:
```
\usecounter{counter}

```

Used in the second argument of the `list` environment (see [`list`](https://latexref.xyz/dev/latex2e.html#list)), this declares that list items will be numbered by counter. It initializes counter to zero, and arranges that when `\item` is called without its optional argument then counter is incremented by `\refstepcounter`, making its value be the current `ref` value (see [`\ref`](https://latexref.xyz/dev/latex2e.html#g_t_005cref)). This command is fragile (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
Put in the document preamble, this example makes a new list environment enumerated with testcounter:
```
\newcounter{testcounter}
\newenvironment{test}{%
  \begin{list}{}{%
    \usecounter{testcounter}
  }
}{%
  \end{list}
}

```

### 13.3 `\value`
Synopsis:
```
\value{counter}

```

Expands to the value of the counter counter. (Note that the name of a counter does not begin with a backslash.)
This example outputs ‘Test counter is 6. Other counter is 5.’.
```
\newcounter{test} \setcounter{test}{5}
\newcounter{other} \setcounter{other}{\value{test}}
\addtocounter{test}{1}

Test counter is \arabic{test}.
Other counter is \arabic{other}.

```

The `\value` command is not used for typesetting the value of the counter. For that, see [`\alph \Alph \arabic \roman \Roman \fnsymbol`: Printing counters](https://latexref.xyz/dev/latex2e.html#g_t_005calph-_005cAlph-_005carabic-_005croman-_005cRoman-_005cfnsymbol).
It is often used in `\setcounter` or `\addtocounter` but `\value` can be used anywhere that LaTeX expects a number, such as in `\hspace{\value{foo}\parindent}`. It must not be preceded by `\protect` (see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect)).
This example inserts `\hspace{4\parindent}`.
```
\setcounter{myctr}{3} \addtocounter{myctr}{1}
\hspace{\value{myctr}\parindent}

```

### 13.4 `\setcounter`
Synopsis:
```
\setcounter{counter}{value}

```

Globally set the counter counter to have the value of the value argument, which must be an integer. Thus, you can set a counter’s value as `\setcounter{section}{5}`. Note that the counter name does not start with a backslash.
In this example if the counter `theorem` has value 12 then the second line will print ‘XII’.
```
\setcounter{exercise}{\value{theorem}}
Here it is in Roman: \Roman{exercise}.

```

### 13.5 `\addtocounter`
Synopsis:
```
\addtocounter{counter}{value}

```

Globally increment counter by the amount specified by the value argument, which may be negative.
In this example the section value appears as ‘VII’.
```
\setcounter{section}{5}
\addtocounter{section}{2}
Here it is in Roman: \Roman{section}.

```

### 13.6 `\refstepcounter`
Synopsis:
```
\refstepcounter{counter}

```

Globally increments the value of counter by one, as does `\stepcounter` (see [`\stepcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005cstepcounter)). The difference is that this command resets the value of any counter numbered within it. (For the definition of “counters numbered within”, see [`\newcounter`: Allocating a counter](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcounter).)
In addition, this command also defines the current `\ref` value to be the result of `\thecounter`.
While the counter value is set globally, the `\ref` value is set locally, i.e., inside the current group.
### 13.7 `\stepcounter`
Synopsis:
```
\stepcounter{counter}

```

Globally adds one to counter and resets all counters numbered within it. (For the definition of “counters numbered within”, see [`\newcounter`: Allocating a counter](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcounter).)
This command differs from `\refstepcounter` in that this one does not influence references; that is, it does not define the current `\ref` value to be the result of `\thecounter` (see [`\refstepcounter`](https://latexref.xyz/dev/latex2e.html#g_t_005crefstepcounter)).
### 13.8 `\day` & `\month` & `\year`
LaTeX defines the counter `\day` for the day of the month (nominally with value between 1 and 31), `\month` for the month of the year (nominally with value between 1 and 12), and `\year` for the year. When TeX starts up, they are set from the current values on the system. The related command `\today` produces a string representing the current day (see [`\today`](https://latexref.xyz/dev/latex2e.html#g_t_005ctoday)).
They counters are not updated as the job progresses so in principle they could be incorrect by the end. In addition, TeX does no sanity check:
```
\day=-2 \month=13 \year=-4 \today

```

gives no error or warning and results in the output ‘-2, -4’ (the bogus month value produces no output).
See [Command line input](https://latexref.xyz/dev/latex2e.html#Command-line-input), to force the date to a given value from the command line.
