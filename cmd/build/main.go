package main

import (
	"fmt"
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/npm-cnb/package_manager"
	"os"
)

func main() {
	builder, err := build.DefaultBuild()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create default builder: %s", err)
		os.Exit(100)
	}

	code, err := runBuild(builder)
	if err != nil {
		build.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runBuild(builder build.Build) (int, error) {
	builder.Logger.FirstLine(build.Logger.PrettyIdentity(build.Buildpack))

	contributor, willContribute, err := npm.NewContributor(builder, package_manager.NodePackageManager{})
	if err != nil {
		return builder.Failure(102), err
	}

	if willContribute {
		if err := contributor.Contribute(); err != nil {
			return builder.Failure(103), err
		}
	}

	return builder.Success(buildplan.BuildPlan{})
}
