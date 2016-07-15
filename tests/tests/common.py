from datadiff.tools import assert_equals
from docker import Client
from docker.utils import kwargs_from_env
import inspect
import json
import logging
import os.path
from os.path import dirname
import requests
import tests
import time

TEST_DIR = os.path.join(dirname(tests.__file__))
CONFIG_OVERRIDE = {}

log = logging.getLogger("common")


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


class Marshaller:
    def __init__(self):
        pass

    def from_string(self, string):
        obj = json.loads(string)
        return JsonObject(obj)

    def to_string(self, obj):
        obj = JsonObject.unwrap(obj)
        return json.dumps(obj)

marshaller = Marshaller()


class Agent():
    def __init__(self):
        pass

    def execute(self, req):
        js = JsonObject.unwrap(req)
        resp = requests.post("http://localhost:8089/events", json=js)
        js_resp = resp.json()
        return js_resp


def json_data(name):
    with open(os.path.join(TEST_DIR, name)) as f:
        return marshaller.from_string(f.read())


def diff_dict(left, right):
    for k in left.keys():
        left_value = left.get(k)
        right_value = right.get(k)
        try:
            diff_dict(dict(left_value), dict(right_value))
            assert_equals(dict(left_value), dict(right_value))
        except AssertionError as e:
            raise e
        except:
            pass


def event_test(agent, name, pre_func=None, post_func=None, diff=True):
    req = json_data(name)
    valid_resp_file = json_data(name + '_resp')
    valid_resp = JsonObject.unwrap(valid_resp_file)

    if pre_func is not None:
        pre_func(req)

    resp = agent.execute(req)
    if post_func is not None:
        insp = inspect.getargspec(post_func)
        if len(insp.args) == 3:
            post_func(req, resp, valid_resp)
        else:
            post_func(req, resp)

    if diff:
        # del resp["id"]
        # del resp["time"]

        diff_dict(valid_resp, JsonObject.unwrap(resp))
        assert_equals(valid_resp, JsonObject.unwrap(resp))

    return req, resp


def delete_container(name):
    client = docker_client()
    for c in client.containers(all=True):
        found = False
        labels = c.get('Labels', {})
        if labels.get('io.rancher.container.uuid', None) == name[1:]:
            found = True

        for container_name in c['Names']:
            if name == container_name:
                found = True
                break

        if found:
            try:
                client.kill(c)
            except:
                pass
            for i in range(10):
                if client.inspect_container(c['Id'])['State']['Pid'] == 0:
                    break
                time.sleep(0.5)
            client.remove_container(c)
            remove_state_file(c)


def docker_client(version=None, base_url_override=None, tls_config=None,
                  timeout=None):
    if DockerConfig.use_boot2docker_connection_env_vars():
        kwargs = kwargs_from_env(assert_hostname=False)
    else:
        kwargs = {'base_url': DockerConfig.url_base()}

    if base_url_override:
        kwargs['base_url'] = base_url_override

    if tls_config:
        kwargs['tls'] = tls_config

    if version is None:
        version = DockerConfig.api_version()

    if timeout:
        kwargs['timeout'] = timeout
    kwargs['version'] = version
    log.debug('docker_client=%s', kwargs)
    return Client(**kwargs)


class DockerConfig:
    def __init__(self):
        pass

    @staticmethod
    def docker_enabled():
        return default_value('DOCKER_ENABLED', 'true') == 'true'

    @staticmethod
    def docker_home():
        return default_value('DOCKER_HOME', '/var/lib/docker')

    @staticmethod
    def url_base():
        return default_value('DOCKER_URL_BASE', None)

    @staticmethod
    def api_version():
        return default_value('DOCKER_API_VERSION', '1.18')

    @staticmethod
    def storage_api_version():
        return default_value('DOCKER_STORAGE_API_VERSION', '1.21')

    @staticmethod
    def use_boot2docker_connection_env_vars():
        use_b2d = default_value('DOCKER_USE_BOOT2DOCKER', 'false')
        return use_b2d.lower() == 'true'


def default_value(name, default):
    if name in CONFIG_OVERRIDE:
        return CONFIG_OVERRIDE[name]
    result = os.environ.get('CATTLE_%s' % name, default)
    if result == '':
        return default
    return result


def state_file_exists(docker_id):
    pass
    # TODO Implement
    # try:
    #     cont_dir = Config.container_state_dir()
    #     file_path = path.join(cont_dir, docker_id)
    #     return os.path.exists(file_path)
    # except:
    #     return False


def remove_state_file(container):
    pass
    # TODO Implement
    # if container:
    #     try:
    #         cont_dir = Config.container_state_dir()
    #         file_path = path.join(cont_dir, container['Id'])
    #         if os.path.exists(file_path):
    #             os.remove(file_path)
    #     except:
    #         pass
