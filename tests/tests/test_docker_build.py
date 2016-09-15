from tests.common import delete_container, JsonObject, \
    instance_activate_common_validation, event_test
from tests.cattle.utils import random_string


def _test_docker_build_from_remote(agent, remote=None,
                                   context=None):

    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    image_uuid = 'image-' + random_string(12)

    def pre(req):
        instance = req.data.instanceHostMap.instance
        # tag is not on the instance, only the image
        instance.data.fields['build'] = JsonObject({
            'remote': remote,
            'context': context,
        })
        instance.data.fields.imageUuid = image_uuid
        instance.image.data['fields'] = JsonObject({
            'build': {
                'remote': remote,
                'context': context,
                'tag': image_uuid,
            },
        })

    def post(req, resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        image = docker_inspect['Config']['Image']
        assert image_uuid + ":latest" == image
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate', diff=False,
               pre_func=pre, post_func=post)

    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')


def test_docker_build_from_github(agent):
    remotes = [
        'github.com/rancherio/tiny-build',
        'git://github.com/rancherio/tiny-build',
        'git://github.com/rancherio/tiny-build.git',
        'git@github.com:rancherio/tiny-build.git',
    ]

    for remote in remotes:
        _test_docker_build_from_remote(agent, remote)


def test_docker_build_from_context(agent):
    url = 'https://github.com/rancherio/tiny-build/raw/master/build.tar'
    _test_docker_build_from_remote(agent, context=url)
