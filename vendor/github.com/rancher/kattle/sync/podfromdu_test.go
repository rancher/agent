package sync

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/go-rancher/v3"
	"github.com/stretchr/testify/assert"
)

func TestGetPodSpec(t *testing.T) {
	assert.Equal(t, getPodSpec(client.DeploymentSyncRequest{
		Containers: []client.Container{
			client.Container{},
		},
	}), v1.PodSpec{
		RestartPolicy: v1.RestartPolicyNever,
		DNSPolicy:     v1.DNSDefault,
	})

	assert.Equal(t, getPodSpec(client.DeploymentSyncRequest{
		Containers: []client.Container{
			client.Container{
				RestartPolicy: &client.RestartPolicy{
					Name: "always",
				},
				PrimaryNetworkId: "1",
				IpcMode:          "host",
				PidMode:          "host",
			},
		},
		Networks: []client.Network{
			client.Network{
				Resource: client.Resource{
					Id: "1",
				},
				Kind: hostNetworkingKind,
			},
		},
	}), v1.PodSpec{
		RestartPolicy: v1.RestartPolicyNever,
		HostIPC:       true,
		HostNetwork:   true,
		HostPID:       true,
		DNSPolicy:     v1.DNSDefault,
	})
}

func TestLivenessProbe(t *testing.T) {
	livenessProbe := getLivenessProbe(client.Container{
		HealthCheck: &client.InstanceHealthCheck{
			ResponseTimeout:    2000,
			Interval:           2000,
			HealthyThreshold:   4,
			UnhealthyThreshold: 4,
		},
	})
	assert.Equal(t, livenessProbe.TimeoutSeconds, int32(2))
	assert.Equal(t, livenessProbe.PeriodSeconds, int32(2))
	assert.Equal(t, livenessProbe.SuccessThreshold, int32(4))
	assert.Equal(t, livenessProbe.FailureThreshold, int32(4))

	livenessProbe = getLivenessProbe(client.Container{
		HealthCheck: &client.InstanceHealthCheck{
			ResponseTimeout: 500,
			Interval:        500,
		},
	})
	assert.Equal(t, livenessProbe.TimeoutSeconds, int32(1))
	assert.Equal(t, livenessProbe.PeriodSeconds, int32(1))

	livenessProbe = getLivenessProbe(client.Container{
		HealthCheck: &client.InstanceHealthCheck{
			Port: 80,
		},
	})
	assert.Equal(t, livenessProbe.TCPSocket.Port, intstr.IntOrString{
		IntVal: 80,
	})

	livenessProbe = getLivenessProbe(client.Container{
		HealthCheck: &client.InstanceHealthCheck{
			RequestLine: "GET /healthcheck HTTP/1.0",
			Port:        80,
		},
	})
	assert.Equal(t, livenessProbe.HTTPGet.Path, "/healthcheck")
	assert.Equal(t, livenessProbe.HTTPGet.Port, intstr.IntOrString{
		IntVal: 80,
	})
}

func TestSecurityContext(t *testing.T) {
	securityContext := getSecurityContext(client.Container{
		Privileged: true,
		ReadOnly:   true,
		CapAdd: []string{
			"capadd1",
			"capadd2",
		},
		CapDrop: []string{
			"capdrop1",
			"capdrop2",
		},
	})
	assert.Equal(t, *securityContext.Privileged, true)
	assert.Equal(t, *securityContext.ReadOnlyRootFilesystem, true)
	assert.Equal(t, securityContext.Capabilities.Add, []v1.Capability{
		v1.Capability("capadd1"),
		v1.Capability("capadd2"),
	})
	assert.Equal(t, securityContext.Capabilities.Drop, []v1.Capability{
		v1.Capability("capdrop1"),
		v1.Capability("capdrop2"),
	})
}

func TestGetVolumes(t *testing.T) {
	assert.Equal(t, getVolumes(client.DeploymentSyncRequest{
		Containers: []client.Container{
			client.Container{
				DataVolumes: []string{
					"/host/path:/container/path",
				},
			},
		},
	}), []v1.Volume{
		v1.Volume{
			Name: "host-path-volume",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/host/path",
				},
			},
		},
	})
	assert.Equal(t, len(getVolumes(client.DeploymentSyncRequest{
		Containers: []client.Container{
			client.Container{
				DataVolumes: []string{
					"/anonymous/volume",
				},
			},
		},
	})), 0)
}

func TestGetVolumeMounts(t *testing.T) {
	assert.Equal(t, getVolumeMounts(client.Container{
		DataVolumes: []string{
			"/host/path:/container/path",
		},
	}), []v1.VolumeMount{
		v1.VolumeMount{
			Name:      "host-path-volume",
			MountPath: "/container/path",
		},
	})
	assert.Equal(t, len(getVolumeMounts(client.Container{
		DataVolumes: []string{
			"/anonymous/volume",
		},
	})), 0)
}
