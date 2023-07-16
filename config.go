package ipkg

import (
	"database/sql"
	"fmt"
	"strings"
)

type PkgConfig struct {
	Name           string
	Version        string
	Dependencies   map[string]bool
	SupportWindows bool
	SupportLinux   bool
	Build          bool
}

func (cfg *PkgConfig) CheckDependencies(root *Root) (bool, error) {
	for id, isRequired := range cfg.Dependencies {
		if !isRequired {
			continue
		}
		var name, version string
		_, err := fmt.Sscanf(id, "%s-$%s", &name, &version)
		if err != nil {
			return false, fmt.Errorf("parsing id %s: %v", id, err)
		}

		_, err = root.db.Query("SELECT * FROM packages WHERE name=? AND version=?", name, version)
		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("finding dependency %s in database: %v", id, err)
		}
	}
	return true, nil
}

func (cfg *PkgConfig) SerializeDependencies() string {
	var result string
	for id, isRequired := range cfg.Dependencies {
		if isRequired {
			result += id + "(!);"
		} else {
			result += id + "(?);"
		}
	}
	return result[:len(result)-1]
}

func UnserializeDependencies(serialized string) map[string]bool {
	result := make(map[string]bool)
	ids := strings.Split(serialized, ";")
	for _, id := range ids {
		clearId := id[:len(id)-3]
		result[clearId] = (id[len(id)-3:] == "(!)")
	}
	return result
}
