# 我的日志项目
我的日志项目是我学习go语言过程中写的记录日志的模块，方法是调用模块函数传递字符串写入文件，控制了文件的大小及日志的期限。
## 目录

- [安装](#安装)
- [使用方法](#使用方法)
- [示例](#示例)
- [贡献](#贡献)
- [许可证](#许可证)
## 安装

使用 `go get` 安装：

```bash
go get github.com/gzjjjfree/loggz@v1.0.0

## 使用方法

导入库：

```go
import "github.com/gzjjjfree/loggz"

## 示例
loggz.WriteTaceLog("我要记录的信息")


## 许可证

本项目使用 [MIT 许可证](LICENSE)。
