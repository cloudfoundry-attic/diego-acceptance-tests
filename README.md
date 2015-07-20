# Diego Acceptance Tests (DATs)

This test suite exercises [Diego](https://github.com/cloudfoundry-incubator/diego-release) when deployed
alongside [CF Runtime](https://github.com/cloudfoundry/cf-release) (Cloud Controller, DEAs, Loggregator, etc.).

## Usage

### Getting the tests

To get these tests, you can either `git clone` this repo:

```bash
git clone https://github.com/cloudfoundry-incubator/diego-acceptance-tests $GOPATH/src/github.com/cloudfoundry-incubator
cd $GOPATH/src/github.com/cloudfoundry-incubator
go get -t -v ./...
```

 or `go get` it:

 ```bash
 go get -t -v github.com/cloudfoundry-incubator/diego-acceptance-tests/...
 ```

 Either way, we assume you have Golang setup on your workstation.

### Test setup

To run the Diego Acceptance tests, you will need:
- a running CF deployment
- a running Diego deployment
- credentials for an Admin user
- an environment variable `CONFIG` which points to a `.json` file that contains the application domain
- the [cf CLI](https://github.com/cloudfoundry/cli)
- the [Diego SSH CLI plugin](https://github.com/cloudfoundry-incubator/diego-ssh/blob/master/README.md#cf-cli-plugin)

The following commands will setup the `CONFIG` for a [bosh-lite](https://github.com/cloudfoundry/bosh-lite)
installation. Replace credentials and URLs as appropriate for your environment.

NOTE: The secure_address must be some inaccessible endpoint from any container, e.g., an etcd endpoint

```bash
cat > integration_config.json <<EOF
{
  "api": "api.10.244.0.34.xip.io",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "10.244.0.34.xip.io",
  "secure_address": "10.244.0.42:4001",
  "skip_ssl_validation": true
}
EOF
export CONFIG=$PWD/integration_config.json
```

### Running the tests

After correctly setting the `CONFIG` environment variable, the following command will run the tests:

```
./bin/test
```

To opt out of the SSH tests, add `-skipPackage ssh`:
```
./bin/test -skipPackage ssh
```
