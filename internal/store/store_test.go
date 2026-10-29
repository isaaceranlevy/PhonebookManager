package store

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/isaaceranlevy/PhonebookManager/internal/book"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, e := Open(t.TempDir(), book.DefaultLimits(), "directory")
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func contacts() []book.Contact {
	return []book.Contact{{ID: "1", Name: "דנה & כהן", Phones: []string{"0012"}}}
}
func TestSaveConflictAndIsolation(t *testing.T) {
	s := open(t)
	b, e := s.Save(0, contacts())
	if e != nil || b.Revision != 1 {
		t.Fatal(b, e)
	}
	if _, e = s.Save(0, contacts()); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	v := s.Snapshot()
	v.Book.Contacts[0].Name = "mutated"
	if s.Snapshot().Book.Contacts[0].Name != "דנה & כהן" {
		t.Fatal("mutable snapshot")
	}
	if _, e = s.Save(1, []book.Contact{{Name: ""}}); e == nil {
		t.Fatal("invalid accepted")
	}
}
func TestFailedWritePreservesState(t *testing.T) {
	s := open(t)
	s.Save(0, contacts())
	before, _ := os.ReadFile(filepath.Join(s.dir, "book.json"))
	os.Mkdir(filepath.Join(s.dir, "book.json.tmp"), 0700)
	if _, e := s.Save(1, contacts()); e == nil {
		t.Fatal("expected failure")
	}
	after, _ := os.ReadFile(filepath.Join(s.dir, "book.json"))
	if !bytes.Equal(before, after) || s.Snapshot().Book.Revision != 1 {
		t.Fatal("state changed")
	}
}
func TestRecoveryAndRestore(t *testing.T) {
	s := open(t)
	s.Save(0, contacts())
	s.Save(1, contacts())
	os.WriteFile(filepath.Join(s.dir, "book.json"), []byte("broken"), 0600)
	r, e := Open(s.dir, book.DefaultLimits(), "directory")
	if e != nil || !r.Snapshot().Degraded || r.Snapshot().Book.Revision != 1 {
		t.Fatal(r, e)
	}
	bs, e := r.Backups()
	if e != nil || len(bs) == 0 {
		t.Fatal(bs, e)
	}
	b, e := r.Restore(bs[0].ID, 1)
	if e != nil || b.Revision != 2 {
		t.Fatal(b, e)
	}
}
func TestConcurrentAndRotation(t *testing.T) {
	s := open(t)
	var w sync.WaitGroup
	w.Add(1)
	go func() {
		defer w.Done()
		for i := 0; i < 100; i++ {
			v := s.Snapshot()
			d := xml.NewDecoder(bytes.NewReader(v.XML))
			for {
				_, e := d.Token()
				if e == io.EOF {
					break
				}
				if e != nil {
					t.Errorf("incomplete XML: %v", e)
					break
				}
			}
		}
	}()
	for i := 0; i < 14; i++ {
		cs := contacts()
		cs[0].Name = fmt.Sprintf("איש קשר %d & בדיקה", i)
		if _, e := s.Save(int64(i), cs); e != nil {
			t.Fatal(e)
		}
	}
	w.Wait()
	b, e := s.Backups()
	if e != nil || len(b) != 10 {
		t.Fatal(len(b), e)
	}
}
func TestCorruptWithoutBackupFails(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "book.json"), []byte("bad"), 0600)
	if _, e := Open(d, book.DefaultLimits(), "directory"); e == nil {
		t.Fatal("silent data loss")
	}
}
func TestMissingPrimaryRecoversBackup(t *testing.T) {
	s := open(t)
	s.Save(0, contacts())
	s.Save(1, contacts())
	os.Remove(filepath.Join(s.dir, "book.json"))
	r, e := Open(s.dir, book.DefaultLimits(), "directory")
	if e != nil || !r.Snapshot().Degraded || r.Snapshot().Book.Revision != 1 {
		t.Fatal("missing primary must recover backup", r, e)
	}
}
func TestStructurallyInvalidPrimaryFails(t *testing.T) {
	for _, data := range []string{`{}`, `null`, `{"revision":1,"updatedAt":"2026-01-01T00:00:00Z","contacts":null}`} {
		d := t.TempDir()
		os.WriteFile(filepath.Join(d, "book.json"), []byte(data), 0600)
		if _, e := Open(d, book.DefaultLimits(), "directory"); e == nil {
			t.Fatal("accepted invalid book", data)
		}
	}
}
