from urlparse import urlparse
from contextlib import closing
import binascii
import os
import json
import urllib2
import logging
import arrow

log = logging.getLogger('cattle')


def memoize(function):
    memo = {}

    def wrapper(*args):
        if args in memo:
            return memo[args]
        else:
            rv = function(*args)
            memo[args] = rv
            return rv
    return wrapper


def _to_json_object(v):
    if isinstance(v, dict):
        return JsonObject(v)
    elif isinstance(v, list):
        ret = []
        for i in v:
            ret.append(_to_json_object(i))
        return ret
    else:
        return v


class JsonObject:
    def __init__(self, data):
        for k, v in data.items():
            self.__dict__[k] = _to_json_object(v)

    def __getitem__(self, item):
        value = self.__dict__[item]
        if isinstance(value, JsonObject):
            return value.__dict__
        return value

    def __getattr__(self, name):
        return getattr(self.__dict__, name)

    @staticmethod
    def unwrap(json_object):
        if isinstance(json_object, list):
            ret = []
            for i in json_object:
                ret.append(JsonObject.unwrap(i))
            return ret

        if isinstance(json_object, dict):
            ret = {}
            for k, v in json_object.items():
                ret[k] = JsonObject.unwrap(v)
            return ret

        if isinstance(json_object, JsonObject):
            ret = {}
            for k, v in json_object.__dict__.items():
                ret[k] = JsonObject.unwrap(v)
            return ret

        return json_object


def get_url_port(url):
    parsed = urlparse(url)

    port = parsed.port

    if port is None:
        if parsed.scheme == 'http':
            port = 80
        elif parsed.scheme == 'https':
            port = 443

    if port is None:
        raise Exception('Failed to find port for {0}'.format(url))

    return port


def random_string(length=64):
    return binascii.hexlify(os.urandom(length/2))


class CadvisorAPIClient(object):
    def __init__(self, host, port, version='v1.2', proto='http://'):
        self.url = '{0}{1}:{2}/api/{3}'.format(proto, host, str(port), version)

    def get_containers(self):
        return self._get(self.url + '/containers')

    def get_latest_stat(self):
        containers = self.get_stats()
        if len(containers) > 1:
            return containers[-1]
        return {}

    def get_stats(self):
        containers = self.get_containers()
        if containers:
            return containers['stats']
        return []

    def get_machine_stats(self):
        machine_data = self._get(self.url + '/machine')
        if machine_data:
            return machine_data
        return {}

    def timestamp_diff(self, time_current, time_prev):
        time_current_conv = self._timestamp_convert(time_current)
        time_prev_conv = self._timestamp_convert(time_prev)

        diff = (time_current_conv - time_prev_conv).total_seconds()
        return round((diff * 10**9))

    def _timestamp_convert(self, stime):
        # Cadvisor handles everything in nanoseconds.
        # Python does not.
        t_conv = arrow.get(stime[0:26])
        return t_conv

    def _marshall_to_python(self, data):
        if isinstance(data, str):
            return json.loads(data)

    def _get(self, url):
        try:
            with closing(urllib2.urlopen(url, timeout=5)) as resp:
                if resp.code == 200:
                    data = resp.read()
                    return self._marshall_to_python(data)
        except:
            log.exception(
                "Could not get stats from cAdvisor at: {0}".format(url))

        return None
