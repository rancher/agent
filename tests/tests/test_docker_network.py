from common import delete_container, event_test, trim, docker_client, \
    JsonObject


def test_network_mode_none(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['nics'][0]['network']['kind'] = 'dockerNone'
        instance['hostname'] = 'nameisset'

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert docker_inspect['Config']['NetworkDisabled']
        assert docker_inspect['HostConfig']['NetworkMode'] == 'none'
        assert docker_inspect['Config']['Hostname'] == 'nameisset'

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    event_test(agent, 'docker/instance_activate', pre_func=pre,
               post_func=post, diff=False)


def test_network_mode_host(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['nics'][0]['network']['kind'] = 'dockerHost'
        instance['hostname'] = 'nameisset'

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        # networkDisabled doesn't exist when mode is set to host
        assert 'NetworkDisabled' not in docker_inspect['Config']
        assert docker_inspect['HostConfig']['NetworkMode'] == 'host'
        assert docker_inspect['Config']['Hostname'] != 'nameisset'

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    event_test(agent, 'docker/instance_activate', pre_func=pre,
               post_func=post, diff=False)


def test_network_mode_container_with_mac_and_hostname(agent):
    delete_container('/network-container')
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    client = docker_client()
    c = client.create_container('ibuildthecloud/helloworld',
                                name='network-container')
    client.start(c)

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['hostname'] = 'no set'
        instance['nics'][0]['network']['kind'] = 'dockerContainer'
        instance['networkContainer'] = JsonObject({
            'uuid': 'network-container'
        })

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert 'MacAddress' not in docker_inspect['Config']
        assert docker_inspect['Config']['Hostname'] != 'no set'
        assert docker_inspect['HostConfig']['NetworkMode'] == \
            'container:{}'.format(c['Id'])

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    event_test(agent, 'docker/instance_activate', pre_func=pre,
               post_func=post, diff=False)


def test_network_mode_container(agent):
    delete_container('/network-container')
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    client = docker_client()
    c = client.create_container('ibuildthecloud/helloworld',
                                name='network-container')
    client.start(c)

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['nics'][0]['network']['kind'] = 'dockerContainer'
        instance['networkContainer'] = JsonObject({
            'uuid': 'network-container'
        })

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert 'NetworkDisabled' not in docker_inspect['Config']
        assert docker_inspect['HostConfig']['NetworkMode'] == \
            'container:{}'.format(c['Id'])

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    event_test(agent, 'docker/instance_activate', pre_func=pre,
               post_func=post, diff=False)


def test_network_mode_bridge(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['nics'][0]['network']['kind'] = 'dockerBridge'

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        docker_data = instance_data['dockerContainer']
        assert 'NetworkDisabled' not in docker_inspect['Config']
        assert len(docker_data['Ports']) == 1
        assert docker_data['Ports'][0]["PublicPort"] == 100

        docker_container = instance_data['dockerContainer']
        fields = instance_data['+fields']
        trim(docker_container, fields, resp, valid_resp)

    event_test(agent, 'docker/instance_activate_bridge', pre_func=pre,
               post_func=post, diff=False)
