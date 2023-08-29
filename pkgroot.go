package ipkg

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Here install all packages
type Root struct {
	path string
	db   *sql.DB
}

// Default path for root
const DefaultPath = "/ira/ipkg"

// Creates package root on specified path. If directory path not exists, it will be created.
func CreateRoot(path string) (*Root, error) {
	// Checking input parameter
	err := checkRootPath(path, true)
	if err != nil {
		return nil, err
	}

	// Creating database
	db, err := os.Create(filepath.Join(path, "db.sqlite3"))
	if err != nil {
		return nil, fmt.Errorf("creating database file: %v", err)
	}
	db.Close()
	// Create package root
	return setupPackageRoot(path)
}

func OpenRoot(path string) (*Root, error) {
	// Checking input parameter
	err := checkRootPath(path, false)
	if err != nil {
		return nil, err
	}

	// Checking is path a package root
	if _, err := os.Stat(filepath.Join(path, "db.sqlite3")); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s is not a package root", err)
	}
	// Create package root
	return setupPackageRoot(path)
}

func checkRootPath(path string, create bool) error {
	pathinfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		if !create {
			return fmt.Errorf("package root %s doesn't exist", path)
		}
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if !pathinfo.IsDir() {
		return fmt.Errorf("incorrect path %s: expected dir, found file", path)
	}
	return nil
}

func setupPackageRoot(path string) (*Root, error) {
	pkgroot := &Root{path: path}
	// Opening database in a temporary variable
	db, err := sql.Open("sqlite3", filepath.Join(pkgroot.path, "db.sqlite3"))
	if err != nil {
		return nil, fmt.Errorf("opening database: %v", err)
	}
	// Creating table IF IT NOT EXISTS.
	// If exists, it won't be truncated
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS packages (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		dependencies TEXT NOT NULL
	);`)
	if err != nil {
		return nil, fmt.Errorf("setup database: %v", err)
	}
	// Setting database
	pkgroot.db = db
	return pkgroot, nil
}

// This function gets package by name and version. If there is no package, returns sql.ErrNoRows
func (r *Root) FindPackage(name string, version string) (*PkgConfig, error) {
	cfg := new(PkgConfig)
	cfg.Name = name
	cfg.Version = version
	var dependencies string
	err := r.db.QueryRow("SELECT dependencies FROM packages WRERE name = ? AND version = ?", name, version).Scan(&dependencies)
	if err != nil {
		return nil, err
	}
	cfg.Dependencies = UnserializeDependencies(dependencies)
	return cfg, nil
}

// This function return all packages with the same name
func (r *Root) FindPackagesByName(name string) ([]PkgConfig, error) {
	var result []PkgConfig
	rows, err := r.db.Query("SELECT version, dependencies FROM packages WHERE name = ?", name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cfg PkgConfig
		var version, dependencies string
		err = rows.Scan(&version, &dependencies)
		if err != nil {
			return nil, err
		}
		cfg.Name = name
		cfg.Version = version
		cfg.Dependencies = UnserializeDependencies(dependencies)
		result = append(result, cfg)
	}
	return result, rows.Err()
}
