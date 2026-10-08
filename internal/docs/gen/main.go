// Command gen writes the command reference from the invox command tree:
// docs/cli/*.md and share/man/man1/*.1, one page per command and help topic.
// Each page holds the command's help, as `invox help` prints it for a user
// whose home directory is $HOME. Run it from the repository root, on Linux or
// macOS: on Windows the paths in the help come out with backslashes.
//
//	go run ./internal/docs/gen
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/fsutil"
	"github.com/0xboris/invox/internal/iostreams"
)

// home stands in for the home directory in the help, and is printed as $HOME.
const home = "/HOME"

// page is one command or help topic: the words after `invox help` that
// print it, its one-line summary, and the pages it links to.
type page struct {
	words   []string
	short   string
	related []*page
}

func main() {
	dir := flag.String("dir", ".", "repository root to write docs/cli and share/man/man1 under")
	flag.Parse()
	if err := generate(*dir); err != nil {
		fmt.Fprintln(iostreams.System().ErrOut, "gen:", err)
		os.Exit(1)
	}
}

func generate(dir string) error {
	pages := commandPages(cli.NewRootCmd(newFactory()))
	root := pages[0]
	for _, topic := range helptext.Topics {
		if topic.Print == nil {
			continue
		}
		topicPage := &page{words: []string{topic.Name}, short: topic.Short, related: []*page{root}}
		root.related = append(root.related, topicPage)
		pages = append(pages, topicPage)
	}

	mdDir := filepath.Join(dir, "docs", "cli")
	manDir := filepath.Join(dir, "share", "man", "man1")
	for _, d := range []string{mdDir, manDir} {
		if err := os.RemoveAll(d); err != nil {
			return err
		}
		if err := fsutil.MkdirAll(d, fsutil.Public); err != nil {
			return err
		}
	}
	for _, p := range pages {
		help, err := helpText(p.words)
		if err != nil {
			return err
		}
		if err := fsutil.WriteFile(filepath.Join(mdDir, p.name("_")+".md"), markdown(p, help), fsutil.Public); err != nil {
			return err
		}
		if err := fsutil.WriteFile(filepath.Join(manDir, p.name("-")+".1"), manPage(p, help), fsutil.Public); err != nil {
			return err
		}
	}
	return nil
}

// commandPages returns a page for root and each available command below
// it, root first. Each links to its parent and its subcommands.
func commandPages(root *cobra.Command) []*page {
	var pages []*page
	var walk func(cmd *cobra.Command, parent *page)
	walk = func(cmd *cobra.Command, parent *page) {
		p := &page{short: cmd.Short}
		if parent != nil {
			p.words = append(append([]string{}, parent.words...), cmd.Name())
			p.related = append(p.related, parent)
			parent.related = append(parent.related, p)
		}
		pages = append(pages, p)
		for _, sub := range cmd.Commands() {
			if sub.IsAvailableCommand() {
				walk(sub, p)
			}
		}
	}
	walk(root, nil)
	return pages
}

// name is the page's file name without extension, such as invox_archive_edit.
// A help topic gets "help" in front of its name.
func (p *page) name(sep string) string {
	words := append([]string{"invox"}, p.words...)
	if p.isTopic() {
		words = []string{"invox", "help", p.words[0]}
	}
	return strings.Join(words, sep)
}

func (p *page) title() string {
	return strings.ReplaceAll(p.name("_"), "_", " ")
}

func (p *page) isTopic() bool {
	if len(p.words) != 1 {
		return false
	}
	topic, ok := helptext.LookupTopic(p.words[0])
	return ok && topic.Print != nil
}

// helpText runs `invox help WORDS` and returns what it prints.
func helpText(words []string) (string, error) {
	f := newFactory()
	ios := f.IOStreams
	if code := cli.Main(append([]string{"help"}, words...), f); code != 0 {
		return "", fmt.Errorf("invox help %s: exit %d: %s", strings.Join(words, " "), code, ios.ErrOut)
	}
	return strings.ReplaceAll(fmt.Sprint(ios.Out), home, "$HOME"), nil
}

// newFactory returns a Factory for a Linux user with no environment
// variables set, so the docs are the same wherever they are generated.
func newFactory() *cmdutil.Factory {
	ios, _, _, _ := iostreams.Test()
	return cmdutil.NewFactory(ios, run.Exec{}, env.Env{
		GOOS:    "linux",
		Getenv:  func(string) string { return "" },
		HomeDir: func() (string, error) { return home, nil },
		Getwd:   func() (string, error) { return home, nil },
		Now:     func() time.Time { return time.Time{} },
	})
}

func markdown(p *page, help string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# %s\n\n```text\n%s```\n", p.title(), help)
	if len(p.related) > 0 {
		b.WriteString("\n## See also\n\n")
		for _, r := range p.related {
			fmt.Fprintf(&b, "- [%s](%s.md): %s\n", r.title(), r.name("_"), r.short)
		}
	}
	return b.Bytes()
}

func manPage(p *page, help string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, ".TH %q 1 \"\" \"invox\" \"invox manual\"\n", strings.ToUpper(p.name("-")))
	fmt.Fprintf(&b, ".SH NAME\n%s \\- %s\n", p.name("-"), roffEscape(p.short))
	b.WriteString(".SH DESCRIPTION\n.nf\n")
	for _, line := range strings.Split(strings.TrimSuffix(help, "\n"), "\n") {
		b.WriteString(roffLine(line) + "\n")
	}
	b.WriteString(".fi\n")
	if len(p.related) > 0 {
		b.WriteString(".SH SEE ALSO\n")
		refs := make([]string, len(p.related))
		for i, r := range p.related {
			refs[i] = fmt.Sprintf("\\fB%s\\fP(1)", r.name("-"))
		}
		b.WriteString(strings.Join(refs, ", ") + "\n")
	}
	return b.Bytes()
}

func roffEscape(s string) string {
	return strings.ReplaceAll(s, `\`, `\e`)
}

// roffLine escapes line for a no-fill block. A line starting with . or '
// would be read as a request, so \& goes in front of it.
func roffLine(line string) string {
	line = roffEscape(line)
	if strings.HasPrefix(line, ".") || strings.HasPrefix(line, "'") {
		line = `\&` + line
	}
	return line
}
