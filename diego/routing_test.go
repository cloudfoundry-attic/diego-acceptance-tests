package diego

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"
)

var _ = Describe("Adding and removing routes", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Standalone, "--no-start", "-b", DIEGO_NULL_BUILDPACK), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("set-env", appName, "CF_DIEGO_BETA", "true")).Should(Exit(0))
		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hi, I'm Bash!"))
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	It("should be able to add and remove routes", func() {
		secondHost := generator.RandomName()

		By("adding a route")
		Eventually(cf.Cf("map-route", appName, helpers.LoadConfig().AppsDomain, "-n", secondHost)).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hi, I'm Bash!"))
		Eventually(helpers.CurlingAppRoot(secondHost)).Should(ContainSubstring("Hi, I'm Bash!"))

		By("removing a route")
		Eventually(cf.Cf("unmap-route", appName, helpers.LoadConfig().AppsDomain, "-n", secondHost)).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(secondHost)).Should(ContainSubstring("404"))
		Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hi, I'm Bash!"))
	})
})
