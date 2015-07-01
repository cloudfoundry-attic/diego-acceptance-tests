// +build !noInternet,!noDocker

package diego

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Docker Application Lifecycle", func() {
	var appName string
	var createDockerAppPayload string

	domain := helpers.LoadConfig().AppsDomain

	BeforeEach(func() {
		appName = generator.RandomName()

		createDockerAppPayload = `{
			"name": "%s",
			"memory": 512,
			"instances": 1,
			"disk_quota": 1024,
			"space_guid": "%s",
			"docker_image": "cloudfoundry/diego-docker-app:latest",
			"command": "/myapp/dockerapp",
			"diego": true
		}`
	})

	JustBeforeEach(func() {
		spaceGuid := guidForSpaceName(context.RegularUserContext().Space)

		payload := fmt.Sprintf(createDockerAppPayload, appName, spaceGuid)
		Eventually(cf.Cf("curl", "/v2/apps", "-X", "POST", "-d", payload)).Should(Exit(0))
		Eventually(cf.Cf("create-route", context.RegularUserContext().Space, domain, "-n", appName)).Should(Exit(0))
		Eventually(cf.Cf("map-route", appName, domain, "-n", appName)).Should(Exit(0))
		Eventually(cf.Cf("start", appName), DOCKER_IMAGE_DOWNLOAD_DEFAULT_TIMEOUT).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(appName)).Should(Equal("0"))
	})

	AfterEach(func() {
		Eventually(cf.Cf("logs", appName, "--recent")).Should(Exit())
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Describe("running the app", func() {
		It("merges the garden and docker environment variables", func() {
			env_json := helpers.CurlApp(appName, "/env")
			var env_vars map[string]string
			json.Unmarshal([]byte(env_json), &env_vars)

			// garden set values should win
			Expect(env_vars).To(HaveKey("HOME"))
			Expect(env_vars).NotTo(HaveKeyWithValue("HOME", "/home/some_docker_user"))
			Expect(env_vars).To(HaveKey("VCAP_APPLICATION"))
			Expect(env_vars).NotTo(HaveKeyWithValue("VCAP_APPLICATION", "{}"))
			// docker image values should remain
			Expect(env_vars).To(HaveKeyWithValue("SOME_VAR", "some_docker_value"))
			Expect(env_vars).To(HaveKeyWithValue("BAD_QUOTE", "'"))
			Expect(env_vars).To(HaveKeyWithValue("BAD_SHELL", "$1"))
		})
	})

	Describe("running a docker app without a start command", func() {
		BeforeEach(func() {
			createDockerAppPayload = `{
				"name": "%s",
				"memory": 512,
				"instances": 1,
				"disk_quota": 1024,
				"space_guid": "%s",
				"docker_image": "cloudfoundry/diego-docker-app:latest",
				"diego": true
			}`
		})

		It("locates and invokes the start command", func() {
			Eventually(helpers.CurlingAppRoot(appName)).Should(Equal("0"))
		})
	})

	Describe("stopping an app", func() {
		It("makes the app unreachable while it is stopped", func() {
			Eventually(helpers.CurlingAppRoot(appName)).Should(Equal("0"))

			Eventually(cf.Cf("stop", appName)).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("404"))

			Eventually(cf.Cf("start", appName), DOCKER_IMAGE_DOWNLOAD_DEFAULT_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(appName)).Should(Equal("0"))
		})
	})
})
