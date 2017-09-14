#!/usr/bin/env python
import sys
import requests

from cattle import from_env

url = sys.argv[1]

r = requests.get(url)

if r.status_code == 200 and r.text.startswith('#!/bin/sh'):
    print url
    sys.exit(0)

r = requests.get(sys.argv[1])
try:
    url = r.headers['X-API-Schemas']
except KeyError:
    url = sys.argv[1]

client = from_env(url=url)

if not client.valid():
    print 'Invalid client'
    sys.exit(1)

if 'POST' not in client.schema.types['registrationToken'].collectionMethods:
    projects = client.list_project(uuid='adminProject')
    if len(projects) == 0:
        print 'Failed to find admin resource group'
        sys.exit(1)

    client = from_env(url=projects[0].links['schemas'])
    if not client.valid():
        print 'Invalid client'
        sys.exit(1)

clusters = client.list_cluster(removed_null=True)

if len(clusters) == 1:
    print clusters[0].registrationToken.registrationUrl
else:
    print url
