package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

type pet struct {
	Name string  `json:"name"`
	Note *string `json:"note"`
}

// pets is a fixture Result: a list of records with one nullable field.
type pets []pet

func (p pets) WriteText(w io.Writer) error {
	for _, item := range p {
		if _, err := fmt.Fprintln(w, item.Name); err != nil {
			return err
		}
	}
	return nil
}

func (p pets) JSONValue() any { return []pet(p) }

func (p pets) CSVHeader() []string { return []string{"name", "note"} }

func (p pets) CSVRows() [][]*string {
	rows := make([][]*string, 0, len(p))
	for _, item := range p {
		rows = append(rows, []*string{&item.Name, item.Note})
	}
	return rows
}

func render(t *testing.T, f Format, r Result) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, f, r); err != nil {
		t.Fatalf("Render(%s): %v", f, err)
	}
	return buf.String()
}

func TestRenderJSONIsCompactWithNullsAndTrailingNewline(t *testing.T) {
	got := render(t, JSON, pets{{Name: "Rex", Note: new("good")}, {Name: "Tib"}})
	const want = `[{"name":"Rex","note":"good"},{"name":"Tib","note":null}]` + "\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderJSONEmptyListIsEmptyArray(t *testing.T) {
	for name, r := range map[string]pets{"nil": nil, "empty": {}} {
		if got := render(t, JSON, r); got != "[]\n" {
			t.Errorf("%s list: got %q, want %q", name, got, "[]\n")
		}
	}
}

func TestRenderCSVQuotesAndWritesNullAsEmptyCell(t *testing.T) {
	got := render(t, CSV, pets{
		{Name: "Rex, Jr.", Note: new(`said "woof"`)},
		{Name: "Tib", Note: new("line one\nline two")},
		{Name: "Moo"},
	})
	const want = "name,note\n" +
		`"Rex, Jr.","said ""woof"""` + "\n" +
		"Tib,\"line one\nline two\"\n" +
		"Moo,\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderCSVEmptyListIsHeaderOnly(t *testing.T) {
	if got := render(t, CSV, pets{}); got != "name,note\n" {
		t.Errorf("got %q, want %q", got, "name,note\n")
	}
}

func TestRenderTextUsesTheResultsOwnRendering(t *testing.T) {
	if got := render(t, Text, pets{{Name: "Rex"}, {Name: "Tib"}}); got != "Rex\nTib\n" {
		t.Errorf("got %q, want %q", got, "Rex\nTib\n")
	}
}

func TestRenderRejectsUnknownFormat(t *testing.T) {
	err := Render(&bytes.Buffer{}, Format(99), pets{})
	if err == nil || !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("err = %v, want an unknown-format error", err)
	}
}

// unmarshalable is a Result whose JSON value cannot be encoded.
type unmarshalable struct{ pets }

func (unmarshalable) JSONValue() any { return make(chan int) }

func TestRenderJSONReportsEncodingFailure(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, JSON, unmarshalable{})
	if err == nil || !strings.Contains(err.Error(), "render json") {
		t.Errorf("err = %v, want a render json error", err)
	}
	if buf.Len() != 0 {
		t.Errorf("nothing should be written on failure; got %q", buf.String())
	}
}
