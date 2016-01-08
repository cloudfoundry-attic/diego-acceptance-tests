package diego

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Getting instance information", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()

		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Standalone, "--no-start", "-b", BINARY_BUILDPACK, "-c", "./bin/start"), CF_PUSH_TIMEOUT).Should(Exit(0))
		enableDiego(appName)
		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("logs", appName, "--recent")).Should(Exit())
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Context("scaling memory", func() {
		BeforeEach(func() {
			context.SetRunawayQuota()
			scale := cf.Cf("scale", appName, "-m", helpers.RUNAWAY_QUOTA_MEM_LIMIT, "-f")
			Eventually(scale).Should(Say("down"))
			scale.Kill()
		})

		It("fails with insufficient resources", func() {
			app := cf.Cf("app", appName)
			Eventually(app).Should(Exit(0))
			Expect(app.Out).To(Say("insufficient resources"))
		})
	})
})
