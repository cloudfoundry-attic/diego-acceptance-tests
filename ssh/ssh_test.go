package ssh

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kr/pty"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"golang.org/x/crypto/ssh"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/diego-acceptance-tests/helpers/assets"
)

const timeFormat = "2006-01-02 15:04:05.00 (MST)"

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

	saySCPRun := func(cmd *exec.Cmd) {
		startColor := ""
		endColor := ""
		if !config.DefaultReporterConfig.NoColor {
			startColor = "\x1b[32m"
			endColor = "\x1b[0m"
		}
		fmt.Fprintf(GinkgoWriter, "\n%s[%s]> %s %s\n", startColor, time.Now().UTC().Format(timeFormat), strings.Join(cmd.Args, " "), endColor)
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

		Context("with the ssh plugin", func() {
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

		Context("without the ssh plugin", func() {
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

			Context("scp", func() {
				var (
					sourceDir, targetDir             string
					generatedFile, generatedFileName string
					generatedFileInfo                os.FileInfo
					err                              error
				)

				BeforeEach(func() {
					Expect(err).NotTo(HaveOccurred())

					sourceDir, err = ioutil.TempDir("", "scp-source")
					Expect(err).NotTo(HaveOccurred())

					fileContents := make([]byte, 1024)
					b, err := rand.Read(fileContents)
					Expect(err).NotTo(HaveOccurred())
					Expect(b).To(Equal(len(fileContents)))

					generatedFileName = "binary.dat"
					generatedFile = filepath.Join(sourceDir, generatedFileName)

					err = ioutil.WriteFile(generatedFile, fileContents, 0664)
					Expect(err).NotTo(HaveOccurred())

					generatedFileInfo, err = os.Stat(generatedFile)
					Expect(err).NotTo(HaveOccurred())

					targetDir, err = ioutil.TempDir("", "scp-target")
					Expect(err).NotTo(HaveOccurred())

					Eventually(func() string {
						return helpers.CurlApp(appName, "/env/INSTANCE_INDEX")
					}).Should(Equal("0"))
				})

				runScp := func(src, dest string) {
					_, sshPort, err := net.SplitHostPort(sshProxyAddress())
					Expect(err).NotTo(HaveOccurred())

					ptyMaster, ptySlave, err := pty.Open()
					Expect(err).NotTo(HaveOccurred())
					defer ptyMaster.Close()

					password := oauthToken() + "\n"

					cmd := exec.Command(scpPath,
						"-r",
						"-P", sshPort,
						fmt.Sprintf("-oUser=cf:%s/0", guidForAppName(appName)),
						"-oUserKnownHostsFile=/dev/null",
						"-oStrictHostKeyChecking=no",
						src,
						dest,
					)

					cmd.Stdin = ptySlave
					cmd.Stdout = ptySlave
					cmd.Stderr = ptySlave

					cmd.SysProcAttr = &syscall.SysProcAttr{
						Setctty: true,
						Setsid:  true,
					}

					saySCPRun(cmd)
					session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					// Close our open reference to ptySlave so that PTY Master recieves EOF
					ptySlave.Close()

					b := make([]byte, 1)
					buf := []byte{}
					passwordPrompt := []byte("password: ")
					for {
						n, err := ptyMaster.Read(b)
						Expect(n).To(Equal(1))
						Expect(err).NotTo(HaveOccurred())
						buf = append(buf, b[0])
						if bytes.HasSuffix(buf, passwordPrompt) {
							break
						}
					}

					n, err := ptyMaster.Write([]byte(password))
					Expect(err).NotTo(HaveOccurred())
					Expect(n).To(Equal(len(password)))

					done := make(chan struct{})
					go func() {
						io.Copy(GinkgoWriter, ptyMaster)
						close(done)
					}()

					Eventually(done).Should(BeClosed())
					Eventually(session).Should(Exit(0))
				}

				It("can send and receive files over scp", func() {
					sshHost, _, err := net.SplitHostPort(sshProxyAddress())
					Expect(err).NotTo(HaveOccurred())

					runScp(sourceDir, fmt.Sprintf("%s:/home/vcap", sshHost))
					runScp(fmt.Sprintf("%s:/home/vcap/%s", sshHost, filepath.Base(sourceDir)), targetDir)

					compareDir(sourceDir, filepath.Join(targetDir, filepath.Base(sourceDir)))
				})
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
})

func compareDir(actualDir, expectedDir string) {
	actualDirInfo, err := os.Stat(actualDir)
	Expect(err).NotTo(HaveOccurred())

	expectedDirInfo, err := os.Stat(expectedDir)
	Expect(err).NotTo(HaveOccurred())

	Expect(actualDirInfo.Mode()).To(Equal(expectedDirInfo.Mode()))

	actualFiles, err := ioutil.ReadDir(actualDir)
	Expect(err).NotTo(HaveOccurred())

	expectedFiles, err := ioutil.ReadDir(actualDir)
	Expect(err).NotTo(HaveOccurred())

	Expect(len(actualFiles)).To(Equal(len(expectedFiles)))
	for i, actualFile := range actualFiles {
		expectedFile := expectedFiles[i]
		if actualFile.IsDir() {
			compareDir(filepath.Join(actualDir, actualFile.Name()), filepath.Join(expectedDir, expectedFile.Name()))
		} else {
			compareFile(filepath.Join(actualDir, actualFile.Name()), filepath.Join(expectedDir, expectedFile.Name()))
		}
	}
}

func compareFile(actualFile, expectedFile string) {
	actualFileInfo, err := os.Stat(actualFile)
	Expect(err).NotTo(HaveOccurred())

	expectedFileInfo, err := os.Stat(expectedFile)
	Expect(err).NotTo(HaveOccurred())

	Expect(actualFileInfo.Mode()).To(Equal(expectedFileInfo.Mode()))
	Expect(actualFileInfo.Size()).To(Equal(expectedFileInfo.Size()))

	actualContents, err := ioutil.ReadFile(actualFile)
	Expect(err).NotTo(HaveOccurred())

	expectedContents, err := ioutil.ReadFile(expectedFile)
	Expect(err).NotTo(HaveOccurred())

	Expect(actualContents).To(Equal(expectedContents))
}

func guidForAppName(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Expect(cfApp.Wait()).To(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}
