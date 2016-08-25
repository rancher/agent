import pytest
import logging
import os
from os.path import dirname
import os.path
import requests
from subprocess import Popen
import time
from common import TEST_DIR, Agent

logging.basicConfig()
log = logging.getLogger("tests")
log.setLevel(logging.INFO)

PROJECT_DIR = dirname(dirname(TEST_DIR))
GOPATH_DIR = dirname(dirname(dirname(dirname(PROJECT_DIR))))


@pytest.fixture(scope='session', autouse=True)
def start_server(request):
    env = dict(os.environ)
    env["GOPATH"] = GOPATH_DIR
    Popen(["go", "run", os.path.join(dirname(TEST_DIR), "main.go")],
          env=env)

    def kill_server():
        try:
            requests.get("http://localhost:8089/die")
        except:
            pass
    request.addfinalizer(kill_server)

    wait = .25
    max_wait = 2
    max_tries = 15
    count = 0
    while True:
        try:
            requests.get("http://localhost:8089/ping")
        except:
            if count > max_tries:
                log.error("Timed out waiting for test event server")
                break
            count += 1
            log.info("Waiting %ss on test event server" % wait)
            time.sleep(wait)
            if wait < max_wait:
                wait *= 2
        else:
            break


@pytest.fixture(scope="module")
def agent():
    return Agent()
