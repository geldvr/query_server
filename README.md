## Run local

### Install

```bash
apt-get update
apt-get install mongodb-org
go get -u github.com/kardianos/govendor
```

### Install go dependencies

```bash
$ cd query-server/vendor
$ govendor sync
```

### Start

```bash
$ mongod
$ cd query-server
$ go run main.go
```
