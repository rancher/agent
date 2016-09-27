package hostInfo

import (
	"github.com/Sirupsen/logrus"
	"github.com/google/cadvisor/container/docker"
	"github.com/google/cadvisor/container/rkt"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/machine"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"os"
)

func GetMachineInfo() (*info.MachineInfo, error) {
	sysFs, err := sysfs.NewRealSysFs()
	if err != nil {
		logrus.Fatalf("Failed to create a system interface: %s", err)
	}
	dockerStatus, err := docker.Status()
	if err != nil {
		logrus.Warnf("Unable to connect to Docker: %v", err)
	}
	rktPath, err := rkt.RktPath()
	if err != nil {
		logrus.Warnf("unable to connect to Rkt api service: %v", err)
	}
	context := fs.Context{
		Docker: fs.DockerContext{
			Root:         docker.RootDir(),
			Driver:       dockerStatus.Driver,
			DriverStatus: dockerStatus.DriverStatus,
		},
		RktPath: rktPath,
	}
	fsInfo, err := fs.NewFsInfo(context)
	if err != nil {
		logrus.Fatalf("Failed to create a system interface: %s", err)
	}
	inHostNamespace := false
	if _, err := os.Stat("/rootfs/proc"); os.IsNotExist(err) {
		inHostNamespace = true
	}
	machineInfo, err := machine.Info(sysFs, fsInfo, inHostNamespace)
	if err != nil {
		return nil, errors.Wrap(err, constants.GetMachineInfoError+"failed to get machine info")
	}
	return machineInfo, nil
}

func GetFileInfo() (fs.FsInfo, error) {
	dockerStatus, err := docker.Status()
	if err != nil {
		logrus.Warnf("Unable to connect to Docker: %v", err)
	}
	rktPath, err := rkt.RktPath()
	if err != nil {
		logrus.Warnf("unable to connect to Rkt api service: %v", err)
	}
	context := fs.Context{
		Docker: fs.DockerContext{
			Root:         docker.RootDir(),
			Driver:       dockerStatus.Driver,
			DriverStatus: dockerStatus.DriverStatus,
		},
		RktPath: rktPath,
	}
	fsInfo, err := fs.NewFsInfo(context)
	if err != nil {
		return nil, errors.Wrap(err, constants.GetFileInfoError+"failed to get machine info")
	}
	return fsInfo, nil
}
