package jobs

import (
	"github.com/pkg/errors"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"os"
	"io"
)

func logToFile(file string, data []byte) error {
	if len(data) > 0 {
		file, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return errors.Wrapf(err, "failed opening file: %s", file)
		}
		defer file.Close()

		_, err = file.Write(data)

		if err != nil {
			return errors.Wrapf(err, "writing log %v failed", file)
		}
	}

	return nil
}

func createArchiveAndCleanup(path, log string) error {
	archive := path + ".gz"
	// create archive
	createArchiveCommand := fmt.Sprintf("tar -czf %v -C %v .", archive, path)
	commandOutput, err := sh.Command("/bin/sh", "-c", createArchiveCommand).CombinedOutput()
	logToFile(log, commandOutput)

	if(err != nil) {
		fmt.Println(err)
		fmt.Println(string(commandOutput))
	}

	if os.RemoveAll(path) != nil {
		// show warning
	}

	return err
}

func isDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// read in ONLY one file
	_, err = f.Readdir(1)

	// and if the file is EOF... well, the dir is empty.
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
