package memory

import (
	"errors"
	"fmt"
	"testing"
	"testing/fstest"

	template "github.com/gasmod/gas-template"
)

func TestRegisterAndGet(t *testing.T) {
	t.Parallel()
	s := NewStore()

	s.Register("emails/welcome.html", []byte("<h1>Welcome</h1>"))

	got, err := s.Get("emails/welcome.html")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if string(got) != "<h1>Welcome</h1>" {
		t.Errorf("Get() = %q, want %q", got, "<h1>Welcome</h1>")
	}
}

func TestGetNotFound(t *testing.T) {
	t.Parallel()
	s := NewStore()

	_, err := s.Get("nonexistent")
	if !errors.Is(err, template.ErrTemplateNotFound) {
		t.Errorf("Get() error = %v, want %v", err, template.ErrTemplateNotFound)
	}
}

func TestRegisterOverwrite(t *testing.T) {
	t.Parallel()
	s := NewStore()

	s.Register("page.html", []byte("v1"))
	s.Register("page.html", []byte("v2"))

	got, err := s.Get("page.html")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if string(got) != "v2" {
		t.Errorf("Get() = %q, want %q", got, "v2")
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	s := NewStore()

	s.Register("b.html", []byte("b"))
	s.Register("a.html", []byte("a"))
	s.Register("c.html", []byte("c"))

	names, err := s.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("List() returned %d names, want 3", len(names))
	}
	if names[0] != "a.html" || names[1] != "b.html" || names[2] != "c.html" {
		t.Errorf("List() = %v, want [a.html b.html c.html]", names)
	}
}

func TestListEmpty(t *testing.T) {
	t.Parallel()
	s := NewStore()

	names, err := s.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("List() returned %d names, want 0", len(names))
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	s := NewStore()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 100 {
			s.Register(fmt.Sprintf("t%d.html", i), []byte("content"))
		}
	}()

	for range 100 {
		_, _ = s.Get("t0.html")
		_, _ = s.List()
	}
	<-done
}

func TestRegisterFS(t *testing.T) {
	t.Parallel()
	s := NewStore()

	fsys := fstest.MapFS{
		"layouts/base.html":    {Data: []byte("<html>{{block \"content\" .}}{{end}}</html>")},
		"partials/header.html": {Data: []byte("<header>Header</header>")},
		"home.html":            {Data: []byte("{{define \"content\"}}Home{{end}}")},
		"readme.md":            {Data: []byte("# Readme")}, // should be skipped
	}

	if err := s.RegisterFS(fsys); err != nil {
		t.Fatalf("RegisterFS() error: %v", err)
	}

	names, _ := s.List()
	if len(names) != 3 {
		t.Fatalf("List() returned %d names, want 3 (non-.html should be skipped)", len(names))
	}

	got, err := s.Get("layouts/base.html")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if string(got) != "<html>{{block \"content\" .}}{{end}}</html>" {
		t.Errorf("Get() unexpected content: %q", got)
	}
}
