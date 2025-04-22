package gqcow2

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"log"
	"syscall"
)

func Convert(image *Image, virtualDisk *VirtualDisk) error {
	cm, err := image.DumpToClusterMap()
	if err != nil {
		return err
	}
	for _, region := range cm.Regions {
		// handle compressed clusters
		if region.Compressed {
			continue
		}

		if !region.Present || !region.Data || region.Zero {
			allZeroBuf := make([]byte, region.Length)
			if _, err := virtualDisk.Handler.WriteAt(allZeroBuf, int64(region.Start)); err != nil {
				return err
			}
			continue
		}

		if region.Offset != 0 {
			if err := convertStandardRegion(image, virtualDisk, region); err != nil {
				return err
			}
			continue
		}

		log.Fatalf("shouldn't happen, region is %#v", region)
	}

	for _, compressed := range cm.CompressedCluster {
		if err := decompress(image, virtualDisk, compressed); err != nil {
			return err
		}
	}

	return nil
}

func decompress(image *Image, virtualDisk *VirtualDisk, compressed GuestCluster) error {
	ErrDecompressFail := errors.New("decompress failed")
	comVDStart := compressed.Start
	comVDLength := compressed.Length

	totalSectors := compressed.L2Info.Compressed.AdditionalSectorCount + 1

	compressedBuf := make([]byte, totalSectors*512)
	// compressedBuf may not be filled fully
	rc, err := image.Handler.ReadAt(compressedBuf, int64(compressed.L2Info.Compressed.DataOffset))
	if err != nil {
		if err != io.EOF {
			return errors.Join(ErrDecompressFail, err)
		} else {
			fmt.Printf("ts: %d\nstart:%d\nlength:%d\nimage_start:%d\nread_count: %d\n",
				totalSectors, comVDStart, comVDLength, compressed.L2Info.Compressed.DataOffset,
				rc)
			compressedBuf = compressedBuf[0:rc]
		}
	}

	// fmt.Printf("ts: %d\nstart:%d\nlength:%d\nimage_start:%d\n", totalSectors, comVDStart, comVDLength, compressed.L2Info.Compressed.DataOffset)
	// fmt.Printf("header: %#v\n", compressedBuf[:2])
	// read and decompress
	// decompressor, err := zlib.NewReader(bytes.NewReader(compressedBuf))
	decompressor := flate.NewReader(bytes.NewReader(compressedBuf))
	defer decompressor.Close()

	// we only need a cluster size of data, or the remaining length
	// whichever the smallest.
	// decompressedBuf := make([]byte, comVDLength)
	decompressedBuf := make([]byte, image.Header.ClusterSize())
	n, err := decompressor.Read(decompressedBuf)
	if err != nil {
		return errors.Join(ErrDecompressFail, err)
	}
	//if n != int(comVDLength) {
	//	return errors.Join(ErrDecompressFail, fmt.Errorf("decompressed less data, %d", n))
	//}
	// Do something with the decompressed data
	if _, err := virtualDisk.Handler.WriteAt(decompressedBuf[:n], int64(comVDStart)); err != nil {
		return errors.Join(ErrDecompressFail, err)
	}

	return nil
}

func convertStandardRegion(image *Image, virtualDisk *VirtualDisk, region VirtualDiskRegion) error {
	imageStart := int64(region.Offset)
	copyLength := int64(region.Length)
	vdStart := int64(region.Start)

	// if both side support send file
	if image.FastMode && virtualDisk.FastMode {
		// max by sendfile
		maxCopySize := int64(0x7ffff000)
		// Get file descriptors
		sh := image.Handler.(FastHandler)
		dh := virtualDisk.Handler.(FastHandler)
		srcFd := int(sh.Fd())
		dstFd := int(dh.Fd())

		var offset int64 = 0
		var remains int64 = copyLength

		for offset < int64(copyLength) {

			desOff := offset + vdStart
			inOff := offset + imageStart
			var copySize int64
			if remains > maxCopySize {
				copySize = maxCopySize
			} else {
				copySize = remains
			}

			// seek the offset for dst first
			_, err := syscall.Seek(dstFd, desOff, io.SeekStart)
			if err != nil {
				return errors.Join(err, fmt.Errorf("seek err"))
			}

			// sendfile max chunk is usually 2GB, be safe
			n, err := syscall.Sendfile(dstFd, srcFd, &inOff, int(copySize))
			if err != nil {
				return errors.Join(err, fmt.Errorf("sendfile err"))
			}
			offset += int64(n)
		}

		return nil
	}

	// use userspace buf
	panic("unexpected")
}
