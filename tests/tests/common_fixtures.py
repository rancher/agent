import os
import random
import sys
import tests
import shutil
import pytest
from cattle import CONFIG_OVERRIDE
CONFIG_OVERRIDE['DOCKER_REQUIRED'] = 'false'  # NOQA

from os.path import dirname

_file = os.path.abspath(__file__)  # NOQA
sys.path.insert(0, dirname(dirname(_file)))  # NOQA

from cattle import plugins


plugins.load()


TEST_DIR = os.path.join(dirname(tests.__file__))
SCRATCH_DIR = os.path.join(TEST_DIR, 'scratch')

if os.path.exists(SCRATCH_DIR):
    shutil.rmtree(SCRATCH_DIR)
os.makedirs(SCRATCH_DIR)


@pytest.fixture(scope='session', autouse=True)
def scratch_dir(request):
    request.addfinalizer(
        lambda: shutil.rmtree(SCRATCH_DIR, ignore_errors=True))


def random_str():
    return 'test-{0}'.format(random_num())


def random_num():
    return random.randint(0, 1000000)
