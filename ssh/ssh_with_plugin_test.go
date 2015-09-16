package ssh

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("SSH With Plugin", func() {
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
		Eventually(cf.Cf("curl", "/v2/apps/"+guid, "-X", "PUT", "-d", `{"diego": true}`)).Should(Exit(0))
		Eventually(cf.Cf("enable-ssh", appName)).Should(Exit(0))
	}

	BeforeEach(func() {
		enableSSH(appName)

		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(func() string {
			return helpers.CurlApp(appName, "/env/INSTANCE_INDEX")
		}).Should(Equal("1"))
	})

	It("can execute a remote command in the container", func() {
		envCmd := cf.Cf("ssh", "-i", "1", appName, "/usr/bin/env")
		Expect(envCmd.Wait()).To(Exit(0))

		output := string(envCmd.Buffer().Contents())

		Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
		Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=1"))

		Eventually(cf.Cf("logs", appName, "--recent")).Should(Say("Successful remote access"))
		Eventually(cf.Cf("events", appName)).Should(Say("audit.app.ssh-authorized"))
	})

	It("runs an interactive session when no command is provided", func() {
		envCmd := exec.Command("cf", "ssh", "-i1", appName)

		stdin, err := envCmd.StdinPipe()
		Expect(err).NotTo(HaveOccurred())

		stdout, err := envCmd.StdoutPipe()
		Expect(err).NotTo(HaveOccurred())

		err = envCmd.Start()
		Expect(err).NotTo(HaveOccurred())

		_, err = stdin.Write([]byte("/usr/bin/env\n"))
		Expect(err).NotTo(HaveOccurred())

		err = stdin.Close()
		Expect(err).NotTo(HaveOccurred())

		output, err := ioutil.ReadAll(stdout)
		Expect(err).NotTo(HaveOccurred())

		err = envCmd.Wait()
		Expect(err).NotTo(HaveOccurred())

		Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
		Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=1"))

		Eventually(cf.Cf("logs", appName, "--recent")).Should(Say("Successful remote access"))
		Eventually(cf.Cf("events", appName)).Should(Say("audit.app.ssh-authorized"))
	})

	It("allows local port forwarding", func() {
		listenCmd := exec.Command("cf", "ssh", "-i1", "-L127.0.0.1:37001:localhost:8080", appName)

		stdin, err := listenCmd.StdinPipe()
		Expect(err).NotTo(HaveOccurred())

		err = listenCmd.Start()
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			stdout := &bytes.Buffer{}
			curlCmd := exec.Command("curl", "http://127.0.0.1:37001/")
			curlCmd.Stdout = stdout
			curlCmd.Run()
			return stdout.String()
		}).Should(ContainSubstring("Hi, I'm Dora"))

		err = stdin.Close()
		Expect(err).NotTo(HaveOccurred())

		err = listenCmd.Wait()
		Expect(err).NotTo(HaveOccurred())
	})
})
