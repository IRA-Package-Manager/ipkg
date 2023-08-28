package ipkg

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ira-package-manager/iscript"
)

// This function install package which should be set in path
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
	config, err := ParseConfig(filepath.Join(workPath, ".ira", "config.json"))
	if os.IsNotExist(err) {
		return fmt.Errorf("package has no config file")
	} else if err != nil {
		return err
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
	// Creating installation folder
	installDir := filepath.Join(r.path, config.Name+"."+config.Version)
	if err = os.Mkdir(installDir, os.ModePerm); !os.IsExist(err) && err != nil {
		return fmt.Errorf("creating installation folder: %v", err)
	}
	// Installing using IScript
	parser, err := iscript.NewParser(
		filepath.Join(workPath, ".ira", "iscript"),
		installDir)
	if err != nil {
		return err
	}
	err = parser.Start(iscript.Install, workPath)
	if err != nil {
		return fmt.Errorf("parsing iscript: %v", err)
	}
	// Copying IScript for future manipulations
	if err = os.Mkdir(filepath.Join(installDir, ".ira"), os.ModePerm); !os.IsExist(err) && err != nil {
		return fmt.Errorf("creating configuration folder: %v", err)
	}

	err = copy(filepath.Join(workPath, ".ira", "iscript"), filepath.Join(installDir, ".ita", "iscript"))
	if err != nil {
		return fmt.Errorf("saving IScript: %v", err)
	}
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
		err := unzipFile(f, filepath.Join(tempDir, destination))
		if err != nil {
			return "", err
		}
	}
	return destination, nil
}

// Prepares package before unzipping
func prepareCompressedPackage(path, tempDir string) (string, error) {
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
