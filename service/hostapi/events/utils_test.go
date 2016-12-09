package events

import (
	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"os"
	"testing"
)

func useEnvVars() bool {
	return os.Getenv("CATTLE_DOCKER_USE_BOOT2DOCKER") == "true"
}

func createContainer(client *client.Client) (types.ContainerJSON, error) {
	config := &container.Config{
		Image: "tianon/true",
	}
	resp, err := client.ContainerCreate(context.Background(), config, nil, nil, "")
	if err != nil {
		return types.ContainerJSON{}, nil
	}
	return client.ContainerInspect(context.Background(), resp.ID)
}

func createNetTestContainerNoLabel(client *client.Client, ip string) (types.ContainerJSON, error) {
	return createTestContainerInternal(client, ip, false, nil, true)
}

func createNetTestContainer(client *client.Client, ip string) (types.ContainerJSON, error) {
	return createTestContainerInternal(client, ip, true, nil, true)
}

func createTestContainer(client *client.Client, ip string, labels map[string]string, isSystem bool) (types.ContainerJSON, error) {
	return createTestContainerInternal(client, ip, true, labels, isSystem)
}

func createTestContainerInternal(client *client.Client, ip string, useLabel bool, inputLabels map[string]string, isSystem bool) (types.ContainerJSON, error) {
	labels := make(map[string]string)
	if inputLabels != nil {
		for k, v := range inputLabels {
			labels[k] = v
		}
	}
	if isSystem {
		labels["io.rancher.container.system"] = "FakeSysContainer"
	}

	env := []string{}
	if ip != "" {
		if useLabel {
			labels["io.rancher.container.ip"] = ip
		} else {
			env = append(env, "RANCHER_IP="+ip)
		}
	}

	config := &container.Config{
		Image:     "busybox:latest",
		Labels:    labels,
		Env:       env,
		OpenStdin: true,
		StdinOnce: false,
	}
	resp, err := client.ContainerCreate(context.Background(), config, nil, nil, "")
	if err != nil {
		return types.ContainerJSON{}, nil
	}
	return client.ContainerInspect(context.Background(), resp.ID)
}

func pullTestImages(client *client.Client) {
	listImageOpts := types.ImageListOptions{}
	images, _ := client.ImageList(context.Background(), listImageOpts)
	imageMap := map[string]bool{}
	for _, image := range images {
		for _, tag := range image.RepoTags {
			imageMap[tag] = true
		}
	}

	var pullImage = func(repo string) {
		if _, ok := imageMap[repo]; !ok {
			client.ImagePull(context.Background(), repo, types.ImagePullOptions{})
		}
	}

	imageName := "tianon/true:latest"
	pullImage(imageName)

	imageName = "busybox:latest"
	pullImage(imageName)
}

func TestMain(m *testing.M) {
	client, _ := NewDockerClient()
	pullTestImages(client)
	result := m.Run()
	os.Exit(result)
}

func prep(t *testing.T) *client.Client {
	dockerClient, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}

	return dockerClient
}
