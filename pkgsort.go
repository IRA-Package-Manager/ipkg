package ipkg

import (
	"sort"

	"golang.org/x/mod/semver"
)

// PkgSortMethod is analog of Less() in sort.Interface:
// gets two PkgConfig and returns should second be earlier than firsy
type PkgSortMethod func(first, second *PkgConfig) bool

type pkgSorter struct {
	pkgs []PkgConfig
	by   func(first, second *PkgConfig) bool
}

func (s *pkgSorter) Len() int      { return len(s.pkgs) }
func (s *pkgSorter) Swap(i, j int) { s.pkgs[i], s.pkgs[j] = s.pkgs[j], s.pkgs[i] }

func (s *pkgSorter) Less(i, j int) bool {
	return s.by(&s.pkgs[i], &s.pkgs[j])
}

// Sort sorts pkgs by PkgSortMethod function (uses sort.Interface internally)
// If reverse was specified, that would be reversed sort
func (method PkgSortMethod) Sort(pkgs []PkgConfig, reverse bool) {
	var sorter sort.Interface = &pkgSorter{
		pkgs: pkgs,
		by:   method,
	}
	if reverse {
		sorter = sort.Reverse(sorter)
	}
	sort.Sort(sorter)
}

// Sort methods
var (
	SortByName PkgSortMethod = func(first, second *PkgConfig) bool {
		return sort.StringsAreSorted([]string{second.Name, first.Name})
	}

	SortByVersion PkgSortMethod = func(first, second *PkgConfig) bool {
		return semver.Compare(first.Version, second.Version) == -1
	}
)
