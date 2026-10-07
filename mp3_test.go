package tag

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMP3VBRDuration(t *testing.T) {
	// 测试 VBR MP3 文件的时长计算
	// 这个测试文件应该有 Xing/LAME 头
	testFiles := []struct {
		name        string
		expectedDur time.Duration
		tolerance   time.Duration
	}{
		{
			name:        "testdata/with_tags/sample.vbr.mp3",
			expectedDur: 249984 * time.Millisecond,
			tolerance:   time.Microsecond,
		},
	}

	for _, tc := range testFiles {
		t.Run(tc.name, func(t *testing.T) {
			file, err := os.Open(tc.name)
			if err != nil {
				t.Skipf("test file not found: %v", err)
			}
			defer file.Close()

			meta, err := ReadFrom(file)
			if err != nil {
				t.Fatalf("failed to read metadata: %v", err)
			}

			duration := meta.Duration()
			if duration == 0 {
				t.Fatal("duration is zero")
			}

			// 检查时长是否在容忍范围内
			diff := duration - tc.expectedDur
			if diff < 0 {
				diff = -diff
			}
			if diff > tc.tolerance {
				t.Errorf("duration %v differs from expected %v by %v (tolerance: %v)",
					duration, tc.expectedDur, diff, tc.tolerance)
			}

			t.Logf("Duration: %v (%.3f seconds)", duration, duration.Seconds())
		})
	}
}

func TestMP3CBRDuration(t *testing.T) {
	// 128kbps、3202000 字节音频应为 200.125s；帧头、CRC、首帧 padding
	// 都不能重复计入。此处无 Xing，测试 CBR 估算而非解码器。
	for _, header := range [][]byte{{0xff, 0xfb, 0x90, 0}, {0xff, 0xfa, 0x90, 0}, {0xff, 0xfb, 0x92, 0}} {
		for _, trailer := range []bool{false, true} {
			t.Run(fmt.Sprintf("header=%x/trailer=%v", header, trailer), func(t *testing.T) {
				data := make([]byte, 10+3202000)
				copy(data, []byte{'I', 'D', '3', 3, 0, 0, 0, 0, 0, 0})
				copy(data[10:], header)
				if trailer {
					data = append(data, make([]byte, 128)...)
					copy(data[len(data)-128:], "TAG")
				}
				meta, err := ReadFrom(bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				if got, want := meta.Duration(), 200125*time.Millisecond; got != want {
					t.Fatalf("duration = %v, want %v", got, want)
				}
			})
		}
	}
}

func TestMP3XingFractionalDuration(t *testing.T) {
	for _, tc := range []struct {
		name       string
		header     []byte
		sideBytes  int
		sampleRate int64
		samples    int64
	}{
		{"mpeg1 stereo", []byte{0xff, 0xfb, 0x90, 0}, 32, 44100, 1152},
		{"mpeg1 mono", []byte{0xff, 0xfb, 0x90, 0xc0}, 17, 44100, 1152},
		{"mpeg2 stereo", []byte{0xff, 0xf3, 0x80, 0}, 17, 22050, 576},
		{"mpeg2 mono", []byte{0xff, 0xf3, 0x80, 0xc0}, 9, 22050, 576},
	} {
		for _, id3v2 := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/id3v2=%v", tc.name, id3v2), func(t *testing.T) {
				data := make([]byte, 417)
				copy(data, tc.header)
				offset := 4 + tc.sideBytes
				copy(data[offset:], "Xing")
				binary.BigEndian.PutUint32(data[offset+4:], 1)
				binary.BigEndian.PutUint32(data[offset+8:], 7793)
				if id3v2 {
					data = append([]byte{'I', 'D', '3', 3, 0, 0, 0, 0, 0, 0}, data...)
				} else {
					data = append(data, make([]byte, 128)...)
					copy(data[len(data)-128:], "TAG")
				}
				meta, err := ReadFrom(bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				want := time.Duration(7793*tc.samples) * time.Second / time.Duration(tc.sampleRate)
				if delta := meta.Duration() - want; delta < -time.Nanosecond || delta > time.Nanosecond {
					t.Fatalf("duration = %v, want %v", meta.Duration(), want)
				}
			})
		}
	}
}

func TestMP3WithPaddingAfterID3v2(t *testing.T) {
	file, err := os.Open("testdata/with_tags/sample.padded.mp3")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer file.Close()

	meta, err := ReadFrom(file)
	if err != nil {
		t.Fatalf("failed to read metadata: %v", err)
	}

	if meta.Title() != "半壶纱" {
		t.Errorf("title = %q, want %q", meta.Title(), "半壶纱")
	}
	if meta.Artist() != "刘珂矣" {
		t.Errorf("artist = %q, want %q", meta.Artist(), "刘珂矣")
	}
	if meta.Duration() == 0 {
		t.Error("duration is zero")
	}

	expected := 222 * time.Second
	diff := meta.Duration() - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > 2*time.Second {
		t.Errorf("duration %v differs from expected %v by %v", meta.Duration(), expected, diff)
	}

	t.Logf("Title: %s, Artist: %s, Duration: %v, BitRate: %d, SampleRate: %d",
		meta.Title(), meta.Artist(), meta.Duration(), meta.BitRate(), meta.SampleRate())
}
