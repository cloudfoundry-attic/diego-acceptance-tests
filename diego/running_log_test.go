package diego

import (
	"fmt"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"
)

var _ = Describe("Logs from apps hosted by Diego", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()

		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Dora, "-b", "ruby_buildpack", "--no-start"), CF_PUSH_TIMEOUT).Should(Exit(0))
		enableDiego(appName)
		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("logs", appName, "--recent")).Should(Exit())
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Context("when the app is running", func() {
		BeforeEach(func() {
			Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hi, I'm Dora!"))
		})

		It("captures logs with the correct tag", func() {
			var message string
			var logs *Session

			By("logging health checks")
			logs = cf.Cf("logs", appName, "--recent")
			Eventually(logs).Should(Exit(0))
			Expect(logs.Out).To(Say("\\[HEALTH/0\\]\\s+OUT healthcheck passed"))
			Expect(logs.Out).To(Say("\\[HEALTH/0\\]\\s+OUT Exit status 0"))

			By("logging application stdout")
			message = "A message from stdout"
			Eventually(helpers.CurlApp(appName, fmt.Sprintf("/echo/stdout/%s", url.QueryEscape(message)))).Should(ContainSubstring(message))

			logs = cf.Cf("logs", appName, "--recent")
			Eventually(logs).Should(Exit(0))
			Expect(logs.Out).To(Say(fmt.Sprintf("\\[APP/0\\]\\s+OUT %s", message)))

			By("logging application stderr")
			message = "A message from stderr"
			Eventually(helpers.CurlApp(appName, fmt.Sprintf("/echo/stderr/%s", url.QueryEscape(message)))).Should(ContainSubstring(message))

			logs = cf.Cf("logs", appName, "--recent")
			Eventually(logs).Should(Exit(0))
			Expect(logs.Out).To(Say(fmt.Sprintf("\\[APP/0\\]\\s+ERR %s", message)))
		})
	})
})
