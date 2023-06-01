package ipkg

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Database struct {
	destination string
	isRoot      bool
}

func (db Database) IsNull() bool {
	return db.destination == ""
}

func (db Database) GetPackage(name string, version string) (Package, error) {
	var pkg Package
	pkg.Name = name
	pkg.Version = version
	pkg.installed = true
	pkgID := fmt.Sprintf("%s-$%s", name, version)
	if _, err := os.Stat(filepath.Join(db.destination, pkgID)); os.IsNotExist(err) {
		return pkg, fmt.Errorf("Finding package %s-$%s: not found", name, version)
	}
	pkg.Destination = filepath.Join(db.destination, pkgID)
	if descFile, err := os.Open(filepath.Join(pkg.Destination, "description")); err == nil {
		defer descFile.Close()
		desc, err := ioutil.ReadAll(descFile)
		if err != nil {
			return pkg, fmt.Errorf("Reading description: %v", err)
		}

		pkg.Description = string(desc)
	} else if !os.IsNotExist(err) {
		return pkg, fmt.Errorf("Reading description: %v", err)
	}

	// If no dependencies specified, package seems to be ready.
	if _, err := os.Stat(filepath.Join(pkg.Destination, "dependencies")); os.IsNotExist(err) {
		return pkg, nil
	}

	if dependenciesFile, err := os.Open(filepath.Join(pkg.Destination, "dependencies", "required.list")); err == nil {
		//TODO: make it work parallel
		defer dependenciesFile.Close()
		deps, err := db.getDependencies(dependenciesFile)
		if err != nil {
			return pkg, fmt.Errorf("Getting regular dependencies: %v", err)
		}
		pkg.Dependencies.Required = deps
	}
	if dependenciesFile, err := os.Open(filepath.Join(pkg.Destination, "dependencies", "recommended.list")); err == nil {
		//TODO: make it work parallel
		defer dependenciesFile.Close()
		deps, err := db.getDependencies(dependenciesFile)
		if err != nil {
			return pkg, err
		}
		pkg.Dependencies.Recommended = deps
	}
	if dependenciesFile, err := os.Open(filepath.Join(pkg.Destination, "dependencies", "optional.list")); err == nil {
		//TODO: make it work parallel
		defer dependenciesFile.Close()
		deps, err := db.getDependencies(dependenciesFile)
		if err != nil {
			return pkg, err
		}
		pkg.Dependencies.Optional = deps
	}
	return pkg, nil
}

func (db Database) getDependencies(dependenciesFile *os.File) ([]Package, error) {
	input := bufio.NewScanner(dependenciesFile)
	var deps []Package
	for input.Scan() {
		parsedID := strings.Split(input.Text(), "\t")
		dep, err := db.GetPackage(parsedID[0], parsedID[1])
		if err != nil {
			return nil, fmt.Errorf("Getting dependency %s-$%s: %v", parsedID[0], parsedID[1], err)
		}
		deps = append(deps, dep)
	}
	return deps, nil
}

func (db Database) getAll(list string, fullPackages bool) ([]Package, error) {
	packages, err := db.read(list)
	if err != nil {
		return nil, fmt.Errorf("Reading database: %v", err)
	}
	if fullPackages {
		for i, pkg := range packages {
			packages[i], err = db.GetPackage(pkg.Name, pkg.Version)
			if err != nil {
				return packages, fmt.Errorf("Getting package %s-$%s: %v", pkg.Name, pkg.Version, err)
			}
		}
	}
	return packages, nil
}

func (db Database) ListUser(fullPackages bool) ([]Package, error) {
	return db.getAll("by-user.list", fullPackages)
}
func (db Database) ListDependencies(fullPackages bool) ([]Package, error) {
	return db.getAll("dependencies.list", fullPackages)
}

func (db Database) read(list string) ([]Package, error) {
	pkglist, err := os.Open(filepath.Join(db.destination, list))
	if os.IsNotExist(err) || os.IsPermission(err) {
		return nil, errors.New("This database is incorrect. Remove it and create again using ipkg.MakeDatabase()")
	}
	scanner := bufio.NewScanner(pkglist)
	var packages []Package
	for scanner.Scan() {
		pair := strings.Split(scanner.Text(), "\t")
		packages = append(packages, Package{Name: pair[0], Version: pair[1]})
	}
	return packages, nil
}

var nullDatabase Database

func MakeDatabase(root bool, presetDestination string) (Database, error) {
	var destination string
	if root {
		if presetDestination == "" {
			destination = "/.ira"
		} else {
			destination = presetDestination
		}
	} else {
		if _, err := os.Stat("/.ira"); os.IsNotExist(err) {
			return nullDatabase, errors.New("Root database not exists. Create it first")
		} else if os.IsPermission(err) {
			return nullDatabase, errors.New("Database for root is incorrect. Remove it and create by ipkg.MakeDatabase()")
		} else if err != nil {
			return nullDatabase, err
		}
		if os.Getenv("HOME") == "" {
			return nullDatabase, errors.New("Cannot determinate where create database: no $HOME variable provided")
		}
		if presetDestination == "" {
			destination = filepath.Join(os.Getenv("HOME"), ".ira")
		}
	}
	if err := os.MkdirAll(destination, 0744); os.IsExist(err) {
		return nullDatabase, errors.New("Database exists")
	} else if err != nil {
		return nullDatabase, err
	}
	pkglist, err := os.Create(filepath.Join(destination, "by-user.list"))
	if err != nil {
		return nullDatabase, err
	}

	err = pkglist.Close()
	if err != nil {
		return nullDatabase, err
	}

	pkglist, err = os.Create(filepath.Join(destination, "dependencies.list"))
	if err != nil {
		return nullDatabase, err
	}

	err = pkglist.Close()

	if err != nil {
		return nullDatabase, err
	}

	return Database{destination: destination, isRoot: root}, nil
}
