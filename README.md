# go-std-ext
Golang Standard Library Extends

## 安装

```bash
$> go get -v -u github.com/PDXbaap/go-std-ext
```

## 使用

### help
```bash
$> go-std-ext --help
NAME:
   /var/folders/b7/q0wwxn550x3_mkt1glwzv7rc0000gn/T/go-build694518831/b001/exe/main - PDX Stdlib 扩展

USAGE:
   main [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --goroot value, -g value  go env 中 GOROOT 对应的目录, pdx 的扩展会被解压到这个目录中
   --revert, -r              还原标准库
   --help, -h                show help
   --version, -v             print the version
```

### setup

```bash
$> go-std-ext
GOROOT :  /usr/local/go/src
VERSION :  go version go1.14.4 darwin/amd64
Success.
```
### revert

```bash
$> go-std-ext --revert
GOROOT :  /usr/local/go/src
VERSION :  go version go1.14.4 darwin/amd64
Success.
```