package bundle

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
)

var (
	//go:embed files/bin/hyperkit files/bin/machine-driver-hyperkit files/bin/qcow-tool files/machine/image.qcow2.bz2 files/machine/kernel files/machine/initrd.img files/machine/UEFI.fd files/machine/machine.yaml
	FS       embed.FS
	Binaries = map[string]string{
		"hyperkit":                "files/bin/hyperkit",
		"machine-driver-hyperkit": "files/bin/machine-driver-hyperkit",
		"qcow-tool":               "files/bin/qcow-tool",
	}
	Machine = map[string]string{
		"image.qcow2.bz2": "files/machine/image.qcow2.bz2",
		"kernel":          "files/machine/kernel",
		"initrd.img":      "files/machine/initrd.img",
		"machine.yaml":    "files/machine/machine.yaml",
		"UEFI.fd":         "files/machine/UEFI.fd",
	}
)

func UnbundleBin(path string) error {
	fs := FS
	dest := fmt.Sprintf("%s/%s", path, "bin")
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}
	for k, v := range Binaries {
		file, err := fs.ReadFile(v)
		if err != nil {
			return err
		}
		fmt.Printf("Extracting %s\n", k)
		if err := writeFile(file, fmt.Sprintf("%s/%s", dest, k), os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func UnbundleMachine(path string) error {
	version, err := NewVersionFromBundle()
	if err != nil {
		return err
	}
	dest := filepath.Join(path, "bundles", version.String())
	if err = os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}
	fs := FS
	for k, v := range Machine {
		file, err := fs.ReadFile(v)
		if err != nil {
			return err
		}
		fmt.Printf("Extracting %s\n", k)
		if err := writeFile(file, fmt.Sprintf("%s/%s\n", dest, k), 0640); err != nil {
			return err
		}
	}
	return nil
}

func writeFile(src []byte, dest string, mode os.FileMode) error {
	write := func(src []byte, dest string, mode os.FileMode) error {
		if err := os.WriteFile(dest, src, mode); err != nil {
			return err
		}
		return nil
	}
	if _, err := os.Stat(dest); err == nil {
		// file exists
		f, err := os.Open(dest)
		if err != nil {
			return err
		}
		defer f.Close()
		if !shaSumMatches(src, f) {
			if err := write(src, dest, mode); err != nil {
				return err
			}
		}
	} else if os.IsNotExist(err) {
		if err := write(src, dest, mode); err != nil {
			return err
		}
	} else {
		return err
	}
	if strings.HasSuffix(dest, ".bz2") {
		if _, err := os.Stat(strings.TrimSuffix(dest, ".bz2")); os.IsNotExist(err) {
			fmt.Printf("  Decompressing %s", dest)
			if err := archiver.DecompressFile(
				dest,
				strings.TrimSuffix(dest, ".bz2")); err != nil {
				return err
			}
		}
	}
	return nil
}

func shaSumMatches(f1 []byte, f2 io.Reader) bool {
	f1Hash := sha256.New()
	f1Hash.Write(f1)
	f2Hash := sha256.New()
	io.Copy(f2Hash, f2)
	return hex.EncodeToString(f1Hash.Sum(nil)) == hex.EncodeToString(f2Hash.Sum(nil))
}
