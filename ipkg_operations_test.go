package ipkg_test

import (
	"database/sql"
	"testing"

	osextra "github.com/ira-package-manager/gobetter/os_extra"
	"github.com/ira-package-manager/ipkg"
)

func TestInstallUncompressed(t *testing.T) {
	// Opening database
	root, err := ipkg.OpenRoot("./test/db")
	if err != nil {
		t.Fatal(err)
	}
	// Checking installisation
	err = root.InstallPackage("./test/pkgs/testpkg", true)
	if err != nil {
		t.Fatal(err)
	}
	testInstalled(root, t)
}

func TestInstallCompressed(t *testing.T) {
	root, err := ipkg.OpenRoot("./test/db")
	if err != nil {
		t.Fatal(err)
	}
	// Checking installisation
	err = root.InstallPackage("./test/pkgs/testpkg.ipkg", true)
	if err != nil {
		t.Fatal(err)
	}
	testInstalled(root, t)
}

func testInstalled(root *ipkg.Root, t *testing.T) {
	if !osextra.Exists("./test/db/testpkg-$1.0") {
		t.Error("package wasn't installed in root")
	} else if !osextra.Exists("./test/db/testpkg-$1.0/scripts", "./test/db/testpkg-$1.0/cfg") {
		t.Error("package has wrong structure")
	} else if !osextra.Exists("./test/db/testpkg-$1.0/scripts/run.sh", "./test/db/testpkg-$1.0/cfg/main.ini") {
		t.Error("package has no files")
	} else if !osextra.Exists("./test/db/testpkg-$1.0/.ira/iscript") {
		t.Error("IScript wasn't saved")
	}
	if _, err := root.FindPackage("testpkg", "1.0"); err == sql.ErrNoRows {
		t.Error("package is not in database")
	}
	if t.Failed() {
		return
	}
	// Checking remove
	testRemove(root, t)
}

func testRemove(root *ipkg.Root, t *testing.T) {
	err := root.RemovePackage("testpkg", "1.0", true)
	if err != nil {
		t.Fatal(err)
	}
	if osextra.Exists("./test/db/testpkg-$1.0") {
		t.Error("package wasn't removed")
	}
	if _, err = root.FindPackage("testpkg", "1.0"); err == nil {
		t.Error("package is still in database")
	} else if err != sql.ErrNoRows {
		t.Error(err)
	}
}
