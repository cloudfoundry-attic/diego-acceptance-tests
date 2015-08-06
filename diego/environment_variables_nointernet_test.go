package diego

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Environment Variables", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		Eventually(cf.Cf("logs", appName, "--recent")).Should(Exit())
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Context("Running environment variables", func() {
		var runningEnv string

		BeforeEach(func() {
			cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
				session := cf.Cf("curl", "/v2/config/environment_variable_groups/running")
				Eventually(session).Should(Exit(0))
				runningEnv = string(session.Out.Contents())

				Eventually(cf.Cf("set-running-environment-variable-group", `{"RUNNING_TEST_VAR":"running_env_value"}`)).Should(Exit(0))
			})

			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Dora, "--no-start", "-b", "ruby_buildpack"), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
		})

		AfterEach(func() {
			cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
				Eventually(cf.Cf("set-running-environment-variable-group", runningEnv)).Should(Exit(0))
			})
		})

		It("Applies correct environment variables while running apps", func() {
			session := cf.Cf("start", appName)
			Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))

			Expect(helpers.CurlApp(appName, "/env/RUNNING_TEST_VAR")).To(ContainSubstring("running_env_value"))
		})
	})
})
