```console
$ make download
$ export AccessKeyID=hoge
$ export SecretAccessKey=fuga
$ make run
```

```console
$ go run main.go -s="2018-12-25" -e="2018-12-26" -n="/aws/lambda/hoge" -l=10 -q="fields @timestamp, @message | sort @timestamp desc"
```

```console
$ go version
go version go1.11.4 darwin/amd64
```
