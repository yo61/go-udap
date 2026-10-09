package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

// Result is what a Data command produced. Every Result supports every
// Output format.
type Result interface {
	// WriteText writes the human-readable rendering.
	WriteText(w io.Writer) error
	// JSONValue returns the value to marshal: a slice for lists, a
	// struct or map otherwise. Absent values marshal as null.
	JSONValue() any
	// CSVHeader returns the column names.
	CSVHeader() []string
	// CSVRows returns one row per record, aligned with CSVHeader. A nil
	// cell is null.
	CSVRows() [][]*string
}

// Render writes r to w in format f.
func Render(w io.Writer, f Format, r Result) error {
	switch f {
	case Text:
		return r.WriteText(w)
	case JSON:
		return renderJSON(w, r.JSONValue())
	case CSV:
		return renderCSV(w, r.CSVHeader(), r.CSVRows())
	default:
		return fmt.Errorf("render: unknown format %d", f)
	}
}

func renderJSON(w io.Writer, v any) error {
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice && rv.IsNil() {
		v = []struct{}{}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("render json: %w", err)
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

func renderCSV(w io.Writer, header []string, rows [][]*string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("render csv: %w", err)
	}
	record := make([]string, len(header))
	for _, row := range rows {
		for i, cell := range row {
			record[i] = ""
			if cell != nil {
				record[i] = *cell
			}
		}
		if err := cw.Write(record); err != nil {
			return fmt.Errorf("render csv: %w", err)
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("render csv: %w", err)
	}
	return nil
}
