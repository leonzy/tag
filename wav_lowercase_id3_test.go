package tag

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// A valid Unicode ID3 tag must remain readable when the RIFF chunk uses
// lowercase id3, as written by common WAV taggers. RIFF INFO may contain
// unrecoverable ASCII question marks while that ID3 still has the real text.
func TestWAVLowercaseID3OverridesPlaceholderINFO(t *testing.T) {
	for _, chunkID := range []string{"ID3 ", "id3 "} {
		t.Run(chunkID, func(t *testing.T) {
			base := createTestWAV(44100, 2, 16, 0.02)
			var body bytes.Buffer
			body.Write(base[12:])
			var info bytes.Buffer
			info.WriteString("INFO")
			for _, field := range []string{"INAM", "IART", "IPRD"} {
				info.WriteString(field)
				binary.Write(&info, binary.LittleEndian, uint32(4))
				info.WriteString("????")
			}
			body.WriteString("LIST")
			binary.Write(&body, binary.LittleEndian, uint32(info.Len()))
			body.Write(info.Bytes())
			tagged := createTestWAVWithID3(WriteOptions{Title: "壁虎漫步", Artist: "潘玮柏", Album: "壁虎漫步"})
			at := bytes.Index(tagged, []byte("ID3 "))
			n := int(binary.LittleEndian.Uint32(tagged[at+4 : at+8]))
			body.WriteString(chunkID)
			body.Write(tagged[at+4 : at+8+n+n%2])
			var wav bytes.Buffer
			wav.WriteString("RIFF")
			binary.Write(&wav, binary.LittleEndian, uint32(4+body.Len()))
			wav.WriteString("WAVE")
			wav.Write(body.Bytes())
			metadata, err := ReadFrom(bytes.NewReader(wav.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if metadata.Title() != "壁虎漫步" || metadata.Artist() != "潘玮柏" || metadata.Album() != "壁虎漫步" {
				t.Fatalf("Unicode ID3 ignored: title=%q artist=%q album=%q", metadata.Title(), metadata.Artist(), metadata.Album())
			}
		})
	}
}
