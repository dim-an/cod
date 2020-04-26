// Copyright 2020 Dmitry Ermolov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parse_doc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseDocker(t *testing.T) {
	parseCompletions := func(args []string, text string) (res []string) {
		ctx, err := makeParseContext(args, text)
		require.NoError(t, err)

		parseResult, err := makeDefaultParser().Parse(ctx)
		require.NoError(t, err)
		for idx := range parseResult.completions {
			res = append(res, parseResult.completions[idx].Flag)
		}
		return
	}
	require.Equal(t, []string{
		"--config",
		"-D",
		"--debug",
		"-H",
		"--host",
		"-l",
		"--log-level",
		"--tls",
		"--tlscacert",
		"--tlscert",
		"--tlskey",
		"--tlsverify",
		"-v",
		"--version",
		"checkpoint",
		"config",
		"container",
		"image",
		"network",
		"node",
		"plugin",
		"secret",
		"service",
		"swarm",
		"system",
		"trust",
		"volume",
		"attach",
		"build",
		"commit",
		"cp",
		"create",
		"deploy",
		"diff",
		"events",
		"exec",
		"export",
		"history",
		"images",
		"import",
		"info",
		"inspect",
		"kill",
		"load",
		"login",
		"logout",
		"logs",
		"pause",
		"port",
		"ps",
		"pull",
		"push",
		"rename",
		"restart",
		"rm",
		"rmi",
		"run",
		"save",
		"search",
		"start",
		"stats",
		"stop",
		"tag",
		"top",
		"unpause",
		"update",
		"version",
		"wait",
	}, parseCompletions([]string{"/usr/bin/docker", "--help"}, dockerHelp))
}

var dockerHelp = `Usage:	docker COMMAND

A self-sufficient runtime for containers

Options:
      --config string      Location of client config files (default "/home/ermolovd/.docker")
  -D, --debug              Enable debug mode
  -H, --host list          Daemon socket(s) to connect to
  -l, --log-level string   Set the logging level ("debug"|"info"|"warn"|"error"|"fatal") (default "info")
      --tls                Use TLS; implied by --tlsverify
      --tlscacert string   Trust certs signed only by this CA (default "/home/ermolovd/.docker/ca.pem")
      --tlscert string     Path to TLS certificate file (default "/home/ermolovd/.docker/cert.pem")
      --tlskey string      Path to TLS key file (default "/home/ermolovd/.docker/key.pem")
      --tlsverify          Use TLS and verify the remote
  -v, --version            Print version information and quit

Management Commands:
  checkpoint  Manage checkpoints
  config      Manage Docker configs
  container   Manage containers
  image       Manage images
  network     Manage networks
  node        Manage Swarm nodes
  plugin      Manage plugins
  secret      Manage Docker secrets
  service     Manage services
  swarm       Manage Swarm
  system      Manage Docker
  trust       Manage trust on Docker images
  volume      Manage volumes

Commands:
  attach      Attach local standard input, output, and error streams to a running container
  build       Build an image from a Dockerfile
  commit      Create a new image from a container's changes
  cp          Copy files/folders between a container and the local filesystem
  create      Create a new container
  deploy      Deploy a new stack or update an existing stack
  diff        Inspect changes to files or directories on a container's filesystem
  events      Get real time events from the server
  exec        Run a command in a running container
  export      Export a container's filesystem as a tar archive
  history     Show the history of an image
  images      List images
  import      Import the contents from a tarball to create a filesystem image
  info        Display system-wide information
  inspect     Return low-level information on Docker objects
  kill        Kill one or more running containers
  load        Load an image from a tar archive or STDIN
  login       Log in to a Docker registry
  logout      Log out from a Docker registry
  logs        Fetch the logs of a container
  pause       Pause all processes within one or more containers
  port        List port mappings or a specific mapping for the container
  ps          List containers
  pull        Pull an image or a repository from a registry
  push        Push an image or a repository to a registry
  rename      Rename a container
  restart     Restart one or more containers
  rm          Remove one or more containers
  rmi         Remove one or more images
  run         Run a command in a new container
  save        Save one or more images to a tar archive (streamed to STDOUT by default)
  search      Search the Docker Hub for images
  start       Start one or more stopped containers
  stats       Display a live stream of container(s) resource usage statistics
  stop        Stop one or more running containers
  tag         Create a tag TARGET_IMAGE that refers to SOURCE_IMAGE
  top         Display the running processes of a container
  unpause     Unpause all processes within one or more containers
  update      Update configuration of one or more containers
  version     Show the Docker version information
  wait        Block until one or more containers stop, then print their exit codes

Run 'docker COMMAND --help' for more information on a command.
`

func TestParseUsageSubCommand(t *testing.T) {
	parseUsage := func(args []string, text string) []string {
		prepared, err := makePreparedText(text)
		require.NoError(t, err)
		return parseUsageSubCommand(args, prepared)
	}

	require.Nil(t, parseUsage([]string{"/usr/bin/docker", "--help"}, dockerHelp))

	{
		usage := "Usage: foo make [OPTION]... [TARGET]...\n" +
			"Build and run tests\n"
		require.Equal(t, []string{"make"}, parseUsage([]string{"/bin/foo", "make", "--help"}, usage))

		require.Equal(t, []string(nil), parseUsage([]string{"/bin/foo", "bake", "--help"}, usage))
	}
	{
		usage := "Usage:\n" +
			"  foo make [OPTION]... [TARGET]...\n" +
			"  foo bake [OPTION]... [TARGET]...\n" +
			"Build and run tests\n"
		require.Equal(t, []string(nil), parseUsage([]string{"/bin/foo", "make", "--help"}, usage))
		require.Equal(t, []string(nil), parseUsage([]string{"/bin/foo", "bake", "--help"}, usage))
	}
}
