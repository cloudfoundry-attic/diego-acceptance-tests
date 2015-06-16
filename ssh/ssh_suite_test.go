package ssh

import (
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

const CF_PUSH_TIMEOUT = 4 * time.Minute

var (
	context helpers.SuiteContext
	scpPath string
)

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	config := helpers.LoadConfig()
	context = helpers.NewContext(config)
	environment := helpers.NewEnvironment(context)

	var _ = SynchronizedBeforeSuite(func() []byte {
		path, err := exec.LookPath("scp")
		Expect(err).NotTo(HaveOccurred())
		return []byte(path)
	}, func(encodedSCPPath []byte) {
		scpPath = string(encodedSCPPath)
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
