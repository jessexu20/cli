package user_test

import (
	"errors"

	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/commands/user"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	"github.com/cloudfoundry/cli/cf/models"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/requirements/requirementsfakes"
	"github.com/cloudfoundry/cli/flags"

	"github.com/cloudfoundry/cli/cf/api/apifakes"
	"github.com/cloudfoundry/cli/cf/api/featureflags/featureflagsfakes"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UnsetSpaceRole", func() {
	var (
		ui         *testterm.FakeUI
		configRepo coreconfig.Repository
		userRepo   *apifakes.FakeUserRepository
		spaceRepo  *apifakes.FakeSpaceRepository
		flagRepo   *featureflagsfakes.FakeFeatureFlagRepository

		cmd         commandregistry.Command
		deps        commandregistry.Dependency
		factory     *requirementsfakes.FakeFactory
		flagContext flags.FlagContext

		loginRequirement        requirements.Requirement
		userRequirement         *requirementsfakes.FakeUserRequirement
		organizationRequirement *requirementsfakes.FakeOrganizationRequirement
	)

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		configRepo = testconfig.NewRepositoryWithDefaults()
		userRepo = new(apifakes.FakeUserRepository)
		repoLocator := deps.RepoLocator.SetUserRepository(userRepo)
		spaceRepo = new(apifakes.FakeSpaceRepository)
		repoLocator = repoLocator.SetSpaceRepository(spaceRepo)
		flagRepo = new(featureflagsfakes.FakeFeatureFlagRepository)
		repoLocator = repoLocator.SetFeatureFlagRepository(flagRepo)

		deps = commandregistry.Dependency{
			UI:          ui,
			Config:      configRepo,
			RepoLocator: repoLocator,
		}

		cmd = &user.UnsetSpaceRole{}
		cmd.SetDependency(deps, false)

		flagContext = flags.NewFlagContext(map[string]flags.FlagSet{})

		factory = new(requirementsfakes.FakeFactory)

		loginRequirement = &passingRequirement{}
		factory.NewLoginRequirementReturns(loginRequirement)

		userRequirement = new(requirementsfakes.FakeUserRequirement)
		userRequirement.ExecuteReturns(nil)
		factory.NewUserRequirementReturns(userRequirement)

		organizationRequirement = new(requirementsfakes.FakeOrganizationRequirement)
		organizationRequirement.ExecuteReturns(nil)
		factory.NewOrganizationRequirementReturns(organizationRequirement)
	})

	Describe("Requirements", func() {
		Context("when not provided exactly four args", func() {
			BeforeEach(func() {
				flagContext.Parse("the-user-name", "the-org-name", "the-space-name")
			})

			It("fails with usage", func() {
				Expect(func() { cmd.Requirements(factory, flagContext) }).To(Panic())
				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Incorrect Usage. Requires USERNAME, ORG, SPACE, ROLE as arguments"},
					[]string{"NAME"},
					[]string{"USAGE"},
				))
			})
		})

		Context("when provided four args", func() {
			BeforeEach(func() {
				flagContext.Parse("the-user-name", "the-org-name", "the-space-name", "SpaceManager")
			})

			It("returns a LoginRequirement", func() {
				actualRequirements := cmd.Requirements(factory, flagContext)
				Expect(factory.NewLoginRequirementCallCount()).To(Equal(1))

				Expect(actualRequirements).To(ContainElement(loginRequirement))
			})

			It("returns an OrgRequirement", func() {
				actualRequirements := cmd.Requirements(factory, flagContext)
				Expect(factory.NewOrganizationRequirementCallCount()).To(Equal(1))
				Expect(factory.NewOrganizationRequirementArgsForCall(0)).To(Equal("the-org-name"))

				Expect(actualRequirements).To(ContainElement(organizationRequirement))
			})

			Context("when the config version is >=2.37.0", func() {
				BeforeEach(func() {
					configRepo.SetAPIVersion("2.37.0")
				})

				It("requests the unset_roles_by_username flag", func() {
					cmd.Requirements(factory, flagContext)
					Expect(flagRepo.FindByNameCallCount()).To(Equal(1))
					Expect(flagRepo.FindByNameArgsForCall(0)).To(Equal("unset_roles_by_username"))
				})

				Context("when the unset_roles_by_username flag exists and is enabled", func() {
					BeforeEach(func() {
						flagRepo.FindByNameReturns(models.FeatureFlag{Enabled: true}, nil)
					})

					It("returns a UserRequirement", func() {
						actualRequirements := cmd.Requirements(factory, flagContext)
						Expect(factory.NewUserRequirementCallCount()).To(Equal(1))
						actualUsername, actualWantGUID := factory.NewUserRequirementArgsForCall(0)
						Expect(actualUsername).To(Equal("the-user-name"))
						Expect(actualWantGUID).To(BeFalse())

						Expect(actualRequirements).To(ContainElement(userRequirement))
					})
				})

				Context("when the unset_roles_by_username flag exists and is disabled", func() {
					BeforeEach(func() {
						flagRepo.FindByNameReturns(models.FeatureFlag{Enabled: false}, nil)
					})

					It("returns a UserRequirement", func() {
						actualRequirements := cmd.Requirements(factory, flagContext)
						Expect(factory.NewUserRequirementCallCount()).To(Equal(1))
						actualUsername, actualWantGUID := factory.NewUserRequirementArgsForCall(0)
						Expect(actualUsername).To(Equal("the-user-name"))
						Expect(actualWantGUID).To(BeTrue())

						Expect(actualRequirements).To(ContainElement(userRequirement))
					})
				})

				Context("when the unset_roles_by_username flag cannot be retrieved", func() {
					BeforeEach(func() {
						flagRepo.FindByNameReturns(models.FeatureFlag{}, errors.New("some error"))
					})

					It("returns a UserRequirement", func() {
						actualRequirements := cmd.Requirements(factory, flagContext)
						Expect(factory.NewUserRequirementCallCount()).To(Equal(1))
						actualUsername, actualWantGUID := factory.NewUserRequirementArgsForCall(0)
						Expect(actualUsername).To(Equal("the-user-name"))
						Expect(actualWantGUID).To(BeTrue())

						Expect(actualRequirements).To(ContainElement(userRequirement))
					})
				})
			})

			Context("when the config version is <2.37.0", func() {
				BeforeEach(func() {
					configRepo.SetAPIVersion("2.36.0")
				})

				It("returns a UserRequirement", func() {
					actualRequirements := cmd.Requirements(factory, flagContext)
					Expect(factory.NewUserRequirementCallCount()).To(Equal(1))
					actualUsername, actualWantGUID := factory.NewUserRequirementArgsForCall(0)
					Expect(actualUsername).To(Equal("the-user-name"))
					Expect(actualWantGUID).To(BeTrue())

					Expect(actualRequirements).To(ContainElement(userRequirement))
				})
			})
		})
	})

	Describe("Execute", func() {
		var org models.Organization

		BeforeEach(func() {
			flagContext.Parse("the-user-name", "the-org-name", "the-space-name", "SpaceManager")
			cmd.Requirements(factory, flagContext)

			org = models.Organization{}
			org.GUID = "the-org-guid"
			org.Name = "the-org-name"
			organizationRequirement.GetOrganizationReturns(org)
		})

		Context("when the space is not found", func() {
			BeforeEach(func() {
				spaceRepo.FindByNameInOrgReturns(models.Space{}, errors.New("space-repo-error"))
			})

			It("doesn't call CC", func() {
				Expect(userRepo.UnsetSpaceRoleByGUIDCallCount()).To(BeZero())
				Expect(userRepo.UnsetSpaceRoleByUsernameCallCount()).To(BeZero())
			})

			It("panics and prints a failure message", func() {
				Expect(func() { cmd.Execute(flagContext) }).To(Panic())
				Expect(ui.Outputs).To(BeInDisplayOrder(
					[]string{"FAILED"},
					[]string{"space-repo-error"},
				))
			})
		})

		Context("when the space is found", func() {
			BeforeEach(func() {
				space := models.Space{}
				space.GUID = "the-space-guid"
				space.Name = "the-space-name"
				space.Organization = org.OrganizationFields
				spaceRepo.FindByNameInOrgReturns(space, nil)
			})

			Context("when the UserRequirement returns a user with a GUID", func() {
				BeforeEach(func() {
					userFields := models.UserFields{GUID: "the-user-guid", Username: "the-user-name"}
					userRequirement.GetUserReturns(userFields)
				})

				It("tells the user it is removing the role", func() {
					cmd.Execute(flagContext)
					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"Removing role", "SpaceManager", "the-user-name", "the-org", "the-user-name"},
						[]string{"OK"},
					))
				})

				It("removes the role using the GUID", func() {
					cmd.Execute(flagContext)
					Expect(userRepo.UnsetSpaceRoleByGUIDCallCount()).To(Equal(1))
					actualUserGUID, actualSpaceGUID, actualRole := userRepo.UnsetSpaceRoleByGUIDArgsForCall(0)
					Expect(actualUserGUID).To(Equal("the-user-guid"))
					Expect(actualSpaceGUID).To(Equal("the-space-guid"))
					Expect(actualRole).To(Equal("SpaceManager"))
				})

				Context("when the call to CC fails", func() {
					BeforeEach(func() {
						userRepo.UnsetSpaceRoleByGUIDReturns(errors.New("user-repo-error"))
					})

					It("panics and prints a failure message", func() {
						Expect(func() { cmd.Execute(flagContext) }).To(Panic())
						Expect(ui.Outputs).To(BeInDisplayOrder(
							[]string{"FAILED"},
							[]string{"user-repo-error"},
						))
					})
				})
			})

			Context("when the UserRequirement returns a user without a GUID", func() {
				BeforeEach(func() {
					userRequirement.GetUserReturns(models.UserFields{Username: "the-user-name"})
				})

				It("removes the role using the given username", func() {
					cmd.Execute(flagContext)
					Expect(userRepo.UnsetSpaceRoleByUsernameCallCount()).To(Equal(1))
					actualUsername, actualSpaceGUID, actualRole := userRepo.UnsetSpaceRoleByUsernameArgsForCall(0)
					Expect(actualUsername).To(Equal("the-user-name"))
					Expect(actualSpaceGUID).To(Equal("the-space-guid"))
					Expect(actualRole).To(Equal("SpaceManager"))
				})

				It("tells the user it is removing the role", func() {
					cmd.Execute(flagContext)
					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"Removing role", "SpaceManager", "the-user-name", "the-org", "the-user-name"},
						[]string{"OK"},
					))
				})

				Context("when the call to CC fails", func() {
					BeforeEach(func() {
						userRepo.UnsetSpaceRoleByUsernameReturns(errors.New("user-repo-error"))
					})

					It("panics and prints a failure message", func() {
						Expect(func() { cmd.Execute(flagContext) }).To(Panic())
						Expect(ui.Outputs).To(BeInDisplayOrder(
							[]string{"FAILED"},
							[]string{"user-repo-error"},
						))
					})
				})
			})
		})
	})
})
