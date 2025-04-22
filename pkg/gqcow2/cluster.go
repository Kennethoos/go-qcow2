package gqcow2

import "io"

// There are cluster that stores metadata such as header
// also there are data cluster
type ClusterType int

const (
	// DataCluster   ClusterType = "data"
	// HeaderCluster ClusterType = "header"
	// RefCountTable ClusterType = "refcount_table"
	// RefCountBlock ClusterType = "refcount_block"
	// L1Table       ClusterType = "l1_table"
	// L2Table       ClusterType = "l2_table"
	Standard ClusterType = iota
	Compressed
)

type GuestCluster struct {
	L2Info L2Entry

	// the start offset of the whole disk
	Start uint64
	// the length of this cluster
	// usually the cluster size defined in the
	// qcow2 image, but could be less in case
	// there is no enough data for one cluster
	Length uint64
}

type ClusterMap struct {
	CompressedCluster []GuestCluster
	Regions           []VirtualDiskRegion
}

func NewClusterMap() *ClusterMap {
	return &ClusterMap{
		CompressedCluster: make([]GuestCluster, 0),
		Regions:           make([]VirtualDiskRegion, 0),
	}
}

// Chunk represent a block of data that should be read from the image,
// then write to the virtual disk file, chunk doesn't do the read/write
// operation, only provide information for the read/write
type Chunk struct {
	// There are 3 types of chunks
	// 1. multiple standard descriptors, have data, can be read and write cluster one by one
	// 2. standard descriptor, no data, or 0 data, concat and write once
	// 3. compressed descriptor, read one chunk decompressed, write
	Type int
}

type DiskHandler interface {
	io.ReaderAt
	io.WriterAt
}

// VirtualDisk represent the whole disk file.
type VirtualDisk struct {
	//// location is used for access
	//Location string

	// operator
	Handler DiskHandler
	// the disk handler support FastHandler
	FastMode bool
}

func NewVirtualDisk(dh DiskHandler) (*VirtualDisk, error) {
	vd := &VirtualDisk{
		Handler: dh,
	}

	if _, ok := dh.(FastHandler); ok {
		vd.FastMode = true
	}

	return vd, nil
}
