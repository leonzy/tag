package tag

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"testing"
)

type cueReadProbe struct {
	*bytes.Reader
	forbidden [][2]int64
	readBytes int
}

func (r *cueReadProbe) Read(p []byte) (int, error) {
	position, _ := r.Reader.Seek(0, io.SeekCurrent)
	for _, span := range r.forbidden {
		if position < span[1] && position+int64(len(p)) > span[0] {
			return 0, fmt.Errorf("CUE scan read an unrelated metadata payload at %d", position)
		}
	}
	n, err := r.Reader.Read(p)
	r.readBytes += n
	return n, err
}

func cueLightFixture(coverSize int, cueFirst, includeCue bool) ([]byte, [][2]int64) {
	data := []byte("fLaC")
	var skipped [][2]int64
	appendBlock := func(kind byte, payload []byte, last bool, skip bool) {
		if last {
			kind |= 0x80
		}
		data = append(data, kind, byte(len(payload)>>16), byte(len(payload)>>8), byte(len(payload)))
		start := int64(len(data))
		data = append(data, payload...)
		if skip {
			skipped = append(skipped, [2]int64{start, int64(len(data))})
		}
	}
	info := make([]byte, 34)
	binary.BigEndian.PutUint64(info[10:18], uint64(44100)<<44|uint64(1)<<41|uint64(15)<<36|uint64(44100*60))
	appendBlock(0, info, false, false)
	cue := make([]byte, 396)
	cue[395] = 3
	for i, number := range []byte{1, 2, 170} {
		track := make([]byte, 36)
		binary.BigEndian.PutUint64(track[:8], uint64(i*44100*30))
		track[8] = number
		copy(track[9:21], "USA111111111")
		track[35] = 1
		index := make([]byte, 12)
		index[8] = 1
		cue = append(cue, track...)
		cue = append(cue, index...)
	}
	var picture bytes.Buffer
	for _, value := range []uint32{3, 10} {
		_ = binary.Write(&picture, binary.BigEndian, value)
	}
	picture.WriteString("image/jpeg")
	for _, value := range []uint32{0, 1, 1, 24, 0, uint32(coverSize)} {
		_ = binary.Write(&picture, binary.BigEndian, value)
	}
	picture.Write(bytes.Repeat([]byte{0x42}, coverSize))
	var comment bytes.Buffer
	_ = binary.Write(&comment, binary.LittleEndian, uint32(0))
	_ = binary.Write(&comment, binary.LittleEndian, uint32(1))
	title := "TITLE=Album image"
	_ = binary.Write(&comment, binary.LittleEndian, uint32(len(title)))
	comment.WriteString(title)
	appendBlock(4, comment.Bytes(), false, true)
	if includeCue && cueFirst {
		appendBlock(5, cue, false, false)
	}
	appendBlock(6, picture.Bytes(), !includeCue || cueFirst, true)
	if includeCue && !cueFirst {
		appendBlock(5, cue, true, false)
	}
	return data, skipped
}

func TestFLACCUESheetSkipsUnrelatedPayloads(t *testing.T) {
	for _, cueFirst := range []bool{false, true} {
		data, forbidden := cueLightFixture(8<<20, cueFirst, true)
		r := &cueReadProbe{Reader: bytes.NewReader(data), forbidden: forbidden}
		meta, err := ReadFLACCUESheet(r)
		if err != nil {
			t.Fatal(err)
		}
		cue := meta.(CUESheetProvider).CUESheetBlock()
		if meta.SampleRate() != 44100 || len(cue.Tracks) != 2 || cue.Tracks[1].OffsetSample != 44100*30 || cue.Tracks[0].ISRC != "USA111111111" || len(cue.Tracks[0].IndexPoints) != 1 {
			t.Fatalf("CUE metadata changed: %+v", cue)
		}
		if meta.Picture() != nil || meta.Title() != "" || r.readBytes > 4096 {
			t.Fatalf("unneeded metadata retained/read: %d bytes", r.readBytes)
		}
	}
}

func TestFLACCUESheetWithoutCueAndTruncatedSkippedBlock(t *testing.T) {
	data, forbidden := cueLightFixture(1<<20, false, false)
	r := &cueReadProbe{Reader: bytes.NewReader(data), forbidden: forbidden}
	meta, err := ReadFLACCUESheet(r)
	if err != nil || meta.(CUESheetProvider).CUESheetBlock() != nil {
		t.Fatal(meta, err)
	}
	if _, err = ReadFLACCUESheet(bytes.NewReader(data[:len(data)-3])); err == nil {
		t.Fatal("truncated skipped picture accepted")
	}
}

func TestNormalFLACMetadataStillReadsArtworkAndTags(t *testing.T) {
	data, _ := cueLightFixture(1234, false, true)
	meta, err := ReadFLACMeta(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Picture() == nil || len(meta.Picture().Data) != 1234 || meta.Title() != "Album image" || len(meta.(CUESheetProvider).CUESheetBlock().Tracks) != 2 {
		t.Fatal("normal tags/artwork changed")
	}
}

func BenchmarkFLACCUEReader(b *testing.B) {
	data, _ := cueLightFixture(8<<20, false, true)
	for _, lightweight := range []bool{false, true} {
		name := "full-metadata"
		if lightweight {
			name = "cue-only"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var err error
				if lightweight {
					_, err = ReadFLACCUESheet(bytes.NewReader(data))
				} else {
					_, err = ReadFLACMeta(bytes.NewReader(data))
				}
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
