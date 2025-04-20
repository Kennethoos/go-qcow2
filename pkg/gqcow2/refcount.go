package gqcow2

import (
	"encoding/binary"
	"fmt"
	"io"
)

const RefCountTableEntrySizeByte = 8

// each entry points to a refcount block
type RefCountTableEntry struct {
	// the index inside the refcount table
	Index int
	// offset into the image file at which
	// the refcount block starts
	RefCountBlockOffset uint64
}

//// each entry stores the reference count for
//// a cluster
//type RefCountBlockEntry struct {
//	// the index of current entry in ALL entries
//	Index    int
//	RefCount int
//}

func (i *Image) LoadRefcountTable() error {
	clusterSize := i.Header.ClusterSize()
	offset := i.Header.RefCountTableOffset
	// table size may expand to MBs based on qcow2 virtual disk size
	totalTableSize := i.Header.RefcountTableClusters * uint32(clusterSize)
	// calculate the total number of refcount table entry
	totalEntryCount := totalTableSize / RefCountTableEntrySizeByte

	tableBuf := make([]byte, totalTableSize)
	rc, err := i.Input.ReadAt(tableBuf, int64(offset))
	if err != nil {
		return err
	}
	// even its read, but corrupted, should abort
	if rc < int(totalTableSize) {
		return io.ErrUnexpectedEOF
	}

	if i.RefCountTable == nil {
		i.RefCountTable = make([]RefCountTableEntry, 0, totalEntryCount)
	}

	for index := range totalEntryCount {
		// each entry takes 8 bytes
		e := binary.BigEndian.Uint64(tableBuf[index : index+8])

		newEntry := RefCountTableEntry{
			Index: int(index),
			// only the last 55bits are meaningful
			RefCountBlockOffset: (e & ((1 << 55) - 1)) << 9,
		}

		i.RefCountTable = append(i.RefCountTable, newEntry)

	}

	return nil
}

// ReadRefCount query the image file to find out given offset's refcount
func (i *Image) ReadRefCount(offset uint64) (int, error) {
	refCountTableIndex := (offset / uint64(i.Header.ClusterSize()) / uint64(i.Header.RefCountBlockEntryCount()))
	refCountBlockIndex := (offset / uint64(i.Header.ClusterSize()) % uint64(i.Header.RefCountBlockEntryCount()))

	blockOffset := i.RefCountTable[refCountTableIndex].RefCountBlockOffset
	if blockOffset == 0 {
		return 0, nil
	}

	// read the block
	rawBlock := make([]byte, 0, i.Header.ClusterSize())
	if _, err := i.Input.ReadAt(rawBlock, int64(blockOffset)); err != nil {
		return 0, err
	}

	refcount, err := extractRefCount(rawBlock, refCountBlockIndex, i.Header.RefCountBit())
	if err != nil {
		return 0, err
	}

	return refcount, nil
}

func extractRefCount(block []byte, index uint64, entryBitSize int) (int, error) {
	offset := index * uint64(entryBitSize)

	// support 16 and 32 bit entry size for now

	byteIndex := offset / 8
	bitOffset := offset % 8

	if bitOffset != 0 {
		return 0, fmt.Errorf("not valid refcount bits[%d]", bitOffset)
	}

	if entryBitSize == 16 {
		return int(binary.BigEndian.Uint16(block[byteIndex : byteIndex+2])), nil
	}

	if entryBitSize == 32 {
		return int(binary.BigEndian.Uint32(block[byteIndex : byteIndex+4])), nil
	}

	return 0, fmt.Errorf("not valid refcount bits[%d]", bitOffset)
}

//func extraUint16(block []byte, offset int) uint16 {
//	// big endian only for now
//	byteIndex := offset / 8
//	bitOffset := offset % 8
//
//	// uint16 will expand two bytes
//
//	// already aligned
//	if bitOffset == 0 {
//		return binary.BigEndian.Uint16(block[byteIndex : byteIndex+2])
//	}
//
//	// like bigEndian implementation
//	return uint16((block[byteIndex])<<bitOffset|block[byteIndex+1]>>(8-bitOffset)) |
//		uint16((block[byteIndex+1]<<(8-bitOffset) | block[byteIndex+2]>>byte(bitOffset)))
//}
