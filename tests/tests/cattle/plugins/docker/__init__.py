import logging

from docker.utils import kwargs_from_env
from tests.cattle import default_value, Config

log = logging.getLogger('docker')

_ENABLED = True


class DockerConfig:
    def __init__(self):
        pass

    @staticmethod
    def docker_enabled():
        return default_value('DOCKER_ENABLED', 'true') == 'true'

    @staticmethod
    def docker_host_ip():
        return default_value('DOCKER_HOST_IP', Config.agent_ip())

    @staticmethod
    def docker_home():
        return default_value('DOCKER_HOME', '/var/lib/docker')

    @staticmethod
    def docker_uuid_file():
        def_value = '{0}/.docker_uuid'.format(Config.state_dir())
        return default_value('DOCKER_UUID_FILE', def_value)

    @staticmethod
    def docker_uuid():
        return Config.get_uuid_from_file('DOCKER_UUID',
                                         DockerConfig.docker_uuid_file())

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
    def docker_required():
        return default_value('DOCKER_REQUIRED', 'true') == 'true'

    @staticmethod
    def delegate_timeout():
        return int(default_value('DOCKER_DELEGATE_TIMEOUT', '120'))

    @staticmethod
    def use_boot2docker_connection_env_vars():
        use_b2d = default_value('DOCKER_USE_BOOT2DOCKER', 'false')
        return use_b2d.lower() == 'true'

    @staticmethod
    def is_host_pidns():
        return default_value('AGENT_PIDNS', 'container') == 'host'


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


try:
    from docker import Client
except:
    log.info('Disabling docker, docker-py not found')
    _ENABLED = False
