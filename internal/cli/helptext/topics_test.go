package helptext

import (
	"io/fs"
	"path"
	"strings"
	"testing"
)

// A page in topics/ shows only when Topics names it.
func TestEveryTopicPageIsRegistered(t *testing.T) {
	files, err := fs.Glob(pages, "topics/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no topic pages embedded")
	}
	for _, file := range files {
		name := strings.TrimSuffix(path.Base(file), ".tmpl")
		if topic, ok := LookupTopic(name); !ok || topic.Name != name {
			t.Errorf("%s: add {Name: %q, Short: ...} to Topics", file, name)
		}
	}
}
