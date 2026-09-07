---
title: "Apache Kafka: Crash Course"
url: "https://x.com/Harry_The_Nerd/status/2062165677033820383"
category: "Engineering Articles"
date: "2026-06-03"
description: "Crash course covering Apache Kafka fundamentals."
lang: "zh-CN"
---

# Apache Kafka 速成课

> 讲解 Apache Kafka 基础的速成课程。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2062165677033820383](https://x.com/Harry_The_Nerd/status/2062165677033820383) · 2026-06-03

![封面图](https://pbs.twimg.com/media/HJ0bH3kawAATxUa.jpg)

## 世界上最大的科技公司如何处理每秒数百万个事件

**问题所在：你的数据库不是消息队列**

设想你是 Uber 的一名工程师。这是一个周五的晚上，全球有几十万名乘客同时打开 App。每一次叫车请求、每一个司机上报的 GPS 坐标、每一次动态调价的重新计算、每一笔支付事件——所有这些都需要被实时记录、处理并作出响应。在峰值负载下，Uber 每天要处理大约 1 万亿个事件。

现在设想这些流量全部直接打到一个 PostgreSQL 数据库上。

数据库开始接收写入，然后开始变慢，接着连接池被耗尽，最后彻底崩溃。你刚刚在一个周五晚上搞垮了 Uber。

这并非假想。在 Uber、LinkedIn、Netflix 这些公司建起像样的事件流基础设施之前，这种故障模式非常普遍。数据库是为事务型负载设计的，不是为消防水管式的数据摄入设计的，于是它成了那个能拖垮整个平台的瓶颈。

LinkedIn 在 2010 年前后深受此苦。他们的基础设施是一团点对点数据管道构成的乱麻。任何一个服务想消费另一个服务的数据，都得自己搭一条定制连接。服务多达几十个之后，这张网就变得无法维护了。更要命的是，当下游服务变慢或宕机时，会产生反压并一路向上游传导，威胁整个系统。活动信息流、分析、搜索索引、监控，全都在争抢读取同一批数据库和 API。整个系统脆弱、缓慢，而且越来越无法扩展。

正是在这样的痛苦中，LinkedIn 的工程师 Jay Kreps、Neha Narkhede 和 Jun Rao 造出了一个新东西。他们把它命名为 Kafka。

## Kafka 是什么？

Apache Kafka 是一个分布式事件流平台。

Kafka 位于你的各个服务之间，是一个高吞吐、容错、持久化的日志。生产者把事件写入 Kafka，消费者按自己的节奏从 Kafka 读取。两边从不直接对话，数据库也永远不会直面涌入的原始流量洪流。

可以把它想象成一个邮件分拣中心。成千上万封信件（事件）源源不断地到达，中心把它们全部接收下来，分拣好，持久保存。收件人（消费者）在自己方便的时候来取信。分拣中心不关心收件人读得多快，它只管持续接收和存储。发件人绝不会因为某个收件人慢而被阻塞。

这种解耦正是 Kafka 背后的根本思想，它立刻解决了「数据库被轰炸」这个核心问题。原本是 5 万个并发写入直冲数据库，现在变成 5 万个写入打到 Kafka——Kafka 正是为此而生的——然后由一个或几个消费者从 Kafka 读取，以受控且可持续的速度写入数据库。

## Kafka 与数据库：更好的架构

Kafka 与数据库的组合，比其中任何一方单独使用都更强大。

在一个朴素的架构里，用户的一次操作（比如下单）会直接写入数据库，然后数据库还得设法触发通知、更新库存、投喂分析、更新搜索索引——全部同步完成，全部在同一个请求里。这样既慢又脆弱，还把所有系统耦合在了一起。

有了 Kafka 之后，下单意味着往 Kafka 里写一个事件：「订单已创建，这是负载数据。」仅此而已。HTTP 响应立刻返回给用户。之后在后台，各个独立的消费者各干各的活：订单服务写入订单数据库，库存服务扣减库存，通知服务发送确认邮件，分析管道记录这个事件。它们中的任何一个失败都不会影响其他，每一个都可以独立扩展，每一个都按自己的节奏处理。

现在数据库接收到的写入，来自一个行为良好的消费者，速率受控可调，而不是成千上万的并发用户直接猛砸。你把一条不受控的消防水管，变成了一股可管理的水流。

这个模式还带来一样极其宝贵的东西：可重放性。数据库反映的是世界当前的状态，而 Kafka 保留了产生这个状态的全部事件历史。如果半年后你上线了一个需要历史数据的新服务，它可以从头重放 Kafka 日志。只有数据库是做不到这一点的。

## Kafka 的架构

理解 Kafka 的内部原理，就能明白它为什么能扛住数据库扛不住的负载。

**事件与主题**

Kafka 中的一切都始于事件——一条记录，说明某件事发生了。一个事件包含键、值、时间戳，以及可选的元数据头。事件被组织成主题（topic）。主题大致相当于数据库中的表或者一个文件夹，区别在于它是一个只追加的有序日志。你不能更新或删除主题中的记录，只能追加新事件。

**分区：并行的基本单元**

每个主题被切分成若干分区（partition）。一个分区就是一段存储在磁盘上的、有序且不可变的事件序列。这是 Kafka 的核心扩展机制。如果单个分区能承受 100MB/s 的吞吐，那么一个有 10 个分区的主题就能承受 1GB/s——你通过增加分区来扩展吞吐量。

具有相同键的事件总是进入同一个分区，这保证了该键下事件的顺序性。这在实践中很关键：同一个用户 ID、同一个订单 ID 的所有事件都会落到同一个分区并按顺序处理，即便同时还有数百万个其他事件正在写入其他分区。

**Broker 与集群**

一个 Kafka 集群由多个 broker（独立的服务器节点）组成。分区分布在这些 broker 上。每个分区有一个 leader broker 负责读写，以及零个或多个 follower broker 复制数据以实现容错。如果 leader broker 挂掉，某个 follower 会被自动提升。正是这种复制让 Kafka 具备持久性：在复制因子为 3 的情况下，你可以损失两个 broker，仍然能无损地提供全部数据。

**生产者**

生产者是那些把事件写入 Kafka 的服务。它们写入指定的主题，也可以选择性地指定分区键。Kafka 内部会对写入做批处理来提升效率——它不会每个事件都做一次磁盘写入，而是先累积事件，再成批刷盘，这也是 Kafka 能达到如此高吞吐的重要原因。生产者可以配置成等待一个 broker 确认、等待所有副本确认，或者不等待确认，以持久性保证换取更低的延迟。

**消费者与消费者组**

消费者从主题中读取事件。这里的核心概念是消费者组（consumer group）。当多个消费者实例属于同一个组时，Kafka 会自动在它们之间分配分区。如果一个主题有 12 个分区，而消费者组里有 4 个实例，那么每个实例分到 3 个分区。增加消费者实例即可扩展吞吐量；减少实例，Kafka 会自动重新平衡。

至关重要的是，消费者用偏移量（offset）来记录自己在日志中的位置——那只是一个简单的整数，表示已经读到第几个事件。这个偏移量归消费者自己所有。Kafka 不会把事件推给消费者，而是由消费者拉取事件并推进自己的偏移量。这意味着一个慢消费者绝不会对 Kafka 或生产者施加反压，它只是暂时落后，等自己有余力时再追上来。

不同的消费者组各自维护独立的偏移量。你的分析管道和通知服务可以完全独立地消费同一个主题，速度完全不同，互不干扰。

**保留策略：Kafka 不是队列**

传统消息队列在消息被消费后就将其删除，Kafka 不会。它会按可配置的时长保留事件——几天、几周，或者按磁盘容量上限来定。这正是可重放性的来源，也是一个根本性的架构差异。Kafka 是日志，不是队列。

**ZooKeeper 与 KRaft**

历史上，Kafka 依赖 Apache ZooKeeper 来管理集群元数据、leader 选举和配置。从 Kafka 2.8 开始引入、并在 Kafka 3.3 完全达到生产可用的 KRaft 模式，是 Kafka 自带的基于 Raft 的共识协议，彻底去掉了对 ZooKeeper 的依赖。现代的 Kafka 部署都使用 KRaft，运维大大简化。

## Kafka 在生产环境中：Netflix 和 Uber 到底拿它做什么

Netflix 把 Kafka 作为整条数据管道的主干。每一次播放事件、每一次暂停、每一条搜索查询、每一行错误日志，都要先流经 Kafka 才会到达任何下游存储。在他们的规模下，这意味着每天数千亿个事件。Kafka 让 Netflix 拥有一条权威的事实流，同时供给实时仪表盘、数据仓库中的离线分析、推荐模型训练以及 A/B 测试系统——全部来自同一条流，而这些消费方彼此互不影响。

Uber 围绕 Kafka 重建了整套数据基础设施。司机位置更新、行程事件、支付处理和欺诈检测，全都流经他们的 Kafka 集群。他们的欺诈检测系统实时消费支付事件流，可以在毫秒级标记出可疑交易——如果这套逻辑必须在每笔交易时都去查数据库，那是不可能做到的。

## 用 Go 写的 Kafka 生产者和消费者

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/segmentio/kafka-go"
)

const (
    brokerAddress = "localhost:9092"
    topic         = "order-events"
)

// OrderEvent represents something that happened in our system
type OrderEvent struct {
    OrderID    string
    CustomerID string
    Amount     float64
    Status     string
}

func runProducer() {
    writer := kafka.NewWriter(kafka.WriterConfig{
        Brokers:      []string{brokerAddress},
        Topic:        topic,
        Balancer:     &kafka.Hash{}, // same key always goes to same partition
        BatchTimeout: 10 * time.Millisecond,
    })
    defer writer.Close()

    events := []kafka.Message{
        {
            Key:   []byte("order-1001"),
            Value: []byte(\`{"order_id":"1001","customer_id":"cust-42","amount":89.99,"status":"placed"}\`),
        },
        {
            Key:   []byte("order-1002"),
            Value: []byte(\`{"order_id":"1002","customer_id":"cust-17","amount":240.00,"status":"placed"}\`),
        },
        {
            Key:   []byte("order-1001"),
            Value: []byte(\`{"order_id":"1001","customer_id":"cust-42","amount":89.99,"status":"payment_confirmed"}\`),
        },
    }

    err := writer.WriteMessages(context.Background(), events...)
    if err != nil {
        log.Fatalf("failed to write messages: %v", err)
    }

    fmt.Printf("Produced %d order events to topic '%s'\n", len(events), topic)
}

func runConsumer(groupID string) {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  []string{brokerAddress},
        Topic:    topic,
        GroupID:  groupID, // consumer group, Kafka tracks offset per group
        MinBytes: 1,
        MaxBytes: 10e6,
        MaxWait:  1 * time.Second,
    })
    defer reader.Close()

    fmt.Printf("Consumer group '%s' waiting for messages...\n", groupID)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    for {
        msg, err := reader.FetchMessage(ctx)
        if err != nil {
            if err == context.DeadlineExceeded {
                fmt.Println("Consumer done (timeout reached)")
                break
            }
            log.Fatalf("fetch error: %v", err)
        }

        fmt.Printf(
            "[%s] partition=%d offset=%d key=%s value=%s\n",
            groupID,
            msg.Partition,
            msg.Offset,
            string(msg.Key),
            string(msg.Value),
        )

        // Commit offset only after successful processing
        // This is important: if processing fails, the offset is not advanced
        // and the message will be redelivered
        if err := reader.CommitMessages(ctx, msg); err != nil {
            log.Printf("failed to commit offset: %v", err)
        }
    }
}

func main() {
    // In a real system these run in separate services
    runProducer()

    // Two different consumer groups both reading the same topic independently
    // e.g., one for writing to DB, one for sending notifications
    go runConsumer("orders-db-writer")
    go runConsumer("orders-notifications")

    time.Sleep(12 * time.Second)
}
```

这段代码中：生产者使用了 Hash 均衡器，意味着订单 ID（也就是键）相同的事件总会落到同一个分区，从而保证订单 1001 的「placed」和「payment confirmed」两个事件按顺序被消费。两个不同的消费者组各自独立地读取同一个主题，数据库写入服务和通知服务都能拿到每一个事件，彼此之间无需任何协调。而偏移量只在处理成功之后才提交，所以如果消费者在处理过程中崩溃，重启后会重新处理那条消息，而不是悄悄跳过它。

以上就是全部内容，各位，干杯！！
