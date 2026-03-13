# 12 Definitions: Class and package commands

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 12.14.1 `\AtBeginDvi` & `\AtEndDvi`
- 12.14.2 `\AtEndOfClass` & `\AtEndOfPackage`
- 12.14.3 `\CheckCommand`
- 12.14.4 `\ClassError` and `\PackageError` and other messages
- 12.14.5 `\CurrentOption`
- 12.14.6 `\DeclareOption`
- 12.14.7 `\DeclareRobustCommand`
- 12.14.8 `\ExecuteOptions`
- 12.14.9 `\IfFileExists` & `\InputIfFileExists`
- 12.14.10 `\LoadClass` & `\LoadClassWithOptions`
- 12.14.11 `\NeedsTeXFormat`
- 12.14.12 `\OptionNotUsed`
- 12.14.13 `\PassOptionsToClass` & `\PassOptionsToPackage`
- 12.14.14 `\ProcessOptions`
- 12.14.15 `\ProvidesClass` & `\ProvidesPackage`
- 12.14.16 `\ProvidesFile`
- 12.14.17 `\RequirePackage` & `\RequirePackageWithOptions`

### 12.14 Class and package commands
These are commands designed to help writers of classes or packages.
  * [`\AtBeginDvi` & `\AtEndDvi`](https://latexref.xyz/dev/latex2e.html#g_t_005cAtBeginDvi-_0026-_005cAtEndDvi)
  * [`\AtEndOfClass` & `\AtEndOfPackage`](https://latexref.xyz/dev/latex2e.html#g_t_005cAtEndOfClass-_0026-_005cAtEndOfPackage)
  * [`\CheckCommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cCheckCommand)
  * [`\ClassError` and `\PackageError` and other messages](https://latexref.xyz/dev/latex2e.html#g_t_005cClassError-and-_005cPackageError-and-other-messages)
  * [`\CurrentOption`](https://latexref.xyz/dev/latex2e.html#g_t_005cCurrentOption)
  * [`\DeclareOption`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareOption)
  * [`\DeclareRobustCommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cDeclareRobustCommand)
  * [`\ExecuteOptions`](https://latexref.xyz/dev/latex2e.html#g_t_005cExecuteOptions)
  * [`\IfFileExists` & `\InputIfFileExists`](https://latexref.xyz/dev/latex2e.html#g_t_005cIfFileExists-_0026-_005cInputIfFileExists)
  * [`\LoadClass` & `\LoadClassWithOptions`](https://latexref.xyz/dev/latex2e.html#g_t_005cLoadClass-_0026-_005cLoadClassWithOptions)
  * [`\NeedsTeXFormat`](https://latexref.xyz/dev/latex2e.html#g_t_005cNeedsTeXFormat)
  * [`\OptionNotUsed`](https://latexref.xyz/dev/latex2e.html#g_t_005cOptionNotUsed)
  * [`\PassOptionsToClass` & `\PassOptionsToPackage`](https://latexref.xyz/dev/latex2e.html#g_t_005cPassOptionsToClass-_0026-_005cPassOptionsToPackage)
  * [`\ProcessOptions`](https://latexref.xyz/dev/latex2e.html#g_t_005cProcessOptions)
  * [`\ProvidesClass` & `\ProvidesPackage`](https://latexref.xyz/dev/latex2e.html#g_t_005cProvidesClass-_0026-_005cProvidesPackage)
  * [`\ProvidesFile`](https://latexref.xyz/dev/latex2e.html#g_t_005cProvidesFile)
  * [`\RequirePackage` & `\RequirePackageWithOptions`](https://latexref.xyz/dev/latex2e.html#g_t_005cRequirePackage-_0026-_005cRequirePackageWithOptions)

#### 12.14.1 `\AtBeginDvi` & `\AtEndDvi`
Synopsis:
```
\AtBeginDvi{code}
\AtEndDvi{code}

```

`\AtBeginDvi` saves, in a box register, code to be executed at the beginning of the shipout of the first page of the document. Despite the name, it applies to DVI, PDF, and XDV output. It fills the `shipout/firstpage` hook; new code should use that hook directly.
Similarly, `\AtEndDvi` (previously available only with the `atenddvi` package) is code executed when finalizing the main output document.
#### 12.14.2 `\AtEndOfClass` & `\AtEndOfPackage`
Synopses:
```
\AtEndOfClass{code}
\AtEndOfPackage{code}

```

Hooks to insert code to be executed when LaTeX finishes processing the current class resp. package.
These hooks can be used multiple times; each `code` segment will be executed in the order called. Many packages and classes use these commands.
See also [`\AtBeginDocument`](https://latexref.xyz/dev/latex2e.html#g_t_005cAtBeginDocument).
#### 12.14.3 `\CheckCommand`
Synopsis:
```
\CheckCommand{cmd}[num][default]{definition}
\CheckCommand* (same parameters)

```

Like `\newcommand` (see [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand)) but does not define cmd; instead it checks that the current definition of cmd is exactly as given by definition and is or is not _`\long`_as expected. A long command is a command that accepts`\par` within an argument.
With the unstarred version of `\CheckCommand`, cmd is expected to be `\long`; with the starred version, cmd must not be `\long`
`\CheckCommand` raises an error when the check fails. This allows you to check before you start redefining `cmd` yourself that no other package has already redefined this command.
#### 12.14.4 `\ClassError` and `\PackageError` and other messages
Produce error, warning, and informational messages for classes:

`\ClassError{class name}{error-text}{help-text}`

`\ClassWarning{class name}{warning-text}`

`\ClassWarningNoLine{class name}{warning-text}`

`\ClassNote{class name}{note-text}`

`\ClassNoteNoLine{class name}{note-text}`

`\ClassInfo{class name}{log-text}`

and the same for packages:

`\PackageError{package name}{error-text}{help-text}`

`\PackageWarning{package name}{warning-text}`

`\PackageWarningNoLine{package name}{warning-text}`

`\PackageNote{package name}{note-text}`

`\PackageNoteNoLine{package name}{note-text}`

`\PackageInfo{package name}{log-text}`

For `\ClassError` and `\PackageError` the message is error-text, followed by TeX’s ‘`?`’ error prompt. If the user then asks for help by typing `h`, they see help-text.
The four `Warning` commands write warning-text on the terminal and log file (with no error prompt), prefixed by the text ‘Warning:’.
The four `Note` commands also write the note-text to the terminal and log file, without the ‘Warning:’ prefix.
The `NoLine` versions omit the number of the input line generating the message, while the other versions do show that number.
The two `Info` commands write log-text only in the transcript file and not to the terminal.
To format the messages, including the help-text: use `\protect` to stop a command from expanding, get a line break with `\MessageBreak`, and get a space with `\space` where a space character is ignored, most commonly after a command.
LaTeX appends a period to the messages.
#### 12.14.5 `\CurrentOption`
Expands to the name of the option currently being processed. This can only be used within the code argument of either `\DeclareOption` or `\DeclareOption*`.
#### 12.14.6 `\DeclareOption`
Synopsis:
```
\DeclareOption{option}{code}
\DeclareOption*{option}{code}

```

Define an option a user can include in their `\documentclass` command. For example, a class `smcmemo` could have an option `logo` allowing users to put the institutional logo on the first page. The document would start with `\documentclass[logo]{smcmemo}`. To enable this, the class file must contain `\DeclareOption{logo}{code}` (and later, `\ProcessOptions`).
If you request an option that has not been declared, by default this will produce a warning like `Unused global option(s): [badoption].` This can be changed by using `\DeclareOption*{code}`, which executes code for any unknown option.
For example, many classes extend an existing class, using code such as `\LoadClass{article}` (see [\LoadClass](https://latexref.xyz/dev/latex2e.html#g_t_005cLoadClass)). In this case, it makes sense to pass any otherwise-unknown options to the underlying class, like this:
```
\DeclareOption*{%
  \PassOptionsToClass{\CurrentOption}{article}%
}

```

As another example, our class `smcmemo` might allow users to keep lists of memo recipients in external files, so the user could invoke `\documentclass[math]{smcmemo}` and it will read the file `math.memo`. This code inputs the file if it exists, while if it doesn’t, the option is passed to the `article` class:
```
\DeclareOption*{\InputIfFileExists{\CurrentOption.memo}
  {}{%
  \PassOptionsToClass{\CurrentOption}{article}}}

```

#### 12.14.7 `\DeclareRobustCommand`
Synopsis:
```
\DeclareRobustCommand{cmd}[num][default]{definition}
\DeclareRobustCommand* (same parameters

```

`\DeclareRobustCommand` and its starred form are generally like `\newcommand` and `\newcommand*` (see [`\newcommand` & `\renewcommand`](https://latexref.xyz/dev/latex2e.html#g_t_005cnewcommand-_0026-_005crenewcommand)), with the addition that they define a so-called _robust_ command, even if some code within the definition is fragile. (For a discussion of robust and fragile commands, see [`\protect`](https://latexref.xyz/dev/latex2e.html#g_t_005cprotect).)
Also unlike `\newcommand`, these do not give an error if macro cmd already exists; instead, a log message is put into the transcript file if a command is redefined. Thus, `\DeclareRobustCommand` can be used to define new robust commands or to redefine existing commands, making them robust.
The starred form, `\DeclareRobustCommand*`, disallows the arguments from containing multiple paragraphs, just like the starred form of `\newcommand` and `\renewcommand`. The meaning of the arguments is the same.
Commands defined this way are a bit less efficient than those defined using `\newcommand` so unless the command’s data is fragile and the command is used within a moving argument, use `\newcommand`.
Related to this, the `etoolbox` package offers three commands and their starred forms: `\newrobustcmd`(`*`) `\renewrobustcmd`(`*`), and `\providerobustcmd`(`*`). They are similar to `\newcommand`, `\renewcommand`, and `\providecommand` and their own starred forms, but define a robust cmd. They have two possible advantages compared to `\DeclareRobustCommand`:
  1. They use the low-level e-TeX protection mechanism rather than the higher-level LaTeX `\protect` mechanism, so they do not incur the slight loss of performance mentioned above, and
  2. They make the same distinction between `\new…`, `\renew…`, and `\provide…`, as the standard commands. That is, they do not just write a log message when you redefine cmd that already exists; you need to use either `\renew…` or `\provide…`, or you get an error. This may or may not be a benefit.

#### 12.14.8 `\ExecuteOptions`
Synopsis:
```
\ExecuteOptions{option-list}

```

For each option option in option-list, in order, this command executes the command `\ds@option`. If this command is not defined then that option is silently ignored.
This can be used to provide a default option list before `\ProcessOptions`. For example, if in a class file you want the default to be 11pt fonts then you could specify `\ExecuteOptions{11pt}\ProcessOptions\relax`.
#### 12.14.9 `\IfFileExists` & `\InputIfFileExists`
Synopses:
```
\IfFileExists{filename}{true-code}{false-code}
\InputIfFileExists{filename}{true-code}{false-code}

```

`\IfFileExists` executes true-code if LaTeX finds the file filename or false-code otherwise. In the first case it executing true-code and then inputs the file. Thus the command
```
\IfFileExists{img.pdf}{%
  \includegraphics{img.pdf}}
  {\typeout{!! img.pdf not found}

```

will include the graphic img.pdf if it is found and otherwise give a warning.
This command looks for the file in all search paths that LaTeX uses, not only in the current directory. To look only in the current directory do something like `\IfFileExists{./filename}{true-code}{false-code}`. If you ask for a filename without a `.tex` extension then LaTeX will first look for the file by appending the `.tex`; for more on how LaTeX handles file extensions see [`\input`](https://latexref.xyz/dev/latex2e.html#g_t_005cinput).
`\InputIfFileExists` is similar, but, as the name states, automatically `\input`s filename if it exists. The true-code is executed just before the `\input`; if the file doesn’t exist, the false-code is executed. An example:
```
\InputIfFileExists{mypkg.cfg}
  {\PackageInfo{Loading mypkg.cfg for configuration information}}
  {\PackageInfo{No mypkg.cfg found}}

```

#### 12.14.10 `\LoadClass` & `\LoadClassWithOptions`
Synopses:
```
\LoadClass[options-list]{class-name}[release-date]
\LoadClassWithOptions{class-name}[release-date]

```

Load a class, as with `\documentclass[options-list]{class-name}[release-date]`. An example: `\LoadClass[twoside]{article}`.
The options-list, if present, is a comma-separated list. The release-date is also optional. If present it must have the form `YYYY/MM/DD`.
If you request release-date and the date of the package installed on your system is earlier, then you get a warning on the screen and in the log like this:
```
You have requested, on input line 4, version `2038/01/19' of
document class article, but only version `2014/09/29 v1.4h
Standard LaTeX document class' is available.

```

The command version `\LoadClassWithOptions` uses the list of options for the current class. This means it ignores any options passed to it via `\PassOptionsToClass`. This is a convenience command that lets you build classes on existing ones, such as the standard `article` class, without having to track which options were passed.
#### 12.14.11 `\NeedsTeXFormat`
Synopsis:
```
\NeedsTeXFormat{format}[format-date]

```

Specifies the format that this class must be run under. Often issued as the first line of a class file, and most often used as: `\NeedsTeXFormat{LaTeX2e}`. When a document using that class is processed, the format being run must exactly match the format name given, including case. If it does not match then execution stops with an error like ‘This file needs format `LaTeX2e' but this is `plain'.’.
To require a version of the format that you know to have certain features, include the optional format-date on which those features were implemented. If present, it must be in the form `YYYY/MM/DD`. If the format version installed on your system is earlier than format date then you get a warning like this.
```
You have requested release `2038/01/20' of LaTeX, but only
release `2016/02/01' is available.

```

#### 12.14.12 `\OptionNotUsed`
Adds the current option to the list of unused options. Can only be used within the code argument of either `\DeclareOption` or `\DeclareOption*`.
#### 12.14.13 `\PassOptionsToClass` & `\PassOptionsToPackage`
Synopses:
```
\PassOptionsToClass{options}{clsname}
\PassOptionsToPackage{option}{pkgname}

```

Adds the options in the comma-separated list options to the options used by any future `\RequirePackage` or `\usepackage` command for the class clsname or the package pkgname, respectively.
The reason for these commands is that although you may load a package any number of times with no options, if you can specify options only the first time you load the package. Loading a package with options more than once will get you an error like `Option clash for package foo.`. LaTeX throws an error even if there is no conflict between the options.
If your own code is bringing in a package twice then you can combine the calls; for example, replacing the two
```
\RequirePackage[landscape]{geometry}
\RequirePackage[margins=1in]{geometry}

```

with the single command
```
\RequirePackage[landscape,margins=1in]{geometry}

```

However, suppose you are loading firstpkg and inside that package it loads secondpkg, and you need `secondpkg` to be loaded with option `draft`. Then before load the first package you must tell LaTeX about the desired options for the second package, like this:
```
\PassOptionsToPackage{draft}{secondpkg}
\RequirePackage{firstpkg}

```

If `firstpkg.sty` loads an option in conflict with what you want then you may have to alter its source, or yours.
These commands are useful for general users as well as class and package writers. For instance, suppose a user wants to load the `graphicx` package with the option `draft` and also wants to use a class `foo` that loads the `graphicx` package, but without that option. The user could start their LaTeX file with `\PassOptionsToPackage{draft}{graphicx} \documentclass{foo}`.
#### 12.14.14 `\ProcessOptions`
Synopsis:
```
\ProcessOptions\@options
\ProcessOptions*\@options

```

Execute the code for each option that the user has invoked. Invoke it in the class file as `\ProcessOptions\relax` (because of the existence of the starred version, described below).
Options come in two types. _Local options_ have been specified for this particular package in `\usepackage[options]`, `\RequirePackage[options]`, or the options argument of `\PassOptionsToPackage{options}`. _Global options_ are those given by the class user in `\documentclass[options]`. If an option is specified both locally and globally then it is local.
When `\ProcessOptions` is called for a package pkg.sty, the following happens:
  1. For each option option so far declared with `\DeclareOption`, `\ProcessOptions` looks to see if that option is either global or local for `pkg`. If so, then it executes the declared code. This is done in the order in which these options were given in pkg.sty.
  2. For each remaining local option, it executes the command `\ds@`option if it has been defined somewhere (other than by a `\DeclareOption`); otherwise, it executes the default option code given in `\DeclareOption*`. If no default option code has been declared then it gives an error message. This is done in the order in which these options were specified.

When `\ProcessOptions` is called for a class it works in the same way except that all options are local, and the default code for `\DeclareOption*` is `\OptionNotUsed` rather than an error.
The starred version `\ProcessOptions*` executes the options in the order specified in the calling commands, rather than in the order of declaration in the class or package. For a package, this means that the global options are processed first.
#### 12.14.15 `\ProvidesClass` & `\ProvidesPackage`
Synopses:
```
\ProvidesClass{clsname}[release-date [info-text]]
\ProvidesPackage{pkgname}[release-date [info-text]]

```

Identifies the class or package being defined, printing a message to the screen and the log file.
When you load a class or package, for example with `\documentclass{smcmemo}` or `\usepackage{test}`, LaTeX inputs a file (smcmemo.cls and test.sty, respectively). If the name of the file does not match the class or package name declared in it then you get a warning. Thus, if you invoke `\documentclass{smcmemo}`, and the file smcmemo.cls has the statement `\ProvidesClass{foo}` then you get a warning like `You have requested document class `smcmemo', but the document class provides 'foo'.` This warning does not prevent LaTeX from processing the rest of the class file normally.
If you include the optional argument then you must include a date, before any spaces, of the form `YYYY/MM/DD`. The rest of the optional argument is free-form, although it traditionally identifies the class. It is written to the screen during compilation and to the log file. Thus, if your file smcmemo.cls contains the line `\ProvidesClass{smcmemo}[2008/06/01 v1.0 SMC memo class]` and your document’s first line is `\documentclass{smcmemo}` then you will see `Document Class: smcmemo 2008/06/01 v1.0 SMC memo class`.
The date in the optional argument allows class and package users to ask to be warned if the version of the class or package is earlier than release date. For instance, a user could enter `\documentclass{smcmemo}[2018/10/12]` or `\usepackage{foo}[[2017/07/07]]` to require a class or package with certain features by specifying that it must be released no earlier than the given date. Perhaps more importantly, the date serves as documentation of the last release. (In practice, package users rarely include a date, and class users almost never do.)
#### 12.14.16 `\ProvidesFile`
Synopsis:
```
\ProvidesFile{filename}[info-text]

```

Declare a file other than the main class and package files, such as a configuration or font definition file. It writes the given information to the log file, essentially like `\ProvidesClass` and `\ProvidesPackage` (see the previous section).
For example:
```
\ProvidesFile{smcmemo.cfg}[2017/10/12 config file for smcmemo.cls]

```

writes this into the log:
```
File: smcmemo.cfg 2017/10/12 config file for smcmemo.cls

```

#### 12.14.17 `\RequirePackage` & `\RequirePackageWithOptions`
Synopsis:
```
\RequirePackage[option-list]{pkgname}[release-date]
\RequirePackageWithOptions{pkgname}[release-date]

```

Load a package, like the command `\usepackage` (see [Additional packages](https://latexref.xyz/dev/latex2e.html#Additional-packages)). An example:
`\RequirePackage[landscape,margin=1in]{geometry}`
The initial optional argument option-list, if present, must be a comma-separated list. The trailing optional argument release-date, if present, must have the form `YYYY/MM/DD`. If the release date of the package as installed on your system is earlier than release-date then you get a warning like ‘You have requested, on input line 9, version `2017/07/03' of package jhtest, but only version `2000/01/01' is available’.
The `\RequirePackageWithOptions` variant uses the list of options for the current class. This means it ignores any options passed to it via `\PassOptionsToClass`. This is a convenience command to allow easily building classes on existing ones without having to track which options were passed.
The difference between `\usepackage` and `\RequirePackage` is small. The `\usepackage` command is intended to be used in documents, while `\RequirePackage` is intended for package and class files. The most significant difference in practice is that `\RequirePackage` can be used in a document before the `\documentclass` command, while `\usepackage` gives an error there. The most common need for this nowadays is for the `\DocumentMetadata` command (see [`\DocumentMetadata`: Producing tagged PDF output](https://latexref.xyz/dev/latex2e.html#g_t_005cDocumentMetadata)).
The LaTeX development team strongly recommends use of these and related commands over plain TeX’s `\input`; see the Class Guide (<https://ctan.org/pkg/clsguide>).
