package sync

import (
	"fmt"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/kattle/labels"
)

// TODO: move this
func Primary(d client.DeploymentSyncRequest) client.Container {
	if len(d.Containers) > -0 {
		return d.Containers[0]
	}
	return client.Container{}
}

const (
	rancherPauseContainerName = "rancher-pause"
	hostNetworkingKind        = "dockerHost"
)

var (
	rancherPauseContainer = v1.Container{
		Name: rancherPauseContainerName,
		// TODO: figure out where to read this so it's not hard-coded
		Image: "gcr.io/google_containers/pause-amd64:3.0",
	}
)

func PodFromDeploymentUnit(deploymentUnit client.DeploymentSyncRequest) v1.Pod {
	//containers := []v1.Container{rancherPauseContainer}
	var containers []v1.Container
	for _, container := range deploymentUnit.Containers {
		containers = append(containers, getContainer(container))
	}

	podSpec := getPodSpec(deploymentUnit)
	podSpec.Containers = containers

	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentUnit.DeploymentUnitUuid,
			Labels: map[string]string{
				labels.RevisionLabel: deploymentUnit.Revision,
			},
		},
		Spec: podSpec,
	}
}

func getContainer(container client.Container) v1.Container {
	return v1.Container{
		Name:            container.Uuid,
		Image:           container.Image,
		Command:         container.EntryPoint,
		Args:            container.Command,
		TTY:             container.Tty,
		Stdin:           container.StdinOpen,
		WorkingDir:      container.WorkingDir,
		Env:             getEnvironment(container),
		SecurityContext: getSecurityContext(container),
		VolumeMounts:    getVolumeMounts(container),
		LivenessProbe:   getLivenessProbe(container),
	}
}

func getPodSpec(deploymentUnit client.DeploymentSyncRequest) v1.PodSpec {
	return v1.PodSpec{
		RestartPolicy: v1.RestartPolicyNever,
		HostNetwork:   getHostNetwork(deploymentUnit),
		HostIPC:       Primary(deploymentUnit).IpcMode == "host",
		HostPID:       Primary(deploymentUnit).PidMode == "host",
		DNSPolicy:     v1.DNSDefault,
		// TODO
		//NodeName:      deploymentUnit.Host.Name,
		NodeSelector: getNodeSelector(Primary(deploymentUnit)),
		Volumes:      getVolumes(deploymentUnit),
	}
}

func getHostNetwork(deploymentUnit client.DeploymentSyncRequest) bool {
	networkId := Primary(deploymentUnit).PrimaryNetworkId
	for _, network := range deploymentUnit.Networks {
		if network.Id == networkId && network.Kind == hostNetworkingKind {
			return true
		}
	}
	return false
}

func getLivenessProbe(container client.Container) *v1.Probe {
	healthcheck := container.HealthCheck
	if healthcheck == nil {
		return nil
	}

	timeoutSeconds := int32(healthcheck.ResponseTimeout / 1000)
	if timeoutSeconds == 0 {
		timeoutSeconds = 1
	}

	periodSeconds := int32(healthcheck.Interval / 1000)
	if periodSeconds == 0 {
		periodSeconds = 1
	}

	probe := v1.Probe{
		TimeoutSeconds:   timeoutSeconds,
		PeriodSeconds:    periodSeconds,
		SuccessThreshold: int32(healthcheck.HealthyThreshold),
		FailureThreshold: int32(healthcheck.UnhealthyThreshold),
	}
	port := intstr.IntOrString{
		IntVal: int32(healthcheck.Port),
	}

	if healthcheck.RequestLine == "" {
		probe.Handler = v1.Handler{
			TCPSocket: &v1.TCPSocketAction{
				Port: port,
			},
		}
	} else {
		parts := strings.Split(healthcheck.RequestLine, " ")
		if len(parts) < 3 {
			return nil
		}
		// TODO: first and third parts completely ignored
		path := parts[1]
		probe.Handler = v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path: path,
				Port: port,
			},
		}
	}

	return &probe
}

func getSecurityContext(container client.Container) *v1.SecurityContext {
	var capAdd []v1.Capability
	for _, cap := range container.CapAdd {
		capAdd = append(capAdd, v1.Capability(cap))
	}
	var capDrop []v1.Capability
	for _, cap := range container.CapDrop {
		capDrop = append(capDrop, v1.Capability(cap))
	}
	return &v1.SecurityContext{
		Privileged:             &container.Privileged,
		ReadOnlyRootFilesystem: &container.ReadOnly,
		Capabilities: &v1.Capabilities{
			Add:  capAdd,
			Drop: capDrop,
		},
	}
}

func getNodeSelector(container client.Container) map[string]string {
	var hostAffinityLabelMap map[string]string
	if label, ok := container.Labels[labels.HostAffinityLabel]; ok {
		hostAffinityLabelMap = labels.Parse(label)
	}
	return hostAffinityLabelMap
}

func getEnvironment(container client.Container) []v1.EnvVar {
	var environment []v1.EnvVar
	for k, v := range container.Environment {
		environment = append(environment, v1.EnvVar{
			Name:  k,
			Value: fmt.Sprint(v),
		})
	}
	return environment
}

func getVolumes(deploymentUnit client.DeploymentSyncRequest) []v1.Volume {
	var volumes []v1.Volume

	for _, container := range deploymentUnit.Containers {
		for _, volume := range container.DataVolumes {
			split := strings.SplitN(volume, ":", -1)
			if len(split) < 2 {
				continue
			}

			hostPath := split[0]

			if !filepath.IsAbs(hostPath) {
				continue
			}

			volumes = append(volumes, v1.Volume{
				Name: createBindMountVolumeName(hostPath),
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: hostPath,
					},
				},
			})
		}
	}

	for _, container := range deploymentUnit.Containers {
		for _, mount := range container.Mounts {
			volumeName := getVolumeName(mount.VolumeName)
			volumes = append(volumes, v1.Volume{
				Name: volumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: volumeName,
					},
				},
			})
		}
	}

	return volumes
}

func getVolumeMounts(container client.Container) []v1.VolumeMount {
	var volumeMounts []v1.VolumeMount

	for _, volume := range container.DataVolumes {
		split := strings.SplitN(volume, ":", -1)
		if len(split) < 2 {
			continue
		}

		hostPath := split[0]
		containerPath := split[1]

		if !filepath.IsAbs(hostPath) {
			continue
		}

		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      createBindMountVolumeName(hostPath),
			MountPath: containerPath,
		})
	}

	for _, mount := range container.Mounts {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      getVolumeName(mount.VolumeName),
			MountPath: mount.Path,
		})
	}

	return volumeMounts
}

func createBindMountVolumeName(path string) string {
	path = strings.TrimLeft(path, "/")
	path = strings.Replace(path, "/", "-", -1)
	return fmt.Sprintf("%s-%s", path, "volume")
}
