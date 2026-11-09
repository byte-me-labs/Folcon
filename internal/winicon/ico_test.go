package winicon

import (
	"encoding/binary"
	"testing"
)

func TestParseICOReadsDeclaredWidth(t *testing.T) {
	largest, err := ParseICO(makeICO(0, []byte{1, 2, 3}))
	if err != nil {
		t.Fatalf("ParseICO() error: %v", err)
	}
	if largest != 256 {
		t.Fatalf("largest = %d, want 256 for a zero width byte", largest)
	}

	largest, err = ParseICO(makeICO(48, []byte{1, 2, 3}))
	if err != nil {
		t.Fatalf("ParseICO() error: %v", err)
	}
	if largest != 48 {
		t.Fatalf("largest = %d, want 48", largest)
	}
}

func TestParseICORejectsBadInput(t *testing.T) {
	for _, data := range [][]byte{
		{},
		{0, 0, 2, 0, 1, 0},
		make([]byte, 22),
	} {
		if _, err := ParseICO(data); err == nil {
			t.Fatalf("ParseICO(% x) accepted bad data", data)
		}
	}
}

func makeICO(width int, payload []byte) []byte {
	out := make([]byte, icoHeaderSize+icoEntrySize+len(payload))
	binary.LittleEndian.PutUint16(out[2:4], 1)
	binary.LittleEndian.PutUint16(out[4:6], 1)
	out[6] = byte(width % 256)
	out[7] = byte(width % 256)
	binary.LittleEndian.PutUint16(out[10:12], 1)
	binary.LittleEndian.PutUint16(out[12:14], 32)
	binary.LittleEndian.PutUint32(out[14:18], uint32(len(payload)))
	binary.LittleEndian.PutUint32(out[18:22], uint32(icoHeaderSize+icoEntrySize))
	copy(out[22:], payload)
	return out
}
