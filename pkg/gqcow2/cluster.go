package gqcow2

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

type Cluster struct {
	RawCluster
	L2Entry
}

// the RawCluster is just the chunk of data
// cut from the image, aligned by cluster size
type RawCluster struct {
	// the offset from the image file
	Offset uint64
	Data   []byte // length is cluster size
}
