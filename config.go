package ipkg

import (
	"database/sql"
	"fmt"
	"strings"
)

// PkgConfig presents file $SRCDIR/.ira/config.json as Go structure
type PkgConfig struct {
	Name           string
	Version        string
	Dependencies   map[string]bool
	SupportWindows bool
	SupportLinux   bool
	Build          bool // true when package needs to be built
}

// This function checks if all dependencies are statisfied or not.
// root is a package root used to package installation.
// Returns boolean means success of check or fail and error if there were some errors
func (cfg *PkgConfig) CheckDependencies(root *Root) (bool, error) {
	for id, isRequired := range cfg.Dependencies { // for each required dependency...
		if !isRequired {
			continue
		}
		// ...parsing ID for name and version...
		var name, version string
		_, err := fmt.Sscanf(id, "%s-$%s", &name, &version)
		if err != nil {
			return false, fmt.Errorf("parsing id %s: %v", id, err)
		}

		// ...and trying to get package from database
		err = root.db.QueryRow("SELECT name FROM packages WHERE name=? AND version=?", name, version).Scan(&name)
		if err == sql.ErrNoRows { // If got no rows
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("finding dependency %s in database: %v", id, err)
		}
	}
	return true, nil
}

// SerializeDependencies prepares package dependencies for saving in database
// by saving them in one string. Format: dependencyID1(flag1);dependencyID2(flag2);...;dependencyIDN(flagN)
// Flag specifies is package required (!) or not (?)
func (cfg *PkgConfig) SerializeDependencies() string {
	var result string
	for id, isRequired := range cfg.Dependencies {
		if isRequired { // setting flag
			result += id + "(!);"
		} else {
			result += id + "(?);"
		}
	}
	return result[:len(result)-1]
}

// UnserializeDependencies get serialized by *PkgConfig.SerializeDependencies() string
// and returns source map (keys are IDs, values are boolean means dependency is required or not)
func UnserializeDependencies(serialized string) map[string]bool {
	result := make(map[string]bool)
	ids := strings.Split(serialized, ";")
	for _, id := range ids {
		clearId := id[:len(id)-3] // Getting ID without flag
		result[clearId] = (id[len(id)-3:] == "(!)")
	}
	return result
}