from datadiff.tools import assert_equals
from docker import Client
from docker.utils import kwargs_from_env
import inspect
import json
import logging
import os
from os import path
from os.path import dirname
import requests
import tests
import time
from docker.utils import compare_version
import re
from tests.cattle import Config
import random

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
    try:
        cont_dir = Config.container_state_dir()
        file_path = path.join(cont_dir, docker_id)
        return os.path.exists(file_path)
    except:
        return False


def remove_state_file(container):
    if container:
        try:
            cont_dir = Config.container_state_dir()
            file_path = os.path.join(cont_dir, container['Id'])
            if os.path.exists(file_path):
                os.remove(file_path)
        except:
            pass


def instance_activate_common_validation(resp):
    docker_container = resp['data']['instanceHostMap']['instance']
    docker_container = docker_container['+data']['dockerContainer']
    docker_id = docker_container['Id']
    container_field_test_boiler_plate(resp)
    fields = resp['data']['instanceHostMap']['instance']['+data']['+fields']
    try:
        del docker_container['Ports'][0]['PublicPort']
        del docker_container['Ports'][1]['PublicPort']
    except KeyError:
        pass
    except IndexError:
        pass
    fields['dockerPorts'].sort()
    for idx, p in enumerate(fields['dockerPorts']):
        if '8080' in p or '12201' in p:
            fields['dockerPorts'][idx] = re.sub(r':.*:', ':1234:', p)
    assert state_file_exists(docker_id)
    instance_activate_assert_host_config(resp)
    instance_activate_assert_image_id(resp)
    del docker_container["State"]
    del docker_container["Mounts"]
    fields["dockerHostIp"] = '1.2.3.4'
    del resp['links']
    del resp['actions']


def newer_than(version):
    client = docker_client()
    ver = client.version()['ApiVersion']
    return compare_version(version, ver) >= 0


def instance_activate_assert_image_id(resp):
    docker_container = resp['data']['instanceHostMap']['instance']
    docker_container = docker_container['+data']['dockerContainer']
    if newer_than('1.20'):
        if 'ImageID' in docker_container:
            del docker_container['ImageID']


def instance_activate_assert_host_config(resp):
    docker_container = resp['data']['instanceHostMap']['instance']
    docker_container = docker_container['+data']['dockerContainer']
    if newer_than('1.20'):
        if 'HostConfig' in docker_container:
            assert docker_container['HostConfig'] == {
                'NetworkMode': 'default'
            } or docker_container['HostConfig'] == {}
            del docker_container['HostConfig']


def container_field_test_boiler_plate(resp):
    instance_data = resp['data']['instanceHostMap']['instance']['+data']
    docker_container = instance_data['dockerContainer']
    assert resp['data']['instanceHostMap']['instance']['externalId'] == \
        instance_data['dockerInspect']['Id']
    del resp['data']['instanceHostMap']['instance']['externalId']
    del instance_data['dockerInspect']
    try:
        del instance_data['dockerMounts']
    except KeyError:
        pass
    fields = instance_data['+fields']
    del docker_container['Created']
    del docker_container['Id']
    del docker_container['Status']
    docker_container.pop('NetworkSettings', None)
    del fields['dockerIp']
    _sort_ports(docker_container)

    if 'Labels' in docker_container and docker_container['Labels'] is None:
        docker_container['Labels'] = {}

    instance_activate_assert_host_config(resp)
    instance_activate_assert_image_id(resp)


def _sort_ports(docker_container):
    docker_container['Ports'] = sorted(docker_container['Ports'],
                                       key=lambda x: 1 - x['PrivatePort'])
    return docker_container


def get_container(name):
    client = docker_client()
    for c in client.containers(all=True):
        for container_name in c['Names']:
            if name == container_name:
                return c
    return None


def trim(docker_container, fields, resp, valid_resp):
    try:
        del docker_container["State"]
        del docker_container["Mounts"]
        fields["dockerHostIp"] = '1.2.3.4'
        del resp['links']
        del resp['actions']
        del valid_resp['previousNames']
    except KeyError:
        pass


def instance_only_activate(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        for nic in instance['nics']:
            nic['macAddress'] = ''

    def post(req, resp):
        docker_inspect = resp['data']['instanceHostMap']['instance']['+data'][
            'dockerInspect']
        labels = docker_inspect['Config']['Labels']
        ip = req['data']['instanceHostMap']['instance']['nics'][
            0]['ipAddresses'][0]
        expected_ip = "{0}/{1}".format(ip.address, ip.subnet.cidrSize)
        assert labels['io.rancher.container.ip'] == expected_ip
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate', pre_func=pre, post_func=post)


def delete_volume(name):
    client = docker_client(version=DockerConfig.storage_api_version())
    try:
        client.remove_volume(name)
    except:
        pass


def random_str():
    return 'test-{0}'.format(random_num())


def random_num():
    return random.randint(0, 1000000)


class ImageValidationError(Exception):
    pass
