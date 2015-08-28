package diego

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"
)

var _ = Describe("Default buildpacks", func() {
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

	Describe("nodeJS", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Node, "--no-start", "-b", "nodejs_buildpack"), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
			session := cf.Cf("start", appName)
			Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(appName)).Should(ContainSubstring("Hello from a node app!"))
		})
	})

	Describe("java", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Java, "--no-start", "-m", "512M"), CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(cf.Cf("set-env", appName, "JAVA_OPTS", "-Djava.security.egd=file:///dev/urandom"), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
			session := cf.Cf("start", appName)
			Eventually(session, 2*CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(appName)).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))
		})
	})

	Describe("golang", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Golang, "--no-start", "-b", "go_buildpack"), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
			session := cf.Cf("start", appName)
			Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(appName)).Should(ContainSubstring("go, world"))
		})
	})

	Describe("python", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Python, "--no-start", "-b", "python_buildpack"), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
			session := cf.Cf("start", appName)
			Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(appName)).Should(ContainSubstring("python, world"))
		})
	})

	Describe("php", func() {
		var phpPushTimeout = CF_PUSH_TIMEOUT + 6*time.Minute

		It("makes the app reachable via its bound route", func() {
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Php, "--no-start", "-b", "php_buildpack"), phpPushTimeout).Should(Exit(0))
			enableDiego(appName)
			session := cf.Cf("start", appName)
			Eventually(session, phpPushTimeout).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(appName)).Should(ContainSubstring("Hello from php"))
		})
	})

	Describe("staticfile", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Staticfile, "--no-start", "-b", "staticfile_buildpack"), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
			session := cf.Cf("start", appName)
			Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(appName)).Should(ContainSubstring("Hello from a staticfile"))
		})
	})
})
