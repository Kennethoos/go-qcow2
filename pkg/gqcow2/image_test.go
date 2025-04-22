package gqcow2_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"go-qcow2/pkg/gqcow2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func Test_NewFileImage(t *testing.T) {
	t.Run("Load image from local file",
		func(t *testing.T) {
			path := "../../test/cirros.img"

			f, err := os.Open(path)
			require.NoError(t, err)

			_, err = gqcow2.NewFileImage(f, "cirros")
			assert.NoError(t, err)
		})
}

func Test_ImageDump(t *testing.T) {
	t.Run("Dump image's map",
		func(t *testing.T) {
			path := "../../test/cirros.img"

			f, err := os.Open(path)
			require.NoError(t, err)

			image, err := gqcow2.NewFileImage(f, "cirros")
			require.NoError(t, err)

			// got
			cirrosMapDump := image.Dump()

			//
			//qemuCmdParams := fmt.Sprintf("qemu-img map -f qcow2 --output=json %s", path)
			params := []string{"map", "-f", "qcow2", "--output=json", path}
			qemuCmd := exec.Command("qemu-img", params...)
			qemuCmd.Dir = getCallerDir()
			rawDump, err := qemuCmd.CombinedOutput()
			require.NoError(t, err)
			require.Truef(t, json.Valid(rawDump), "fail to dump using qemu-img, command: [%s %s]", "qemu-img", params)

			wantedDump := make([]gqcow2.VirtualDiskRegion, 0)
			json.Unmarshal(rawDump, &wantedDump)

			assert.Equal(t, wantedDump, cirrosMapDump)
		})
}

func Test_ImageConvertToRaw(t *testing.T) {
	t.Run("Convert qcow2 to raw",
		func(t *testing.T) {
			path := "../../test/cirros.img"
			outputPath := "../../test/myraw.raw"

			f, err := os.Open(path)
			require.NoError(t, err)

			image, err := gqcow2.NewFileImage(f, "cirros")
			//_, err = gqcow2.NewFileImage(f, "cirros")
			require.NoError(t, err)

			rawFile, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
			//_, err = os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
			require.NoError(t, err)

			vd, err := gqcow2.NewVirtualDisk(rawFile)
			require.NoError(t, err)

			err = gqcow2.Convert(image, vd)
			assert.NoError(t, err)
		})
}

func Test_FindL2Entry(t *testing.T) {
	t.Run("Find l2 entry",
		func(t *testing.T) {
			path := "../../test/cirros.img"

			f, err := os.Open(path)
			require.NoError(t, err)

			image, err := gqcow2.NewFileImage(f, "cirros")
			//_, err = gqcow2.NewFileImage(f, "cirros")
			require.NoError(t, err)

			l2entry, err := image.FindL2Entry(0)
			require.NoError(t, err)

			fmt.Printf("%#v\n", l2entry)
			fmt.Printf("%#v\n", l2entry.Compressed)
		})
}

func getCallerDir() string {
	_, filename, _, ok := runtime.Caller(0) // 0 = current function
	if !ok {
		panic("Could not get caller info")
	}
	return filepath.Dir(filename)
}
