package gqcow2

type StandardDescriptor struct {
	AllZero bool
	// bits 9 - 55
	// if DataOffset is 0, and L2Entry Flag is false
	// this cluster is unallocated
	// if DataOffset is 0, and L2Entry Flag is true,
	// external data file is used.
	DataOffset uint64
}

type CompressedDescriptor struct {
	// not aligned to cluster or sector boundary
	// Starting from this offset, there is ONE sector of
	// 512bytes as the start of the compressed data
	DataOffset uint64
	// each sector is 512bytes, all sectors together
	// stores the compressed data
	//
	// they are after the DataOffset's chunk
	// and may expand to the next contiguous host cluster
	//
	// compressed data does not necessarily occupy
	// all of the bytes in the final sector; rather,
	// decompression stops when it has produced a cluster of data.
	//
	// Another compressed cluster may map to the tail of the final
	// sector used by this compressed cluster.
	// (this is why the DataOffset is NOT aligned to the cluster size)
	AdditionalSectorCount int
}
