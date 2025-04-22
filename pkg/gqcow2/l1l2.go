package gqcow2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type L1Entry struct {
	Index         int
	L2TableOffset uint64
	// false means refcount == 0 or refcont >=2
	// true means refcount == 1
	// from read perspective, false means need to
	// further look into the refcount
	// true means read
	RefCountBit bool
}

type L2Entry struct {
	// flase for cluster that are:
	//  unused, compressed or require COW
	// true for standard clusters whose refcount == 1
	Flag bool

	// only one may exist
	Standard   *StandardDescriptor
	Compressed *CompressedDescriptor
}

func (l2e L2Entry) Valid() bool {
	// must be one of it
	return l2e.Standard != nil || l2e.Compressed != nil
}

// LoadL1Table load the l1 table content from the Image.
func (i *Image) LoadL1Table() error {
	clusterSize := i.Header.ClusterSize()
	offset := i.Header.L1TableOffset
	totalTableSize := i.Header.L1Size * uint32(clusterSize)
	totalEntryCount := totalTableSize / 8 // each L1 table entry is 64bit

	tableBuf := make([]byte, totalTableSize)

	rc, err := i.Handler.ReadAt(tableBuf, int64(offset))
	if err != nil {
		return err
	}
	// even its read, but corrupted, should abort
	if rc < int(totalTableSize) {
		return io.ErrUnexpectedEOF
	}

	if i.L1Table == nil {
		i.L1Table = make([]L1Entry, 0, totalEntryCount)
	}
	// only the 9-55bits are meaningful
	l2OffsetMask := uint64((1 << 47) - 1<<9)

	for index := range totalEntryCount {
		// each entry takes 8 bytes
		e := binary.BigEndian.Uint64(tableBuf[index : index+8])

		newEntry := L1Entry{
			Index:         int(index),
			L2TableOffset: e & l2OffsetMask,
		}

		if newEntry.L2TableOffset%uint64(clusterSize) != 0 {
			return errors.New("corrupted L1 table, L2 offset not aligned to cluster boundary")
		}

		if (e>>63)&1 == 1 {
			newEntry.RefCountBit = true
		} else {
			newEntry.RefCountBit = false
		}

		i.L1Table = append(i.L1Table, newEntry)
	}

	return nil
}

// FindL2Entry takes virtual disk's offset as input, and return provide the l2 table entry
func (i *Image) FindL2Entry(vdOffset uint64) (L2Entry, error) {
	// each L2 table entry take 64bits, 8bytes
	// and each L2 table takes 1 cluster size
	l2EntryCountPerTable := i.Header.ClusterSize() / 8

	l1Index := (vdOffset / uint64(i.Header.ClusterSize())) / uint64(l2EntryCountPerTable)
	l2Index := (vdOffset / uint64(i.Header.ClusterSize())) % uint64(l2EntryCountPerTable)

	l2TableStart := i.L1Table[l1Index].L2TableOffset
	fmt.Printf("l1 table :%#v", i.L1Table[l1Index])

	// read the l2 table
	rawL2Table := make([]byte, i.Header.ClusterSize())
	if _, err := i.Handler.ReadAt(rawL2Table, int64(l2TableStart)); err != nil {
		return L2Entry{}, errors.Join(
			fmt.Errorf("reading l2 entry failed, offset %d at image file", l2TableStart),
			err)
	}

	return extractL2Entry(rawL2Table, l2Index, i.Header.ClusterBits), nil
}

func extractL2Entry(block []byte, index uint64, cb uint32) L2Entry {
	offset := index * 8

	rawEntry := binary.BigEndian.Uint64(block[offset : offset+8])
	fmt.Printf("rawentry: %#v", rawEntry)

	descriptorType := (rawEntry >> 62) & 1
	flag := (rawEntry >> 63) & 1
	entry := L2Entry{
		Flag: flag == 1,
	}
	if descriptorType == 0 {
		sd := &StandardDescriptor{}
		mask := uint64((1 << 47) - 1<<9)
		sd.DataOffset = mask & rawEntry
		sd.AllZero = rawEntry&1 == 1
		entry.Standard = sd
	} else {
		cd := &CompressedDescriptor{}
		split := 62 - (cb - 8)

		cd.DataOffset = rawEntry & uint64((1<<split)-1)
		// cd.AdditionalSectorCount = int((rawEntry >> split) & (1 << (62 - split)))
		cd.AdditionalSectorCount = int((rawEntry >> split) & ((1 << (62 - split)) - 1))
		entry.Compressed = cd
	}

	return entry
}
