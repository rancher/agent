from .common import event_test, delete_container, \
    instance_activate_common_validation


# new added test case for go agent
def test_env_variable(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instance']
        fields = instance['data']['fields']
        fields['environment'] = {'foo': 'bar'}

    def post(req, resp, valid_resp):
        instance_data = resp['data']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        assert 'foo=bar' in docker_inspect['Config']['Env']
        instance_activate_common_validation(resp)

    schema = 'docker/instance_activate'
    event_test(agent, schema, pre_func=pre, post_func=post)
