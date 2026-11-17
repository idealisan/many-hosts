package main

import (
	"strings"
	"testing"
)

func TestParseHostsBlocks(t *testing.T) {
	data := strings.Join([]string{
		"127.0.0.1 localhost",
		";==hosts-profile-work-start",
		"",
		"127.0.0.1 work.local",
		";==hosts-profile-work-end",
		"",
		";==hosts-profile-dev-start",
		"",
		"; 127.0.0.1 dev.local",
		";==hosts-profile-dev-end",
		"",
	}, "\n")

	profiles := (&app{parts: loadParts(data)}).profiles()
	if len(profiles) != 3 {
		t.Fatalf("profiles = %d, want 3", len(profiles))
	}
	if profiles[0].Name != "原始内容" || profiles[0].Content != "127.0.0.1 localhost" {
		t.Fatalf("adopted = %+v", profiles[0])
	}
	if profiles[1].Name != "work" || !profiles[1].Enabled || profiles[1].Content != "127.0.0.1 work.local" {
		t.Fatalf("work = %+v", profiles[1])
	}
	if profiles[2].Name != "dev" || profiles[2].Enabled || profiles[2].Content != "127.0.0.1 dev.local" {
		t.Fatalf("dev = %+v", profiles[2])
	}
}

func TestAdoptRawHostsLines(t *testing.T) {
	data := strings.Join([]string{
		"",
		"127.0.0.1 localhost",
		"::1 localhost",
		"",
		";==hosts-profile-work-start",
		"",
		"127.0.0.1 work.local",
		";==hosts-profile-work-end",
		"",
	}, "\n")

	profiles := (&app{parts: loadParts(data)}).profiles()
	if len(profiles) != 2 {
		t.Fatalf("profiles = %d, want 2", len(profiles))
	}
	if profiles[0].Name != "原始内容" || !profiles[0].Enabled ||
		profiles[0].Content != "127.0.0.1 localhost\n::1 localhost" {
		t.Fatalf("adopted = %+v", profiles[0])
	}
	if profiles[1].Name != "work" {
		t.Fatalf("second = %+v", profiles[1])
	}
}

func TestAppleQuote(t *testing.T) {
	in := `C:\Users\a"b`
	want := `C:\\Users\\a\"b`
	if got := appleQuote(in); got != want {
		t.Fatalf("appleQuote = %q, want %q", got, want)
	}
}

func TestParseNewMarkers(t *testing.T) {
	data := strings.Join([]string{
		"#==hosts-profile-work-start",
		"",
		"# 127.0.0.1 work.local",
		"#==hosts-profile-work-end",
		"",
	}, "\n")
	profiles := (&app{parts: loadParts(data)}).profiles()
	if len(profiles) != 1 || profiles[0].Name != "work" || profiles[0].Enabled ||
		profiles[0].Content != "127.0.0.1 work.local" {
		t.Fatalf("profiles = %+v", profiles)
	}
}

func TestEncodeBlockToggle(t *testing.T) {
	p := &Profile{Name: "p", Content: "127.0.0.1 p.local", Enabled: true}
	wantOn := []string{
		"#==hosts-profile-p-start",
		"",
		"127.0.0.1 p.local",
		"#==hosts-profile-p-end",
	}
	gotOn := encodeBlock(p)
	if strings.Join(gotOn, "|") != strings.Join(wantOn, "|") {
		t.Fatalf("on = %q, want %q", gotOn, wantOn)
	}

	p.Enabled = false
	wantOff := []string{
		"#==hosts-profile-p-start",
		"",
		"# 127.0.0.1 p.local",
		"#==hosts-profile-p-end",
	}
	gotOff := encodeBlock(p)
	if strings.Join(gotOff, "|") != strings.Join(wantOff, "|") {
		t.Fatalf("off = %q, want %q", gotOff, wantOff)
	}
}
