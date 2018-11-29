## Run local

### Install

```bash
apt-get update
apt-get install mongodb-org
go get -u github.com/kardianos/govendor
```

### Install go dependencies

```bash
$ cd itv/vendor
$ govendor sync
```

### Start

```bash
$ mongod
$ cd itv/query-server
$ go run main.go
```
