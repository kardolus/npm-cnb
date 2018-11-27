package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	libbuildpackV3 "github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/npm-cnb/detect"
	"github.com/cloudfoundry/npm-cnb/utils"
	"github.com/fatih/color"
)

func CreateLaunchMetadata() libbuildpackV3.LaunchMetadata {
	return libbuildpackV3.LaunchMetadata{
		Processes: libbuildpackV3.Processes{
			libbuildpackV3.Process{
				Type:    "web",
				Command: "npm start",
			},
		},
	}
}

type ModuleInstaller interface {
	InstallToLayer(string, string) error
	RebuildLayer(string, string) error
	CleanAndCopyToDst(string, string) error
}

type Modules struct {
	BuildContribution, LaunchContribution bool
	App                                   libbuildpackV3.Application
	CacheLayer                            libbuildpackV3.CacheLayer
	LaunchLayer                           libbuildpackV3.LaunchLayer
	Logger                                libjavabuildpack.Logger
	NPM                                   ModuleInstaller
}

type Metadata struct {
	SHA256 string `toml:"sha256"`
}

func NewModules(builder libjavabuildpack.Build, npm ModuleInstaller) (m Modules, planExists bool, e error) {
	bp, planExists := builder.BuildPlan[detect.NPMDependency]
	if !planExists {
		return Modules{}, false, nil
	}

	modules := Modules{
		NPM:         npm,
		App:         builder.Application,
		Logger:      builder.Logger,
		CacheLayer:  builder.Cache.Layer(detect.NPMDependency),
		LaunchLayer: builder.Launch.Layer(detect.NPMDependency),
	}

	var isBool bool
	if val, contributeBuild := bp.Metadata["build"]; contributeBuild {
		modules.BuildContribution, isBool = val.(bool)
		if !isBool {
			return Modules{}, false, errors.New("NPM build plan build contribution must be boolean")
		}
	}

	if val, contributeLaunch := bp.Metadata["launch"]; contributeLaunch {
		modules.LaunchContribution, isBool = val.(bool)
		if !isBool {
			return Modules{}, false, errors.New("NPM build plan launch contribution must be boolean")
		}
	}
	return modules, true, nil
}

func (m Modules) Contribute() error {
	if !m.BuildContribution && !m.LaunchContribution {
		return nil
	}

	if m.BuildContribution {
		if !m.LaunchContribution {
			m.Logger.FirstLine("%s: %s to cache", logHeader(), color.YellowString("Contributing"))
			if err := m.installInCache(); err != nil {
				return fmt.Errorf("failed to install in cache for build : %v", err)
			}
		}

		m.Logger.SubsequentLine("Writing NODE_PATH")
		if err := m.CacheLayer.AppendPathEnv("NODE_PATH", filepath.Join(m.CacheLayer.Root, "node_modules")); err != nil {
			return err
		}
	}

	if m.LaunchContribution {
		if sameSHASums, err := m.packageLockMatchesMetadataSHA(); err != nil {
			return err
		} else if sameSHASums {
			m.Logger.FirstLine("%s: %s cached launch layer", logHeader(), color.GreenString("Reusing"))
			return nil
		}

		m.Logger.FirstLine("%s: %s to launch", logHeader(), color.YellowString("Contributing"))

		if err := m.installInCache(); err != nil {
			return fmt.Errorf("failed to install in cache for launch : %v", err)
		}

		if err := m.installInLaunch(); err != nil {
			return fmt.Errorf("failed to install in launch : %v", err)
		}

		if err := m.writeProfile(); err != nil {
			return fmt.Errorf("failed to write profile.d : %v", err)
		}
	}

	appModulesDir := filepath.Join(m.App.Root, "node_modules")
	if err := os.RemoveAll(appModulesDir); err != nil {
		return fmt.Errorf("failed to clean up the node_modules: %v", err)
	}

	return nil
}

func (m Modules) packageLockMatchesMetadataSHA() (bool, error) {
	packageLockPath := filepath.Join(m.App.Root, "package-lock.json")
	if exists, err := libjavabuildpack.FileExists(packageLockPath); err != nil {
		return false, fmt.Errorf("failed to check for package-lock.json: %v", err)
	} else if !exists {
		return false, fmt.Errorf("there is no package-lock.json in the app")
	}

	buf, err := ioutil.ReadFile(packageLockPath)
	if err != nil {
		return false, fmt.Errorf("failed to read package-lock.json: %v", err)
	}

	var metadata Metadata
	if err := m.LaunchLayer.ReadMetadata(&metadata); err != nil {
		return false, err
	}

	metadataHash, err := hex.DecodeString(metadata.SHA256)
	if err != nil {
		return false, err
	}

	hash := sha256.Sum256(buf)
	return bytes.Equal(metadataHash, hash[:]), nil
}

func (m Modules) writeMetadataSHA(path string) error {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", path, err)
	}

	hash := sha256.Sum256(buf)
	return m.LaunchLayer.WriteMetadata(Metadata{SHA256: hex.EncodeToString(hash[:])})
}

func (m *Modules) copyModulesToLayer(src, dest string) error {
	if exist, err := libjavabuildpack.FileExists(dest); err != nil {
		return err
	} else if !exist {
		if err := os.MkdirAll(dest, 0777); err != nil {
			return err
		}
	}
	return utils.CopyDirectory(src, dest)
}

func (m Modules) installInCache() error {
	appModulesDir := filepath.Join(m.App.Root, "node_modules")

	vendored, err := libjavabuildpack.FileExists(appModulesDir)
	if err != nil {
		return fmt.Errorf("could not locate app modules directory : %s", err)
	}

	if vendored {
		m.Logger.SubsequentLine("%s node_modules", color.YellowString("Rebuilding"))

		if err := m.NPM.RebuildLayer(m.App.Root, m.CacheLayer.Root); err != nil {
			return fmt.Errorf("failed to rebuild node_modules: %v", err)
		}
	} else {
		m.Logger.SubsequentLine("%s node_modules", color.YellowString("Installing"))

		cacheModulesDir := filepath.Join(m.CacheLayer.Root, "node_modules")
		if exists, err := libjavabuildpack.FileExists(cacheModulesDir); err != nil {
			return err
		} else if !exists {
			if err := os.MkdirAll(cacheModulesDir, 0777); err != nil {
				return fmt.Errorf("could not make node modules directory : %s", err)
			}
		}

		if err := os.Symlink(cacheModulesDir, appModulesDir); err != nil {
			return fmt.Errorf("could not symlink node modules directory : %s", err)
		}
		defer os.Remove(appModulesDir)

		if err := m.NPM.InstallToLayer(m.App.Root, m.CacheLayer.Root); err != nil {
			return fmt.Errorf("failed to install and copy node_modules: %v", err)
		}
	}

	return nil
}

func (m Modules) installInLaunch() error {
	cacheModulesDir := filepath.Join(m.CacheLayer.Root, "node_modules")
	launchModulesDir := filepath.Join(m.LaunchLayer.Root, "node_modules")

	if err := m.NPM.CleanAndCopyToDst(cacheModulesDir, launchModulesDir); err != nil {
		return fmt.Errorf("failed to copy the node_modules to the launch layer: %v", err)
	}

	if err := m.writeMetadataSHA(filepath.Join(m.App.Root, "package-lock.json")); err != nil {
		return fmt.Errorf("failed to write metadata to package-lock.json: %v", err)
	}

	return nil
}

func (m Modules) writeProfile() error {
	m.Logger.SubsequentLine("Writing profile.d/NODE_PATH")

	launchModulesDir := filepath.Join(m.LaunchLayer.Root, "node_modules")
	if err := m.LaunchLayer.WriteProfile("NODE_PATH", fmt.Sprintf("export NODE_PATH=\"%s\"", launchModulesDir)); err != nil {
		return fmt.Errorf("failed to write NODE_PATH in the launch layer: %v", err)
	}
	return nil
}

func logHeader() string {
	return color.New(color.FgBlue, color.Bold).Sprint("Node Modules")
}
