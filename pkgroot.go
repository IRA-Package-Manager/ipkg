package ipkg

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Root struct {
	path string
	db   *sql.DB
}

const DefaultPath = "/ira/ipkg"

func CreateRoot(path string) (*Root, error) {
	pathinfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if !pathinfo.IsDir() {
		return nil, fmt.Errorf("incorrect path %s: expected dir, found file", path)
	}

	db, err := os.Create(filepath.Join(path, "db.sqlite3"))
	if err != nil {
		return nil, fmt.Errorf("creating database file: %v", err)
	}
	db.Close()
	pkgroot := &Root{path: path}
	pkgroot.db, err = sql.Open("sqlite3", filepath.Join(pkgroot.path, "db.sqlite3"))
	if err != nil {
		return nil, fmt.Errorf("opening new database: %v", err)
	}
	_, err = pkgroot.db.Exec(`CREATE TABLE IF NOT EXISTS packages (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		dependencies TEXT NOT NULL
	);`)
	if err != nil {
		return nil, fmt.Errorf("setup database: %v", err)
	}
	return pkgroot, nil
}
