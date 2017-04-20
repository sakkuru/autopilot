package main_test

import (
	"errors"
	"testing"

	"code.cloudfoundry.org/cli/plugin/pluginfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	plugin_models "code.cloudfoundry.org/cli/plugin/models"
	rollback "github.com/sakkuru/rollback-push"
)

func TestAutopilot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Blue Green Push/Rollback plugin")
}

var _ = Describe("Flag Parsing", func() {
	It("parses a complete set of args", func() {
		appName, manifestPath, appPath, err := rollback.ParseArgs(
			[]string{
				"blue-green-push",
				"appname",
				"-f", "manifest-path",
				"-p", "app-path",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		Expect(appName).To(Equal("appname"))
		Expect(manifestPath).To(Equal("manifest-path"))
		Expect(appPath).To(Equal("app-path"))
	})

	It("requires a manifest", func() {
		_, _, _, err := rollback.ParseArgs(
			[]string{
				"blue-green-push",
				"appname",
				"-p", "app-path",
			},
		)
		Expect(err).To(MatchError(rollback.ErrNoManifest))
	})

	It("parses a complete set of args for rollback", func() {
		appName, version, err := rollback.ParseRollbackArgs(
			[]string{
				"blue-green-rollback",
				"appname",
				"g1",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		Expect(appName).To(Equal("appname"))
		Expect(version).To(Equal("g1"))
	})

})

var _ = Describe("ApplicationRepo", func() {
	var (
		cliConn *pluginfakes.FakeCliConnection
		repo    *rollback.ApplicationRepo
	)

	BeforeEach(func() {
		cliConn = &pluginfakes.FakeCliConnection{}
		repo = rollback.NewApplicationRepo(cliConn)
	})

	Describe("RenameApplication", func() {
		It("renames the application", func() {
			err := repo.RenameApplication("old-name", "new-name")
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{"rename", "old-name", "new-name"}))
		})

		It("returns an error if one occurs", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("no app"))

			err := repo.RenameApplication("old-name", "new-name")
			Expect(err).To(MatchError("no app"))
		})
	})

	Describe("DoesAppExist", func() {

		It("returns an error if the cli returns an error", func() {
			cliConn.CliCommandWithoutTerminalOutputReturns([]string{}, errors.New("you shall not curl"))
			_, err := repo.DoesAppExist("app-name")

			Expect(err).To(MatchError("you shall not curl"))
		})

		It("returns an error if the cli response is invalid JSON", func() {
			response := []string{
				"}notjson{",
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.DoesAppExist("app-name")

			Expect(err).To(HaveOccurred())
		})

		It("returns an error if the cli response doesn't contain total_results", func() {
			response := []string{
				`{"brutal_results":2}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.DoesAppExist("app-name")

			Expect(err).To(MatchError("Missing total_results from api response"))
		})

		It("returns an error if the cli response contains a non-number total_results", func() {
			response := []string{
				`{"total_results":"sandwich"}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.DoesAppExist("app-name")

			Expect(err).To(MatchError("total_results didn't have a number sandwich"))
		})

		It("returns true if the app exists", func() {
			response := []string{
				`{"total_results":1}`,
			}
			spaceGUID := "4"

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			cliConn.GetCurrentSpaceReturns(
				plugin_models.Space{
					SpaceFields: plugin_models.SpaceFields{
						Guid: spaceGUID,
					},
				},
				nil,
			)

			result, err := repo.DoesAppExist("app-name")

			Expect(cliConn.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
			args := cliConn.CliCommandWithoutTerminalOutputArgsForCall(0)
			Expect(args).To(Equal([]string{"curl", "v2/apps?q=name:app-name&q=space_guid:4"}))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("URL encodes the application name", func() {
			response := []string{
				`{"total_results":1}`,
			}
			spaceGUID := "4"

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			cliConn.GetCurrentSpaceReturns(
				plugin_models.Space{
					SpaceFields: plugin_models.SpaceFields{
						Guid: spaceGUID,
					},
				},
				nil,
			)

			result, err := repo.DoesAppExist("app name")

			Expect(cliConn.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
			args := cliConn.CliCommandWithoutTerminalOutputArgsForCall(0)
			Expect(args).To(Equal([]string{"curl", "v2/apps?q=name:app+name&q=space_guid:4"}))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("returns false if the app does not exist", func() {
			response := []string{
				`{"total_results":0}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			result, err := repo.DoesAppExist("app-name")

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeFalse())
		})

	})

	Describe("PushApplication", func() {
		It("pushes an application with both a manifest and a path", func() {
			err := repo.PushApplication("appName", "/path/to/a/manifest.yml", "/path/to/the/app")
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{
				"push",
				"appName",
				"-f", "/path/to/a/manifest.yml",
				"-p", "/path/to/the/app",
			}))
		})

		It("pushes an application with only a manifest", func() {
			err := repo.PushApplication("appName", "/path/to/a/manifest.yml", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{
				"push",
				"appName",
				"-f", "/path/to/a/manifest.yml",
			}))
		})

		It("returns errors from the push", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad app"))

			err := repo.PushApplication("appName", "/path/to/a/manifest.yml", "/path/to/the/app")
			Expect(err).To(MatchError("bad app"))
		})
	})

	Describe("DeleteApplication", func() {
		It("deletes all trace of an application", func() {
			err := repo.DeleteApplication("app-name")
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{
				"delete", "app-name",
				"-f",
			}))
		})

		It("returns errors from the delete", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad app"))

			err := repo.DeleteApplication("app-name")
			Expect(err).To(MatchError("bad app"))
		})
	})

	Describe("ListApplications", func() {
		It("lists all the applications", func() {
			err := repo.ListApplications()
			Expect(err).ToNot(HaveOccurred())

			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{"apps"}))
		})

		It("returns errors from the list", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad apps"))

			err := repo.ListApplications()
			Expect(err).To(MatchError("bad apps"))
		})
	})

	Describe("StartApplication", func() {
		It("starts an instance", func() {
			err := repo.StartApplication("appName")
			Expect(err).ToNot(HaveOccurred())
			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{"start", "appName"}))
		})
	})

	Describe("Map Route Application", func() {

		It("map route an application", func() {
			cliConn.GetAppReturns(plugin_models.GetAppModel{
				Routes: []plugin_models.GetApp_RouteSummary{plugin_models.GetApp_RouteSummary{
					Guid: "aaabbbbccc",
					Host: "hostName",
					Domain: plugin_models.GetApp_DomainFields{
						Name: "microsoft.com",
					},
				},
				},
			}, nil)

			err := repo.MapRouteApplication("appName", "hostName")
			Expect(err).ToNot(HaveOccurred())
			Expect(cliConn.CliCommandCallCount()).To(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Expect(args).To(Equal([]string{"map-route", "appName", "microsoft.com", "-n", "hostName"}))
		})
	})
})
