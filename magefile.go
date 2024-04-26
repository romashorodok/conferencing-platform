//go:build mage
// +build mage

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	DOCKER_DEFAULT_CONTEXT       = "default"
	DOCKER_BUILDX_CACHE_DIR_NAME = ".dockercache"
)

func fmtPanic(format string, val ...any) {
	panic(fmt.Sprintf(format, val...))
}

type DockerServiceBuild struct {
	Target string            `json:"target"`
	Args   map[string]string `json:"args"`
}

type DockerComposeService struct {
	Name  string
	Build DockerServiceBuild `json:"build"`
}

type DockerComposeFile struct {
	Name        string `json:"name"`
	RawServices any    `json:"services"`
	Services    []DockerComposeService
}

func parseDockerComposeFile(src []byte) DockerComposeFile {
	var dockerComposeFile DockerComposeFile

	if err := json.NewDecoder(bytes.NewReader(src)).Decode(&dockerComposeFile); err != nil {
		fmtPanic("Unable decode. Err: %s", err)
	}

	rawServices, err := json.Marshal(&dockerComposeFile.RawServices)
	if err != nil {
		fmtPanic("Unable conver raw services. Err: %s", err)
	}

	var servicesMap map[string]any
	if err := json.Unmarshal(rawServices, &servicesMap); err != nil {
		fmtPanic("Unable decode services. Err: %s", err)
	}

	services := make([]DockerComposeService, 0)

	for key, val := range servicesMap {
		rawServiceEntry, err := json.Marshal(val)
		if err != nil {
			fmtPanic("Unable get service entry. Err: %s", err)
		}

		var serviceEntry DockerComposeService
		if err := json.Unmarshal(rawServiceEntry, &serviceEntry); err != nil {
			fmtPanic("Unable decode service entry. Err: %s", err)
		}

		serviceEntry.Name = key
		services = append(services, serviceEntry)
	}

	dockerComposeFile.Services = services
	return dockerComposeFile
}

func dockerContextUse(name string) error {
	cmd := exec.Command("docker", "context", "use", name)
	stderr, _ := cmd.StderrPipe()

	errBus := make(chan []byte)

	go func() {
		result, err := io.ReadAll(stderr)
		if err != nil {
			fmtPanic("Unable exec docker context use %s Err: %s", name, err)
		}
		errBus <- result
		close(errBus)
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s", <-errBus)
	}

	fmt.Printf("[Docker] Use context %s.\n", name)

	return nil
}

const DOCKER_ERR_BUILDX_CREATE_EXISTING_INSTANCE = "ERROR: existing instance"

var ErrExistingBuilder = errors.New("Already existing docker buildkit builder")

func buildxCreateBuilder(builderName, contextName string) error {
	cmd := exec.Command("docker", "buildx", "create",
		"--name", builderName,
		"--driver=docker-container",
		contextName,
	)
	stderr, _ := cmd.StderrPipe()
	errBus := make(chan []byte)

	go func() {
		result, err := io.ReadAll(stderr)
		if err != nil {
			fmtPanic("Unable exec docker buildx create of builder: %s for context: %s %s Err: %s", builderName, contextName, err)
		}
		errBus <- result
		close(errBus)
	}()

	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("%s", <-errBus)
		if strings.HasPrefix(err.Error(), DOCKER_ERR_BUILDX_CREATE_EXISTING_INSTANCE) {
			fmt.Println("[Docker]", ErrExistingBuilder.Error(), "name:", builderName)
		} else {
			return err
		}
	}

	return nil
}

func buildxBuildTarget(builder, cache, target, label string, args map[string]string) error {
	var buildArgs []string
	for argName, argVal := range args {
		buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", argName, argVal))
	}

	command := []string{
		"buildx", "build",
		fmt.Sprintf("--builder=%s", builder),
		fmt.Sprintf("--cache-to=type=local,dest=%s", cache),
		fmt.Sprintf("--cache-from=type=local,src=%s", cache),
		"--target", target,
		"--label", label,
	}
	command = append(command, buildArgs...)
	command = append(command, ".")

	cmd := exec.Command("docker", command...)

	stderr, _ := cmd.StderrPipe()
	go io.Copy(os.Stdout, stderr)

	fmt.Printf("[Docker] Exec image %s build with:\n %s \n", label, cmd.String())
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func buildKitBuild(composeFile DockerComposeFile) {
	if err := dockerContextUse(DOCKER_DEFAULT_CONTEXT); err != nil {
		fmtPanic("Unable select docker context. Err: %s", err)
	}
	defer dockerContextUse(DOCKER_DEFAULT_CONTEXT)

	BUILDER_NAME := "container"
	if err := buildxCreateBuilder(BUILDER_NAME, DOCKER_DEFAULT_CONTEXT); err != nil {
		fmtPanic("Unable create buildx %s builder for %s context. Err: %s", BUILDER_NAME, DOCKER_DEFAULT_CONTEXT, err)
	}

	dirPath, err := os.Getwd()
	if err != nil {
		fmtPanic("Unable get pwd of project root. Err: %s", err)
	}

	CACHE_DIR := path.Join(dirPath, DOCKER_BUILDX_CACHE_DIR_NAME)
	fmt.Printf("[Docker] Use CACHE_DIR: %s | BUILDER_NAME: %s\n", CACHE_DIR, BUILDER_NAME)

	for _, service := range composeFile.Services {
		err := buildxBuildTarget(BUILDER_NAME, CACHE_DIR, service.Build.Target, fmt.Sprintf("%s-%s", composeFile.Name, service.Name), service.Build.Args)
		if err != nil {
			fmt.Printf("[Docker] Build | Error: %s", err)
		}
	}
}

func Buildx(composeFilePath string) {
	config := exec.Command("docker-compose", "-f", composeFilePath, "config", "--format", "json")
	stdout, err := config.StdoutPipe()
	if err != nil {
		fmtPanic("Unable init docker compose stdout pipe. Err: %s", err)
	}
	defer stdout.Close()

	stdoutBytes := make(chan []byte)

	go func() {
		out, err := io.ReadAll(stdout)
		if err != nil {
			fmtPanic("Unable read stdout Err: %s", err)
		}
		stdoutBytes <- out
		close(stdoutBytes)
	}()

	if err := config.Run(); err != nil {
		fmtPanic("Unable start docker compse file. Err: %s", err)
	}

	composeFile := parseDockerComposeFile(<-stdoutBytes)
	buildKitBuild(composeFile)
}

func buildxPushTarget(builder, cache, target, label string, args map[string]string) error {
	var buildArgs []string
	for argName, argVal := range args {
		buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", argName, argVal))
	}

	command := []string{
		"buildx", "build",
		fmt.Sprintf("--builder=%s", builder),
		fmt.Sprintf("--cache-from=type=local,src=%s", cache),
		"--target", target,
		"--label", label,
		"--tag", fmt.Sprintf("%s:latest", label),
		"--load",
	}
	command = append(command, buildArgs...)
	command = append(command, ".")

	cmd := exec.Command("docker", command...)

	stderr, _ := cmd.StderrPipe()
	go io.Copy(os.Stdout, stderr)

	fmt.Printf("[Docker] Exec image %s build with:\n %s \n", label, cmd.String())
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func buildKitDeploy(contextName string, file DockerComposeFile) {
	if err := dockerContextUse(contextName); err != nil {
		fmtPanic("Unable select docker context. Err: %s", err)
	}
	defer dockerContextUse(DOCKER_DEFAULT_CONTEXT)

	BUILDER_NAME := "container"
	if err := buildxCreateBuilder(BUILDER_NAME, DOCKER_DEFAULT_CONTEXT); err != nil {
		fmtPanic("Unable create buildx %s builder for %s context. Err: %s\n", BUILDER_NAME, DOCKER_DEFAULT_CONTEXT, err)
	}

	dirPath, err := os.Getwd()
	if err != nil {
		fmtPanic("Unable get pwd of project root. Err: %s", err)
	}

	CACHE_DIR := path.Join(dirPath, DOCKER_BUILDX_CACHE_DIR_NAME)
	fmt.Printf("[Docker] Use CACHE_DIR: %s | BUILDER_NAME: %s\n", CACHE_DIR, BUILDER_NAME)

	for _, service := range file.Services {
		err := buildxPushTarget(BUILDER_NAME, CACHE_DIR, service.Build.Target, fmt.Sprintf("%s-%s", file.Name, service.Name), service.Build.Args)
		if err != nil {
			fmt.Printf("[Docker] Build | Error: %s\n", err)
		}
	}
}

func BuildxDeploy(contextName, composeFilePath string) {
	config := exec.Command("docker-compose", "-f", composeFilePath, "config", "--format", "json")
	stdout, err := config.StdoutPipe()
	if err != nil {
		fmtPanic("Unable init docker compose stdout pipe. Err: %s", err)
	}
	defer stdout.Close()

	stdoutBytes := make(chan []byte)

	go func() {
		out, err := io.ReadAll(stdout)
		if err != nil {
			fmtPanic("Unable read stdout Err: %s", err)
		}
		stdoutBytes <- out
		close(stdoutBytes)
	}()

	if err := config.Run(); err != nil {
		fmtPanic("Unable start docker compse file. Err: %s", err)
	}

	composeFile := parseDockerComposeFile(<-stdoutBytes)
	buildKitDeploy(contextName, composeFile)
}
