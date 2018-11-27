package detect

import (
	"fmt"
	"path/filepath"

	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/npm-cnb/packagejson"
)

const NodeDependency = "node"
const NPMDependency = "npm"

func UpdateBuildPlan(libDetect *libbuildpack.Detect) error {
	packageJSONPath := filepath.Join(libDetect.Application.Root, "package.json")
	if exists, err := libjavabuildpack.FileExists(packageJSONPath); err != nil {
		return fmt.Errorf("error checking filepath %s", packageJSONPath)
	} else if !exists {
		return fmt.Errorf("no package.json found in %s", packageJSONPath)
	}

	pkgJSON, err := packagejson.LoadPackageJSON(packageJSONPath, libDetect.Logger)
	if err != nil {
		return err
	}

	libDetect.BuildPlan[NodeDependency] = libbuildpack.BuildPlanDependency{
		Version: pkgJSON.Engines.Node,
		Metadata: libbuildpack.BuildPlanDependencyMetadata{
			"build":  true,
			"launch": true,
		},
	}

	libDetect.BuildPlan[NPMDependency] = libbuildpack.BuildPlanDependency{
		Metadata: libbuildpack.BuildPlanDependencyMetadata{
			"launch": true,
		},
	}

	return nil
}
