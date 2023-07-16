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
	pkginfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("package %q doesn't exist", path)
	} else if os.IsPermission(err) {
		return fmt.Errorf("working with %s: permission denied", path)
	} else if err != nil {
		return fmt.Errorf("os.Stat(%q): %v", path, err)
	}
	var workPath string
	if pkginfo.IsDir() {
		workPath = path
	} else if filepath.Ext(path) == ".ipkg" {
		workPath, err = unzipPackage(path)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("file %s is not IRA package", path)
	}
	var config *PkgConfig
	configFile, err := os.OpenFile(filepath.Join(workPath, ".ira", "config.json"), os.O_RDONLY, os.ModePerm)
	if os.IsNotExist(err) {
		return fmt.Errorf("package has no config file")
	} else if err != nil {
		return fmt.Errorf("opening config file: %v", err)
	}

	configJson, err := io.ReadAll(configFile)
	if err != nil {
		return fmt.Errorf("reading config file: %v", err)
	}
	err = json.Unmarshal(configJson, config)
	if err != nil {
		return fmt.Errorf("parsing config as JSON: %v", err)
	}
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
	ok, err := config.CheckDependencies(r)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("not all required dependencies statisfied")
	}
	if config.Build {
		var out bytes.Buffer
		buildscript.Stdout = &out
		err = buildscript.Run()
		if err != nil {
			return fmt.Errorf("executing build script: %v", err)
		}
		log.Println("script output: ", out.String())
	}
	// TODO: parse install script

	_, err = r.db.Exec("INSERT INTO packages VALUES (NULL, ?, ?, ?)", config.Name, config.Version, config.SerializeDependencies())
	if err != nil {
		return fmt.Errorf("adding package to database: %v", err)
	}
	return nil
}

func unzipPackage(path string) (string, error) {
	tempDir := filepath.Join(os.TempDir(), "ira", "ipkg", "install")
	archivePath, err := preparePackage(path, tempDir)
	if err != nil {
		return "", err
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("opening %s as archive: %v", path, err)
	}
	defer archive.Close()
	archivePath, err = filepath.Abs(archivePath)
	if err != nil {
		return "", err
	}
	destination := strings.TrimSuffix(archivePath, ".zip")

	for _, f := range archive.File {
		err := unzipFile(f, filepath.Join(tempDir, destination))
		if err != nil {
			return "", err
		}
	}
	return destination, nil
}
func preparePackage(path, tempDir string) (string, error) {

	err := os.MkdirAll(tempDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("making temporary dir: %v", err)
	}
	archivePath := filepath.Join(tempDir, strings.TrimSuffix(filepath.Base(path), ".ipkg")+".zip")
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
	_, err = io.Copy(dest, src)
	if err != nil {
		return "", fmt.Errorf("copying package %s to temporary place %s: %v", path, archivePath, err)
	}
	return archivePath, nil
}

func unzipFile(f *zip.File, destination string) error {
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

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

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil

}
