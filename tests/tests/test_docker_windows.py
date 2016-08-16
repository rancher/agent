from .common import event_test, delete_container, \
    instance_activate_common_validation


def test_docker_windows_no_names(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    def pre(req):
        instance = req['data']['instanceHostMap']['instance']
        instance['data']['fields']['imageUuid'] = 'microsoft/iis'
        instance['image']['data']['dockerImage']['fullName'] = 'microsoft/iis'

    def post(req, resp, valid_resp):
        data = valid_resp['data']['instanceHostMap']['instance']['+data']
        docker_con = data['dockerContainer']
        docker_con['Names'] = ['/c861f990-4472-4fa1-960f-65171b544c28']
        instance_activate_common_validation(resp)

    schema = 'docker/instance_activate'
    event_test(agent, schema, pre_func=pre, post_func=post)
