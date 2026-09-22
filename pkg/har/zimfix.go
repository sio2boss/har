package har

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	"github.com/cookiengineer/gozim/archive/zim"
)

// RepairZim fills in the pieces Kiwix requires and gozim's writer leaves out:
// a terminated MIME list, a v0 title index, a non-zero UUID, and a real MD5.
// It does not link libzim or any C++ code.
func RepairZim(path string) error {
	archive, err := zim.Open(path)
	if err != nil {
		return err
	}
	dirents := archive.Dirents()
	keys := make([]zimTitleKey, len(dirents))
	for i, d := range dirents {
		title := d.Title
		if title == "" {
			title = d.Path
		}
		keys[i] = zimTitleKey{index: uint32(i), ns: byte(d.Namespace), title: title, path: d.Path}
	}
	archive.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < zim.HeaderSize+zim.ChecksumSize {
		return fmt.Errorf("zim: file too small")
	}

	if at, ok := mimeListNeedsTerminator(data, int(u64(data, 56))); ok {
		data = insertAt(data, at, []byte{0})
		shiftAbsoluteOffsets(data, uint64(at), 1, true)
	}

	if u64(data, 40) == ^uint64(0) {
		order := titleOrder(keys)
		raw := make([]byte, len(order)*4)
		for i, idx := range order {
			binary.LittleEndian.PutUint32(raw[i*4:], idx)
		}
		at := int(u64(data, 48))
		data = insertAt(data, at, raw)
		shiftAbsoluteOffsets(data, uint64(at), uint64(len(raw)), false)
		binary.LittleEndian.PutUint64(data[40:48], uint64(at))
	} else if u64(data, 40) == u64(data, 48) {
		// A previous repair inserted the index but then slid the header
		// pointer onto the cluster list. The index is the bytes just before it.
		entries := uint64(u32(data, 24))
		at := u64(data, 48) - entries*4
		if titleIndexLooksValid(data, at, int(entries)) {
			binary.LittleEndian.PutUint64(data[40:48], at)
		}
	}

	if bytes.Equal(data[8:24], make([]byte, 16)) {
		if _, err := rand.Read(data[8:24]); err != nil {
			return err
		}
	}

	checksumPos := u64(data, 72)
	if checksumPos+uint64(zim.ChecksumSize) != uint64(len(data)) {
		return fmt.Errorf("zim: checksum is not at end of file (%d vs %d)", checksumPos, len(data))
	}
	sum := md5.Sum(data[:checksumPos])
	copy(data[checksumPos:], sum[:])

	return os.WriteFile(path, data, 0644)
}

type zimTitleKey struct {
	index uint32
	ns    byte
	title string
	path  string
}

func titleIndexLooksValid(data []byte, at uint64, entries int) bool {
	if entries == 0 || at+uint64(entries*4) > uint64(len(data)) {
		return false
	}
	seen := make(map[uint32]struct{}, entries)
	for i := 0; i < entries; i++ {
		idx := binary.LittleEndian.Uint32(data[int(at)+i*4:])
		if int(idx) >= entries {
			return false
		}
		seen[idx] = struct{}{}
	}
	return len(seen) == entries
}

func titleOrder(keys []zimTitleKey) []uint32 {
	sorted := append([]zimTitleKey(nil), keys...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ns != sorted[j].ns {
			return sorted[i].ns < sorted[j].ns
		}
		if sorted[i].title != sorted[j].title {
			return sorted[i].title < sorted[j].title
		}
		return sorted[i].path < sorted[j].path
	})
	out := make([]uint32, len(sorted))
	for i, k := range sorted {
		out[i] = k.index
	}
	return out
}

func mimeListNeedsTerminator(data []byte, start int) (int, bool) {
	i := start
	for i < len(data) {
		if data[i] == 0 {
			return i, false
		}
		end := bytes.IndexByte(data[i:], 0)
		if end < 0 {
			return i, true
		}
		if !isMimeToken(data[i : i+end]) {
			return i, true
		}
		i += end + 1
	}
	return i, true
}

func isMimeToken(s []byte) bool {
	if len(s) == 0 || len(s) > 128 || !bytes.Contains(s, []byte{'/'}) {
		return false
	}
	for _, b := range s {
		if b < 0x20 || b > 0x7e {
			return false
		}
	}
	return true
}

func insertAt(data []byte, at int, extra []byte) []byte {
	out := make([]byte, len(data)+len(extra))
	copy(out, data[:at])
	copy(out[at:], extra)
	copy(out[at+len(extra):], data[at:])
	return out
}

// shiftAbsoluteOffsets adds delta to file offsets at or after threshold.
// includeURL is false when the insert sits after the URL pointer list, so
// those pointers still address the same directory entries.
func shiftAbsoluteOffsets(data []byte, threshold, delta uint64, includeURL bool) {
	bump64(data, 32, threshold, delta) // url pointer list
	bump64(data, 40, threshold, delta) // title index
	bump64(data, 48, threshold, delta) // cluster pointer list
	bump64(data, 56, threshold, delta) // mime list
	bump64(data, 72, threshold, delta) // checksum

	entries := int(u32(data, 24))
	if includeURL {
		urlPos := int(u64(data, 32))
		for i := 0; i < entries; i++ {
			bump64(data, urlPos+i*8, threshold, delta)
		}
	}
	clusters := int(u32(data, 28))
	clusterPos := int(u64(data, 48))
	for i := 0; i < clusters; i++ {
		bump64(data, clusterPos+i*8, threshold, delta)
	}
}

func bump64(data []byte, at int, threshold, delta uint64) {
	if at < 0 || at+8 > len(data) {
		return
	}
	v := binary.LittleEndian.Uint64(data[at:])
	if v == ^uint64(0) || v < threshold {
		return
	}
	binary.LittleEndian.PutUint64(data[at:], v+delta)
}

func u64(data []byte, at int) uint64 {
	return binary.LittleEndian.Uint64(data[at : at+8])
}

func u32(data []byte, at int) uint32 {
	return binary.LittleEndian.Uint32(data[at : at+4])
}
