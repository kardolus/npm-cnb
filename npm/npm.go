package npm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/npm-cnb/utils"
	"github.com/pkg/errors"
)

type NPM struct {
	Runner Runner
}

func NewNPM() *NPM {
	return &NPM{Runner: &npmCmd{}}
}

func (n *NPM) InstallToLayer(srcLayer, dstLayer string) error {
	srcPackageJsonPath := filepath.Join(srcLayer, "package.json")
	if exists, err := libjavabuildpack.FileExists(srcPackageJsonPath); err != nil || !exists {
		return fmt.Errorf("failed to find file %s ", srcPackageJsonPath)
	}

	return n.Runner.Run(srcLayer, "install", "--unsafe-perm", "--cache", filepath.Join(srcLayer, "npm-cache"))
}

func (n *NPM) RebuildModules(srcLayer, dstLayer string) error {
	srcPackageJsonPath := filepath.Join(srcLayer, "package.json")
	if exists, err := libjavabuildpack.FileExists(srcPackageJsonPath); err != nil || !exists {
		return fmt.Errorf("failed to find file %s ", srcPackageJsonPath)
	}

	if err := n.Runner.Run(srcLayer, "rebuild"); err != nil {
		return err
	}

	srcModulesDir := filepath.Join(srcLayer, "node_modules")
	dstModulesDir := filepath.Join(dstLayer, "node_modules")

	return n.CleanAndCopyToDst(srcModulesDir, dstModulesDir)
}

func (n *NPM) CleanAndCopyToDst(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to remove modules in %s", dst))
	}

	if err := n.copyModules(src, dst); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to copy the src modules from %s to %s", src, dst))
	}

	return n.copyModules(src, dst)
}

func (n *NPM) copyModules(src, dst string) error {
	exists, err := libjavabuildpack.FileExists(dst)
	if err != nil {
		return err
	}

	if !exists {
		if err := os.MkdirAll(dst, 0777); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to make directory %s", dst))
		}
	}

	return utils.CopyDirectory(src, dst)
}
