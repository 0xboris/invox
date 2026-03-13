# 26 Letters

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 26.1 `\address`
- 26.2 `\cc`
- 26.3 `\closing`
- 26.4 `\encl`
- 26.5 `\location`
- 26.6 `\makelabels`
- 26.7 `\name`
- 26.8 `\opening`
- 26.9 `\ps`
- 26.10 `\signature`
- 26.11 `\telephone`

## 26 Letters
Synopsis:
```
\documentclass{letter}
\address{senders address}   % return address
\signature{sender name}
\begin{document}
\begin{letter}{recipient address}
\opening{salutation}
  letter body
\closing{closing text}
\end{letter}
   ...
\end{document}

```

Produce one or more letters.
Each letter is in a separate `letter` environment, whose argument recipient address often contains multiple lines separated with a double backslash, (`\\`). For example, you might have:
```
 \begin{letter}{Ninon de l'Enclos \\
                l'h\^otel Sagonne}
   ...
 \end{letter}

```

The start of the `letter` environment resets the page number to 1, and the footnote number to 1 also.
The sender address and sender name are common to all of the letters, whether there is one or more, so these are best put in the preamble. As with the recipient address, often sender address contains multiple lines separated by a double backslash (`\\`). LaTeX will put the sender name under the closing, after a vertical space for the traditional hand-written signature.
Each `letter` environment body begins with a required `\opening` command such as `\opening{Dear Madam or Sir:}`. The letter body text is ordinary LaTeX so it can contain everything from enumerated lists to displayed math, except that commands such as `\chapter` that make no sense in a letter are turned off. Each `letter` environment body typically ends with a `\closing` command such as `\closing{Yours,}`.
Additional material may come after the `\closing`. You can say who is receiving a copy of the letter with a command like `\cc{the Boss \\ the Boss's Boss}`. There’s a similar `\encl` command for a list of enclosures. And, you can add a postscript with `\ps`.
LaTeX’s default is to indent the sender name and the closing above it by a length of `\longindentation`. By default this is `0.5\textwidth`. To make them flush left, put `\setlength{\longindentation}{0em}` in your preamble.
To set a fixed date use something like `\renewcommand{\today}{1958-Oct-12}`. If put in your preamble then it will apply to all the letters.
This example shows only one `letter` environment. The three lines marked as optional are typically omitted.
```
\documentclass{letter}
\address{Sender's street \\ Sender's town}
\signature{Sender's name \\ Sender's title}
% optional: \location{Mailbox 13}
% optional: \telephone{(102) 555-0101}
\begin{document}
\begin{letter}{Recipient's name \\ Recipient's address}
\opening{Sir:}
% optional: \thispagestyle{firstpage}
I am not interested in entering a business arrangement with you.
\closing{Your most humble, etc.,}
\end{letter}
\end{document}

```

These commands are used with the `letter` class.
  * [`\address`](https://latexref.xyz/dev/latex2e.html#g_t_005caddress)
  * [`\cc`](https://latexref.xyz/dev/latex2e.html#g_t_005ccc)
  * [`\closing`](https://latexref.xyz/dev/latex2e.html#g_t_005cclosing)
  * [`\encl`](https://latexref.xyz/dev/latex2e.html#g_t_005cencl)
  * [`\location`](https://latexref.xyz/dev/latex2e.html#g_t_005clocation)
  * [`\makelabels`](https://latexref.xyz/dev/latex2e.html#g_t_005cmakelabels)
  * [`\name`](https://latexref.xyz/dev/latex2e.html#g_t_005cname)
  * [`\opening`](https://latexref.xyz/dev/latex2e.html#g_t_005copening)
  * [`\ps`](https://latexref.xyz/dev/latex2e.html#g_t_005cps)
  * [`\signature`](https://latexref.xyz/dev/latex2e.html#g_t_005csignature)
  * [`\telephone`](https://latexref.xyz/dev/latex2e.html#g_t_005ctelephone)

### 26.1 `\address`
Synopsis:
```
\address{senders address}

```

Specify the return address, as it appears on the letter and on the envelope. Separate multiple lines in senders address with a double backslash, `\\`.
Because it can apply to multiple letters this declaration is often put in the preamble. However, it can go anywhere, including inside an individual `letter` environment.
This command is optional: if you do not use it then the letter is formatted with some blank space on top, for copying onto pre-printed letterhead paper. If you do use the `\address` declaration then it is formatted as a personal letter.
Here is an example.
```
\address{Stephen Maturin \\
         The Grapes of the Savoy}

```

### 26.2 `\cc`
Synopsis:
```
\cc{name0 \\
     ... }

```

Produce a list of names to which copies of the letter were sent. This command is optional. If it appears then typically it comes after `\closing`. Put the names on different lines by separating them with a double backslash, `\\`, as in:
```
\cc{President \\
    Vice President}

```

### 26.3 `\closing`
Synopsis:
```
\closing{text}

```

Produce the letter’s closing. This is optional, but usual. It appears at the end of a letter, above a handwritten signature. For example:
```
\closing{Regards,}

```

### 26.4 `\encl`
Synopsis:
```
\encl{first enclosed object \\
       ... }

```

Produce a list of things included with the letter. This command is optional; when it is used, it typically is put after `\closing`. Separate multiple lines with a double backslash, `\\`.
```
\encl{License \\
      Passport}

```

### 26.5 `\location`
Synopsis:
```
\location{text}

```

The text appears centered at the bottom of the page. It only appears if the page style is `firstpage`.
### 26.6 `\makelabels`
Synopsis:
```
\makelabels   % in preamble

```

Optional, for a document that contains `letter` environments. If you just put `\makelabels` in the preamble then at the end of the document you will get a sheet with labels for all the recipients, one for each letter environment, that you can copy to a sheet of peel-off address labels.
Customize the labels by redefining the commands `\startlabels`, `\mlabel`, and `\returnaddress` (and perhaps `\name`) in the preamble. The command `\startlabels` sets the width, height, number of columns, etc., of the page onto which the labels are printed. The command `\mlabel{return address}{recipient address}` produces the two labels (or one, if you choose to ignore the return address) for each letter environment. The first argument, return address, is the value returned by the macro `\returnaddress`. The second argument, recipient address, is the value passed in the argument to the `letter` environment. By default `\mlabel` ignores the first argument, the return address, causing the default behavior described in the prior paragraph.
This illustrates customization. Its output includes a page with two columns having two labels each.
```
\documentclass{letter}
\renewcommand*{\returnaddress}{Fred McGuilicuddy \\
                               Oshkosh, Mineola 12305}
\newcommand*\originalMlabel{}
\let\originalMlabel\mlabel
\def\mlabel#1#2{\originalMlabel{}{#1}\originalMlabel{}{#2}}
\makelabels
  ...
\begin{document}
\begin{letter}{A Einstein \\
               112 Mercer Street \\
               Princeton, New Jersey, USA 08540}
  ...
\end{letter}
\begin{letter}{K G\"odel \\
               145 Linden Lane \\
               Princeton, New Jersey, USA 08540}
  ...
\end{letter}
\end{document}

```

The first column contains the return address twice. The second column contains the address for each recipient.
The package `envlab` makes formatting the labels easier, with standard sizes already provided. The preamble lines `\usepackage[personalenvelope]{envlab}` and `\makelabels` are all that you need to print envelopes.
### 26.7 `\name`
Synopsis:
```
\name{name}

```

Optional. Sender’s name, used for printing on the envelope together with the return address.
### 26.8 `\opening`
Synopsis:
```
\opening{salutation}

```

Required. Follows the `\begin{letter}{...}`. The argument salutation is mandatory. For instance:
```
\opening{Dear John:}

```

### 26.9 `\ps`
Synopsis:
```
\ps{text}

```

Add a postscript. This command is optional and usually is used after `\closing`.
```
\ps{P.S. After you have read this letter, burn it. Or eat it.}

```

### 26.10 `\signature`
Synopsis:
```
\signature{first line \\
            ... }

```

The sender’s name. This command is optional, although its inclusion is usual.
The argument text appears at the end of the letter, after the closing. LaTeX leaves some vertical space for a handwritten signature. Separate multiple lines with a double backslash, `\\`. For example:
```
\signature{J Fred Muggs \\
           White House}

```

LaTeX’s default for the vertical space from the `\closing` text down to the `\signature` text is `6\medskipamount`, which is six times `\medskipamount` (where `\medskipamount` is equal to a `\parskip`, which in turn is defined by default here to 0.7em).
This command is usually in the preamble, to apply to all the letters in the document. To have it apply to one letter only, put it inside a `letter` environment and before the `\closing`.
You can include a graphic in the signature as here.
```
\signature{\vspace{-6\medskipamount}\includegraphics{sig.png}\\
             My name}

```

For this you must put `\usepackage{graphicx}` in the preamble (see [Graphics](https://latexref.xyz/dev/latex2e.html#Graphics)).
### 26.11 `\telephone`
Synopsis:
```
\telephone{number}

```

The sender’s telephone number. This is typically in the preamble, where it applies to all letters. This only appears if the `firstpage` pagestyle is selected. If so, it appears on the lower right of the page.
