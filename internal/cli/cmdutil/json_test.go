package cmdutil

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/iostreams"
)

type itemJSON struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Total *string `json:"total"`
	Tags  []string
}

// runJSONCommand runs `invox list ARGS` with --json fields from itemJSON and
// returns the exporter the command saw and the error.
func runJSONCommand(args ...string) (*Exporter, error) {
	var exporter *Exporter
	var seen *Exporter
	root := &cobra.Command{Use: "invox"}
	cmd := &cobra.Command{Use: "list", RunE: func(*cobra.Command, []string) error {
		seen = exporter
		return nil
	}}
	AddJSONFlags(cmd, &exporter, itemJSON{})
	root.AddCommand(cmd)
	root.SetFlagErrorFunc(FlagErrorFunc)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs(append([]string{"list"}, args...))
	err := root.Execute()
	return seen, err
}

func TestAddJSONFlags(t *testing.T) {
	tests := []struct {
		args       []string
		wantFields []string
		wantErr    string
		wantUsage  bool
	}{
		{args: nil, wantFields: nil},
		{args: []string{"--json", "name,id"}, wantFields: []string{"name", "id"}},
		{args: []string{"--json= id , ,total"}, wantFields: []string{"id", "total"}},
		{args: []string{"--json"}, wantErr: "specify one or more comma-separated fields for --json:\n  id\n  name\n  total"},
		{args: []string{"--json", ","}, wantErr: "specify one or more comma-separated fields for --json:\n  id\n  name\n  total"},
		{args: []string{"--json", "id,Tags"}, wantErr: "unknown JSON field: \"Tags\"\nAvailable fields:\n  id\n  name\n  total", wantUsage: true},
	}
	for _, tc := range tests {
		exporter, err := runJSONCommand(tc.args...)
		if tc.wantErr != "" {
			var flagErr *FlagError
			if err == nil || err.Error() != tc.wantErr || errors.As(err, &flagErr) != tc.wantUsage {
				t.Errorf("%q: error = %#v, want %q (usage error: %v)", tc.args, err, tc.wantErr, tc.wantUsage)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: error = %v", tc.args, err)
			continue
		}
		var got []string
		if exporter != nil {
			got = exporter.fields
		}
		if (exporter == nil) != (tc.wantFields == nil) || len(got) != len(tc.wantFields) {
			t.Errorf("%q: fields = %q, want %q", tc.args, got, tc.wantFields)
			continue
		}
		for i := range got {
			if got[i] != tc.wantFields[i] {
				t.Errorf("%q: fields = %q, want %q", tc.args, got, tc.wantFields)
			}
		}
	}
}

func TestExporterWrite(t *testing.T) {
	total := "120.00"
	tests := []struct {
		name   string
		fields []string
		tty    bool
		data   any
		want   string
	}{
		{name: "empty list", fields: []string{"id"}, data: []itemJSON{}, want: "[]\n"},
		{name: "nil list", fields: []string{"id"}, data: []itemJSON(nil), want: "[]\n"},
		{name: "list picks fields", fields: []string{"total", "id"}, data: []itemJSON{{ID: "a", Name: "A", Total: &total}, {ID: "<b&c>"}}, want: `[{"id":"a","total":"120.00"},{"id":"<b&c>","total":null}]` + "\n"},
		{name: "object", fields: []string{"name"}, data: itemJSON{ID: "a", Name: `C:\x`}, want: `{"name":"C:\\x"}` + "\n"},
		{name: "indented on a terminal", fields: []string{"id", "name"}, tty: true, data: []itemJSON{{ID: "a", Name: "A"}}, want: "[\n  {\n    \"id\": \"a\",\n    \"name\": \"A\"\n  }\n]\n"},
	}
	for _, tc := range tests {
		ios, _, out, errOut := iostreams.Test()
		ios.SetStdoutTTY(tc.tty)
		exporter := &Exporter{fields: tc.fields}
		if err := exporter.Write(ios, tc.data); err != nil {
			t.Fatalf("%s: Write returned error: %v", tc.name, err)
		}
		if out.String() != tc.want || errOut.Len() != 0 {
			t.Errorf("%s: stdout = %q, stderr = %q, want stdout %q", tc.name, out.String(), errOut.String(), tc.want)
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestExporterWriteReportsWriteErrors(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	ios.Out = failingWriter{}
	exporter := &Exporter{fields: []string{"id"}}
	if err := exporter.Write(ios, []itemJSON{{ID: "a"}}); err == nil || err.Error() != "disk full" {
		t.Fatalf("Write error = %v, want disk full", err)
	}
}
