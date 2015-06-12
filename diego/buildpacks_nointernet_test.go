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

var _ = Describe("Buildpacks without internet", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		Eventually(cf.Cf("logs", appName, "--recent")).Should(Exit())
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	It("stages with auto detect and runs on diego", func() {
		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().HelloWorld, "--no-start"), CF_PUSH_TIMEOUT).Should(Exit(0))
		enableDiego(appName)
		Eventually(cf.Cf("start", appName), 2*CF_PUSH_TIMEOUT).Should(Exit(0)) // Double timeout to allow for all buildpacks.
		Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hello, world!"))
	})

	It("stages with a named buildpack and runs on diego", func() {
		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().HelloWorld, "--no-start", "-b", "ruby_buildpack"), CF_PUSH_TIMEOUT).Should(Exit(0))
		enableDiego(appName)
		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hello, world!"))
	})
})
