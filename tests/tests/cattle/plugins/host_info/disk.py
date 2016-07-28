import platform

from tests.cattle.utils import CadvisorAPIClient
from tests.cattle import Config


class DiskCollector(object):
    def __init__(self, docker_client=None):
        self.unit = 1048576
        self.cadvisor = CadvisorAPIClient(Config.cadvisor_ip(),
                                          Config.cadvisor_port())

        self.docker_client = docker_client
        self.docker_storage_driver = None

        if self.docker_client:
            self.docker_storage_driver = \
                self.docker_client.info().get("Driver", None)

    def _convert_units(self, number):
        # Return in MB
        return round(float(number)/self.unit, 3)

    def _get_dockerstorage_info(self):
        data = {}

        if self.docker_client:
            for item in self.docker_client.info().get("DriverStatus"):
                data[item[0]] = item[1]

        return data

    def _include_in_filesystem(self, device):
        include = True

        if self.docker_storage_driver == "devicemapper":
            pool = self._get_dockerstorage_info()

            pool_name = pool.get("Pool Name", "/dev/mapper/docker-")
            if pool_name.endswith("-pool"):
                pool_name = pool_name[:-5]

            if pool_name in device:
                include = False

        return include

    def _get_mountpoints_cadvisor(self):
        data = {}
        stat = self.cadvisor.get_latest_stat()

        if 'filesystem' in stat.keys():
            for fs in stat['filesystem']:
                device = fs['device']
                percent_used = \
                    float(fs['usage']) / float(fs['capacity']) * 100

                data[device] = {
                    'free': self._convert_units(fs['capacity'] - fs['usage']),
                    'total': self._convert_units(fs['capacity']),
                    'used': self._convert_units(fs['usage']),
                    'percentUsed': round(percent_used, 2)
                }

        return data

    def _get_machine_filesystems_cadvisor(self):
        data = {}
        machine_info = self.cadvisor.get_machine_stats()

        if 'filesystems' in machine_info.keys():
            for filesystem in machine_info['filesystems']:
                if self._include_in_filesystem(filesystem['device']):
                    data[filesystem['device']] = {
                        'capacity': self._convert_units(filesystem['capacity'])
                    }

        return data

    def key_name(self):
        return 'diskInfo'

    def get_data(self):
        data = {
            'fileSystems': {},
            'mountPoints': {},
            'dockerStorageDriverStatus': {},
            'dockerStorageDriver': self.docker_storage_driver
        }

        if platform.system() == 'Linux':
            data['fileSystems'].update(
                self._get_machine_filesystems_cadvisor())
            data['mountPoints'].update(self._get_mountpoints_cadvisor())

        data['dockerStorageDriverStatus'].update(
            self._get_dockerstorage_info())

        return data
