package server

import (
	"encoding/json"
	"github.com/isaaceranlevy/PhonebookManager/internal/book"
	"github.com/isaaceranlevy/PhonebookManager/internal/config"
	"github.com/isaaceranlevy/PhonebookManager/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setup(t *testing.T) *App {
	s, e := store.Open(t.TempDir(), book.DefaultLimits(), "directory")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(s, config.Default())
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func req(a *App, method, path, body, host, origin, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:8765"+path, strings.NewReader(body))
	r.Host = host
	r.Header.Set("Origin", origin)
	r.Header.Set("X-Phonebook-Token", token)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.Admin.ServeHTTP(w, r)
	return w
}
func TestAdminSecurity(t *testing.T) {
	a := setup(t)
	for _, x := range []struct {
		host, origin, token string
		status              int
	}{{"evil:8765", "", a.Token, 403}, {"127.0.0.1:8765", "http://evil", a.Token, 403}, {"127.0.0.1:8765", "", "", 403}, {"127.0.0.1:8765", "http://localhost:8765", a.Token, 403}, {"127.0.0.1:8765", "", a.Token, 200}} {
		w := req(a, "PUT", "/api/book", `{"revision":0,"contacts":[]}`, x.host, x.origin, x.token)
		if w.Code != x.status {
			t.Fatalf("%+v: %d %s", x, w.Code, w.Body)
		}
	}
}
func TestCRUDAndErrors(t *testing.T) {
	a := setup(t)
	body := `{"revision":0,"contacts":[{"id":"1","name":"דנה & כהן","phones":["00123"]}]}`
	w := req(a, "PUT", "/api/book", body, "127.0.0.1:8765", "", a.Token)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w = req(a, "PUT", "/api/book", body, "127.0.0.1:8765", "", a.Token)
	if w.Code != 409 {
		t.Fatal(w.Code)
	}
	w = req(a, "PUT", "/api/book", `{"revision":1,"contacts":[{"id":"1","name":"","phones":[]}]}`, "127.0.0.1:8765", "", a.Token)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	w = req(a, "GET", "/api/book", "", "localhost:8765", "", "")
	var b book.Book
	json.Unmarshal(w.Body.Bytes(), &b)
	if b.Revision != 1 || b.Contacts[0].Phones[0] != "00123" {
		t.Fatal(w.Body)
	}
}
func TestPhoneIsolationConditional(t *testing.T) {
	a := setup(t)
	for _, p := range []string{"/", "/api/book", "/app.js", "/phonebook.xml/"} {
		w := httptest.NewRecorder()
		a.Phones.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != 404 {
			t.Fatal(p, w.Code)
		}
	}
	w := httptest.NewRecorder()
	a.Phones.ServeHTTP(w, httptest.NewRequest("GET", "/phonebook.xml", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "YealinkIPPhoneDirectory") {
		t.Fatal(w)
	}
	r := httptest.NewRequest("GET", "/phonebook.xml", nil)
	r.Header.Set("If-None-Match", w.Header().Get("ETag"))
	w = httptest.NewRecorder()
	a.Phones.ServeHTTP(w, r)
	if w.Code != 304 {
		t.Fatal(w.Code)
	}
	w = httptest.NewRecorder()
	a.Phones.ServeHTTP(w, httptest.NewRequest("HEAD", "/phonebook.xml", nil))
	if w.Code != 200 || w.Body.Len() != 0 {
		t.Fatal(w)
	}
	w = httptest.NewRecorder()
	a.Phones.ServeHTTP(w, httptest.NewRequest("POST", "/phonebook.xml", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatal(w.Code)
	}
}
func TestSizeAndMalformed(t *testing.T) {
	a := setup(t)
	for _, b := range []string{strings.Repeat("x", (1<<20)+1), `{} {}`, `{"revision":0,"contacts":[],"extra":1}`} {
		w := req(a, "PUT", "/api/book", b, "127.0.0.1:8765", "", a.Token)
		if w.Code != 400 && w.Code != 413 {
			t.Fatal(w.Code)
		}
	}
}
func TestRequiredSaveFields(t *testing.T) {
	a := setup(t)
	for _, body := range []string{`{}`, `{"revision":0}`, `{"contacts":[]}`, `{"revision":0,"contacts":null}`} {
		w := req(a, "PUT", "/api/book", body, "127.0.0.1:8765", "", a.Token)
		if w.Code != 400 {
			t.Fatalf("accepted missing/null save fields %s: %d", body, w.Code)
		}
	}
}
func TestRealBodyLimits(t *testing.T) {
	a := setup(t)
	body := `{"revision":0,"contacts":[{"id":"1","name":"` + strings.Repeat("a", 1<<20) + `","phones":["1"]}]}`
	w := req(a, "PUT", "/api/book", body, "127.0.0.1:8765", "", a.Token)
	if w.Code != 413 {
		t.Fatal("oversized valid JSON", w.Code)
	}
	r := httptest.NewRequest("PUT", "http://127.0.0.1:8765/api/book", strings.NewReader(`{"revision":0,"contacts":[]}`))
	r.Header.Set("X-Phonebook-Token", a.Token)
	r.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	a.Admin.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatal(w.Code)
	}
}
func TestPhoneSubnet(t *testing.T) {
	s, e := store.Open(t.TempDir(), book.DefaultLimits(), "directory")
	if e != nil {
		t.Fatal(e)
	}
	c := config.Default()
	c.AllowedSubnet = "192.168.1.0/24"
	a, e := New(s, c)
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range []struct {
		ip   string
		code int
	}{{"192.168.1.12:1234", 200}, {"192.168.2.12:1234", 403}} {
		r := httptest.NewRequest("GET", "/phonebook.xml", nil)
		r.RemoteAddr = v.ip
		w := httptest.NewRecorder()
		a.Phones.ServeHTTP(w, r)
		if w.Code != v.code {
			t.Fatal(v, w.Code)
		}
	}
}
