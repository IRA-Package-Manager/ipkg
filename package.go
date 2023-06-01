package ipkg

import (
	"os/user"
)

type Package struct {
	Name         string
	Version      string
	Owner        user.User
	Description  string
	Dependencies struct {
		Required    []Package
		Recommended []Package
		Optional    []Package
	}
	Destination string
	installed   bool
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func (p *Package) Install(db Database) error {
	if p.installed {
		return nil
	}

	return nil

}
