package npm_test

import (
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

//go:generate mockgen -source=npm.go -destination=mocks_test.go -package=npm_test

func TestUnitNPM(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "NPM", testNPM, spec.Report(report.Terminal{}))
}

func testNPM(t *testing.T, when spec.G, it spec.S) {
	when("npm.NewContributor", func() {
		var (
			mockCtrl       *gomock.Controller
			mockPkgManager *MockPackageManager
			factory        *test.BuildFactory
		)

		it.Before(func() {
			mockCtrl = gomock.NewController(t)
			mockPkgManager = NewMockPackageManager(mockCtrl)

			factory = test.NewBuildFactory(t)

			Expect(os.Mkdir(factory.Build.Application.Root, 0777)).To(Succeed())
		})

		it.After(func() {
			mockCtrl.Finish()
		})

		when("there is no package-lock.json", func() {
			it("fails", func() {
				factory.AddBuildPlan(t, npm.Dependency, buildplan.Dependency{})

				_, _, err := npm.NewContributor(factory.Build, mockPkgManager)
				Expect(err).To(HaveOccurred())
			})
		})

		when("there is a package-lock.json", func() {
			it.Before(func() {
				lockFile := filepath.Join(factory.Build.Application.Root, "package-lock.json")
				Expect(ioutil.WriteFile(lockFile, []byte("package lock"), 0666)).To(Succeed())
			})


			it("returns true if a build plan exists", func() {
				factory.AddBuildPlan(t, npm.Dependency, buildplan.Dependency{})

				_, willContribute, err := npm.NewContributor(factory.Build, mockPkgManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
			})

			it("returns false if a build plan does not exist", func() {
				_, willContribute, err := npm.NewContributor(factory.Build, mockPkgManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeFalse())
			})

			it("uses package-lock.json for identity", func() {
				factory.AddBuildPlan(t, npm.Dependency, buildplan.Dependency{})

				contributor, _, _ := npm.NewContributor(factory.Build, mockPkgManager)
				name, version := contributor.Identity()
				Expect(name).To(Equal(npm.Dependency))
				Expect(version).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
			})

			when("the app is vendored", func() {
				it.Before(func() {
					nodeModules := filepath.Join(factory.Build.Application.Root, "node_modules")
					Expect(os.Mkdir(nodeModules, 0777)).To(Succeed())

					mockPkgManager.EXPECT().Rebuild(factory.Build.Application.Root).Do(func(location string) {
						Expect(ioutil.WriteFile(filepath.Join(nodeModules, "test_module"), []byte("some module"), 0666)).To(Succeed())
					})
				})

				it("contributes modules to the cache layer when included in the build plan", func() {
					factory.AddBuildPlan(t, npm.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"build": true},
					})

					contributor, _, err := npm.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					layer := factory.Build.Layers.Layer(npm.Dependency)
					test.BeLayerLike(t, layer, true, true, false)
					test.BeFileLike(t, filepath.Join(layer.Root, "test_module"), 0644, "some module")
					test.BeOverrideSharedEnvLike(t, layer, "NODE_PATH", layer.Root)

					Expect(filepath.Join(factory.Build.Application.Root, "node_modules")).NotTo(BeADirectory())
				})

				it("contributes modules to the launch layer when included in the build plan", func() {
					factory.AddBuildPlan(t, npm.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"launch": true},
					})

					contributor, _, err := npm.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					layer := factory.Build.Layers.Layer(npm.Dependency)
					test.BeLayerLike(t, layer, false, true, true)
					test.BeFileLike(t, filepath.Join(layer.Root, "test_module"), 0644, "some module")
					test.BeOverrideSharedEnvLike(t, layer, "NODE_PATH", layer.Root)

					Expect(filepath.Join(factory.Build.Application.Root, "node_modules")).NotTo(BeADirectory())
				})
			})

			when("the app is not vendored", func() {
				it.Before(func() {
					mockPkgManager.EXPECT().Install(factory.Build.Application.Root).Do(func(location string) {
						nodeModules := filepath.Join(factory.Build.Application.Root, "node_modules")
						Expect(os.Mkdir(nodeModules, 0777)).To(Succeed())
						Expect(ioutil.WriteFile(filepath.Join(nodeModules, "test_module"), []byte("some module"), 0666)).To(Succeed())
					})
				})

				it("contributes modules to the cache layer when included in the build plan", func() {
					factory.AddBuildPlan(t, npm.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"build": true},
					})

					contributor, _, err := npm.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					layer := factory.Build.Layers.Layer(npm.Dependency)
					test.BeLayerLike(t, layer, true, true, false)
					test.BeFileLike(t, filepath.Join(layer.Root, "test_module"), 0644, "some module")
					test.BeOverrideSharedEnvLike(t, layer, "NODE_PATH", layer.Root)

					Expect(filepath.Join(factory.Build.Application.Root, "node_modules")).NotTo(BeADirectory())
				})

				it("contributes modules to the launch layer when included in the build plan", func() {
					factory.AddBuildPlan(t, npm.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"launch": true},
					})

					contributor, _, err := npm.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					layer := factory.Build.Layers.Layer(npm.Dependency)
					test.BeLayerLike(t, layer, false, true, true)
					test.BeFileLike(t, filepath.Join(layer.Root, "test_module"), 0644, "some module")
					test.BeOverrideSharedEnvLike(t, layer, "NODE_PATH", layer.Root)

					Expect(filepath.Join(factory.Build.Application.Root, "node_modules")).NotTo(BeADirectory())
				})
			})
		})
	})
}
