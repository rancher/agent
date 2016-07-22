from docker.errors import APIError
from .common import docker_client, event_test, instance_only_activate, \
    delete_container, json_data, random_str


def test_image_activate(agent):
    try:
        docker_client().remove_image('ibuildthecloud/helloworld:latest')
    except APIError:
        pass

    def post(req, resp):
        del resp['links']
        del resp['actions']
    event_test(agent, 'docker/image_activate', post_func=post)


def test_instance_activate_need_pull_image(agent):
    try:
        docker_client().remove_image('ibuildthecloud/helloworld:latest')
    except APIError:
        pass

    instance_only_activate(agent)


def test_image_activate_no_reg_cred_pull_image(agent):
    try:
        docker_client().remove_image('ibuildthecloud/helloworld:latest')
    except APIError:
        pass

    def pre(req):
        image = req['data']['imageStoragePoolMap']['image']
        image['registryCredential'] = None

    def post(req, resp):
        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/image_activate', pre_func=pre, post_func=post)


def test_image_pull_variants(agent):
    image_names = [
        'ibuildthecloud/helloworld:latest',
        'ibuildthecloud/helloworld',
        'tianon/true',
        'tianon/true:latest',
        # 'registry.rancher.io/rancher/scratch', Need to make our registry
        # 'registry.rancher.io/rancher/scratch:latest', Support non-authed
        # 'registry.rancher.io/rancher/scratch:new_stuff',  pulls.
        'cirros',
        'cirros:latest',
        'cirros:0.3.3',
        'docker.io/tianon/true',
        'docker.io/library/cirros',
        'docker.io/cirros',
        'index.docker.io/tianon/true',
        'index.docker.io/library/cirros',
        'index.docker.io/cirros',
        'rocket.chat',
        'rocket.chat:latest',
        'docker.io/rocket.chat',
        'index.docker.io/rocket.chat',
        'index.docker.io/rocket.chat:latest'
    ]

    for i in image_names:
        _pull_image_by_name(agent, i)


def _pull_image_by_name(agent, image_name):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    try:
        docker_client().remove_image(image_name, noprune=True)
    except APIError:
        pass

    def pre(req):
        image = req['data']['imageStoragePoolMap']['image']
        remap_dockerImage(image, image_name)

    event_test(agent, 'docker/image_activate', pre_func=pre, diff=False)


def remap_dockerImage(dockerImage, image_name):
    image = dockerImage
    parsed = _parse_repo_tag(image_name)
    image['name'] = parsed['fullName']
    image['uuid'] = 'docker:' + parsed['fullName']
    image['data']['dockerImage']['fullName'] = parsed['fullName']
    image['data']['dockerImage']['server'] = parsed['server']
    image['data']['dockerImage']['repository'] = parsed['repository']
    image['data']['dockerImage']['lookUpName'] = parsed['lookUpName']
    image['data']['dockerImage']['qualifiedName'] = parsed['qualifiedName']
    image['data']['dockerImage']['namespace'] = parsed['namespace']
    image['data']['dockerImage']['tag'] = parsed['tag']


def _parse_repo_tag(image):
    namespace = None
    repo = None
    tag = None
    server = 'index.docker.io'
    if image is None:
        return None
    forwardSlash = image.split("/")
    if len(forwardSlash) <= 3:
        if len(forwardSlash) == 1:
            split2 = forwardSlash[0].split(":")
            if len(split2) == 1:
                tag = "latest"
                repo = image
            elif len(split2) == 2:
                tag = split2[1]
                repo = split2[0]
        elif len(forwardSlash) == 2:
            first = forwardSlash[0]
            second = forwardSlash[1].split(":")
            if '.' in first or ':' in first or \
                    'localhost' in first and first != 'docker.io':
                server = first
            else:
                namespace = first
            if len(second) == 2:
                repo = second[0]
                tag = second[1]
            else:
                repo = forwardSlash[1]
                tag = 'latest'
        elif len(forwardSlash) == 3:
            server = forwardSlash[0]
            namespace = forwardSlash[1]
            split2 = forwardSlash[2].split(':')
            if len(split2) == 1:
                repo = forwardSlash[2]
                tag = 'latest'
            else:
                repo = split2[0]
                tag = split2[1]
        else:
            return None
    if namespace is not None:
        lookUpName = namespace + '/' + repo
    else:
        lookUpName = repo

    if server == "index.docker.io" or server == "docker.io":
        if namespace is None:
            qualifiedName = repo
        else:
            qualifiedName = namespace + "/" + repo

    else:
        if namespace is None:
            qualifiedName = server + "/" + repo
        else:
            qualifiedName = server + "/" + namespace + "/" + repo
    if server == "docker.io":
        server = "index.docker.io"

    return dict(repository=repo,
                lookUpName=lookUpName,
                server=server,
                namespace=namespace,
                tag=tag,
                fullName=image,
                qualifiedName=qualifiedName)


def _test_image_pull_credential(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    image_name = 'registry.rancher.io/rancher/loop'

    try:
        docker_client().remove_image(image_name)
    except APIError:
        pass

    def pre(req):
        image = req['data']['imageStoragePoolMap']['image']
        remap_dockerImage(image, image_name)
        image['registryCredential'] = {
            'publicValue': 'rancher',
            'secretValue': 'rancher',
            'data': {
                'fields': {
                    'email': 'test@rancher.com',
                }
            },
            'registry': {
                'data': {
                    'fields': {
                        'serverAddress': 'registry.rancher.io'
                    }
                }
            }
        }

    def post(req, resp):
        responseImage = resp['data']['imageStoragePoolMap']['+data']
        responseImage = responseImage['dockerImage']
        correct = False
        sent_parsed = _parse_repo_tag(image_name)
        for resp_img_uuid in responseImage['RepoTags']:
            parsed_name = _parse_repo_tag(resp_img_uuid)
            assert parsed_name['repository'] == sent_parsed['repository']
            if sent_parsed['tag'] != '':
                if sent_parsed['tag'] == 'latest':
                    if parsed_name['tag'] is not None:
                        correct = True
                else:
                    if parsed_name['tag'] == sent_parsed['tag']:
                        correct = True
            else:
                correct = True
        assert correct is True

    event_test(agent, 'docker/image_activate', pre_func=pre, post_func=post,
               diff=False)


# TODO ADD ASSERTIONS TO TEST
def test_image_pull_invalid_credential(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    image_name = 'tianon/true'

    try:
        docker_client().remove_image(image_name)
    except APIError:
        pass

    def pre(req):
        image = req['data']['imageStoragePoolMap']['image']
        remap_dockerImage(image, image_name)
        image['registryCredential'] = {
            'publicValue': random_str(),
            'secretValue': random_str(),
            'data': {
                'fields': {
                    'email': 'test@rancher.com',
                }
            },
            'registry': {
                'data': {
                    'fields': {
                        'serverAddress': 'index.docker.io'
                    }
                }
            }
        }
    req = json_data('docker/image_activate')
    pre(req)
    '''
    if newer_than('1.22'):
        error_class = APIError
    else:
        error_class = ImageValidationError
    '''
    # with pytest.raises(error_class) as e:
    agent.execute(req)
    # assert 'auth' in str(e.value.message).lower()


# TODO ADD ASSERTIONS TO TEST
def test_image_pull_invalid_image(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    image_name = random_str() + random_str()

    def pre(req):
        image = req['data']['imageStoragePoolMap']['image']
        remap_dockerImage(image, image_name)
    req = json_data('docker/image_activate')
    pre(req)
    # with pytest.raises(ImageValidationError) as e:
    agent.execute(req)
    # assert 'not found' in e.value.message


def test_image_activate_no_op(agent):
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')
    repo = 'ubuntu'
    tag = '10.04'
    image_name = repo + ':' + tag
    client = docker_client()
    try:
        client.remove_image(image_name)
    except APIError:
        pass

    def pre(req):
        image = req['data']['imageStoragePoolMap']['image']
        remap_dockerImage(image, image_name)
        image['data']['processData'] = {}
        image['data']['processData']['containerNoOpEvent'] = True

    def post(req, resp):
        images = client.images(name=repo)
        for i in images:
            for t in i['RepoTags']:
                assert tag not in t
        assert not resp['data']['imageStoragePoolMap']
        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/image_activate', pre_func=pre,
               post_func=post, diff=False)
