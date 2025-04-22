package gqcow2

import (
	"encoding/binary"
	"errors"
)

const QCOW2MagicNumber = "QFI\xfb"

// for V2, fixed format

//Byte  0 -  3:   magic
//                QCOW magic string ("QFI\xfb")
//
//      4 -  7:   version
//                Version number (valid values are 2 and 3)
//
//      8 - 15:   backing_file_offset
//                Offset into the image file at which the backing file name
//                is stored (NB: The string is not null terminated). 0 if the
//                image doesn't have a backing file.
//
//     16 - 19:   backing_file_size
//                Length of the backing file name in bytes. Must not be
//                longer than 1023 bytes. Undefined if the image doesn't have
//                a backing file.
//
//     20 - 23:   cluster_bits
//                Number of bits that are used for addressing an offset
//                within a cluster (1 << cluster_bits is the cluster size).
//                Must not be less than 9 (i.e. 512 byte clusters).
//
//                Note: qemu as of today has an implementation limit of 2 MB
//                as the maximum cluster size and won't be able to open images
//                with larger cluster sizes.
//
//     24 - 31:   size
//                Virtual disk size in bytes
//
//     32 - 35:   crypt_method
//                0 for no encryption
//                1 for AES encryption
//
//     36 - 39:   l1_size
//                Number of entries in the active L1 table
//
//     40 - 47:   l1_table_offset
//                Offset into the image file at which the active L1 table
//                starts. Must be aligned to a cluster boundary.
//
//     48 - 55:   refcount_table_offset
//                Offset into the image file at which the refcount table
//                starts. Must be aligned to a cluster boundary.
//
//     56 - 59:   refcount_table_clusters
//                Number of clusters that the refcount table occupies
//
//     60 - 63:   nb_snapshots
//                Number of snapshots contained in the image
//
//     64 - 71:   snapshots_offset
//                Offset into the image file at which the snapshot table
//                starts. Must be aligned to a cluster boundary.

// expanded format for v3

//     72 -  79:  incompatible_features
//                Bitmask of incompatible features. An implementation must
//                fail to open an image if an unknown bit is set.
//
//                Bit 0:      Dirty bit.  If this bit is set then refcounts
//                            may be inconsistent, make sure to scan L1/L2
//                            tables to repair refcounts before accessing the
//                            image.
//
//                Bit 1:      Corrupt bit.  If this bit is set then any data
//                            structure may be corrupt and the image must not
//                            be written to (unless for regaining
//                            consistency).
//
//                Bits 2-63:  Reserved (set to 0)
//
//     80 -  87:  compatible_features
//                Bitmask of compatible features. An implementation can
//                safely ignore any unknown bits that are set.
//
//                Bit 0:      Lazy refcounts bit.  If this bit is set then
//                            lazy refcount updates can be used.  This means
//                            marking the image file dirty and postponing
//                            refcount metadata updates.
//
//                Bits 1-63:  Reserved (set to 0)
//
//     88 -  95:  autoclear_features
//                Bitmask of auto-clear features. An implementation may only
//                write to an image with unknown auto-clear features if it
//                clears the respective bits from this field first.
//
//                Bit 0:      Bitmaps extension bit
//                            This bit indicates consistency for the bitmaps
//                            extension data.
//
//                            It is an error if this bit is set without the
//                            bitmaps extension present.
//
//                            If the bitmaps extension is present but this
//                            bit is unset, the bitmaps extension data must be
//                            considered inconsistent.
//
//                Bits 1-63:  Reserved (set to 0)
//
//     96 -  99:  refcount_order
//                Describes the width of a reference count block entry (width
//                in bits: refcount_bits = 1 << refcount_order). For version 2
//                images, the order is always assumed to be 4
//                (i.e. refcount_bits = 16).
//                This value may not exceed 6 (i.e. refcount_bits = 64).
//
//    100 - 103:  header_length
//                Length of the header structure in bytes. For version 2
//                images, the length is always assumed to be 72 bytes.

// there are other optional sections called header extension directly after image header

type Header struct {
	// for v2, there are fixed 72 bytes in the header, big-endian.

	// valid: 2 or 3
	Version uint32
	// 8 bytes
	BackingFileOffset uint64
	BackingFileSize   uint32

	// cluster size is 1 << cluster bites
	ClusterBits uint32
	// virtual disk size in bytes
	Size        uint64
	CryptMethod bool // 0 for no encryption, 1 for AES, 2 for LUKS

	L1Size        uint32
	L1TableOffset uint64

	RefCountTableOffset   uint64
	RefcountTableClusters uint32

	NumSnapshots   uint32
	SnapshotOffset uint64

	// these fields only meaningful for v3
	// ... other ignored fields...
	RefCountOrder uint32
	// 4bytes, 100 - 103
	Length uint32
}

// ClusetrSize is in bytes
func (h *Header) ClusterSize() int {
	return 1 << h.ClusterBits
}

// RefCountBit is in bits, this is the
// refcount block entry's length, using bits
// is because the length maybe sub-byte
func (h *Header) RefCountBit() int {
	if h.Version == 2 {
		return 16 // refcount_bits is 16 for v2
	}

	return 1 << h.RefCountOrder
}

// The number of refcount block entries inside 1
// refcount block
func (h *Header) RefCountBlockEntryCount() int {
	return h.ClusterSize() * 8 / h.RefCountBit()
}

func ParseHeader(r FileHandler) (*Header, error) {
	// get the first 104 bytes
	hdr, err := readAt(r, 0, 104)
	if err != nil {
		return nil, err
	}

	if string(hdr[0:4]) != QCOW2MagicNumber {
		return nil, errors.New("invalid QCOW2 magic")
	}

	h := &Header{
		Version:               binary.BigEndian.Uint32(hdr[4:8]),
		BackingFileOffset:     binary.BigEndian.Uint64(hdr[8:16]),
		BackingFileSize:       binary.BigEndian.Uint32(hdr[16:20]),
		ClusterBits:           binary.BigEndian.Uint32(hdr[20:24]),
		Size:                  binary.BigEndian.Uint64(hdr[24:32]),
		CryptMethod:           binary.BigEndian.Uint32(hdr[32:36]) == 1,
		L1Size:                binary.BigEndian.Uint32(hdr[36:40]),
		L1TableOffset:         binary.BigEndian.Uint64(hdr[40:48]),
		RefCountTableOffset:   binary.BigEndian.Uint64(hdr[48:56]),
		RefcountTableClusters: binary.BigEndian.Uint32(hdr[56:60]),
		NumSnapshots:          binary.BigEndian.Uint32(hdr[60:64]),
		SnapshotOffset:        binary.BigEndian.Uint64(hdr[64:72]),
		RefCountOrder:         binary.BigEndian.Uint32(hdr[96:100]),
		Length:                binary.BigEndian.Uint32(hdr[100:104]),
	}

	if h.Version != 2 && h.Version != 3 {
		return nil, errors.New("invalid version")
	}
	// 1 << 9 == 512, which is the smallest cluster size
	if h.ClusterBits < 9 {
		return nil, errors.New("invalid cluster size")
	}

	return h, nil
}

func (i *Image) LoadHeader() error {
	var err error
	if i.Header, err = ParseHeader(i.Handler); err != nil {
		return err
	}
	return nil
}
