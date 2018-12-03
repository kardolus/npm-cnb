package package_manager

import (
	"os"
	"os/exec"
	"path/filepath"
)

type NodePackageManager struct {
}

func (n NodePackageManager) Install(location string) error {
	return run(location, "install", "--unsafe-perm", "--cache", filepath.Join(location, "npm-cache"))
}

func (n NodePackageManager) Rebuild(location string) error {
	return run(location, "rebuild")
}

func run(dir string, args ...string) error {
	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
