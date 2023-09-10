package ipkg

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func unzipPackage(path string) (string, error) {
	// Getting paths used for unzipping
	tempDir := filepath.Join(os.TempDir(), "ira", "ipkg", "install")
	archivePath, err := prepareCompressedPackage(path, tempDir)
	if err != nil {
		return "", err
	}

	// Opening archive
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("opening %s as archive: %v", path, err)
	}
	defer archive.Close()
	archivePath, err = filepath.Abs(archivePath) // needed in security purposes
	if err != nil {
		return "", err
	}
	destination := strings.TrimSuffix(archivePath, ".zip")
	// Unzipping archive
	for _, f := range archive.File {
		err := unzipFile(f, destination)
		if err != nil {
			return "", err
		}
	}
	return destination, nil
}

func unzipFile(f *zip.File, destination string) error {
	filePath := filepath.Join(destination, f.Name)
	// For security purposes
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// Creating FS tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	// Unzipping file
	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil

}
