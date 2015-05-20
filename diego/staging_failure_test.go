package diego

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("When staging fails", func() {
	var appName string

	Context("due to a custom buildpack that cannot be downloaded", func() {
		BeforeEach(func() {
			appName = generator.RandomName()

			//Diego needs a custom buildpack until the ruby buildpack lands
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Dora, "--no-start", "-b=http://example.com/so-not-a-thing/adlfijaskldjlkjaslbnalwieulfjkjsvas.zip"), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
		})

		AfterEach(func() {
			Eventually(cf.Cf("logs", appName, "--recent")).Should(Exit())
			Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
		})

		It("informs the user in the CLI output and the logs", func() {
			start := cf.Cf("start", appName)
			Eventually(start, CF_PUSH_TIMEOUT).Should(Exit(1))
			Expect(start.Out).To(gbytes.Say("StagingError"))

			Eventually(func() *Session {
				logs := cf.Cf("logs", appName, "--recent")
				Expect(logs.Wait()).To(Exit(0))
				return logs
			}).Should(gbytes.Say("Failed to download buildpack"))
		})
	})

	Context("due to insufficient resources", func() {
		BeforeEach(func() {
			appName = generator.RandomName()

			context.SetRunawayQuota()

			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Dora, "--no-start", "-b=ruby_buildpack", "-m", helpers.RUNAWAY_QUOTA_MEM_LIMIT), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
		})

		AfterEach(func() {
			Eventually(cf.Cf("logs", appName, "--recent")).Should(Exit())
			Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
		})

		It("informs the user in the CLI output and the logs", func() {
			start := cf.Cf("start", appName)
			Eventually(start, CF_PUSH_TIMEOUT).Should(Exit(1))
			Expect(start.Out).To(gbytes.Say("InsufficientResources"))

			Eventually(func() *Session {
				logs := cf.Cf("logs", appName, "--recent")
				Expect(logs.Wait()).To(Exit(0))
				return logs
			}).Should(gbytes.Say("Failed to stage application: insufficient resources"))

			app := cf.Cf("app", appName)
			Eventually(app).Should(Exit(0))
			Expect(app.Out).To(gbytes.Say("requested state: stopped"))
		})
	})
})
