// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"reflect"
	"testing"
)

// readTFrameValues must split an ID3v2.4 null-separated multi-value text frame
// into individual artists and drop empty parts. The encoding byte (0=ISO-8859-1)
// is the first byte of a T-frame body; values are separated by singleZero.
func TestReadTFrameValuesMultiValue(t *testing.T) {
	// encoding=ISO-8859-1(0), value = "A\x00B\x00C"
	body := append([]byte{0}, []byte("A\x00B\x00C")...)
	got, err := readTFrameValues(body)
	if err != nil {
		t.Fatalf("readTFrameValues: %v", err)
	}
	want := []string{"A", "B", "C"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A single-value frame yields a one-element slice (no null byte present).
func TestReadTFrameValuesSingle(t *testing.T) {
	body := append([]byte{0}, []byte("AC/DC")...)
	got, err := readTFrameValues(body)
	if err != nil {
		t.Fatalf("readTFrameValues: %v", err)
	}
	want := []string{"AC/DC"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// Band names containing separators must NOT be split: this is the whole
	// point of reading structured multi-value instead of guessing on the merged
	// string.
}

// Empty/whitespace-only parts are dropped.
func TestReadTFrameValuesDropsEmpty(t *testing.T) {
	body := append([]byte{0}, []byte("A\x00\x00B")...)
	got, err := readTFrameValues(body)
	if err != nil {
		t.Fatalf("readTFrameValues: %v", err)
	}
	want := []string{"A", "B"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// singleArtistList wraps a non-empty string and returns nil for empty.
func TestSingleArtistList(t *testing.T) {
	if got := singleArtistList(""); got != nil {
		t.Errorf("empty: got %v, want nil", got)
	}
	if got := singleArtistList("  "); got != nil {
		t.Errorf("blank: got %v, want nil", got)
	}
	if got := singleArtistList("X"); !reflect.DeepEqual(got, []string{"X"}) {
		t.Errorf("got %v, want [X]", got)
	}
}

// nonEmptyArtists trims and drops empties.
func TestNonEmptyArtists(t *testing.T) {
	if got := nonEmptyArtists([]string{" A ", "", "B", "  "}); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("got %v, want [A B]", got)
	}
	if got := nonEmptyArtists([]string{"", "  "}); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// metadataVorbis must preserve multiple ARTIST fields from a Vorbis comment
// block so a duet exposes each artist via Artists(). We feed a hand-built
// comment block: vendor(0)+count(2)+two ARTIST= lines.
func TestVorbisArtistsMultiField(t *testing.T) {
	var buf bytes.Buffer
	// vendor string: length + bytes (empty)
	buf.Write([]byte{0, 0, 0, 0})
	// comment count = 2 (little-endian uint32)
	buf.Write([]byte{2, 0, 0, 0})
	writeComment := func(s string) {
		b := []byte(s)
		// little-endian length
		buf.Write([]byte{byte(len(b)), 0, 0, 0})
		buf.Write(b)
	}
	writeComment("ARTIST=Alice")
	writeComment("ARTIST=Bob")

	m := newMetadataVorbis()
	if err := m.readVorbisComment(&buf); err != nil {
		t.Fatalf("readVorbisComment: %v", err)
	}
	got := m.Artists()
	want := []string{"Alice", "Bob"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	// AlbumArtists falls back to nil (no albumartist field).
	if got := m.AlbumArtists(); got != nil {
		t.Errorf("AlbumArtists: got %v, want nil", got)
	}
}
