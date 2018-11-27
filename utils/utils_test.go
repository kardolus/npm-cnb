package utils_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cloudfoundry/npm-cnb/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	const windowsFileModeWarning = "Windows does not respect file mode bits as Linux does, see https://golang.org/pkg/os/#Chmod"

	Describe("CopyFile", func() {
		var (
			tmpdir   string
			err      error
			fileInfo os.FileInfo
			oldMode  os.FileMode
			oldUmask int
		)
		BeforeEach(func() {
			var fh *os.File
			sourceFile := "fixtures/source.txt"

			tmpdir, err = ioutil.TempDir("", "copy")
			Expect(err).To(BeNil())

			fileInfo, err = os.Stat(sourceFile)
			Expect(err).To(BeNil())
			oldMode = fileInfo.Mode()

			fh, err = os.Open(sourceFile)
			Expect(err).To(BeNil())
			defer fh.Close()

			if runtime.GOOS != "windows" {
				err = fh.Chmod(0742)
				Expect(err).To(BeNil())
			}

			oldUmask = umask(0000)
		})
		AfterEach(func() {
			var fh *os.File
			sourceFile := "fixtures/source.txt"

			fh, err = os.Open(sourceFile)
			Expect(err).To(BeNil())
			defer fh.Close()

			if runtime.GOOS != "windows" {
				err = fh.Chmod(oldMode)
				Expect(err).To(BeNil())
			}

			err = os.RemoveAll(tmpdir)
			Expect(err).To(BeNil())

			umask(oldUmask)
		})

		Context("with a valid source file", func() {
			It("copies the file", func() {
				err = utils.CopyFile("fixtures/source.txt", filepath.Join(tmpdir, "out.txt"))
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "out.txt")).To(BeAnExistingFile())
				Expect(ioutil.ReadFile(filepath.Join(tmpdir, "out.txt"))).To(Equal([]byte("a file\n")))
			})

			It("preserves the file mode", func() {
				if runtime.GOOS == "windows" {
					Skip(windowsFileModeWarning)
				}

				err = utils.CopyFile("fixtures/source.txt", filepath.Join(tmpdir, "out.txt"))
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "out.txt")).To(BeAnExistingFile())
				fileInfo, err = os.Stat(filepath.Join(tmpdir, "out.txt"))

				Expect(fileInfo.Mode()).To(Equal(os.FileMode(0742)))
			})
		})
	})

	Describe("CopyDirectory", func() {
		var (
			destDir string
			err     error
		)

		BeforeEach(func() {
			destDir, err = ioutil.TempDir("", "destDir")
			Expect(err).To(BeNil())
		})

		It("copies source to destination", func() {
			srcDir := filepath.Join("fixtures", "copydir")
			err = utils.CopyDirectory(srcDir, destDir)
			Expect(err).To(BeNil())

			Expect(filepath.Join(srcDir, "source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(srcDir, "standard", "manifest.yml")).To(BeAnExistingFile())

			Expect(filepath.Join(destDir, "source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(destDir, "standard", "manifest.yml")).To(BeAnExistingFile())
		})

		It("handles symlink to directory", func() {
			if runtime.GOOS == "windows" {
				Skip("Symlinks require administrator privileges on windows and are not used")
			}

			srcDir := filepath.Join("fixtures", "copydir_symlinks")
			err = utils.CopyDirectory(srcDir, destDir)
			Expect(err).To(BeNil())

			Expect(filepath.Join(srcDir, "source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(srcDir, "sym_source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(srcDir, "standard", "manifest.yml")).To(BeAnExistingFile())
			Expect(filepath.Join(srcDir, "sym_standard", "manifest.yml")).To(BeAnExistingFile())

			Expect(filepath.Join(destDir, "source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(destDir, "sym_source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(destDir, "standard", "manifest.yml")).To(BeAnExistingFile())
			Expect(filepath.Join(destDir, "sym_standard", "manifest.yml")).To(BeAnExistingFile())
		})
	})

	Describe("JSON", func() {
		var (
			json   *utils.JSON
			tmpDir string
			err    error
		)

		BeforeEach(func() {
			tmpDir, err = ioutil.TempDir("", "json")
			Expect(err).To(BeNil())

			json = &utils.JSON{}
		})

		AfterEach(func() {
			err = os.RemoveAll(tmpDir)
			Expect(err).To(BeNil())
		})

		Describe("Load", func() {
			Context("file is valid json", func() {
				Context("that starts with BOM", func() {
					BeforeEach(func() {
						ioutil.WriteFile(filepath.Join(tmpDir, "valid.json"), []byte("\uFEFF"+`{"key": "value"}`), 0666)
					})
					It("returns an error", func() {
						obj := make(map[string]string)
						err = json.Load(filepath.Join(tmpDir, "valid.json"), &obj)

						Expect(err).To(BeNil())
						Expect(obj["key"]).To(Equal("value"))
					})
				})
				Context("that does not start with BOM", func() {
					BeforeEach(func() {
						ioutil.WriteFile(filepath.Join(tmpDir, "valid.json"), []byte(`{"key": "value"}`), 0666)
					})
					It("returns an error", func() {
						obj := make(map[string]string)
						err = json.Load(filepath.Join(tmpDir, "valid.json"), &obj)

						Expect(err).To(BeNil())
						Expect(obj["key"]).To(Equal("value"))
					})
				})
			})

			Context("file is NOT valid json", func() {
				BeforeEach(func() {
					ioutil.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("not valid json"), 0666)
				})
				It("returns an error", func() {
					obj := make(map[string]string)
					err = json.Load(filepath.Join(tmpDir, "invalid.json"), &obj)
					Expect(err).ToNot(BeNil())
				})
			})

			Context("file does not exist", func() {
				It("returns an error", func() {
					obj := make(map[string]string)
					err = json.Load(filepath.Join(tmpDir, "does_not_exist.json"), &obj)
					Expect(err).ToNot(BeNil())
				})
			})
		})

		Describe("Write", func() {
			Context("directory exists", func() {
				It("writes the json to a file ", func() {
					obj := map[string]string{
						"key": "val",
					}
					err = json.Write(filepath.Join(tmpDir, "file.json"), obj)
					Expect(err).To(BeNil())

					Expect(ioutil.ReadFile(filepath.Join(tmpDir, "file.json"))).To(Equal([]byte(`{"key":"val"}`)))
				})
			})

			Context("directory does not exist", func() {
				It("creates the directory", func() {
					obj := map[string]string{
						"key": "val",
					}
					err = json.Write(filepath.Join(tmpDir, "extradir", "file.json"), obj)
					Expect(err).To(BeNil())

					Expect(ioutil.ReadFile(filepath.Join(tmpDir, "extradir", "file.json"))).To(Equal([]byte(`{"key":"val"}`)))
				})
			})
		})
	})
})
