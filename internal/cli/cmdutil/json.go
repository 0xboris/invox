package cmdutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/iostreams"
)

// JSONFieldsAnnotation holds a command's --json fields, comma-separated. The
// help page lists them and the shell completes them.
const JSONFieldsAnnotation = "help:json-fields"

// Exporter writes the --json output of a command: the fields it was asked
// for, of each value it is given.
type Exporter struct {
	fields []string
}

// AddJSONFlags adds --json to cmd. Its fields are the json tags of the
// fields of exportType, a struct. When --json is given, *exporter is set
// before cmd runs; it stays nil otherwise. An unknown field is a usage
// error that lists the fields; --json without a value lists them and fails.
func AddJSONFlags(cmd *cobra.Command, exporter **Exporter, exportType any) {
	fields := jsonFieldNames(reflect.TypeOf(exportType))
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[JSONFieldsAnnotation] = strings.Join(fields, ",")

	var requested string
	cmd.Flags().StringVar(&requested, "json", "", "Output JSON with the specified `fields`")
	_ = cmd.RegisterFlagCompletionFunc("json", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		prefix := ""
		if i := strings.LastIndex(toComplete, ","); i >= 0 {
			prefix = toComplete[:i+1]
		}
		chosen := strings.Split(prefix, ",")
		var completions []string
		for _, field := range fields {
			if !slices.Contains(chosen, field) {
				completions = append(completions, prefix+field)
			}
		}
		return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	})

	preRun := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			var selected []string
			for _, field := range strings.Split(requested, ",") {
				field = strings.TrimSpace(field)
				if field == "" {
					continue
				}
				if !slices.Contains(fields, field) {
					return FlagErrorf(CommandPath(cmd), "unknown JSON field: %q\nAvailable fields:\n%s", field, fieldList(fields))
				}
				selected = append(selected, field)
			}
			if len(selected) == 0 {
				return jsonFieldsRequired(fields)
			}
			*exporter = &Exporter{fields: selected}
		}
		if preRun != nil {
			return preRun(cmd, args)
		}
		return nil
	}
}

// jsonFieldsRequired is the error for --json without fields. Like gh, it is
// a runtime error (exit 1) that lists the fields, not a usage error.
func jsonFieldsRequired(fields []string) error {
	return fmt.Errorf("specify one or more comma-separated fields for --json:\n%s", fieldList(fields))
}

// jsonFlagWithoutValue returns the error for --json given without a value
// on cmd, or nil when err is about something else.
func jsonFlagWithoutValue(cmd *cobra.Command, err error) error {
	fields, ok := cmd.Annotations[JSONFieldsAnnotation]
	if !ok || err.Error() != "flag needs an argument: --json" {
		return nil
	}
	return jsonFieldsRequired(strings.Split(fields, ","))
}

func fieldList(fields []string) string {
	return "  " + strings.Join(fields, "\n  ")
}

// jsonFieldNames returns the json tag names of struct type t, sorted.
func jsonFieldNames(t reflect.Type) []string {
	var names []string
	for i := range t.NumField() {
		if name := jsonName(t.Field(i)); name != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

func jsonName(field reflect.StructField) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// Write encodes the requested fields of data, a struct or a slice of
// structs, as one JSON document on ios.Out: indented on a terminal, compact
// otherwise. A slice is always an array, [] when it is empty. The whole
// document is encoded before anything is written.
func (e *Exporter) Write(ios *iostreams.IOStreams, data any) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if ios.IsStdoutTTY() {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(e.pick(reflect.ValueOf(data))); err != nil {
		return err
	}
	_, err := ios.Out.Write(buf.Bytes())
	return err
}

func (e *Exporter) pick(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Slice:
		items := make([]any, 0, v.Len())
		for i := range v.Len() {
			items = append(items, e.pick(v.Index(i)))
		}
		return items
	case reflect.Struct:
		object := make(map[string]any, len(e.fields))
		for i := range v.NumField() {
			if name := jsonName(v.Type().Field(i)); slices.Contains(e.fields, name) {
				object[name] = v.Field(i).Interface()
			}
		}
		return object
	}
	panic(fmt.Sprintf("cmdutil: cannot export %s", v.Type()))
}
