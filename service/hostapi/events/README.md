### A Note on testing from a Mac

Some of these tests actualy spin up containers and use nsenter to manipulate IPs. In a typical Mac/b2d setup, the won't pass out of the box. There's some setup you need to do:

```
boot2docker ssh
cp /Users/PATH/TO/PROJECT/host-api/src/github.com/rancher/host-api/scripts/lib/net-util.sh .
cp /Users/PATH/TO/PROJECT/python-agent/vendor/nsenter .
vi net-util.sh # change /bin/bash to /bin/sh. b2d doesnt have bash. This is a hack but works
cp net-util.sh nsenter /usr/local/bin/
sudo mkdir /host
sudo ln -s /proc/ /host/proc
```

TODO Write a script for this
