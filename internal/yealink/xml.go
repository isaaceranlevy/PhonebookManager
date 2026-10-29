package yealink

import (
	"encoding/xml"
	"fmt"
	"github.com/isaaceranlevy/PhonebookManager/internal/book"
)

type entry struct {
	Name      string   `xml:"Name"`
	Telephone []string `xml:"Telephone"`
}
type directory struct {
	XMLName xml.Name `xml:"YealinkIPPhoneDirectory"`
	Title   string   `xml:"Title"`
	Entries []entry  `xml:"DirectoryEntry"`
}
type unit struct {
	Name string `xml:"Name,attr"`
	P1   string `xml:"Phone1,attr"`
	P2   string `xml:"Phone2,attr"`
	P3   string `xml:"Phone3,attr"`
}
type menu struct {
	Name  string `xml:"Name,attr"`
	Units []unit `xml:"Unit"`
}
type phonebook struct {
	XMLName xml.Name `xml:"YealinkIPPhoneBook"`
	Menu    menu     `xml:"Menu"`
}

func Encode(cs []book.Contact, format string) ([]byte, error) {
	var v any
	switch format {
	case "directory":
		d := directory{Title: "Nimbus Phonebook"}
		for _, c := range cs {
			d.Entries = append(d.Entries, entry{c.Name, c.Phones})
		}
		v = d
	case "phonebook":
		p := phonebook{Menu: menu{Name: "Nimbus Phonebook"}}
		for _, c := range cs {
			var ps [3]string
			copy(ps[:], c.Phones)
			p.Menu.Units = append(p.Menu.Units, unit{c.Name, ps[0], ps[1], ps[2]})
		}
		v = p
	default:
		return nil, fmt.Errorf("unknown Yealink format %q", format)
	}
	b, e := xml.MarshalIndent(v, "", "  ")
	if e != nil {
		return nil, e
	}
	return append([]byte(xml.Header), b...), nil
}
