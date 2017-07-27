from docker.errors import APIError

from .common import docker_client, instance_only_activate


def test_instance_activate_need_pull_image(agent):
    try:
        docker_client().remove_image('ibuildthecloud/helloworld:latest')
    except APIError:
        pass

    instance_only_activate(agent)
