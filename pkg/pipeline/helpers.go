package pipeline

import (
	"io"
	"os"
)

func copyFile(sourceName, destName string) error {
	// Open source for reading
	in, err := os.Open(sourceName)
	if err != nil {
		return err
	}
	defer in.Close()

	// Open destination for writing
	out, err := os.Create(destName)
	if err != nil {
		return err
	}
	defer out.Close()

	// Allocate a 4k buffer
	buf := make([]byte, 4096)

	// Go!
	_, err = io.CopyBuffer(out, in, buf)
	return err
}
