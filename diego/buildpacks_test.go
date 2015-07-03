// +build !noInternet

package diego

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"
)

var _ = Describe("Buildpacks", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		Eventually(cf.Cf("logs", appName, "--recent")).Should(Exit())
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	It("stages with a zip buildpack and runs on diego", func() {
		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Standalone, "--no-start", "-b", ZIP_NULL_BUILDPACK), CF_PUSH_TIMEOUT).Should(Exit(0))
		enableDiego(appName)
		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hi, I'm Bash!"))
	})

	It("stages with a git buildpack and runs on diego", func() {
		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Standalone, "--no-start", "-b", GIT_NULL_BUILDPACK), CF_PUSH_TIMEOUT).Should(Exit(0))
		enableDiego(appName)
		session := cf.Cf("start", appName)
		Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))
		Expect(session).To(Say("LANG=en_US.UTF-8"))
		Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("Hi, I'm Bash!"))
	})

	It("advertises the stack as CF_STACK when staging on diego", func() {
		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Standalone, "--no-start", "-b", GIT_NULL_BUILDPACK, "-s", "cflinuxfs2"), CF_PUSH_TIMEOUT).Should(Exit(0))
		enableDiego(appName)
		session := cf.Cf("start", appName)
		Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))
		Expect(session).To(Say("CF_STACK=cflinuxfs2"))
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
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Java, "--no-start", "-b", "java_buildpack", "-m", "512M"), CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(cf.Cf("set-env", appName, "JAVA_OPTS", "-Djava.security.egd=file:///dev/urandom"), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
			session := cf.Cf("start", appName)
			Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(appName)).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))
		})
	})

	Describe("golang", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Golang, "--no-start", "-b", "golang_buildpack"), CF_PUSH_TIMEOUT).Should(Exit(0))
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
			Eventually(helpers.CurlAppRoot(appName)).Should(ContainSubstring("Hello from a node app!"))
		})
	})

	Describe("staticfile", func() {
		It("makes the app reachable via its bound route", func() {
			Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Staticfile, "--no-start", "-b", "static_buildpack"), CF_PUSH_TIMEOUT).Should(Exit(0))
			enableDiego(appName)
			session := cf.Cf("start", appName)
			Eventually(session, CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(appName)).Should(ContainSubstring("Hello from a staticfile"))
		})
	})
})
