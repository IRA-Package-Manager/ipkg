package ipkg

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func (r *Root) InstallPackage(path string) error {
	// Checking input argument
	pkginfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("package %q doesn't exist", path)
	} else if os.IsPermission(err) {
		return fmt.Errorf("working with %s: permission denied", path)
	} else if err != nil {
		return fmt.Errorf("os.Stat(%q): %v", path, err)
	}

	// Setting working path (if package is IPKG, we need to unzip it and use temporary folder)
	var workPath string
	if pkginfo.IsDir() {
		workPath = path // if package is a directory (unpacked), we can work there
	} else if filepath.Ext(path) == ".ipkg" {
		workPath, err = unzipPackage(path) // if package is IPKG, we need unpack it before working.
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("file %s is not IRA package", path)
	}

	// Parsing configuration file
	var config *PkgConfig
	// Firstly, we're opening config file
	configFile, err := os.OpenFile(filepath.Join(workPath, ".ira", "config.json"), os.O_RDONLY, os.ModePerm)
	if os.IsNotExist(err) {
		return fmt.Errorf("package has no config file")
	} else if err != nil {
		return fmt.Errorf("opening config file: %v", err)
	}
	defer configFile.Close()
	// Secondary, we read this file (and close it)
	configJson, err := io.ReadAll(configFile)
	if err != nil {
		return fmt.Errorf("reading config file: %v", err)
	}
	configFile.Close()
	// And finally, we unmarshal JSON content and getting config
	err = json.Unmarshal(configJson, config)
	if err != nil {
		return fmt.Errorf("parsing config as JSON: %v", err)
	}

	// Checking operating system and setting path to build script
	var buildscript *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		buildscript = exec.Command(filepath.Join(workPath, ".ira", "build"))
		if !config.SupportLinux {
			return fmt.Errorf("unsupported os: %v", runtime.GOOS)
		}
	case "windows":
		buildscript = exec.Command(filepath.Join(workPath, ".ira", "build.bat"))
		if !config.SupportWindows {
			return fmt.Errorf("unsupported os: %v", runtime.GOOS)
		}
	default:
		return fmt.Errorf("unsupported os: %v", runtime.GOOS)
	}
	// Checking dependencies
	ok, err := config.CheckDependencies(r)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("not all required dependencies statisfied")
	}
	// Running build script if build option enabled
	if config.Build {
		var out bytes.Buffer
		buildscript.Stdout = &out
		err = buildscript.Run()
		if err != nil {
			return fmt.Errorf("executing build script: %v", err)
		}
		log.Println("script output: ", out.String())
	}
	// Installing using IScript
	// TODO: parse install script

	// After all, adding package to database
	_, err = r.db.Exec("INSERT INTO packages VALUES (NULL, ?, ?, ?)", config.Name, config.Version, config.SerializeDependencies())
	if err != nil {
		return fmt.Errorf("adding package to database: %v", err)
	}
	return nil
}

func unzipPackage(path string) (string, error) {
	// Getting paths used for unzipping
	tempDir := filepath.Join(os.TempDir(), "ira", "ipkg", "install")
	archivePath, err := preparePackage(path, tempDir)
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
		err := unzipFile(f, filepath.Join(tempDir, destination))
		if err != nil {
			return "", err
		}
	}
	return destination, nil
}

// Prepares package before unzipping
func preparePackage(path, tempDir string) (string, error) {
	// Creating temporary dir if not exists
	err := os.MkdirAll(tempDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("making temporary dir: %v", err)
	}

	archivePath := filepath.Join(tempDir, strings.TrimSuffix(filepath.Base(path), ".ipkg")+".zip") // path to new zip archive
	src, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("creating temporary file %s: %v", archivePath, err)
	}
	defer src.Close()

	dest, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening package %s: %v", path, err)
	}
	defer dest.Close()

	// Copying IPKG to temporary folder as ZIP archive
	if _, err = io.Copy(dest, src); err != nil {
		return "", fmt.Errorf("copying package %s to temporary place %s: %v", path, archivePath, err)
	}
	return archivePath, nil
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
