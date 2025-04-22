package gqcow2

import (
	"fmt"
	"io"
)

// FileHandler handles the read/write operation against
// the image resource. No matter its local/remote
// fs file, http served file, or other file system.
type FileHandler interface {
	io.ReaderAt
	// io.WriterAt
	// io.Closer

	// FastHandler
}

// FastHandler tries to hook up the read, write file
// descriptor
type FastHandler interface {
	// exposed linux file descriptor
	Fd() uintptr
}

type Image struct {
	// some resource may only support read
	RWMode   bool
	FastMode bool

	// mostly for print
	Name string

	Handler FileHandler

	// layout info
	Header        *Header
	RefCountTable []RefCountTableEntry
	L1Table       []L1Entry
}

func NewFileImage(f FileHandler, name string) (*Image, error) {
	var err error
	image := &Image{Name: name, Handler: f}

	if _, ok := f.(io.WriterAt); ok {
		image.RWMode = true
	}
	if _, ok := f.(FastHandler); ok {
		image.FastMode = true
	}

	if err = image.LoadHeader(); err != nil {
		return nil, err
	}

	if err = image.LoadRefcountTable(); err != nil {
		return nil, err
	}

	if err = image.LoadL1Table(); err != nil {
		return nil, err
	}

	return image, nil
}

func (i *Image) String() string {
	return fmt.Sprintf(`image:%s
    format:qcow2
    version:%d
    virtual size: %d(bytes)
    cluster size: %d
    `,
		i.Name,
		i.Header.Version,
		i.Header.Size,
		i.Header.ClusterSize(),
	)
}
