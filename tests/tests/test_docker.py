from .common import event_test, delete_container, \
    instance_activate_common_validation, \
    instance_activate_assert_host_config, \
    instance_activate_assert_image_id, \
    docker_client, \
    container_field_test_boiler_plate, \
    trim, CONFIG_OVERRIDE, JsonObject, Config, get_container, \
    instance_only_activate, delete_volume, DockerConfig, \
    newer_than, json_data

import time
from docker.errors import APIError
import pytest
import platform
from cattle.plugins.host_info.main import HostInfo


@pytest.fixture(scope='module')
def pull_images():
    client = docker_client()
    images = [('ibuildthecloud/helloworld', 'latest'),
              ('rancher/agent', 'v0.7.9'),
              ('rancher/agent', 'latest')]
    for i in images:
        try:
            client.inspect_image(':'.join(i))
        except APIError:
            client.pull(i[0], i[1])


def test_example(agent):
    """
    This test is the same as test_instance_activate_no_name except that it
    passes diff=False to event_test
    :param agent:
    :return:
    """
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['name'] = None

    def post(req, resp, valid_resp):
        data = valid_resp['data']['instanceHostMap']['instance']['+data']
        docker_con = data['dockerContainer']
        del docker_con['Labels']['io.rancher.container.name']
        docker_con['Names'] = ['/c861f990-4472-4fa1-960f-65171b544c28']

    schema = 'docker/instance_activate'
    event_test(agent, schema, pre_func=pre, post_func=post, diff=False)


def test_instance_activate_no_name(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['name'] = None

    def post(req, resp, valid_resp):
        data = valid_resp['data']['instanceHostMap']['instance']['+data']
        docker_con = data['dockerContainer']
        del docker_con['Labels']['io.rancher.container.name']
        docker_con['Names'] = ['/c861f990-4472-4fa1-960f-65171b544c28']
        # TODO Pull over below method
        instance_activate_common_validation(resp)

    schema = 'docker/instance_activate'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_duplicate_name(agent):
    dupe_name_uuid = 'dupename-c861f990-4472-4fa1-960f-65171b544c28'
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    delete_container('/' + dupe_name_uuid)

    schema = 'docker/instance_activate'
    event_test(agent, schema, diff=False)

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['uuid'] = dupe_name_uuid

    def post(req, resp, valid_resp):
        data = valid_resp['data']['instanceHostMap']['instance']['+data']
        docker_con = data['dockerContainer']
        docker_con['Labels']['io.rancher.container.uuid'] = dupe_name_uuid
        docker_con['Names'] = ['/' + dupe_name_uuid]
        instance_activate_common_validation(resp)

    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_no_mac_address(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        for nic in instance['nics']:
            nic['macAddress'] = ''

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
    # mac_received = docker_inspect['Config']['NetworkSettings']['MacAddress']
        mac_nic_received = docker_inspect['NetworkSettings']['MacAddress']
        # assert mac_received == ''
        assert mac_nic_received is not None
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate', pre_func=pre, post_func=post)


def test_instance_activate_mac_address(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        mac_received = docker_inspect['Config']['MacAddress']
        mac_nic_received = docker_inspect['NetworkSettings']['MacAddress']
        assert mac_nic_received == '02:03:04:05:06:07'
        assert mac_received == '02:03:04:05:06:07'
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate', post_func=post)


def test_instance_activate_ports(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        del instance_data['dockerInspect']
        del instance_data['dockerMounts']
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        fields['dockerPorts'].sort()
        del docker_container['Created']
        del docker_container['Id']
        del docker_container['Status']
        docker_container.pop('NetworkSettings', None)
        del fields['dockerIp']
        del resp['data']['instanceHostMap']['instance']['externalId']

        assert len(docker_container['Ports']) == 4
        for port in docker_container['Ports']:
            if port['PrivatePort'] == 8080:
                assert port['Type'] == 'tcp'
                assert 'HostIp' not in port
            elif port['PrivatePort'] == 12201:
                assert port['Type'] == 'udp'
                assert 'HostIp' not in port
            elif port['PrivatePort'] == 6666 and port['PublicPort'] == 7777:
                assert port['Type'] == 'tcp'
                assert port['IP'] == '127.0.0.1'
            elif port['PrivatePort'] == 6666 and port['PublicPort'] == 8888:
                assert port['Type'] == 'tcp'
                assert port['IP'] == '0.0.0.0'
            else:
                assert False, 'Found unknown port: %s' % port

        del docker_container['Ports']
        del docker_container["State"]
        del docker_container["Mounts"]
        fields["dockerHostIp"] = '1.2.3.4'
        del resp['links']
        del resp['actions']
        for i in range(len(fields['dockerPorts'])):
            if '12201/udp' in fields['dockerPorts'][i] or \
                    '8080/tcp' in fields['dockerPorts'][i]:
                fields['dockerPorts'][i] = fields[
                    'dockerPorts'][i].split(':')[-1]

        del valid_resp['previousNames']
        fields['dockerPorts'].sort()
        instance_activate_assert_host_config(resp)
        instance_activate_assert_image_id(resp)

    event_test(agent, 'docker/instance_activate_ports', post_func=post)


def test_instance_activate_links_null_ports(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        links = req.data.instanceHostMap.instance.instanceLinks
        links.append(JsonObject({
            'type': 'instanceLink',
            'linkName': 'null',
            'data': {
                'fields': {
                    'ports': None
                }
            },
            'targetInstanceId': None,
        }))

    def post(req, resp, valid_resp):
        id = resp['data']['instanceHostMap']['instance']
        id = id['+data']['dockerContainer']['Id']
        instance_activate_common_validation(resp)
        del valid_resp['previousNames']

    event_test(agent, 'docker/instance_activate_links', pre_func=pre,
               post_func=post)


def test_instance_activate_double_links(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp, valid_resp):
        id = resp['data']['instanceHostMap']['instance']
        id = id['+data']['dockerContainer']['Id']
        inspect = docker_client().inspect_container(id)
        instance_activate_common_validation(resp)
        del valid_resp['previousNames']

        env = inspect['Config']['Env']

        for line in env:
            assert 'LVL' not in line

        assert 'MYSQL_NAME=/cattle/mysql' in env
        assert 'MYSQL_PORT=udp://mysql:3307' in env
        assert 'MYSQL_PORT_3307_UDP=udp://mysql:3307' in env
        assert 'MYSQL_PORT_3307_UDP_ADDR=mysql' in env
        assert 'MYSQL_PORT_3307_UDP_PORT=3307' in env
        assert 'MYSQL_PORT_3307_UDP_PROTO=udp' in env

        assert 'MYSQL_PORT_3306_TCP=tcp://mysql:3306' in env
        assert 'MYSQL_PORT_3306_TCP_ADDR=mysql' in env
        assert 'MYSQL_PORT_3306_TCP_PORT=3306' in env
        assert 'MYSQL_PORT_3306_TCP_PROTO=tcp' in env

        assert 'REDIS_NAME=/cattle/redis' in env
        assert 'REDIS_PORT=udp://redis:26' in env
        assert 'REDIS_PORT_26_UDP=udp://redis:26' in env
        assert 'REDIS_PORT_26_UDP_ADDR=redis' in env
        assert 'REDIS_PORT_26_UDP_PORT=26' in env
        assert 'REDIS_PORT_26_UDP_PROTO=udp' in env

        assert 'REDIS_ENV_ONE=TWO' in env
        assert 'REDIS_ENV_THREE=FOUR' in env
        assert 'REDIS_1_ENV_ONE=TWO' in env
        assert 'REDIS_1_ENV_THREE=FOUR' in env
        assert 'REDIS_2_ENV_ONE=TWO' in env
        assert 'REDIS_2_ENV_THREE=FOUR' in env
        assert 'ENV_REDIS_1_ENV_ONE=TWO' in env
        assert 'ENV_REDIS_1_ENV_THREE=FOUR' in env
        assert 'ENV_REDIS_2_ENV_ONE=TWO' in env
        assert 'ENV_REDIS_2_ENV_THREE=FOUR' in env

        assert 'REDIS_1_PORT=udp://redis:26' in env
        assert 'REDIS_1_PORT_26_UDP=udp://redis:26' in env
        assert 'REDIS_1_PORT_26_UDP_ADDR=redis' in env
        assert 'REDIS_1_PORT_26_UDP_PORT=26' in env
        assert 'REDIS_1_PORT_26_UDP_PROTO=udp' in env

        assert 'ENV_REDIS_1_PORT=udp://redis:26' in env
        assert 'ENV_REDIS_1_PORT_26_UDP=udp://redis:26' in env
        assert 'ENV_REDIS_1_PORT_26_UDP_ADDR=redis' in env
        assert 'ENV_REDIS_1_PORT_26_UDP_PORT=26' in env
        assert 'ENV_REDIS_1_PORT_26_UDP_PROTO=udp' in env

    event_test(agent, 'docker/instance_activate_double_links', post_func=post)


def test_instance_activate_links(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp, valid_resp):
        id = resp['data']['instanceHostMap']['instance']
        id = id['+data']['dockerContainer']['Id']
        inspect = docker_client().inspect_container(id)
        instance_activate_common_validation(resp)

        env = inspect['Config']['Env']

        assert 'MYSQL_NAME=/cattle/mysql' in env
        assert 'MYSQL_PORT=udp://mysql:3307' in env
        assert 'MYSQL_PORT_3307_UDP=udp://mysql:3307' in env
        assert 'MYSQL_PORT_3307_UDP_ADDR=mysql' in env
        assert 'MYSQL_PORT_3307_UDP_PORT=3307' in env
        assert 'MYSQL_PORT_3307_UDP_PROTO=udp' in env

        assert 'MYSQL_PORT_3306_TCP=tcp://mysql:3306' in env
        assert 'MYSQL_PORT_3306_TCP_ADDR=mysql' in env
        assert 'MYSQL_PORT_3306_TCP_PORT=3306' in env
        assert 'MYSQL_PORT_3306_TCP_PROTO=tcp' in env

        assert 'REDIS_NAME=/cattle/redis' in env
        assert 'REDIS_PORT=udp://redis:26' in env
        assert 'REDIS_PORT_26_UDP=udp://redis:26' in env
        assert 'REDIS_PORT_26_UDP_ADDR=redis' in env
        assert 'REDIS_PORT_26_UDP_PORT=26' in env
        assert 'REDIS_PORT_26_UDP_PROTO=udp' in env

        assert 'REDIS_ENV_ONE=TWO' in env
        assert 'REDIS_ENV_THREE=FOUR' in env
        assert 'REDIS_1_ENV_ONE=TWO' in env
        assert 'REDIS_1_ENV_THREE=FOUR' in env
        assert 'REDIS_2_ENV_ONE=TWO' in env
        assert 'REDIS_2_ENV_THREE=FOUR' in env
        assert 'ENV_REDIS_1_ENV_ONE=TWO' in env
        assert 'ENV_REDIS_1_ENV_THREE=FOUR' in env
        assert 'ENV_REDIS_2_ENV_ONE=TWO' in env
        assert 'ENV_REDIS_2_ENV_THREE=FOUR' in env

        assert 'REDIS_1_PORT=udp://redis:26' in env
        assert 'REDIS_1_PORT_26_UDP=udp://redis:26' in env
        assert 'REDIS_1_PORT_26_UDP_ADDR=redis' in env
        assert 'REDIS_1_PORT_26_UDP_PORT=26' in env
        assert 'REDIS_1_PORT_26_UDP_PROTO=udp' in env

        assert 'ENV_REDIS_1_PORT=udp://redis:26' in env
        assert 'ENV_REDIS_1_PORT_26_UDP=udp://redis:26' in env
        assert 'ENV_REDIS_1_PORT_26_UDP_ADDR=redis' in env
        assert 'ENV_REDIS_1_PORT_26_UDP_PORT=26' in env
        assert 'ENV_REDIS_1_PORT_26_UDP_PROTO=udp' in env

        del valid_resp['previousNames']

    event_test(agent, 'docker/instance_activate_links', post_func=post)


def test_instance_activate_links_no_service(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    delete_container('/target_redis')
    delete_container('/target_mysql')

    client = docker_client()
    c = client.create_container('ibuildthecloud/helloworld',
                                ports=[(3307, 'udp'), (3306, 'tcp')],
                                name='target_mysql')
    client.start(c, port_bindings={
        '3307/udp': ('127.0.0.2', 12346),
        '3306/tcp': ('127.0.0.2', 12345)
    })

    c = client.create_container('ibuildthecloud/helloworld',
                                name='target_redis')
    client.start(c)

    def post(req, resp, valid_resp):
        id = resp['data']['instanceHostMap']['instance']
        id = id['+data']['dockerContainer']['Id']
        inspect = docker_client().inspect_container(id)
        instance_activate_common_validation(resp)

        assert {'/target_mysql:/r-test/mysql',
                '/target_redis:/r-test/redis'} == \
            set(inspect['HostConfig']['Links'])
        del valid_resp['previousNames']

    event_test(agent, 'docker/instance_activate_links_no_service',
               post_func=post)


def test_instance_activate_cpu_set(agent):

    def pre(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        # instance['data']['fields']['cpuSet'] = '0,1'
        instance['data']['fields']['cpuSetCpus'] = '1,3'

    def preNull(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['cpuSetCpus'] = None

    def preEmpty(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        # instance['data']['fields']['cpuSet'] = ''
        instance['data']['fields']['cpuSetCpus'] = ''

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        # assert docker_inspect['Config']['Cpuset'] == '0,1'
        assert docker_inspect['HostConfig']['CpusetCpus'] == '1,3'
        container_field_test_boiler_plate(resp)
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    def postNull(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        # assert docker_inspect['Config']['Cpuset'] == ''
        assert docker_inspect['HostConfig']['CpusetCpus'] == ''
        container_field_test_boiler_plate(resp)
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    def postEmpty(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        # assert docker_inspect['Config']['Cpuset'] == ''
        assert docker_inspect['HostConfig']['CpusetCpus'] == ''
        container_field_test_boiler_plate(resp)
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)
    event_test(agent, schema, pre_func=preNull, post_func=postNull)
    event_test(agent, schema, pre_func=preEmpty, post_func=postEmpty)


def test_instance_activate_read_only(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    schema = 'docker/instance_activate_fields'

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['readOnly'] = True

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['HostConfig']['ReadonlyRootfs']
        container_field_test_boiler_plate(resp)
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    event_test(agent, schema, pre_func=pre, post_func=post)

    # Now test default value is False
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert not docker_inspect['HostConfig']['ReadonlyRootfs']
        container_field_test_boiler_plate(resp)
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, post_func=post)


def test_instance_activate_memory_swap(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    client = docker_client()
    swap = client.info()['SwapLimit']

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['memory'] = 12000000
        instance['data']['fields']['memorySwap'] = 16000000

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        if swap:
            # assert docker_inspect['Config']['MemorySwap'] == 16000000
            assert docker_inspect['HostConfig']['MemorySwap'] == 16000000
        else:
            # assert docker_inspect['Config']['MemorySwap'] == -1
            assert docker_inspect['HostConfig']['MemorySwap'] == -1
        # assert docker_inspect['Config']['Memory'] == 12000000
        assert docker_inspect['HostConfig']['Memory'] == 12000000
        container_field_test_boiler_plate(resp)
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_extra_hosts(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['extraHosts'] = [
            'host:1.1.1.1', 'b:2.2.2.2']

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['HostConfig']['ExtraHosts'] == ['host:1.1.1.1',
                                                              'b:2.2.2.2']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_pid_mode(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['pidMode'] = 'host'

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['HostConfig']['PidMode'] == 'host'
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_log_config(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['logConfig'] = \
            JsonObject({'driver': 'json-file',
                        'config': {
                            'max-size': '10',
                        }})

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['HostConfig']['LogConfig'] == {
            'Type': 'json-file',
            'Config': {
                'max-size': '10',
            }
        }
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_log_config_null(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['logConfig'] = JsonObject({'driver': None,
                                                              'config': None})

    def pre2(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['logConfig'] = None

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['HostConfig']['LogConfig']['Type'] == 'json-file'
        # Note: This is obscuring the fact that LogConfig.Config can be either
        # None or an empty map, but thats ok.
        assert not docker_inspect['HostConfig']['LogConfig']['Config']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    event_test(agent, schema, pre_func=pre2, post_func=post)
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    event_test(agent, schema, post_func=post)


def test_instance_activate_security_opt(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['securityOpt'] = ["label:foo", "label:bar"]

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['HostConfig']['SecurityOpt'] == ["label:foo",
                                                               "label:bar"]
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_working_dir(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['workingDir'] = "/tmp"

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['Config']['WorkingDir'] == "/tmp"
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_entrypoint(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['entryPoint'] = ["./sleep.sh"]

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['Config']['Entrypoint'] == ["./sleep.sh"]
        docker_container = instance_data['dockerContainer']
        docker_container['Command'] = "/sleep.sh"
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_memory(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['memory'] = 12000000

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        # assert docker_inspect['Config']['Memory'] == 12000000
        assert docker_inspect['HostConfig']['Memory'] == 12000000
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_tty(agent):

    def preFalse(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['tty'] = False

    def pre(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['tty'] = True

    def postFalse(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert not docker_inspect['Config']['Tty']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['Config']['Tty']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)
    event_test(agent, schema, pre_func=preFalse, post_func=postFalse)


def test_instance_activate_stdinOpen(agent):

    def preTrueDetach(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['stdinOpen'] = True
        # instance['data']['fields']['detach'] = True

    def preFalse(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['stdinOpen'] = False
        # instance['data']['fields']['detach'] = False

    def pre(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['stdinOpen'] = True
        # instance['data']['fields']['detach'] = False

    def postTrueDetach(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        # assert not docker_inspect['Config']['StdinOnce']
        assert docker_inspect['Config']['OpenStdin']
        # assert not docker_inspect['Config']['AttachStdin']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    def postFalse(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        # assert not docker_inspect['Config']['StdinOnce']
        assert not docker_inspect['Config']['OpenStdin']
        # assert not docker_inspect['Config']['AttachStdin']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        # assert docker_inspect['Config']['StdinOnce']
        assert docker_inspect['Config']['OpenStdin']
        # assert docker_inspect['Config']['AttachStdin']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)
    event_test(agent, schema, pre_func=preFalse, post_func=postFalse)
    event_test(agent, schema, pre_func=preTrueDetach, post_func=postTrueDetach)


def test_instance_activate_domainname(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['domainName'] = "rancher.io"

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['Config']['Domainname'] == "rancher.io"
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_devices(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    input_devices = ['/dev/null:/dev/xnull', '/dev/random:/dev/xrandom:rw']
    expected_devices = {}
    for input_device in input_devices:
        parts_of_device = input_device.split(':')
        key = parts_of_device[0]
        expected_devices[key] = {
            "PathOnHost": parts_of_device[0],
            "PathInContainer": parts_of_device[1]
        }
        if len(parts_of_device) == 3:
            expected_devices[key]["CgroupPermissions"] = parts_of_device[2]
        else:
            expected_devices[key]["CgroupPermissions"] = "rwm"

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['devices'] = input_devices

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        actual_devices = docker_inspect['HostConfig']['Devices']

        assert len(expected_devices) == len(actual_devices)

        for act_dvc in actual_devices:
            exp_dvc = expected_devices[act_dvc['PathOnHost']]
            assert exp_dvc['PathOnHost'] == act_dvc['PathOnHost']
            assert exp_dvc['PathInContainer'] == act_dvc['PathInContainer']
            assert exp_dvc['CgroupPermissions'] == act_dvc['CgroupPermissions']

        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


'''
@pytest.mark.skip("wait to implement host info api")
def test_instance_activate_device_options(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    # Note, can't test weight as it isn't supported in kernel by default
    device_options = {'/dev/sda': {
        'readIops': 1000,
        'writeIops': 2000,
        'readBps': 1024,
        'writeBps': 2048
    }
    }

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['blkioDeviceOptions'] = device_options

    def post(req, resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        host_config = instance_data['dockerInspect']['HostConfig']
        assert host_config['BlkioDeviceReadIOps'] == [
            {'Path': '/dev/sda', 'Rate': 1000}]
        assert host_config['BlkioDeviceWriteIOps'] == [
            {'Path': '/dev/sda', 'Rate': 2000}]
        assert host_config['BlkioDeviceReadBps'] == [
            {'Path': '/dev/sda', 'Rate': 1024}]
        assert host_config['BlkioDeviceWriteBps'] == [
            {'Path': '/dev/sda', 'Rate': 2048}]
        container_field_test_boiler_plate(resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)

    # Test DEFAULT_DISK functionality
    dc = DockerCompute()

    device = '/dev/mock'

    class MockHostInfo(object):

        def get_default_disk(self):
            return device

    dc.host_info = MockHostInfo()
    instance = JsonObject({'data': {}})
    instance.data['fields'] = {
        'blkioDeviceOptions': {
            'DEFAULT_DISK': {'readIops': 10}
        }
    }
    config = {}
    dc._setup_device_options(config, instance)
    assert config['BlkioDeviceReadIOps'] == [{'Path': '/dev/mock', 'Rate': 10}]

    config = {}
    device = None
    dc._setup_device_options(config, instance)
    assert not config  # config should be empty
'''


def test_instance_activate_single_device_option(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    device_options = {'/dev/sda': {
        'writeIops': 2000,
    }
    }

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['blkioDeviceOptions'] = device_options

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        host_config = instance_data['dockerInspect']['HostConfig']
        assert host_config['BlkioDeviceWriteIOps'] == [
            {'Path': '/dev/sda', 'Rate': 2000}]
        assert host_config['BlkioDeviceReadIOps'] is None
        assert host_config['BlkioDeviceReadBps'] is None
        assert host_config['BlkioDeviceWriteBps'] is None
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_dns(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['dns'] = ["1.2.3.4", "8.8.8.8"]
        instance['data']['fields']['dnsSearch'] = ["5.6.7.8", "7.7.7.7"]

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        actual_dns = docker_inspect['HostConfig']['Dns']
        actual_dns_search = docker_inspect['HostConfig']['DnsSearch']
        assert set(actual_dns) == set(["8.8.8.8", "1.2.3.4"])
        assert set(actual_dns_search) == set(["7.7.7.7", "5.6.7.8"])
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_caps(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['capAdd'] = ["MKNOD", "SYS_ADMIN"]
        instance['data']['fields']['capDrop'] = ["MKNOD", "SYS_ADMIN"]

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        set_actual_cap_add = set(docker_inspect['HostConfig']['CapAdd'])
        set_expected_cap_add = set(["MKNOD", "SYS_ADMIN"])
        assert set_actual_cap_add == set_expected_cap_add
        set_actual_cap_drop = set(docker_inspect['HostConfig']['CapDrop'])
        set_expected_cap_drop = set(["MKNOD", "SYS_ADMIN"])
        assert set_actual_cap_drop == set_expected_cap_drop
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_privileged(agent):

    def preTrue(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['privileged'] = True

    def preFalse(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['privileged'] = False

    def postTrue(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['HostConfig']['Privileged']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    def postFalse(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert not docker_inspect['HostConfig']['Privileged']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=preTrue, post_func=postTrue)
    event_test(agent, schema, pre_func=preFalse, post_func=postFalse)


def test_instance_restart_policy(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    expected_restart_pol_1 = {"maximumRetryCount": 0,
                              "name": "always"}
    expected_restart_pol_2 = {"name": "on-failure",
                              "maximumRetryCount": 2,
                              }
    expected_restart_pol_3 = {"name": "always"}

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['restartPolicy'] = expected_restart_pol_1

    def pre_failure_policy(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['restartPolicy'] = expected_restart_pol_2

    def pre_name_policy(req):
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['restartPolicy'] = expected_restart_pol_3

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        act_restart_pol = docker_inspect['HostConfig']['RestartPolicy']
        assert act_restart_pol['Name'] == expected_restart_pol_1['name']
        assert act_restart_pol['MaximumRetryCount'] == expected_restart_pol_1[
            'maximumRetryCount']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    def post_failure_policy(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        act_restart_pol = docker_inspect['HostConfig']['RestartPolicy']
        assert act_restart_pol['Name'] == expected_restart_pol_2['name']
        assert act_restart_pol['MaximumRetryCount'] == expected_restart_pol_2[
            'maximumRetryCount']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    def post_name_policy(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        act_restart_pol = docker_inspect['HostConfig']['RestartPolicy']
        assert act_restart_pol['Name'] == expected_restart_pol_3['name']
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)
    event_test(agent, schema, pre_func=pre_failure_policy,
               post_func=post_failure_policy)
    event_test(agent, schema, pre_func=pre_name_policy,
               post_func=post_name_policy)


def test_instance_activate_cpu_shares(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['cpuShares'] = 400

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        # assert docker_inspect['Config']['CpuShares'] == 400
        assert docker_inspect['HostConfig']['CpuShares'] == 400
        container_field_test_boiler_plate(resp)

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    schema = 'docker/instance_activate_fields'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_instance_activate_ipsec(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp, valid_resp):
        instance_activate_common_validation(resp)

        del valid_resp['previousNames']

    event_test(agent, 'docker/instance_activate_ipsec', post_func=post)


def test_instance_activate_agent_instance_localhost(agent):
    CONFIG_OVERRIDE['CONFIG_URL'] = 'https://localhost:1234/a/path'
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp):
        id = resp['data']['instanceHostMap']['instance']
        id = id['+data']['dockerContainer']['Id']
        inspect = docker_client().inspect_container(id)
        instance_activate_common_validation(resp)

        port = Config.api_proxy_listen_port()
        assert 'CATTLE_CONFIG_URL_SCHEME=https' in inspect['Config']['Env']
        assert 'CATTLE_CONFIG_URL_PATH=/a/path' in inspect['Config']['Env']
        assert 'CATTLE_CONFIG_URL_PORT={0}'.format(port) in \
            inspect['Config']['Env']


def test_instance_activate_agent_instance(agent):
    CONFIG_OVERRIDE['CONFIG_URL'] = 'https://something.fake:1234/a/path'
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp):
        id = resp['data']['instanceHostMap']['instance']
        id = id['+data']['dockerContainer']['Id']

        inspect = docker_client().inspect_container(id)
        instance_activate_common_validation(resp)

        port = Config.api_proxy_listen_port()
        assert 'CATTLE_CONFIG_URL={0}'.format(Config.config_url()) in \
               inspect['Config']['Env']
        assert 'CATTLE_CONFIG_URL_SCHEME=https' not in inspect['Config']['Env']
        assert 'CATTLE_CONFIG_URL_PATH=/a/path' not in inspect['Config']['Env']
        assert 'CATTLE_CONFIG_URL_PORT={0}'.format(port) not in \
               inspect['Config']['Env']
        assert 'ENV1=value1' in inspect['Config']['Env']


def test_instance_activate_start_fails(agent):
    delete_container('/r-start-fails')
    start_fails(agent)
    container = get_container('/r-start-fails')
    assert container is None


def start_fails(agent):
    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['name'] = 'start-fails'
        instance['uuid'] = 'start-fails'
        instance['data']['fields']['command'] = ["willfail"]

    # with pytest.raises(APIError):
    event_test(agent, 'docker/instance_activate',
               pre_func=pre, diff=False)


def test_instance_activate_volumes(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    delete_container('/target_volumes_from_by_uuid')
    delete_container('/target_volumes_from_by_id')

    client = docker_client()
    labels = {'io.rancher.container.uuid': 'target_volumes_from_by_uuid'}
    c = client.create_container('ibuildthecloud/helloworld',
                                volumes=['/volumes_from_path_by_uuid'],
                                labels=labels,
                                name='target_volumes_from_by_uuid')
    client.start(c)

    c2 = client.create_container('ibuildthecloud/helloworld',
                                 volumes=['/volumes_from_path_by_id'],
                                 name='target_volumes_from_by_id')
    client.start(c2)

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['dataVolumesFromContainers'][1]['externalId'] = c2['Id']

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        inspect = instance_data['dockerInspect']

        assert inspect['Config']['Volumes']['/host/proc'] is not None
        assert inspect['Config']['Volumes']['/host/sys'] is not None
        assert inspect['Config']['Volumes']['/random'] is not None
        # volumes from outside container can only be shown in Mount point
        # assert inspect['Config']['Volumes']
        # ['/volumes_from_path_by_uuid'] is not None
        # assert inspect['Config']['Volumes']
        # ['/volumes_from_path_by_id'] is not None

        assert len(inspect['Mounts']) == 6

        '''
        assert inspect['VolumesRW'] == {
            '/host/proc': True,
            '/host/sys': False,
            '/random': True,
            '/volumes_from_path_by_uuid': True,
            '/volumes_from_path_by_id': True,
            '/slave_test': True,
        }
        '''

        assert {'/sys:/host/sys:ro',
                '/proc:/host/proc:rw',
                '/slave_test:/slave_test:Z'} \
            == set(inspect['HostConfig']['Binds'])

        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate_volumes', pre_func=pre,
               post_func=post)


def test_instance_activate_null_command(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp):
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate_command_null', post_func=post)


def test_instance_activate_command(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp):
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate_command', post_func=post)


def test_instance_activate_command_args(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp):
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate_command_args', post_func=post)


def test_instance_activate_labels(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp):
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate_labels',
               post_func=post)


def test_instance_deactivate(agent):
    instance_only_activate(agent)

    def post(req, resp, valid_resp):
        container_field_test_boiler_plate(resp)

        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    start = time.time()
    event_test(agent, 'docker/instance_deactivate', post_func=post)
    end = time.time()

    # assert end - start < 1.5

    def pre(req):
        req['data']['processData']['timeout'] = 1

    instance_only_activate(agent)
    start = time.time()
    event_test(agent, 'docker/instance_deactivate', pre_func=pre,
               post_func=post)
    end = time.time()

    assert end - start > 1


def test_instance_activate_ipsec_network_agent(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp, valid_resp):
        instance_activate_common_validation(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    event_test(agent, 'docker/instance_activate_ipsec_network_agent',
               post_func=post)


def test_instance_activate_ipsec_lb_agent(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def post(req, resp, valid_resp):
        instance_activate_common_validation(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    event_test(agent, 'docker/instance_activate_ipsec_lb_agent',
               post_func=post)


def test_instance_force_stop(agent):
    delete_container('/force-stop-test')

    client = docker_client()
    c = client.create_container('ibuildthecloud/helloworld',
                                name='force-stop-test')
    client.start(c)
    inspect = client.inspect_container(c)
    assert inspect['State']['Running'] is True

    def pre(req):
        req['data']['instanceForceStop']['id'] = c['Id']

    def post(req, resp):
        inspect = client.inspect_container(c)
        assert inspect['State']['Running'] is False

    event_test(agent, 'docker/instance_force_stop',
               pre_func=pre, post_func=post, diff=False)

    # Assert that you can call on a stop container without issue
    event_test(agent, 'docker/instance_force_stop',
               pre_func=pre, post_func=post, diff=False)

    # And a non-existent one
    client.remove_container(c)
    event_test(agent, 'docker/instance_force_stop', pre_func=pre, diff=False)


def test_instance_remove(agent):
    instance_only_activate(agent)
    container = get_container('/r-test')
    assert container is not None

    def post(req, resp, valid_resp):
        c = get_container('/r-test')
        assert c is None
        del valid_resp['data']
        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/instance_remove', post_func=post)

    # Test finding and removing by externalId instead of uuid
    instance_only_activate(agent)
    container = get_container('/r-test')
    assert container is not None

    def pre(req):
        req['data']['instanceHostMap']['instance']['externalId'] = container[
            'Id']
        req['data']['instanceHostMap']['instance']['uuid'] = 'wont be found'

    def post(req, resp, valid_resp):
        c = get_container('/r-test')
        assert c is None
        del valid_resp['data']
        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/instance_remove', pre_func=pre, post_func=post)


def test_instance_links_net_host(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    delete_container('/target_redis')
    delete_container('/target_mysql')

    client = docker_client()
    c = client.create_container('ibuildthecloud/helloworld',
                                ports=[(3307, 'udp'), (3306, 'tcp')],
                                name='target_mysql')
    client.start(c, port_bindings={
        '3307/udp': ('127.0.0.2', 12346),
        '3306/tcp': ('127.0.0.2', 12345)
    })

    c = client.create_container('ibuildthecloud/helloworld',
                                name='target_redis')
    client.start(c)

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['nics'][0]['network']['kind'] = 'dockerHost'

    def post(req, resp, valid_resp):
        id = resp['data']['instanceHostMap']['instance']
        id = id['+data']['dockerContainer']['Id']
        inspect = docker_client().inspect_container(id)
        assert inspect['HostConfig']['Links'] is None

        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    event_test(agent, 'docker/instance_activate_links_no_service',
                      pre_func=pre, post_func=post, diff=False)


def test_volume_delete_orphaning(agent):
    # This test emulates the situatoin we've seen in docker 1.10 where we need
    # to delete a volume but docker's ref count is off and it won't let us
    # delete it. In this scenario, we'll now just orphan the volume and return
    # success
    delete_container('/orphan_test')
    vol_name = 'orphan_test_vol'
    delete_volume(vol_name)

    v = DockerConfig.storage_api_version()
    docker_client(version=v).create_volume(vol_name, 'local')

    client = docker_client()
    c = client.create_container('ibuildthecloud/helloworld',
                                name='orphan_test',
                                host_config=client.create_host_config(
                                    binds=['%s:/tmp/1' % vol_name]))
    client.start(c)

    def pre(req):
        vol = req['data']['volumeStoragePoolMap']['volume']
        vol['name'] = vol_name
        vol['data'] = {'fields': {'driver': 'local'}}
        vol['uri'] = 'local:///%s' % vol_name

    def post(req, resp, valid_resp):
        found_vol = docker_client(version=v).inspect_volume(vol_name)
        assert found_vol is not None
        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/volume_remove', pre_func=pre, post_func=post)


def test_volume_from_data_volume_mounts_with_opt(agent, request):
    driver_opts = JsonObject(
        {'foo': 'bar'}
    )
    volumes_from_data_volume_mounts_test(agent, request,
                                         driver_opts=driver_opts)


def test_volume_from_data_volume_mounts(agent, request):
    volumes_from_data_volume_mounts_test(agent, request)


def test_volume_from_data_volume_mounts_empty_opts(agent, request):
    volumes_from_data_volume_mounts_test(agent, request,
                                         driver_opts=JsonObject({}))


def volumes_from_data_volume_mounts_test(agent, request,
                                         driver_opts=None):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    delete_container('/convoy')
    client = docker_client(version='1.21')
    dr = 'convoytest'
    _launch_convoy_container(client, dr)

    vol_name = 'test-vol1'

    # Doing redundant cleanup as a finalizer because things can get weird if
    # volume drivers just disappear while volumes for it are still around
    def remove_vol():
        delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
        client.remove_volume(vol_name)
        delete_container('/convoy')
    request.addfinalizer(remove_vol)

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['dataVolumes'] = ['%s:/con/path' % vol_name]
        mounts = [JsonObject(
            {
                'name': vol_name,
                'data': {
                    'fields': {
                        'driver': dr,
                        'driverOpts': driver_opts,
                    },
                },
            })]
        instance['volumesFromDataVolumeMounts'] = mounts

    def post(req, resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        mounts = instance_data['dockerMounts']
        assert len(mounts) == 1
        assert mounts[0]['Name'] == vol_name
        assert mounts[0]['Driver'] == dr

    event_test(agent, 'docker/instance_activate', pre_func=pre, post_func=post,
               diff=False)


def _launch_convoy_container(client, dr):
    client.pull('cjellick/convoy-local', 'v0.4.3-longhorn-2')
    container = client. \
        create_container('cjellick/convoy-local:v0.4.3-longhorn-2',
                         name='/convoy',
                         environment={
                             'CONVOY_SOCKET': '/var/run/%s.sock' % dr,
                             'CONVOY_DATA_DIR': '/tmp/%s' % dr,
                             'CONVOY_DRIVER_NAME': '%s' % dr},
                         volumes=['/var/run/',
                                  '/etc/docker/plugins',
                                  '/tmp/%s' % dr],
                         host_config=client.
                         create_host_config(privileged=True, binds=[
                             '/var/run:/var/run',
                             '/etc/docker/plugins/:/etc/docker/plugins',
                             '/tmp/%s:/tmp/%s' % (dr, dr)])
                         )
    client.start(container)
    return container


def test_volume_activate(agent):

    def post(req, resp, valid_resp):
        del resp['links']
        del resp['actions']
        del valid_resp['previousNames']

    event_test(agent, 'docker/volume_activate', post_func=post)


def test_volume_activate_driver1(agent):
    def pre(req):
        vol = req['data']['volumeStoragePoolMap']['volume']
        vol['data'] = {'fields': {'driver': 'local',
                                  'driverOpts': None}}
        vol['name'] = 'test_vol'

    def post(req, resp, valid_resp):
        v = DockerConfig.storage_api_version()
        vol = docker_client(version=v).inspect_volume('test_vol')
        assert vol['Driver'] == 'local'
        assert vol['Name'] == 'test_vol'
        docker_client(version=v).remove_volume('test_vol')

        del resp['links']
        del resp['actions']
        del valid_resp['previousNames']

    event_test(agent, 'docker/volume_activate', pre_func=pre, post_func=post)


def test_volume_activate_driver2(agent):
    def pre(req):
        vol = req['data']['volumeStoragePoolMap']['volume']
        vol['data'] = {'fields': {'driver': 'local',
                                  'driverOpts': {'size': '10G'}}}
        vol['name'] = 'test_vol'

    def post(req, resp, valid_resp):
        v = DockerConfig.storage_api_version()
        vol = docker_client(version=v).inspect_volume('test_vol')
        assert vol['Driver'] == 'local'
        assert vol['Name'] == 'test_vol'
        docker_client(version=v).remove_volume('test_vol')

        del resp['links']
        del resp['actions']
        del valid_resp['previousNames']

    event_test(agent, 'docker/volume_activate', pre_func=pre, post_func=post)


def test_instance_activate_volume_driver(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['volumeDriver'] = 'local'

    def post(req, resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        if newer_than('1.19'):
            if newer_than('1.21'):
                assert docker_inspect['HostConfig']['VolumeDriver'] == 'local'
            else:
                assert docker_inspect['Config']['VolumeDriver'] == 'local'
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate', pre_func=pre, post_func=post)


def test_volume_remove_driver(agent):
    def pre(req):
        v = DockerConfig.storage_api_version()
        docker_client(version=v).create_volume('test_vol',
                                               'local')
        vol = req['data']['volumeStoragePoolMap']['volume']
        vol['data'] = {'fields': {'driver': 'local',
                                  'driverOpts': {'size': '10G'}}}
        vol['name'] = 'test_vol'
        vol['uri'] = 'local:///test_vol'

    def post(req, resp, valid_resp):
        v = DockerConfig.storage_api_version()
        with pytest.raises(APIError) as e:
            docker_client(version=v).inspect_volume('test_vol')
        assert e.value.explanation == 'no such volume' or \
            e.value.explanation == 'get test_vol: no such volume'

        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/volume_remove', pre_func=pre, post_func=post)


def test_ping(agent, pull_images, mocker):
    mocker.patch.object(HostInfo, 'collect_data',
                        return_value=json_data('docker/host_info_resp'))

    client = docker_client()

    delete_container('/named-running')
    delete_container('/named-stopped')
    delete_container('/named-created')
    delete_container('/named-system')
    delete_container('/named-sys-nover')
    delete_container('/named-agent-instance')

    client.create_container('ibuildthecloud/helloworld',
                            name='named-created', labels={
                                'io.rancher.container.uuid': 'uuid-created'})
    running = client.create_container('ibuildthecloud/helloworld:latest',
                                      name='named-running', labels={
                                          'io.rancher.container.uuid':
                                          'uuid-running'})
    client.start(running)
    stopped = client.create_container('ibuildthecloud/helloworld:latest',
                                      name='named-stopped', labels={
                                          'io.rancher.container.uuid':
                                          'uuid-stopped'})
    client.start(stopped)
    client.kill(stopped, signal='SIGKILL')

    system_con = client.create_container('rancher/agent:v0.7.9',
                                         name='named-system', labels={
                                             'io.rancher.container.uuid':
                                             'uuid-system'})
    client.start(system_con)
    client.kill(system_con, signal='SIGKILL')

    sys_nover = client.create_container('rancher/agent',
                                        name='named-sys-nover', labels={
                                            'io.rancher.container.uuid':
                                            'uuid-sys-nover'})
    client.start(sys_nover)
    client.kill(sys_nover, signal='SIGKILL')

    agent_inst_con = client.create_container(
        'ibuildthecloud/helloworld:latest',
        name='named-agent-instance',
        labels={
            'io.rancher.container.uuid':
                'uuid-agent-instance',
            'io.rancher.container.system':
                'networkAgent'},
        command='true')
    client.start(agent_inst_con)
    client.kill(agent_inst_con, signal='SIGKILL')

    CONFIG_OVERRIDE['DOCKER_UUID'] = 'testuuid'
    CONFIG_OVERRIDE['PHYSICAL_HOST_UUID'] = 'hostuuid'

    event_test(agent, 'docker/ping', post_func=ping_post_process)


def ping_post_process(req, resp, valid_resp):
    resources = resp['data']['resources']

    labels = {'io.rancher.host.docker_version': '1.6',
              'io.rancher.host.linux_kernel_version': '4.1'}

    uuids = {'uuid-running': 0, 'uuid-stopped': 1, 'uuid-created': 2,
             'uuid-system': 3, 'uuid-sys-nover': 4, 'uuid-agent-instance': 5}
    instances = []
    for r in resources:
        if r['type'] == 'host':
            if platform.system() == 'Linux':
                # check whether the system is Linux.
                # If so, execute the test script below
                if 'io.rancher.host.kvm' in r['labels']:
                    assert r['labels']['io.rancher.host.kvm'] == 'true'
                    del r['labels']['io.rancher.host.kvm']
                assert len(r['labels']) == 2
            r['labels'] = labels
            r['info'] = HostInfo.collect_data()
            r['physicalHostUuid'] = 'hostuuid'
            r['uuid'] = 'testuuid'
        if r['type'] == 'storagePool':
            r['hostUuid'] = 'testuuid'
            r['uuid'] = 'testuuid-pool'
        if r['type'] == 'instance' and r['uuid'] in uuids:
            if r['uuid'] == 'uuid-running':
                assert r['state'] == 'running'
            elif r['uuid'] in ['uuid-stopped', 'uuid-agent-instance']:
                assert r['state'] == 'stopped'
            elif r['uuid'] == 'uuid-system' or r['uuid'] == 'uuid-sys-nover':
                assert r['state'] == 'stopped'
                # Account for docker 1.7/1.8 difference
                try:
                    del r['labels']['io.rancher.container.system']
                except KeyError:
                    pass

            # Account for docker 1.6 where ':latest' is appended
            if r['uuid'] == 'uuid-sys-nover' and r[
                    'image'] == 'rancher/agent:latest':
                r['image'] = 'rancher/agent'

            assert r['dockerId'] is not None
            del r['dockerId']
            assert r['created'] is not None
            del r['created']
            if r['systemContainer'] == '':
                r['systemContainer'] = None
            instances.append(r)

    def ping_sort(item):
        return uuids[item['uuid']]

    instances.sort(key=ping_sort)

    assert len(instances) == 5

    resources = filter(lambda x: x.get('kind') == 'docker', resources)
    resources += instances
    resp['data']['resources'] = resources
    assert_ping_stat_resources(resp)
    del valid_resp['previousNames']
    del resp['links']
    del resp['actions']


def assert_ping_stat_resources(resp):
    hostname = Config.hostname()
    pool_name = hostname + ' Storage Pool'
    assert resp['data']['resources'][0]['hostname'] == hostname
    assert resp['data']['resources'][1]['name'] == pool_name
    resp['data']['resources'][0]['hostname'] = 'localhost'
    resp['data']['resources'][1]['name'] = 'localhost Storage Pool'


# @pytest.skip("this test doesn't make sense to go agent")
# def test_ping_stat_exception(agent, mocker):
#     mocker.patch.object(HostInfo, 'collect_data',
#                         side_effect=ValueError('Bad Value Found'))
#
#     CONFIG_OVERRIDE['DOCKER_UUID'] = 'testuuid'
#     CONFIG_OVERRIDE['PHYSICAL_HOST_UUID'] = 'hostuuid'
#
#     event_test(agent, 'docker/ping_stat_exception',
#                post_func=ping_post_process_state_exception)
#
#
# def ping_post_process_state_exception(req, resp, valid_resp):
#     labels = {'io.rancher.host.docker_version': '1.6',
#               'io.rancher.host.linux_kernel_version': '4.1'}
#
#     # This filters down the returned resources to just the stat-based ones.
#     # In other words, it gets rid of all containers from the response.
#     resp['data']['resources'] = filter(lambda x: x.get('kind') == 'docker',
#                                        resp['data']['resources'])
#     for r in resp['data']['resources']:
#         if r['type'] == 'host':
#             if platform.system() == 'Linux':
#                 # check whether the system is Linux.
#                 # If so, execute the test script below
#                 if 'io.rancher.host.kvm' in r['labels']:
#                     assert r['labels']['io.rancher.host.kvm'] == 'true'
#                     del r['labels']['io.rancher.host.kvm']
#                 assert len(r['labels']) == 2
#             r['labels'] = labels
#             try:
#                 r['info'] = HostInfo.collect_data()
#             except:
#                 pass
#             assert len(r['info']['osInfo']) == 0
#             assert len(r['info']['memoryInfo']) == 0
#             assert len(r['info']['diskInfo']['fileSystem']) == 0
#             assert len(r['info']['diskInfo']['mountPoints']) == 0
#         assert len(r['info']['diskInfo']['dockerStorageDriverStatus']) == 0
#             assert len(r['info']['cpuInfo']) == 0
#             assert len(r['info']['iopsInfo']) == 0
#             r['info'] = None
#             r['physicalHostUuid'] = 'hostuuid'
#             r['uuid'] = 'testuuid'
#
#         if r['type'] == 'storagePool':
#             r['hostUuid'] = 'testuuid'
#             r['uuid'] = 'testuuid-pool'
#
#     assert_ping_stat_resources(resp)
#     del valid_resp['previousNames']
#     del resp['links']
#     del resp['actions']


# new added test case for go agent
def test_env_variable(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        fields = instance['data']['fields']
        fields['environment'] = {'foo': 'bar'}

    def post(req, resp, valid_resp):
        instance_activate_assert_host_config(resp)
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert 'foo=bar' in docker_inspect['Config']['Env']
        instance_activate_common_validation(resp)

    schema = 'docker/instance_activate'
    event_test(agent, schema, pre_func=pre, post_func=post)

