package npm

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/helper"
)

type Logger interface {
	Info(format string, args ...interface{})
}

type NPM struct {
	Logger Logger
}

func (n NPM) Install(cache, location string) error {
	nodeModules, existingNodeModules := filepath.Join(location, "node_modules"), filepath.Join(cache, "node_modules")
	if exists, err := helper.FileExists(existingNodeModules); err != nil {
		return err
	} else if exists {
		n.Logger.Info("Reusing existing node_modules")
		if err := helper.CopyDirectory(existingNodeModules, nodeModules); err != nil {
			return err
		}
		defer os.RemoveAll(existingNodeModules)
	}

	npmCache, existingNPMCache := filepath.Join(location, "npm-cache"), filepath.Join(cache, "npm-cache")
	if exists, err := helper.FileExists(existingNPMCache); err != nil {
		return err
	} else if exists {
		n.Logger.Info("Reusing existing npm-cache")
		if err := helper.CopyDirectory(existingNPMCache, npmCache); err != nil {
			return err
		}
		defer os.RemoveAll(existingNPMCache)
	}

	if err := run(location, "install", "--unsafe-perm", "--cache", npmCache); err != nil {
		return err
	}

	return run(location, "cache", "verify", "--cache", npmCache)
}

func (n NPM) Rebuild(location string) error {
	return run(location, "rebuild")
}

func run(dir string, args ...string) error {
	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
