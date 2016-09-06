from common import delete_container, docker_client, event_test


def test_inspect_by_name(agent):
    delete_container('/inspect_test')

    client = docker_client()
    c = client.create_container('ibuildthecloud/helloworld',
                                name='inspect_test')
    inspect = docker_client().inspect_container(c['Id'])

    def post(req, resp):
        response_inspect = resp['data']['instanceInspect']
        # diff_dict(inspect, response_inspect)
        assert response_inspect['Id'] == inspect['Id']
        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/instance_inspect',
               post_func=post, diff=False)


def test_inspect_by_id(agent):
    delete_container('/inspect_test')

    client = docker_client()
    c = client.create_container('ibuildthecloud/helloworld',
                                name='inspect_test')
    inspect = docker_client().inspect_container(c['Id'])

    def pre(req):
        instance_inspect = req['data']['instanceInspect']
        instance_inspect['id'] = c['Id']
        del instance_inspect['name']

    def post(req, resp, valid_resp):
        response_inspect = resp['data']['instanceInspect']

        # can't compare the inspect from go api and py api
        # TODO find a new way to assert
        assert response_inspect['Id'] == inspect['Id']
        # diff_dict(inspect, response_inspect)

    event_test(agent, 'docker/instance_inspect', pre_func=pre,
               post_func=post, diff=False)


def test_inspect_not_found(agent):
    delete_container('/inspect_test')

    def post(req, resp):
        assert "Id" not in resp['data']['instanceInspect']
        assert "Name" not in resp['data']['instanceInspect']

    event_test(agent, 'docker/instance_inspect', post_func=post, diff=False)
