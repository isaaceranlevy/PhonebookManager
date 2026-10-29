package book

import (
	"bytes"
	"golang.org/x/text/encoding/charmap"
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	good := []Contact{{ID: "1", Name: "ישראל & כהן", Phones: []string{"03-0123456", "+972501234567"}}}
	if e := Validate(good, DefaultLimits()); e != nil {
		t.Fatal(e)
	}
	for _, c := range []Contact{{ID: "1", Name: "", Phones: []string{"03"}}, {ID: "1", Name: "א", Phones: []string{"x"}}, {ID: "1", Name: "א", Phones: []string{}}, {ID: "1", Name: strings.Repeat("א", 65), Phones: []string{"1"}}} {
		if Validate([]Contact{c}, DefaultLimits()) == nil {
			t.Fatalf("accepted %#v", c)
		}
	}
	if Validate(append(good, good...), DefaultLimits()) == nil {
		t.Fatal("duplicate ids")
	}
}
func TestCSVRoundTrip(t *testing.T) {
	c := []Contact{{ID: "1", Name: "=שם & עברית", Phones: []string{"+97250123", "0012"}}}
	data, e := ExportCSV(c)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.HasPrefix(data, []byte{239, 187, 191}) {
		t.Fatal("missing BOM")
	}
	p, e := ImportCSV(data, DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	if p.Contacts[0].Name != c[0].Name || p.Contacts[0].Phones[0] != "+97250123" || p.Contacts[0].Phones[1] != "0012" {
		t.Fatalf("%+v", p)
	}
}
func TestCP1255(t *testing.T) {
	b, e := charmap.Windows1255.NewEncoder().Bytes([]byte("name,phone1,phone2,phone3\nדנה,0012,,\n"))
	if e != nil {
		t.Fatal(e)
	}
	p, e := ImportCSV(b, DefaultLimits())
	if e != nil || p.Contacts[0].Name != "דנה" {
		t.Fatalf("%+v %v", p, e)
	}
}
func TestCSVBadRows(t *testing.T) {
	for _, s := range []string{"garbage\n", "name,phone1,phone2,phone3\nא,x,,\n", "name,phone1,phone2,phone3\n"} {
		if _, e := ImportCSV([]byte(s), DefaultLimits()); e == nil {
			t.Fatal("accepted invalid CSV")
		}
	}
}
