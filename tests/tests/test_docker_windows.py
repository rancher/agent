from .common import event_test, delete_container, \
    instance_activate_common_validation, DockerConfig, \
    docker_client, newer_than, APIError, instance_only_activate
import pytest
import time

docker_host = "tcp://192.168.42.175:2375"


def test_windows_volume_activate(agent):

    def post(req, resp):
        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/volume_activate', post_func=post)


def test_docker_windows_image1(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['imageUuid'] = 'microsoft/iis'
        instance['image']['data']['dockerImage']['fullName'] = 'microsoft/iis'

    def post(req, resp, valid_resp):
        data = resp['data']['instanceHostMap']['instance']['+data']
        docker_con = data['dockerContainer']
        assert docker_con['Image'] == 'microsoft/iis'
        instance_activate_common_validation(resp)

    schema = 'docker/instance_activate'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_docker_windows_image2(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['imageUuid'] = 'microsoft/sqlite'
        instance['image']['data']['dockerImage']['fullName'] = 'microsoft/sqlite'

    def post(req, resp, valid_resp):
        data = resp['data']['instanceHostMap']['instance']['+data']
        docker_con = data['dockerContainer']
        assert docker_con['Image'] == 'microsoft/sqlite'
        instance_activate_common_validation(resp)

    schema = 'docker/instance_activate'
    event_test(agent, schema, pre_func=pre, post_func=post)


def test_windows_volume_activate_driver1(agent):
    def pre(req):
        vol = req['data']['volumeStoragePoolMap']['volume']
        vol['data'] = {'fields': {'driver': 'local',
                                  'driverOpts': None}}
        vol['name'] = 'test_vol'

    def post(req, resp):
        v = DockerConfig.storage_api_version()
        vol = docker_client(version=v, base_url_override=docker_host).inspect_volume('test_vol')
        assert vol['Driver'] == 'local'
        assert vol['Name'] == 'test_vol'
        docker_client(version=v, base_url_override=docker_host).remove_volume('test_vol')

        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/volume_activate', pre_func=pre, post_func=post)


def test_volume_activate_driver2(agent):
    def pre(req):
        vol = req['data']['volumeStoragePoolMap']['volume']
        vol['data'] = {'fields': {'driver': 'local',
                                  'driverOpts': {'size': '10G'}}}
        vol['name'] = 'test_vol'

    def post(req, resp):
        v = DockerConfig.storage_api_version()
        vol = docker_client(version=v, base_url_override=docker_host).inspect_volume('test_vol')
        assert vol['Driver'] == 'local'
        assert vol['Name'] == 'test_vol'
        docker_client(version=v, base_url_override=docker_host).remove_volume('test_vol')

        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/volume_activate', pre_func=pre, post_func=post)


def test_volume_deactivate_driver(agent):
    def pre(req):
        v = DockerConfig.storage_api_version()
        docker_client(version=v, base_url_override=docker_host).create_volume('test_vol',
                                               'local')
        vol = req['data']['volumeStoragePoolMap']['volume']
        vol['data'] = {'fields': {'driver': 'local',
                                  'driverOpts': {'size': '10G'}}}
        vol['name'] = 'test_vol'

    def post(req, resp):
        v = DockerConfig.storage_api_version()
        vol = docker_client(version=v, base_url_override=docker_host).inspect_volume('test_vol')
        assert vol['Driver'] == 'local'
        assert vol['Name'] == 'test_vol'
        docker_client(version=v, base_url_override=docker_host).remove_volume('test_vol')
        del resp['data']['volumeStoragePoolMap']
        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/volume_deactivate', pre_func=pre, post_func=post)


def test_volume_deactivate(agent):
    def post(req, resp):
        del resp['data']['volumeStoragePoolMap']
        del resp['links']
        del resp['actions']
    event_test(agent, 'docker/volume_deactivate', post_func=post)


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
        docker_client(version=v, base_url_override=docker_host).create_volume('test_vol',
                                               'local')
        vol = req['data']['volumeStoragePoolMap']['volume']
        vol['data'] = {'fields': {'driver': 'local',
                                  'driverOpts': {'size': '10G'}}}
        vol['name'] = 'test_vol'
        vol['uri'] = 'local:///test_vol'

    def post(req, resp):
        v = DockerConfig.storage_api_version()
        with pytest.raises(APIError) as e:
            docker_client(version=v, base_url_override=docker_host).inspect_volume('test_vol')
        assert e.value.explanation == 'no such volume' or \
            e.value.explanation == 'get test_vol: no such volume'

        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/volume_remove', pre_func=pre, post_func=post)


def test_instance_deactivate(agent):
    instance_only_activate(agent)

    def post(req, resp, valid_resp):
       #  container_field_test_boiler_plate(resp)

        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        # trim(docker_container, fields, resp, valid_resp)

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
