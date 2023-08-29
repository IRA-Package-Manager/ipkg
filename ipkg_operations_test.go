package ipkg_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/ira-package-manager/ipkg"
)

func TestInstall(t *testing.T) {
	root, err := ipkg.OpenRoot("./test/db")
	if err != nil {
		t.Fatal(err)
	}
	err = root.InstallPackage("./test/pkgs/testpkg", true)
	if err != nil {
		t.Fatal(err)
	}
	if !exists("./test/db/testpkg-$1.0") {
		t.Error("package wasn't installed in root")
	} else if !exists("./test/db/testpkg-$1.0/scripts", "./test/db/testpkg-$1.0/cfg") {
		t.Error("package has wrong structure")
	} else if !exists("./test/db/testpkg-$1.0/scripts/run.sh", "./test/db/testpkg-$1.0/cfg/main.ini") {
		t.Error("package has no files")
	} else if !exists("./test/db/testpkg-$1.0/.ira/iscript") {
		t.Error("IScript wasn't saved")
	}
	if _, err = root.FindPackage("testpkg", "1.0"); err == sql.ErrNoRows {
		t.Error("package is not in database")
	}
}

func exists(filePaths ...string) bool {
	for _, path := range filePaths {
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			return false
		}
	}
	return true
}
