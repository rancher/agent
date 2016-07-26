import platform


class MemoryCollector(object):
    def __init__(self):
        self.key_map = {'memtotal': 'memTotal',
                        'memfree': 'memFree',
                        'memavailable': 'memAvailable',
                        'buffers': 'buffers',
                        'cached': 'cached',
                        'swapcached': 'swapCached',
                        'active': 'active',
                        'inactive': 'inactive',
                        'swaptotal': 'swapTotal',
                        'swapfree': 'swapFree'
                        }

        self.unit = 1024.0

    def _get_meminfo_data(self):
        with open('/proc/meminfo') as f:
            return f.readlines()

    def _parse_linux_meminfo(self):
        data = {k: None for k in self.key_map.values()}

        # /proc/meminfo file has all values in kB
        mem_data = self._get_meminfo_data()
        for line in mem_data:
            line_list = line.split(':')
            key_lower = line_list[0].lower()
            possible_mem_value = line_list[1].strip().split(' ')[0]

            if self.key_map.get(key_lower):
                converted_mem_val = float(possible_mem_value)/self.unit
                data[self.key_map[key_lower]] = round(converted_mem_val, 3)

        return data

    def key_name(self):
        return "memoryInfo"

    def get_data(self):
        if platform.system() == 'Linux':
            return self._parse_linux_meminfo()
        else:
            return {}
