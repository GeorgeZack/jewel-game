package text

import "testing"

func TestLoadNestedAndFallback(t *testing.T) {
	t.Cleanup(func() { Reset("en") })
	err := Load("fr", []byte("ui:\n  title: Joyaux\nlog:\n  cast: \"%s lance %s.\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if Language() != "fr" {
		t.Fatalf("language %q", Language())
	}
	if got := T("ui.title", "Jewel Game"); got != "Joyaux" {
		t.Fatalf("title %q", got)
	}
	if got := T("ui.missing", "Keep"); got != "Keep" {
		t.Fatalf("fallback %q", got)
	}
	if got := Tf("log.cast", "%s casts %s.", "Ada", "Smite"); got != "Ada lance Smite." {
		t.Fatalf("format %q", got)
	}
	// an empty translation is a real value, not a missing key
	if err := Load("fr", []byte("ui:\n  title: \"\"\n")); err != nil {
		t.Fatal(err)
	}
	if got := T("ui.title", "Jewel Game"); got != "" {
		t.Fatalf("empty %q", got)
	}
}

func TestBadYAMLKeepsTheOldCatalog(t *testing.T) {
	t.Cleanup(func() { Reset("en") })
	if err := Load("en", []byte("ui:\n  title: Hello\n")); err != nil {
		t.Fatal(err)
	}
	if err := Load("en", []byte("ui: [\n")); err == nil {
		t.Fatal("expected an error")
	}
	if got := T("ui.title", "nope"); got != "Hello" {
		t.Fatalf("catalog changed to %q", got)
	}
}
