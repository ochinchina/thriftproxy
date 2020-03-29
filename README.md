## thriftproxy

This is a golang implemented proxy for thrift binary protocol over TCP/IP.

## How to compile it

Download golang 1.14+, set your GOROOT and your GOPATH for the thriftproxy properly, like:

```shell
# cd ~
# wget https://dl.google.com/go/go1.14.1.linux-amd64.tar.gz
# tar -zxvf go1.14.1.linux-amd64.tar.gz
# export GOROOT=$HOME/go
# export PATH=$GOROOT/bin:$PATH
# mkdir ~/thriftproxy
# export GOPATH=~/thriftproxy
# go get -u github.com/ochinchina/thriftproxy
```

After executing above commands under linux, the thriftproxy binary will be available in the ~/thriftproxy/bin directory.

## Start the thriftproxy

Before starting the thriftproxy, you need to prepare a configuration file for thriftproxy. The sample configuration file test-proxy.yaml can be found in the git. After preparing the configuration file, you can start the thriftproxy with flag "-c" like:

```shell
# cat test-proxy.yaml
admin:
  addr: ":7890"
proxies:
  - name: test-1
    listen: ":9090"
    backends:
      - addr: "127.0.0.1:9091"
        readiness:
          protocol: tcp
          port: 7890
      - addr: "127.0.0.1:9092"
        readiness:
          protocol: tcp
          port: 7891
  - name: test-2
    listen: ":9020"
    backends:
      - addr: "127.0.0.1:9022"
      - addr: "127.0.0.1:9021"

  - name: test-3
    listen: ":9030"
    backends:
      - addr: "127.0.0.1:9032"
      - addr: "127.0.0.1:9031"
# ~/thriftproxy/bin/thriftproxy -c test-proxy.yaml
```

## rest API for adding/removing backend

The thriftproxy will listen on the admin address to accept the restful call to add/remove backend servers. In the above test-proxy.yaml example, the admin address is ":7890" which means it will listen on port 7890 in all network ip address.

```shell
# cat backends.yaml
{
    "proxies": [
        {
            "name": "test-1",
            "backends": [
              { "addr": "127.0.0.1:6666",
                "readiness": {
                  "protocol": "http",
                  "port": 7893,
                  "path": "/healthz"
                }
              }]
        }
    ]
}
# curl http://localhost:7890/addbackend -d@backends.yaml
# curl http://localhost:7890/removebackend -d@backends.yaml

```
