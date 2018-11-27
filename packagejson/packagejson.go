package packagejson

// TODO: This file is common between node-related buildpacks. Find a shared place to put it.

import (
	"errors"
	"os"

	"github.com/cloudfoundry/npm-cnb/utils"
)

type PackageJSON struct {
	Engines Engines `json:"engines"`
}

type Engines struct {
	Node string `json:"node"`
	Yarn string `json:"yarn"`
	NPM  string `json:"npm"`
	IOJS string `json:"iojs"`
}

type logger interface {
	Info(format string, args ...interface{})
}

func LoadPackageJSON(path string, logger logger) (PackageJSON, error) {
	var p PackageJSON

	err := utils.NewJSON().Load(path, &p)
	if err != nil && !os.IsNotExist(err) {
		return PackageJSON{}, err
	}

	if p.Engines.IOJS != "" {
		return PackageJSON{}, errors.New("io.js not supported by this buildpack")
	}

	if p.Engines.Node != "" {
		logger.Info("engines.node (package.json): %s", p.Engines.Node)
	} else {
		logger.Info("engines.node (package.json): unspecified")
	}

	if p.Engines.NPM != "" {
		logger.Info("engines.npm (package.json): %s", p.Engines.NPM)
	} else {
		logger.Info("engines.npm (package.json): unspecified (use default)")
	}

	return p, nil
}
