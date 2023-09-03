package ipkg

import (
	"archive/zip"
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ira-package-manager/iscript"
)

// This function install package which should be set in path. If package is installed by user, asDependency must be false
// If package must be installed for another program (as dependency), you should set it as true
func (r *Root) InstallPackage(path string, asDependency bool) error {
	pkginfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("package %q doesn't exist", path)
	} else if os.IsPermission(err) {
		return fmt.Errorf("working with %s: permission denied", path)
	} else if err != nil {
		return fmt.Errorf("os.Stat(%q): %w", path, err)
	}
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
	err = checkPackage(config, r)
	if err != nil {
		return err
	}
	// Running build script if build option enabled
	if config.Build {
		err = buildPackage(workPath)
		if err != nil {
			return err
		}
	}
	// Creating installation folder
	installDir := filepath.Join(r.path, config.Name+"-$"+config.Version)
	if err = os.Mkdir(installDir, os.ModePerm); !os.IsExist(err) && err != nil {
		return fmt.Errorf("creating installation folder: %w", err)
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
		return fmt.Errorf("parsing iscript: %w", err)
	}
	// Copying IScript for future manipulations
	if err = os.Mkdir(filepath.Join(installDir, ".ira"), os.ModePerm); !os.IsExist(err) && err != nil {
		return fmt.Errorf("creating configuration folder: %w", err)
	}

	err = copy(filepath.Join(workPath, ".ira", "iscript"), filepath.Join(installDir, ".ira", "iscript"))
	if err != nil {
		return fmt.Errorf("saving IScript: %w", err)
	}
	// We need only add package in database
	var byUser int
	if asDependency {
		byUser = 0
	} else {
		byUser = 1
	}
	_, err = r.db.Exec("INSERT INTO packages VALUES (NULL, ?, ?, ?, ?, 0)", config.Name, config.Version, config.SerializeDependencies(), byUser)
	if err != nil {
		return fmt.Errorf("adding package to database: %v", err)
	}
	return nil
}

func (r *Root) activate(name, version string) error {
	if _, err := r.FindPackage(name, version); err == sql.ErrNoRows {
		return fmt.Errorf("package %s-$%s is not installed", name, version)
	}
	path := filepath.Join(r.path, name+"-$"+version)
	if !exists(filepath.Join(path, ".ira", "deactivated")) {
		return nil // activated
	}
	log := filepath.Join(path, ".ira", "activate.log")
	if exists(log) {
		file, err := os.Open(log)
		if err != nil {
			return fmt.Errorf("opening activation log: %w", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			parsedLog := strings.Split(scanner.Text(), " ")
			// Note: ignoring errors
			os.Symlink(parsedLog[0], parsedLog[1])
		}
		if scanner.Err() != nil {
			return fmt.Errorf("scanning activation log: %w", err)
		}
		file.Close()
		flag, err := os.Create(filepath.Join(path, ".ira", "deactivated"))
		if err != nil {
			return fmt.Errorf("creating flag file: %w", err)
		}
		flag.Close()
	}
	return nil
}

func (r *Root) deactivate(name, version string) error {
	if _, err := r.FindPackage(name, version); err == sql.ErrNoRows {
		return fmt.Errorf("package %s-$%s is not installed", name, version)
	}
	path := filepath.Join(r.path, name+"-$"+version)
	if exists(filepath.Join(path, ".ira", "deactivated")) {
		return nil // deactivated
	}
	log := filepath.Join(path, ".ira", "activate.log")
	if exists(log) {
		file, err := os.Open(log)
		if err != nil {
			return fmt.Errorf("opening activation log: %w", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			// Note: igroring errors
			os.Remove(strings.Split(scanner.Text(), " ")[1])
		}
		if scanner.Err() != nil {
			return fmt.Errorf("scanning activation log: %w", err)
		}
		file.Close()
		flag, err := os.Create(filepath.Join(path, ".ira", "deactivated"))
		if err != nil {
			return fmt.Errorf("creating flag file: %w", err)
		}
		flag.Close()
	}
	return nil
}

func (r *Root) ActivatePackage(name, version string) error {
	err := r.activate(name, version)
	if err != nil {
		return err
	}
	pkgs, err := r.FindPackagesByName(name)
	if err != nil {
		return fmt.Errorf("getting all packages: %w", err)
	}
	for _, pkg := range pkgs {
		if pkg.Version == version {
		} else {
			err = r.deactivate(pkg.Name, pkg.Version)
			if err != nil {
				return fmt.Errorf("deactivating %s-$%s: %w", name, version, err)
			}
		}
	}
	return nil
}

func (r *Root) RemovePackage(name, version string, removeDependencies bool) error {
	pkg, err := r.FindPackage(name, version)
	if err == sql.ErrNoRows {
		return fmt.Errorf("package %s-$%s is not installed", name, version)
	}
	if removeDependencies {
		err = pkg.ForEachDependency(r.removeDependency)
		if err != nil {
			return err
		}
	}
	err = r.deactivate(name, version)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("DELETE FROM packages WHERE name = ? AND version = ?", name, version)
	if err != nil {
		return fmt.Errorf("removing package from database: %v", err)
	}
	path := filepath.Join(r.path, name+"-$"+version)
	parser, err := iscript.NewParser(filepath.Join(path, ".ira", "iscript"), path)
	if err != nil {
		return err
	}
	err = parser.Start(iscript.Remove, "")
	if err != nil {
		return fmt.Errorf("parsing iscript: %w", err)
	}
	err = os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing package files: %v", err)
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
		err := unzipFile(f, destination)
		if err != nil {
			return "", err
		}
	}
	return destination, nil
}

func (r *Root) removeDependency(name, version string, isRequired bool) error {
	isDependency, err := r.IsDependency(name, version)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return fmt.Errorf("checking is %s-$%s a dependency : %v", name, version, err)
	}
	if !isDependency {
		return nil
	}
	canBeRemoved, err := r.CanBeRemoved(name, version)
	if err != nil {
		return fmt.Errorf("checking can %s-$%s be removed: %v", name, version, err)
	}
	if !canBeRemoved {
		return nil
	}
	err = r.RemovePackage(name, version, true)
	if err != nil {
		return fmt.Errorf("removing dependency %s-$%s: %v", name, version, err)
	}
	return nil
}

// Prepares package before unzipping
func prepareCompressedPackage(path, tempDir string) (string, error) {
	// Creating temporary dir if not exists
	err := createIfNotExists(tempDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("making temporary dir: %v", err)
	}

	archivePath := filepath.Join(tempDir, strings.TrimSuffix(filepath.Base(path), ".ipkg")+".zip") // path to new zip archive

	// Copying IPKG to temporary folder as ZIP archive
	if err = copy(path, archivePath); err != nil {
		return "", fmt.Errorf("copying package %s to temporary place %s: %v", path, archivePath, err)
	}
	return archivePath, nil
}

func buildPackage(pkgPath string) error {
	buildscriptPath := filepath.Join(pkgPath, ".ira", "build")
	if runtime.GOOS == "windows" {
		buildscriptPath += ".bat"
	}
	buildscript := exec.Command(buildscriptPath)
	var out bytes.Buffer
	buildscript.Stdout = &out
	err := buildscript.Run()
	if err != nil {
		return fmt.Errorf("executing build script: %w", err)
	}
	log.Println("script output: ", out.String())
	return nil
}

func checkPackage(config *PkgConfig, r *Root) error {
	// Checking operating system
	switch runtime.GOOS {
	case "linux":
		if !config.SupportLinux {
			return fmt.Errorf("unsupported os: %v", runtime.GOOS)
		}
	case "windows":
		if !config.SupportWindows {
			return fmt.Errorf("unsupported os: %v", runtime.GOOS)
		}
	default:
		return fmt.Errorf("unsupported os: %v", runtime.GOOS)
	}

	// Checking is package installed
	_, err := r.FindPackage(config.Name, config.Version)
	if err == nil {
		return fmt.Errorf("package %s-$%s is already installed", config.Name, config.Version)
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("checking is package installed: %w", err)
	}
	// Checking dependencies
	ok, err := config.CheckDependencies(r)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("not all required dependencies statisfied")
	}
	return nil
}
