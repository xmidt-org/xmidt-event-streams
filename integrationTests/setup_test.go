// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package integrationTests

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/xmidt-org/idock"

	"eventstream/internal/install"
)

func TestMain(m *testing.M) {
	fmt.Printf("in test main")

	if testFlags() == "" {
		fmt.Println("Skipping integration tests.")
		os.Exit(0)
	}

	infra := idock.New(
		idock.DockerComposeFile("docker.yml"),
		// NOTE - UNABLE TO RUN LOCALLY UNLESS BELOW LINE IS COMMENTED, docker startup times out, same on github action
		//idock.RequireDockerTCPPorts(4567, 6100, 6101, 6102, 6103, 6600, 6601, 6602, 6603, 4566, 6379),
		//idock.AfterDocker(waitForAnemoi),
		idock.Program(func() { _ = install.EventStream([]string{"--dev", "-f", "eventstream.yml"}, true) }),
		//idock.RequireProgramTCPPorts(18111,18112,18113),
	)

	err := infra.Start()
	if err != nil {
		panic(err)
	}

	// docker compose should be creating the stream, but it currently is not working
	scriptPath := "./createStream.sh"

	// Create a new *Cmd instance for the shell script
	// The first argument is the interpreter (e.g., "bash", "/bin/sh")
	// The subsequent arguments are passed to the interpreter, including the script itself
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	// temp wait to allow services to fully start
	time.Sleep(5 * time.Second)

	returnCode := m.Run()

	infra.Stop()

	if returnCode != 0 {
		os.Exit(returnCode)
	}
}

// testFlags returns "" to run no tests, "all" to run all tests, "broken" to
// run only broken tests, and "working" to run only working tests.
func testFlags() string {
	env := os.Getenv("INTEGRATION_TESTS_RUN")
	env = strings.ToLower(env)
	env = strings.TrimSpace(env)

	switch env {
	case "all":
	case "broken":
	case "":
	default:
		return "working"
	}

	return env
}
