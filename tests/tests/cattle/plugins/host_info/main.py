import logging

from tests.cattle.plugins.host_info.memory import MemoryCollector
from tests.cattle.plugins.host_info.os_c import OSCollector
from tests.cattle.plugins.host_info.cpu import CpuCollector
from tests.cattle.plugins.host_info.disk import DiskCollector
from tests.cattle.plugins.host_info.iops import IopsCollector

log = logging.getLogger('host_info')


class HostInfo(object):
    def __init__(self, docker_client=None):
        self.docker_client = docker_client
        self.iops_collector = IopsCollector()
        self.collectors = [MemoryCollector(),
                           OSCollector(self.docker_client),
                           DiskCollector(self.docker_client),
                           CpuCollector(),
                           self.iops_collector]

    def collect_data(self):
        data = {}
        for collector in self.collectors:
            try:
                data[collector.key_name()] = collector.get_data()
            except:
                log.exception(
                    "Error collecting {0} stats".format(collector.key_name()))
                data[collector.key_name()] = {}

        return data

    def host_labels(self, label_pfx="io.rancher.host"):
        labels = {}
        for collector in self.collectors:
            try:
                get_labels = getattr(collector, "get_labels", None)
                if callable(get_labels):
                    labels.update(get_labels(label_pfx))
            except:
                log.exception(
                    "Error getting {0} labels".format(collector.key_name()))

        return labels if len(labels) > 0 else None

    def get_default_disk(self):
        return self.iops_collector.get_default_disk()
