package server

import (
	"testing"
)

func TestEntry(t *testing.T) {
	var ent Entry
	root := CreateFSEntry("~/")

	if root.PathDot != "~/" ||
		root.level != 0 || root.Name() != "" {
		t.Fatal(root)
	}

	ent = root.Child("")
	if ent == nil || ent.(*FSEntry).PathDot != "~/" ||
		ent.(*FSEntry).level != 0 || ent.Name() != "" {
		t.Fatal(ent)
	}

	ent = root.Child("/")
	if ent == nil || ent.(*FSEntry).PathDot != "~/" ||
		ent.(*FSEntry).level != 0 || ent.Name() != "" {
		t.Fatal(ent)
	}

	ent = root.Child("").Child("//")
	if ent == nil || ent.(*FSEntry).PathDot != "~/" ||
		ent.(*FSEntry).level != 0 || ent.Name() != "" {
		t.Fatal(ent)
	}

	ent = root.Child("robots.txt")
	if ent == nil || ent.(*FSEntry).PathDot != "~/robots.txt." ||
		ent.(*FSEntry).level != 1 || ent.Name() != "robots.txt" {
		t.Fatal(ent)
	}

	ent = root.Child("html")
	if ent == nil || ent.(*FSEntry).PathDot != "~/html." ||
		ent.(*FSEntry).level != 1 || ent.Name() != "html" {
		t.Fatal(ent)
	}

	ent = root.Child("html/")
	if ent == nil || ent.(*FSEntry).PathDot != "~/html.dir/" ||
		ent.(*FSEntry).level != 1 || ent.Name() != "html" {
		t.Fatal(ent)
	}

	ent = root.Child("html").Child("index.html")
	if ent == nil || ent.(*FSEntry).PathDot != "~/html.dir/index.html." ||
		ent.(*FSEntry).level != 2 || ent.Name() != "index.html" {
		t.Fatal(ent)
	}

	ent = root.Child("html").Child("/")
	if ent == nil || ent.(*FSEntry).PathDot != "~/html.dir/" ||
		ent.(*FSEntry).level != 1 || ent.Name() != "html" {
		t.Fatal(ent)
	}

	ent = root.Child("html/").Child("/")
	if ent == nil || ent.(*FSEntry).PathDot != "~/html.dir/" ||
		ent.(*FSEntry).level != 1 || ent.Name() != "html" {
		t.Fatal(ent)
	}

	ent = root.Child("html/").Child("index.html")
	if ent == nil || ent.(*FSEntry).PathDot != "~/html.dir/index.html." ||
		ent.(*FSEntry).level != 2 || ent.Name() != "index.html" {
		t.Fatal(ent)
	}

	ent = root.Child("html/index.html")
	if ent == nil || ent.(*FSEntry).PathDot != "~/html.dir/index.html." ||
		ent.(*FSEntry).level != 2 || ent.Name() != "index.html" {
		t.Fatal(ent)
	}
}
