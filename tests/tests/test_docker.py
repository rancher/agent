from common import event_test, delete_container
import pytest


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


@pytest.mark.skip("Must finish implementing for this to pass")
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
        # instance_activate_common_validation(resp)

    schema = 'docker/instance_activate'
    event_test(agent, schema, pre_func=pre, post_func=post)
