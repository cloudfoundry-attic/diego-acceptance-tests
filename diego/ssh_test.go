package diego

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"golang.org/x/crypto/ssh"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"
)

var _ = Describe("SSH", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
		Eventually(cf.Cf("push", appName, "-p", assets.NewAssets().Dora, "--no-start", "-b", "ruby_buildpack", "-i", "2"), CF_PUSH_TIMEOUT).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("logs", appName, "--recent")).Should(Exit())
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	enableSSH := func(appName string) {
		guid := guidForAppName(appName)
		Eventually(cf.Cf("curl", "/v2/apps/"+guid, "-X", "PUT", "-d", `{"diego": true,"enable_ssh": true}`)).Should(Exit(0))
	}

	oauthToken := func() string {
		oauthToken := cf.Cf("oauth-token")
		Expect(oauthToken.Wait()).To(Exit(0))

		tokenStr := string(oauthToken.Buffer().Contents())
		index := strings.Index(tokenStr, "bearer")

		token := strings.TrimSpace(string(tokenStr[index:]))
		Expect(token).To(ContainSubstring("bearer "))
		return token
	}

	type infoResponse struct {
		AppSSHEndpoint string `json:"app_ssh_endpoint"`
	}

	sshProxyAddress := func() string {
		infoCommand := cf.Cf("curl", "/v2/info")
		Expect(infoCommand.Wait()).To(Exit(0))

		var response infoResponse
		err := json.Unmarshal(infoCommand.Buffer().Contents(), &response)
		Expect(err).NotTo(HaveOccurred())

		return response.AppSSHEndpoint
	}

	Describe("An App running on Diego with enable_ssh on", func() {
		BeforeEach(func() {
			enableSSH(appName)

			Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(func() string {
				return helpers.CurlApp(appName, "/env/INSTANCE_INDEX")
			}).Should(Equal("1"))
		})

		It("can be sshed to and records its success", func() {
			password := oauthToken()

			clientConfig := &ssh.ClientConfig{
				User: fmt.Sprintf("cf:%s/%d", guidForAppName(appName), 1),
				Auth: []ssh.AuthMethod{ssh.Password(password)},
			}

			client, err := ssh.Dial("tcp", sshProxyAddress(), clientConfig)
			Expect(err).NotTo(HaveOccurred())

			session, err := client.NewSession()
			Expect(err).NotTo(HaveOccurred())

			output, err := session.Output("/usr/bin/env")
			Expect(err).NotTo(HaveOccurred())

			Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=1"))

			Eventually(cf.Cf("logs", appName, "--recent")).Should(Say("Successful remote access"))
			Eventually(cf.Cf("events", appName)).Should(Say("audit.app.ssh-authorized"))
		})

		It("records failed ssh attempts", func() {
			clientConfig := &ssh.ClientConfig{
				User: fmt.Sprintf("cf:%s/%d", guidForAppName(appName), 0),
				Auth: []ssh.AuthMethod{ssh.Password("bogus password")},
			}

			_, err := ssh.Dial("tcp", sshProxyAddress(), clientConfig)
			Expect(err).To(HaveOccurred())

			Eventually(cf.Cf("events", appName)).Should(Say("audit.app.ssh-unauthorized"))
		})
	})
})
