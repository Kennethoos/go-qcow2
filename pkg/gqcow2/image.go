package gqcow2

import (
	"fmt"
	"io"
	"os"
)

type File interface{}

type Image struct {
	Name          string
	Input         io.ReaderAt
	Ouput         io.WriterAt
	Header        *Header
	RefCountTable []RefCountTableEntry
	L1Table       []L1Entry
}

func NewFileImage(path string) (*Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	name := f.Name()
	hdr, err := ParseHeader(f)
	if err != nil {
		return nil, err
	}

	return &Image{
		Name:   name,
		Input:  f,
		Header: hdr,
	}, nil
}

func (i *Image) String() string {
	return fmt.Sprintf("image:%s\nformat:qcow2\nversion:%d\nvirtual size: %d(bytes)\ncluster size: %d\n",
		i.Name,
		i.Header.Version,
		i.Header.Size,
		i.Header.ClusterSize(),
	)
}
