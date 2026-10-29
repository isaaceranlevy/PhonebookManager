package yealink

import (
	"bytes"
	"encoding/xml"
	"github.com/isaaceranlevy/PhonebookManager/internal/book"
	"io"
	"testing"
)

func TestFormats(t *testing.T) {
	for _, f := range []string{"directory", "phonebook"} {
		b, e := Encode([]book.Contact{{ID: "1", Name: "עברית & < \" ' שלום", Phones: []string{"00123", "+972"}}}, f)
		if e != nil {
			t.Fatal(e)
		}
		d := xml.NewDecoder(bytes.NewReader(b))
		var text []byte
		for {
			v, e := d.Token()
			if e == io.EOF {
				break
			}
			if e != nil {
				t.Fatal(e)
			}
			if c, ok := v.(xml.CharData); ok {
				text = append(text, c...)
			}
			if st, ok := v.(xml.StartElement); ok {
				for _, a := range st.Attr {
					text = append(text, []byte(a.Value)...)
				}
			}
		}
		if !bytes.Contains(text, []byte("עברית & < \" ' שלום")) || !bytes.Contains(text, []byte("00123")) {
			t.Fatal(string(b))
		}
	}
}
func TestInvalidFormat(t *testing.T) {
	if _, e := Encode(nil, "unknown"); e == nil {
		t.Fatal("invalid format")
	}
}
