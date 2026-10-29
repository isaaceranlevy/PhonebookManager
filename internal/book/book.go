package book

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

type Contact struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Phones []string `json:"phones"`
}
type Book struct {
	Revision  int64     `json:"revision"`
	UpdatedAt time.Time `json:"updatedAt"`
	Contacts  []Contact `json:"contacts"`
}
type Limits struct {
	Contacts  int   `json:"contacts"`
	Name      int   `json:"name"`
	Number    int   `json:"number"`
	JSONBytes int64 `json:"jsonBytes"`
	CSVBytes  int64 `json:"csvBytes"`
}

func DefaultLimits() Limits { return Limits{1000, 64, 32, 1 << 20, 2 << 20} }

type ValidationError struct {
	Errors []string `json:"errors"`
}

func (e *ValidationError) Error() string { return strings.Join(e.Errors, "; ") }
func Validate(cs []Contact, l Limits) error {
	var es []string
	if len(cs) > l.Contacts {
		es = append(es, fmt.Sprintf("מקסימום %d אנשי קשר", l.Contacts))
	}
	ids := map[string]bool{}
	for i, c := range cs {
		bad := func(s string) { es = append(es, fmt.Sprintf("שורה %d: %s", i+1, s)) }
		if c.ID == "" || ids[c.ID] {
			bad("מזהה חסר או כפול")
		}
		ids[c.ID] = true
		if strings.TrimSpace(c.Name) == "" || utf8.RuneCountInString(c.Name) > l.Name || !validXML(c.Name) {
			bad("שם ריק, ארוך או מכיל תו אסור")
		}
		if len(c.Phones) < 1 || len(c.Phones) > 3 {
			bad("נדרשים 1–3 מספרים")
		}
		has := false
		for _, p := range c.Phones {
			if strings.TrimSpace(p) != "" {
				has = true
			}
			if utf8.RuneCountInString(p) > l.Number || strings.Trim(p, "0123456789+*#,; -()") != "" {
				bad("מספר לא תקין")
			}
		}
		if !has {
			bad("נדרש מספר טלפון")
		}
	}
	if len(es) > 0 {
		return &ValidationError{es}
	}
	return nil
}
func validXML(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !(r == 9 || r == 10 || r == 13 || r >= 32 && r <= 0xd7ff || r >= 0xe000 && r <= 0xfffd || r >= 0x10000 && r <= 0x10ffff) {
			return false
		}
	}
	return true
}
func NewID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}

type Preview struct {
	Contacts []Contact `json:"contacts"`
	Warnings []string  `json:"warnings"`
	Encoding string    `json:"encoding"`
}

func ImportCSV(data []byte, l Limits) (Preview, error) {
	p := Preview{Warnings: []string{"פתיחה ישירה ב־Excel עלולה למחוק אפסים מובילים; יש לייבא את עמודות הטלפון כטקסט."}, Encoding: "UTF-8"}
	if int64(len(data)) > l.CSVBytes {
		return p, fmt.Errorf("קובץ CSV גדול מדי")
	}
	data = bytes.TrimPrefix(data, []byte{239, 187, 191})
	if !utf8.Valid(data) {
		var e error
		data, e = charmap.Windows1255.NewDecoder().Bytes(data)
		if e != nil {
			return p, e
		}
		p.Encoding = "Windows-1255"
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = 4
	h, e := r.Read()
	if e != nil {
		return p, fmt.Errorf("כותרת CSV חסרה")
	}
	expected := []string{"name", "phone1", "phone2", "phone3"}
	for i, v := range h {
		if strings.ToLower(strings.TrimSpace(v)) != expected[i] {
			return p, fmt.Errorf("נדרשת הכותרת name,phone1,phone2,phone3")
		}
	}
	seen := map[string]bool{}
	for {
		row, e := r.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return p, e
		}
		for i := range row {
			row[i] = strings.TrimPrefix(row[i], "'")
		}
		phones := row[1:]
		for len(phones) > 0 && phones[len(phones)-1] == "" {
			phones = phones[:len(phones)-1]
		}
		c := Contact{NewID(), row[0], phones}
		key := strings.TrimSpace(c.Name) + "\x00" + strings.Join(c.Phones, "\x00")
		if seen[key] {
			p.Warnings = append(p.Warnings, fmt.Sprintf("שורה %d כפולה", len(p.Contacts)+2))
		}
		seen[key] = true
		p.Contacts = append(p.Contacts, c)
		if len(p.Contacts) > l.Contacts {
			return p, fmt.Errorf("יותר מדי אנשי קשר")
		}
	}
	if len(p.Contacts) == 0 {
		return p, fmt.Errorf("הקובץ אינו מכיל אנשי קשר")
	}
	if e := Validate(p.Contacts, l); e != nil {
		return p, e
	}
	return p, nil
}
func ExportCSV(cs []Contact) ([]byte, error) {
	var b bytes.Buffer
	b.Write([]byte{239, 187, 191})
	w := csv.NewWriter(&b)
	w.UseCRLF = true
	w.Write([]string{"name", "phone1", "phone2", "phone3"})
	for _, c := range cs {
		row := []string{c.Name, "", "", ""}
		copy(row[1:], c.Phones)
		for i, s := range row {
			if s != "" && (strings.ContainsRune("=+-@", rune(s[0])) || strings.HasPrefix(s, "'")) {
				row[i] = "'" + s
			}
		}
		if e := w.Write(row); e != nil {
			return nil, e
		}
	}
	w.Flush()
	return b.Bytes(), w.Error()
}
