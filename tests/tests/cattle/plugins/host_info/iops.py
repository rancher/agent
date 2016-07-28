import logging
import platform
import json

log = logging.getLogger('iops')


class IopsCollector(object):
    def __init__(self):
        self.data = {}

    def _get_iops_data(self, read_or_write):
        with open('/var/lib/rancher/state/' + read_or_write + '.json') as f:
            return json.load(f)

    def _parse_iops_file(self):
        data = {}

        try:
            read_json_data = self._get_iops_data('read')
            write_json_data = self._get_iops_data('write')
        except IOError:
            # File doesn't exist. Silently skip.
            return {}

        read_iops = read_json_data['jobs'][0]['read']['iops']
        write_iops = write_json_data['jobs'][0]['write']['iops']
        device = read_json_data['disk_util'][0]['name']
        key = '/dev/' + device.encode('ascii', 'ignore')
        data[key] = {'read': read_iops, 'write': write_iops}
        return data

    def key_name(self):
        return "iopsInfo"

    def get_data(self):
        if platform.system() == 'Linux':
            if not self.data:
                self.data = self._parse_iops_file()
            return self.data
        else:
            return {}

    def get_default_disk(self):
        data = self.get_data()
        if not data:
            return None

        # Return the first item in the dict
        return data.keys()[0]
