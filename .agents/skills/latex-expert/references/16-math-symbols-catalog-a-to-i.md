# 16 Math formulas: Symbol catalog A through I

Source: extracted from [`latex_syntax.md`](../../latex_syntax.md), based on the January 2025 LaTeX2e unofficial reference manual.

## Included sections

- 16.2 Math symbols

### 16.2 Math symbols
LaTeX provides almost any mathematical or technical symbol that anyone uses. For example, if you include `$\pi$` in your source, you will get the pi symbol π. See the “Comprehensive LaTeX Symbol List” package at <https://ctan.org/pkg/comprehensive>.
Here is a list of commonly-used symbols. It is by no means exhaustive. Each symbol is described with a short phrase, and its symbol class, which determines the spacing around it, is given in parenthesis. Unless said otherwise, the commands for these symbols can be used only in math mode. To redefine a command so that it can be used whatever the current mode, see [`\ensuremath`](https://latexref.xyz/dev/latex2e.html#g_t_005censuremath).

`\|`

∥ Parallel (relation). Synonym: `\parallel`.

`\aleph`

ℵ Aleph, transfinite cardinal (ordinary).

`\alpha`

α Lowercase Greek letter alpha (ordinary).

`\amalg`

⨿ Disjoint union (binary)

`\angle`

∠ Geometric angle (ordinary). Similar: less-than sign `<` and angle bracket `\langle`.

`\approx`

≈ Almost equal to (relation).

`\ast`

∗ Asterisk operator, convolution, six-pointed (binary). Synonym: `*`, which is often a superscript or subscript, as in the Kleene star. Similar: `\star`, which is five-pointed, and is sometimes used as a general binary operation, and sometimes reserved for cross-correlation.

`\asymp`

≍ Asymptotically equivalent (relation).

`\backslash`

\ Backslash (ordinary). Similar: set minus `\setminus`, and `\textbackslash` for backslash outside of math mode.

`\beta`

β Lowercase Greek letter beta (ordinary).

`\bigcap`

⋂ Variable-sized, or n-ary, intersection (operator). Similar: binary intersection `\cap`.

`\bigcirc`

⚪ Circle, larger (binary). Similar: function composition `\circ`.

`\bigcup`

⋃ Variable-sized, or n-ary, union (operator). Similar: binary union `\cup`.

`\bigodot`

⨀ Variable-sized, or n-ary, circled dot operator (operator).

`\bigoplus`

⨁ Variable-sized, or n-ary, circled plus operator (operator).

`\bigotimes`

⨂ Variable-sized, or n-ary, circled times operator (operator).

`\bigtriangledown`

▽ Variable-sized, or n-ary, open triangle pointing down (binary). Synonym: \varbigtriangledown.

`\bigtriangleup`

△ Variable-sized, or n-ary, open triangle pointing up (binary). Synonym: \varbigtriangleup.

`\bigsqcup`

⨆ Variable-sized, or n-ary, square union (operator).

`\biguplus`

⨄ Variable-sized, or n-ary, union operator with a plus (operator). (Note that the name has only one p.)

`\bigvee`

⋁ Variable-sized, or n-ary, logical-or (operator).

`\bigwedge`

⋀ Variable-sized, or n-ary, logical-and (operator).

`\bot`

⊥, Up tack, bottom, least element of a partially ordered set, or a contradiction (ordinary). See also `\top`.

`\bowtie`

⋈ Natural join of two relations (relation).

`\Box`

□ Modal operator for necessity; square open box (ordinary). Not available in plain TeX. In LaTeX you need to load the `amssymb` package.

`\bullet`

• Bullet (binary). Similar: multiplication dot `\cdot`.

`\cap`

∩ Intersection of two sets (binary). Similar: variable-sized operator `\bigcap`.

`\cdot`

⋅ Multiplication (binary). Similar: Bullet dot `\bullet`.

`\chi`

χ Lowercase Greek chi (ordinary).

`\circ`

∘ Function composition, ring operator (binary). Similar: variable-sized operator `\bigcirc`.

`\clubsuit`

♣ Club card suit (ordinary).

`\complement`

∁, Set complement, used as a superscript as in `$S^\complement$` (ordinary). Not available in plain TeX. In LaTeX you need to load the `amssymb` package. Also used: `$S^{\mathsf{c}}$` or `$\bar{S}$`.

`\cong`

≅ Congruent (relation).

`\coprod`

∐ Coproduct (operator).

`\cup`

∪ Union of two sets (binary). Similar: variable-sized operator `\bigcup`.

`\dagger`

† Dagger relation (binary).

`\dashv`

⊣ Dash with vertical, reversed turnstile (relation). Similar: turnstile `\vdash`.

`\ddagger`

‡ Double dagger relation (binary).

`\Delta`

Δ Greek uppercase delta, used for increment (ordinary).

`\delta`

δ Greek lowercase delta (ordinary).

`\Diamond`

◇ Large diamond operator (ordinary). Not available in plain TeX. In LaTeX you need to load the `amssymb` package.

`\diamond`

⋄ Diamond operator (binary). Similar: large diamond `\Diamond`, circle bullet `\bullet`.

`\diamondsuit`

♢ Diamond card suit (ordinary).

`\div`

÷ Division sign (binary).

`\doteq`

≐ Approaches the limit (relation). Similar: geometrically equal to `\Doteq`.

`\downarrow`

↓ Down arrow, converges (relation). Similar: `\Downarrow` double line down arrow.

`\Downarrow`

⇓ Double line down arrow (relation). Similar: `\downarrow` single line down arrow.

`\ell`

ℓ Lowercase cursive letter l (ordinary).

`\emptyset`

∅ Empty set symbol (ordinary). The variant form is `\varnothing`.

`\epsilon`

ϵ Lowercase lunate epsilon (ordinary). Similar to Greek text letter. More widely used in mathematics is the script small letter epsilon `\varepsilon` ε. Related: the set membership relation `\in` ∈.

`\equiv`

≡ Equivalence (relation).

`\eta`

η Lowercase Greek letter (ordinary).

`\exists`

∃ Existential quantifier (ordinary).

`\flat`

♭ Musical flat (ordinary).

`\forall`

∀ Universal quantifier (ordinary).

`\frown`

⌢ Downward curving arc (ordinary).

`\Gamma`

Γ uppercase Greek letter (ordinary).

`\gamma`

γ Lowercase Greek letter (ordinary).

`\ge`

≥ Greater than or equal to (relation). This is a synonym for `\geq`.

`\geq`

≥ Greater than or equal to (relation). This is a synonym for `\ge`.

`\gets`

← Is assigned the value (relation). Synonym: `\leftarrow`.

`\gg`

≫ Much greater than (relation). Similar: much less than `\ll`.

`\hbar`

ℏ Planck constant over two pi (ordinary).

`\heartsuit`

♡ Heart card suit (ordinary).

`\hookleftarrow`

↩ Hooked left arrow (relation).

`\hookrightarrow`

↪ Hooked right arrow (relation).

`\iff`

⟷ If and only if (relation). It is `\Longleftrightarrow` with a `\thickmuskip` on either side.

`\Im`

ℑ Imaginary part (ordinary). See: real part `\Re`.

`\imath`

Dotless i; used when you are putting an accent on an i (see [Math accents](https://latexref.xyz/dev/latex2e.html#Math-accents)).

`\in`

∈ Set element (relation). See also: lowercase lunate epsilon `\epsilon`ϵ and small letter script epsilon `\varepsilon`.

`\infty`

∞ Infinity (ordinary).

`\int`

∫ Integral (operator).

`\iota`

ι Lowercase Greek letter (ordinary).
