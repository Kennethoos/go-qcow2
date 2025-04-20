package main

import (
	"encoding/json"
	"fmt"
	"log"

	"go-qcow2/pkg/gqcow2"
)

func main() {
	// http.HandleFunc("/upload", uploadHandler)
	// log.Println("Listening on :8080")
	// log.Fatal(http.ListenAndServe(":8080", nil))

	img, err := gqcow2.NewFileImage("./cirros.img")
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Printf("%s\n", img)
	// fmt.Println("=== headers ===")
	// fmt.Printf("%#v\n", img.Header)

	err = img.LoadRefcountTable()
	if err != nil {
		log.Fatal(err)
	}
	err = img.LoadL1Table()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println("=== refcount table ===")
	// fmt.Printf("%#v\n", img.RefCountTable)

	//refcount, err := img.ReadRefCount(33423360)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Printf("refcount is %d\n", refcount)

	// entry, _ := img.FindL2Entry(41091072)
	// fmt.Printf("%#v\n", entry)
	// fmt.Printf("%#v\n", entry.Standard)

	result := MapVirtualDisk(img)
	// fmt.Printf("all regsions: %d", len(result))
	output, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", output)
	//_ = output
}

type VirtualDiskRegion struct {
	Start      uint64 `json:"start"`
	Length     uint64 `json:"length"`
	Depth      int    `json:"depth"`
	Present    bool   `json:"present"`
	Zero       bool   `json:"zero"`
	Data       bool   `json:"data"`
	Compressed bool   `json:"compressed"`
	Offset     uint64 `json:"offset,omitempty"`
}

func MapVirtualDisk(image *gqcow2.Image) []VirtualDiskRegion {
	virtualSize := image.Header.Size
	clusterSize := image.Header.ClusterSize()
	// clusterCount := virtualSize / uint64(clusterSize)
	// trailingPartLength := virtualSize % uint64(clusterSize)
	// fmt.Printf("virtual size: %d\n", virtualSize)
	// fmt.Printf("cluster size: %d\n", clusterSize)
	// fmt.Printf("clusterCount: %d\n", clusterCount)
	// fmt.Printf("trailing: %d\n", trailingPartLength)
	// fmt.Printf("======\n")

	depth := 0
	if image.Header.BackingFileOffset == 0 {
		depth = 0
	}

	// for loop all clusters
	regions := make([]VirtualDiskRegion, 0)
	activeRegion := VirtualDiskRegion{}
	offset := uint64(0)
	for offset < virtualSize {
		entry, err := image.FindL2Entry(offset)
		if err != nil {
			log.Fatalf("reading l2 entry failed, offset %d", offset)
		}
		//if virtualSize-offset < uint64(clusterSize) {
		//	fmt.Printf("\n%d\n", offset)
		//}

		// every cluster generate a new region,
		// if same, then merge the length
		// if not, create a new one
		newRegion := VirtualDiskRegion{}

		nrStart := offset
		nrDepth := depth
		nrLength := 0
		if virtualSize-offset >= uint64(clusterSize) {
			nrLength = clusterSize
		} else {
			nrLength = int(virtualSize - offset)
		}
		var nrPresent, nrCompressed, nrZero, nrData bool
		nrOffset := uint64(0)

		// present means either is preallocated, or used
		// zero, if present, could be true (not yet written)
		// Data, if present, could be false (not yet written)

		// unused, compressed or require COW
		if entry.Compressed != nil {
			nrPresent = true
			nrCompressed = true
			nrData = true // if compressed, then must have data
			nrZero = false
		} else if entry.Standard != nil {
			nrCompressed = false
			// preallocated
			if entry.Standard.AllZero {
				nrPresent = true
				nrZero = true
				nrData = false
			} else if entry.Standard.DataOffset == 0 && !entry.Flag {
				// unallocated
				nrPresent = false
				nrZero = true
				nrData = false
			} else if entry.Standard.DataOffset != 0 {
				nrPresent = true
				nrZero = false
				nrData = true
				nrOffset = entry.Standard.DataOffset
			}

		} else {
			log.Fatalf("corrupted l2 entry, offset %d", offset)
		}

		newRegion.Start = nrStart
		newRegion.Length = uint64(nrLength)
		newRegion.Depth = nrDepth
		newRegion.Present = nrPresent
		newRegion.Zero = nrZero
		newRegion.Compressed = nrCompressed
		newRegion.Data = nrData
		newRegion.Offset = nrOffset

		if offset == 0 {
			activeRegion = newRegion
			offset = offset + uint64(clusterSize)
			continue
		}

		if newRegion.SameAs(activeRegion) {
			// update active region
			activeRegion.Length = activeRegion.Length + newRegion.Length
		} else {
			// or current cluster belongs to a new region
			if offset != 0 {
				regions = append(regions, activeRegion)
				lastStart := activeRegion.Start
				lastLength := activeRegion.Length
				// a new region, start = last one's start + length
				newRegion.Start = lastStart + lastLength
			}
			activeRegion = newRegion
		}

		offset = offset + uint64(clusterSize)
	}

	// get the last one
	regions = append(regions, activeRegion)

	return regions
}

func (vdr VirtualDiskRegion) SameAs(another VirtualDiskRegion) bool {
	return vdr.Present == another.Present &&
		vdr.Zero == another.Zero &&
		vdr.Data == another.Data &&
		vdr.Compressed == another.Compressed
}
