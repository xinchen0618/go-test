# Golang

## 技术栈

|        技术        |         名称         | 地址                                   |
|:----------------:|:------------------:|--------------------------------------|
|       API        |        Gin         | https://github.com/gin-gonic/gin     |
|      MySQL       |        GORM        | https://github.com/go-gorm/gorm      |
|      Redis       |      go-redis      | https://github.com/go-redis/redis    |
|      cache       |    Redis cache     | https://github.com/go-redis/cache    |
|        登录        |       jwt-go       | https://github.com/golang-jwt/jwt    |
|        日志        |        zap         | https://github.com/uber-go/zap       |
|       优雅停止       |      endless       | https://github.com/fvbock/endless    |
|       命令行        |     urfave/cli     | https://github.com/urfave/cli        |
|       计划任务       |       gocron       | https://github.com/go-co-op/gocron   |
|   Goroutine 池    |        pond        | https://github.com/alitto/pond       |
|       消息队列       |       Asynq        | https://github.com/hibiken/asynq     |
|       类型转换       |        cast        | https://github.com/spf13/cast        |
|       json       |      go-json       | https://github.com/goccy/go-json     |
|    WebSocket     | Gorilla WebSocket  | https://github.com/gorilla/websocket |

## 规范

- 项目布局
  
  - [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
  
- 编码规范 
  
  - [Uber Go 语言编码规范](https://github.com/xxjwxc/uber_go_guide_cn)
  - [Google Style Guides](https://google.github.io/styleguide/go/)

## 目录结构

这里是完整的目录结构, 实际项目未使用的目录可以删除.

```
- cmd/                  项目入口
  - demo-api/           API   
  - demo-cli/           命令行
  - demo-cron/          计划任务
  - demo-queue/         消息队列
  - demo-websocket/     WebSocket
- config/               配置
  - di/                 服务注入
    - db.go             DB 服务
    - logger.go         日志服务
    - queue.go          消息队列服务
    - redis.go          Redis 服务
    - pool.go           Goroutine 池服务
    - cache.go          go-redis cache
  - cfg.go              配置实现
  - common.go           公共配置
  - prod.go             生产环境配置
  - testing.go          测试环境配置
- internal/             内部应用代码. 处理业务的代码
  - action/             命令行 action
  - cron/               计划任务  
  - controller/         API 控制器
  - router/             API 路由
  - middleware/         API 中间件  
  - task/               消息队列任务 
  - service/            内部应用业务原子级服务. 需要公共使用的业务逻辑在这里实现
  - ws/                 websocket 业务
  - consts/             业务相关常量定义
  - types/              业务相关结构体定义
  - model/              表 Model
- pkg/                  外部应用可以使用的代码. 不依赖内部应用的代码
  - ginx/               Gin 增强函数. 此包中出现 error 会向客户端输出 4xx/500 错误, 调用时捕获到 error 直接结束业务逻辑即可
  - gox/                Golang 增强函数
  - gormx/              GORM 初始化函数
  - queuex/             消息队列操作函数
- go.mod                包管理  
```

## 环境定义

环境定义使用`DTAP`, 参考 [Deployment environment](https://en.wikipedia.org/wiki/Deployment_environment)

环境变量`RUNTIME_ENV`指定运行环境, 可以在系统中设置, 也可以在命令行中指定, 默认为生产环境.

- `dev`       开发环境
- `testing`   测试环境
- `stage`     预发布环境
- `prod`      生产环境

## 配置

- 为什么从项目中移除了`viper`

  `viper`提供了运行时修改配置的功能, 而且无法限制.

- 多环境配置

  local 配置, 可选, 功能类似`.env`, 优先级最高, local 配置文件不参与 git 管理;

  环境配置, 可选, 与环境定义相符的配置才会生效, 优先级次之;

  common 配置, 公共配置, 优先级最低;

  同键名配置, 优先级高的覆盖优先级低的;

  配置文件会按分类拆分为多个`<RUNTIME_ENV>_<TYPE>.go`, 比如`testing_db.go`, `testing_app.go`;

- 使用

  获取配置值`config.GetInt()`, `config.GetString()`, `config.GetBool()`, `config.GetIntSlice()`, `config.GetStringSlice()`, `config.GetStringMapString()`

## 依赖注入

DI 实现参考 [Dependency Injection / Service Location](https://docs.phalcon.io/5.0/en/di#dependency-injection--service-location)

`config/di`下每一个导出函数即为一个服务, 需要添加服务, 添加导出函数即可; 包中文件按服务类型分为了多个, 方便管理.

仅日志服务为初始化时加载, 日志服务加载不成功程序不允许启动, 因为在生产环境日志加载不成功开发者就失去了与生产环境程序的联系.

其他服务均为惰性加载, 即第一次使用时才加载.

## 日志

日志文件路径通过`config/`中`error_log`项配置, 注意文件需要读写权限, 未配置文件路径日志将输出到控制台.

日志编码格式为`JSON`.

内部应用使用`di.Logger().Error()`, `di.Logger().Warn()`, `di.Logger().Info()`, `di.Logger().Debug()`记录,

其他, 使用`zap.L(),Error()`, `zap.L().Warn()`, `zap.L().Info()`, `zap.L().Debug()`记录.

`Error()`日志会记录栈信息.

SQL 日志会记录到 zap.

## Goroutine 池 

使用 Goroutine 池旨在解决两个问题:

- Goroutine 使用资源上限
- 优雅处理 Goroutine 中`panic`
 
### 使用

- 公共 Goroutine 池

  ```
  # go func
  for i := 0; i < 10; i++ {
    di.Pool().Submit(func () {
      // do something
    })
  }
  
  # Wait Group
  pg := di.Pool().Group()
  for i := 0; i < 10; i++ {
    pg.Submit(func () {
      // do something
    })
  }
  pg.Wait()
  ```

- 独享 Goroutine 池

  独享 Goroutine 池通常起到类似限流的作用  

  ```
  # go func
  ps := di.PoolSeparate(100)
  for i := 0; i < 10000; i++ {
    ps.Submit(func () {
      // do something
    })
  }
  
  # Wait Group
  psg := di.PoolSeparate(100).Group()
  for i := 0; i < 10000; i++ {
    psg.Submit(func () {
      // do something
    })
  }
  psg.Wait()  
  ```

## API

### 规范

遵循 RESTful 规范, 参考指南 [Best Practices for Designing a Pragmatic RESTful API](https://www.vinaysahni.com/best-practices-for-a-pragmatic-restful-api)

### 流程

`cmd/demo-api/main.go` -> `internal/router/` [-> `internal/middleware/`] -> `internal/controller/` [-> `internal/service/`]

- `internal/router/` 路由, API 版本在此控制
- `internal/middleware/` 中间件, 可选
- `internal/controller/` 业务处理
- `internal/service/` 原子级服务, 可选, 业务应优先考虑是否可以封装为原子级操作以提高代码复用性. 
  
### 登录

- 登录流程

  - 校验账户信息
  - 生成 JWT Token
  - 以`<userType>:<userID>:jwt:<md5(jwtToken)>`的格式记录入 Redis 白名单
  - JWT Token 返回给客户端

- 校验登录

  - 客户端请求时 Header 携带 JWT Token `Authorization: Bearer <token>`
  - 校验 JWT Token
  - 校验 Redis 白名单
  
- 退出登录
 
  - 校验登录
  - 删除对应 Redis 白名单

### 运行

- 开发&测试环境使用 air 实时热重载

  注意, 是否配置了 Go mod 代理`go env -w GOPROXY=https://goproxy.cn,direct`, 是否安装了 air `go install github.com/cosmtrek/air@latest`, 是否配置了 Go bin 路径`export PATH=$PATH:$HOME/go/bin`

  ```
  cd cmd/demo-api
  RUNTIME_ENV=testing air
  ```

- 预发布&生产环境执行编译好的程序

  实际上会提前编译好, 在机器上直接部署可执行文件. 如果程序不需要使用 C 库或者嵌入 C 代码，那么`CGO_ENABLED=0`可以让编译更简单和快速, 如果程序里调用了 cgo 命令, 此参数必须设置为1, 否则编译时将出错.

  ```
  # 启动
  cd cmd/demo-api
  go build -ldflags="-s -w"
  (RUNTIME_ENV=prod ./demo-api &> /dev/null &)

  # 优雅重启
  pkill -SIGHUP -f "demo-api"

  # 优雅停止
  pkill -SIGINT -f "demo-api"
  ```

## CLI

### 流程

`cmd/demo-cli/main.go` -> `internal/action/` [-> `internal/service/`]

- `cmd/demo-cli/main.go` 定义 CLI 路由, 按业务维度分两级
- `internal/action/` 执行逻辑

### 使用

```
cd cmd/demo-cli
go build -ldflags="-s -w"
RUNTIME_ENV=testing ./demo-cli <commond> <action> [ARG...]
```

## Cron

Cron 的停止并非优雅停止, 尤其要注意数据完整性的问题.

### 流程

`cmd/demo-cron/main.go` -> `internal/cron/` [-> `internal/service/`]  

- `cmd/demo-cron/main.go` 定义计划任务
- `internal/cron/` 执行逻辑

### 启动

```
cd cmd/demo-cron
go build -ldflags="-s -w"
(RUNTIME_ENV=testing ./demo-cron &> /dev/null &)
```

## Queue

### 流程

`cmd/demo-queue/main.go` -> `internal/task/` [-> `internal/service/`]

- `cmd/demo-queue/main.go` 定义队列任务
- `internal/task/` 执行逻辑

### 使用

- 启动 Worker

  ```
  cd cmd/demo-queue
  go build -ldflags="-s -w"
  (RUNTIME_ENV=testing ./demo-queue &> /dev/null &)
  ```

- 优雅停止 Worker

  ```
  pkill -TSTP -f "demo-queue" && pkill -TERM -f "demo-queue"
  ```

- 发送 Job

  消息队列按任务优先级分两个队列: 默认队列, 该队列分配了较多的系统资源, 任务一般发送至此队列; 低优先级队列, 该队列分配了较少的系统资源, 数据量大不紧急的任务发送至此队列.

  默认队列: 及时消息`queuex.Enqueue()`, 延时消息`queuex.EnqueueIn()`, 定时消息`queuex.EnqueueAt()`

  低优先级队列: 及时消息`queuex.LowEnqueue()`, 延时消息`queuex.LowEnqueueIn()`, 定时消息`queuex.LowEnqueueAt()`

## WebSocket

### 鉴权 

与 API 鉴权保持一致, 使用的JWT. 客户端通过 URL 参数`client_id`, 值为`url_base64(userID:md5(jwtToken))`, 传入鉴权信息.

### 通信

客户端与服务端通信的消息格式为`{type: "", data: {}}`, `type`-消息类型, `data`-消息内容.
  
比如, 客户端发送消息
```
{
  "type": "MicroChat:SendMessage",
  "data": {
    "content": "Hello, word!"
  }
}
```

服务端响应消息
```
{
  "type": "ClientError",
  "data": {
    "code": "UserUnauthorized",
    "message": "您未登录或登录已过期, 请重新登录"
  }
}
```

### 消息推送

服务端主动向客户端推送消息, 通过 Redis 订阅来实现, 服务端监听名为`WSMessageChannel`的 Redis 频道.

向频道发送消息的格式为 json 字符串 `{"user_id": int, "type": string, data: {}}`

`user_id` 为 0 表示向所有用户推送消息, 否则为向指定用户推送消息.

## Redis

`key`统一在`internal/consts/redis_key.go`中定义, 避免冲突.

### 规范

[阿里云Redis开发规范](https://developer.aliyun.com/article/531067)
