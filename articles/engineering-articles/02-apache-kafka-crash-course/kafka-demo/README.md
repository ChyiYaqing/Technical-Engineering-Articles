# Kafka Go Demo

这个示例用 [`kafka-go`](https://github.com/segmentio/kafka-go) 连接运行在 Kubernetes 中的 Kafka 集群。程序会向 `order-events` topic 写入 3 条订单事件，再启动两个独立的 consumer group 分别消费这些消息：

- `orders-db-writer`
- `orders-notification`

两个 consumer group 各自维护 offset，因此都能收到完整的事件流。

## 环境要求

- Go 1.26.3，版本以 `go.mod` 为准
- 能访问目标 Kubernetes 集群的 `kubectl`
- 集群已安装 Strimzi，并存在名为 `kafka` 的 Kafka 集群
- 本机能够访问 Kafka 外部地址 `10.1.101.231:9094`

当前外部 listener 使用 `SCRAM-SHA-512` 认证。用户名和密码通过环境变量传给程序，不写入源码。

## 创建 Kafka 用户

仓库里的 `kafka-user.yaml` 会在 `kafka` namespace 创建一个名为 `kafka-demo` 的用户：

```bash
kubectl apply -f kafka-user.yaml
kubectl -n kafka wait kafkauser/kafka-demo \
  --for=condition=Ready \
  --timeout=120s
```

Strimzi 会生成同名 Secret，其中的 `password` 字段是 SCRAM 密码。把用户名和密码注入当前终端：

```bash
export KAFKA_USERNAME=kafka-demo
export KAFKA_PASSWORD="$(kubectl -n kafka get secret kafka-demo \
  -o jsonpath='{.data.password}' | base64 -d)"
```

确认变量是否已设置时不要直接打印密码：

```bash
test -n "$KAFKA_USERNAME" && test -n "$KAFKA_PASSWORD"
```

## 运行示例

```bash
go mod download
go run .
```

正常运行时，程序先输出生产结果，然后两个 consumer group 分别读取 3 条消息：

```text
Produced 3 order events to topic 'order-events'
Consumer group 'orders-db-writer' is listening for order events...
Consumer group 'orders-notification' is listening for order events...
[orders-db-writer] partition:0 offset:0 key:order-1 value:{...}
[orders-notification] partition:0 offset:0 key:order-1 value:{...}
```

consumer 等待 10 秒后会正常退出，`Consumer timeout reached, exiting...` 不是错误。

## 运行测试

```bash
go test -race ./...
go vet ./...
```

测试会检查缺少凭据时是否返回错误，以及客户端是否使用 `SCRAM-SHA-512`。

## 常见问题

### `KAFKA_USERNAME and KAFKA_PASSWORD must be set`

当前终端没有 Kafka 凭据。重新执行上面的 `export` 命令，再运行程序。

### `unexpected EOF`

Kafka listener 要求进行 SASL 握手，客户端却以未认证方式发送请求时，broker 会关闭连接。确认程序使用当前版本的 `main.go`，并检查 `KAFKA_USERNAME` 和 `KAFKA_PASSWORD` 是否已设置。

### 无法连接 `10.1.101.231:9094`

检查外部 Service 和 Kafka listener 状态：

```bash
kubectl -n kafka get svc kafka-kafka-external-bootstrap
kubectl -n kafka get kafka kafka \
  -o jsonpath='{.status.listeners[?(@.name=="external")].bootstrapServers}{"\n"}'
```

如果集群的外部地址发生变化，需要同步修改 `main.go` 中的 `brokerAddress`。

## 安全说明

当前集群的外部 listener 是 `SASL_PLAINTEXT`：SCRAM 用于认证，但 Kafka 流量没有 TLS 加密。集群也没有启用 Kafka ACL，因此 `kafka-demo` 用户目前没有 topic 和 consumer group 级别的权限隔离。

生产环境应为外部 listener 启用 TLS，并让客户端信任 Strimzi 集群 CA；同时启用 Kafka authorization，仅允许该用户读写 `order-events`，并访问 `orders-db-writer`、`orders-notification` 两个 consumer group。修改 listener 或 authorization 会滚动更新 broker，也可能影响现有客户端，变更前需要一起检查 `admin`、`kafbat-ui` 等已有用户的配置。
