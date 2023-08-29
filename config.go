package ipkg

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
		_, err = root.FindPackage(name, version)
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
	if len(cfg.Dependencies) == 0 {
		return result // ""
	}
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
	if serialized == "" {
		return result // map[string]bool{}
	}
	ids := strings.Split(serialized, ";")
	for _, id := range ids {
		clearId := id[:len(id)-3] // Getting ID without flag
		result[clearId] = (id[len(id)-3:] == "(!)")
	}
	return result
}

// ParseConfig parse config file set in path and return PkgConfig
func ParseConfig(path string) (*PkgConfig, error) {
	var config *PkgConfig
	// Firstly, we're opening config file
	configFile, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return config, err
	}
	defer configFile.Close()
	// Secondary, we read this file (and close it)
	configJson, err := io.ReadAll(configFile)
	if err != nil {
		return config, fmt.Errorf("reading config file: %v", err)
	}
	configFile.Close()
	// And finally, we unmarshal JSON content and getting config
	err = json.Unmarshal(configJson, &config)
	if err != nil {
		return config, fmt.Errorf("parsing config as JSON: %v", err)
	}
	return config, nil
}
