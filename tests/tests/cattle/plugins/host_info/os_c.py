import platform
from .utils import semver_trunk


class OSCollector(object):
    def __init__(self, docker_client=None):
        self.docker_client = docker_client

    def key_name(self):
        return "osInfo"

    def _zip_fields_values(self, keys, values):
        data = {}
        for key, value in zip(keys, values):
            if len(value) > 0:
                data[key] = value
            else:
                data[key] = None

        return data

    def _docker_version_request(self):
        if self.docker_client:
            return self.docker_client.version()

        return None

    def _get_docker_version(self, verbose=True):
        data = {}

        if platform.system() == 'Linux':
            ver_resp = self._docker_version_request()

            if verbose and ver_resp:
                version = "Docker version {0}, build {1}".format(
                    ver_resp.get("Version", "Unknown"),
                    ver_resp.get("GitCommit", "Unknown"))

            elif ver_resp:
                version = "{0}".format(
                    semver_trunk(ver_resp.get("Version", "Unknown"), 2))

            else:
                version = "Unknown"

            data['dockerVersion'] = version

        return data

    def _get_os(self):
        data = {}
        if platform.system() == 'Linux':
            if self.docker_client:
                data["operatingSystem"] = \
                    self.docker_client.info().get("OperatingSystem",
                                                  None)

            data['kernelVersion'] = \
                platform.release() if len(platform.release()) > 0 else None

        return data

    def get_data(self):
        data = self._get_os()
        data.update(self._get_docker_version())

        return data

    def get_labels(self, pfx="rancher"):
        labels = {
            ".".join([pfx, "docker_version"]):
            self._get_docker_version(verbose=False)["dockerVersion"],
            ".".join([pfx, "linux_kernel_version"]):
            semver_trunk(self._get_os()["kernelVersion"], 2)
        }

        return labels
