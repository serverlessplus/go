![](serverless+.png)

# Serverless + Go

## 简介

`serverlessplus` 是一个简单易用的工具，它可以帮助你将现有的 `beego` / `go gin` 等框架构建的应用借助 [API 网关](https://cloud.tencent.com/product/apigateway) 迁移到 [腾讯云无服务云函数](https://cloud.tencent.com/product/scf)（Tencent Cloud Serverless Cloud Function）上。

## 开始使用

```shell
$ go get github.com/serverlessplus/go
```

假设有如下 `go gin` 应用：
```go
package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/go-gin-example", func(c *gin.Context) {
		c.Data(200, "text/html", []byte("hello world"))
	})
	r.Run()
}
```

进行如下简单修改, 即可迁移到[腾讯云无服务云函数](https://cloud.tencent.com/product/scf)（Tencent Cloud Serverless Cloud Function）上
- 指定 `HTTP` 服务监听的端口
- 将 `r.Run` 替换为 `net.Listen` 和 `http.Serve`
- 初始化 `Handler`, 指定端口及需要进行 `base64` 编码的 `MIME` 类型
```go
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	serverlessplus "github.com/serverlessplus/go"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
)

const (
	port = 1216
)

var handler *serverlessplus.Handler

func init() {
	// start your server
	r := gin.Default()
	r.GET("/go-gin-example", func(c *gin.Context) {
		c.Data(200, "text/html", []byte("hello world"))
	})
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", serverlessplus.Host, port))
	if err != nil {
		fmt.Printf("failed to listen on port %d: %v\n", port, err)
		// panic to force the runtime to restart
		panic(err)
	}
	go http.Serve(l, r)

	// setup handler
	types := make(map[string]struct{})
	types["image/png"] = struct{}{}
	handler = serverlessplus.NewHandler(port).WithBinaryMIMETypes(types)
}

func entry(ctx context.Context, req *serverlessplus.APIGatewayRequest) (*serverlessplus.APIGatewayResponse, error) {
	return handler.Handle(ctx, req)
}

func main() {
	cloudfunction.Start(entry)
}
```

## 示例

- [go gin 示例](https://github.com/serverlessplus/go-gin-example)
- [beego 示例](https://github.com/serverlessplus/beego-example)

## 支持的框架

`serverlessplus` 被设计为 `HTTP` 协议与框架进行交互, 对框架并没有限制

## 路线图

`serverlessplus` 处于活跃开发中，`API` 可能在未来的版本中发生变更，我们十分欢迎来自社区的贡献，你可以通过 pull request 或者 issue 来参与。
