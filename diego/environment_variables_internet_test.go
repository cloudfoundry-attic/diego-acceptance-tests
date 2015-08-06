// +build !noInternet

package diego

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
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

	Context("Staging environment variables", func() {
		var stagingEnv string

		BeforeEach(func() {
			cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
				session := cf.Cf("curl", "/v2/config/environment_variable_groups/staging")
				Eventually(session).Should(Exit(0))
				stagingEnv = string(session.Out.Contents())

				Eventually(cf.Cf("set-staging-environment-variable-group", `{"STAGING_TEST_VAR":"staging_env_value"}`)).Should(Exit(0))
			})

			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Standalone, "--no-start", "-b", GIT_NULL_BUILDPACK), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
		})

		AfterEach(func() {
			cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
				Eventually(cf.Cf("set-staging-environment-variable-group", stagingEnv)).Should(Exit(0))
			})
		})

		It("Applies environment variables while staging apps", func() {
			session := cf.Cf("start", appName)
			Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))

			Expect(session).To(Say("STAGING_TEST_VAR=staging_env_value"))
		})
	})
})
