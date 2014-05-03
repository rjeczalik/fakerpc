fakerpc [![GoDoc](https://godoc.org/github.com/rjeczalik/fakerpc?status.png)](https://godoc.org/github.com/rjeczalik/fakerpc) [![Build Status](https://travis-ci.org/rjeczalik/fakerpc.png?branch=master)](https://travis-ci.org/rjeczalik/fakerpc)
=======

A fake server for recording and mocking HTTP-based RPC services.

### Installation

```
~ $ go get -u github.com/rjeczalik/fakerpc
```

### Usage

```
~/src/foo $ git diff
diff --git a/foo_test.go b/foo_test.go
index 6ffdfa3..d5e18d0 100644
--- a/foo_test.go
+++ b/foo_test.go
@@ -3,10 +3,13 @@ package foo
 import (
        "net/rpc"
        "testing"
+
+       "github.com/rjeczalik/fakerpc"
 )

 func TestFoo(t *testing.T) {
-       const addr = "http://rpc.int.myservice.com"
+       addr, cleanup := fakerpc.Fixture(t)
+       defer cleanup()
        client, err := rpc.DialHTTP("tcp", addr)
        if err != nil {

~/src/foo $ FAKERPC_RECORD="http://rpc.int.myservice.com" go test foo
ok      foo     0.005s
~/src/foo $ git add testdata/testfoo.gzob
~/src/foo $ go test foo
ok      foo     0.003s
```

## cmd/fakerpc [![GoDoc](https://godoc.org/github.com/rjeczalik/fakerpc/cmd/fakerpc?status.png)](https://godoc.org/github.com/rjeczalik/fakerpc/cmd/fakerpc)

### Installation

```
~ $ go get -u github.com/rjeczalik/fakerpc/cmd/fakerpc
~ $ go install github.com/rjeczalik/fakerpc/cmd/fakerpc
```

### Usage

```
NAME:
   fakerpc - use gentle and with great care

USAGE:
   fakerpc [global options] command [command options] [arguments...]

VERSION:
   0.1.0

COMMANDS:
   record       Proxies connections recording them all to the record-log
   reply        Serves connections with recorded responses from the record-log
   show         Shows record-log as a ngrep output
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --addr 'localhost:0'                         An address to listen on
   --log '/home/rjeczalik/fakerpc.gzob.0'       A path to the record-log file (or ngrep output)
   --version, -v                                print the version
   --help, -h                                   show help
```
