package ssh

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

const CF_PUSH_TIMEOUT = 4 * time.Minute

var (
	context  helpers.SuiteContext
	scpPath  string
	sftpPath string
)

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	config := helpers.LoadConfig()
	context = helpers.NewContext(config)
	environment := helpers.NewEnvironment(context)

	type sshPaths struct {
		SCP  string `json:"scp"`
		SFTP string `json:"sftp"`
	}

	var _ = SynchronizedBeforeSuite(func() []byte {
		scp, err := exec.LookPath("scp")
		Expect(err).NotTo(HaveOccurred())

		sftp, err := exec.LookPath("sftp")
		Expect(err).NotTo(HaveOccurred())

		paths, err := json.Marshal(sshPaths{
			SCP:  scp,
			SFTP: sftp,
		})
		Expect(err).NotTo(HaveOccurred())

		return []byte(paths)
	}, func(encodedSSHPaths []byte) {
		var sshPaths sshPaths
		err := json.Unmarshal(encodedSSHPaths, &sshPaths)
		Expect(err).NotTo(HaveOccurred())

		scpPath = sshPaths.SCP
		sftpPath = sshPaths.SFTP

		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	componentName := "SSH"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func guidForAppName(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Expect(cfApp.Wait()).To(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}
