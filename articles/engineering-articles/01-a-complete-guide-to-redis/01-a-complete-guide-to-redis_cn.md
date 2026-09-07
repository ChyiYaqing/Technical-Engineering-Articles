---
title: "A Complete Guide to Redis"
url: "https://x.com/Harry_The_Nerd/status/2061076806552392055"
category: "Engineering Articles"
date: "2026-05-31"
description: "In-depth guide covering Redis internals and use cases."
lang: "zh-CN"
---

# Redis 完全指南

> 深入讲解 Redis 内部原理与使用场景的指南。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2061076806552392055](https://x.com/Harry_The_Nerd/status/2061076806552392055) · 2026-05-31

![封面图](https://pbs.twimg.com/media/HJAmXK_bsAAL9Dp.jpg)

## Redis 是什么？

Redis（Remote Dictionary Server）是一个开源的内存数据结构存储系统，可以用作数据库、缓存、消息代理和流式引擎。它由 Salvatore Sanfilippo 于 2009 年创建，如今已是世界上最流行的数据库之一。

传统数据库把数据存在磁盘上，Redis 则把所有数据放在内存里。这让它快得惊人，每秒可以处理数百万次操作，延迟低于毫秒级。

## 为什么需要 Redis？

传统数据库从磁盘读写，天然带来延迟。Redis 完全在内存中工作，消除了这部分开销。因此它非常适合：

- 缓存高频访问的数据
- 管理用户会话
- 实时排行榜和分析
- 消息队列与发布/订阅系统
- 限流和分布式锁

## Redis 与传统数据库的对比

**Redis vs SQL vs NoSQL** **Redis** • 内存存储 • 亚毫秒级速度 • 键值数据模型 • 持久化可选 • 最适合缓存、会话、队列

**SQL 数据库** • 基于磁盘的存储 • 毫秒级延迟 • 基于表的关系模型 • 默认持久化 • 最适合事务和复杂查询

**NoSQL 数据库** • 基于磁盘的存储 • 毫秒级延迟 • 文档/列式模型 • 默认持久化 • 最适合灵活的 schema 和水平扩展

## 核心概念

**键值模型：** Redis 中的一切都以「键指向值」的形式存储。值可以是字符串、列表、集合、哈希等等。

**单线程：** Redis 一次只处理一条命令，因此避免了竞态条件，操作天然是原子的。虽然是单线程，但由于全部在内存中操作，速度依然极快。

**持久化：** Redis 并非纯粹的易失存储。它提供了把数据落盘的机制，重启后数据仍在。

**过期：** 每个键都可以设置 TTL（生存时间），到期后 Redis 会自动删除它。这让 Redis 天生适合做缓存。

## Redis 中的数据结构

Redis 不只是一个简单的键值存储。它支持一整套丰富的数据结构，每一种都针对特定场景设计。这正是 Redis 区别于其他缓存的地方。

**1\. 字符串（Strings）：** Redis 中最基础的类型。一个字符串值可以存放文本、整数或二进制数据，最大 512MB。

```
SET name "Alice"
GET name          → "Alice"

SET counter 10
INCR counter      → 11
INCRBY counter 5  → 16
```

使用场景：缓存 HTML、存储配置项、计数器、限流。

**2\. 列表（Lists）：** 按插入顺序排列的字符串有序集合。可以从头部或尾部添加元素。

```
LPUSH tasks "task1"
LPUSH tasks "task2"
RPUSH tasks "task3"

LRANGE tasks 0 -1   → ["task2", "task1", "task3"]
LPOP tasks          → "task2"
```

使用场景：消息队列、动态信息流、栈和队列。

**3\. 集合（Sets）：** 无序的唯一字符串集合。重复元素会被自动忽略。

```
SADD tags "redis"
SADD tags "database"
SADD tags "redis"     → still only one "redis"

SMEMBERS tags         → {"redis", "database"}
SISMEMBER tags "redis" → 1 (true)
```

还可以在集合之间做并集、交集和差集运算。

```
SUNION set1 set2
SINTER set1 set2
SDIFF set1 set2
```

使用场景：独立访客统计、标签、共同好友、权限。

**4\. 有序集合（Sorted Sets / ZSets）：** 类似集合，但每个成员都带一个分值。成员会按分值自动排序。

```
ZADD leaderboard 100 "Alice"
ZADD leaderboard 200 "Bob"
ZADD leaderboard 150 "Carol"

ZRANGE leaderboard 0 -1 WITHSCORES
→ Alice 100, Carol 150, Bob 200

ZRANK leaderboard "Alice"  → 0 (lowest score = rank 0)
```

**5\. 哈希（Hashes）：** 一个键下存放一组字段-值对，就像数据库表里的一行。

```
HSET user:1 name "Alice" age "30" city "NYC"
HGET user:1 name       → "Alice"
HGETALL user:1         → {name: Alice, age: 30, city: NYC}
HINCRBY user:1 age 1   → 31
```

使用场景：存储对象、用户资料、商品数据。

**6\. 位图（Bitmaps）：** 并不是一种独立的数据类型。位图就是被当作位数组来处理的字符串。用来跟踪布尔状态时内存效率极高。

```
SETBIT active_users 1001 1   → user 1001 is active
GETBIT active_users 1001     → 1
BITCOUNT active_users        → total active users
```

使用场景：日活用户、功能开关、考勤记录。

**7\. HyperLogLog：** 一种概率型数据结构，用来统计唯一元素数量，而不需要存储元素本身。无论输入规模多大，它都只占用固定的约 12KB 内存，误差约 0.81%。

```
PFADD visitors "user1" "user2" "user3"
PFCOUNT visitors   → 3
```

使用场景：统计独立页面浏览量、独立搜索词、近似分析。

**8\. 流（Streams）：** 一种只追加的日志结构，类似 Kafka。每条记录都有一个自动生成的唯一 ID，内容是字段-值对。

```
XADD events * action "login" user "Alice"
XADD events * action "purchase" user "Bob"

XRANGE events - +    → all entries in order
```

消费者既可以单独读取流，也可以作为消费者组的一部分来读取。

使用场景：事件溯源、活动日志、实时数据管道。

**9\. 地理空间索引（Geospatial Indexes）：** Redis 可以原生存储经纬度坐标并执行基于位置的查询。

```java
GEOADD locations 13.361 38.115 "Palermo"
GEOADD locations 15.087 37.502 "Catania"

GEODIST locations Palermo Catania km   → 166.27
GEOSEARCH locations FROMMEMBER Palermo BYRADIUS 200 km ASC
```

## Redis 的核心特性

**1\. 过期与 TTL**

Redis 中的每个键都可以设置 TTL（生存时间），到期后 Redis 会自动删除它。这是缓存的基础能力。

```
SET session "abc123"
EXPIRE session 3600       → expires in 3600 seconds (1 hour)

TTL session               → returns seconds remaining
PTTL session              → returns milliseconds remaining

PERSIST session           → removes the expiry, key lives forever
```

也可以在创建键的同时设置过期时间：

```java
SET session "abc123" EX 3600     → seconds
SET session "abc123" PX 3600000  → milliseconds
```

当 TTL 归零时，Redis 会惰性删除（访问时删）或主动删除（后台清理）。使用场景：会话 token、验证码、临时缓存、限流窗口。

**2\. 淘汰策略**

当 Redis 内存耗尽时，它需要决定删除哪些键。你通过 maxmemory-policy 配置项来控制这一点。

noeviction -\> 内存满时直接返回错误 allkeys-lru -\> 淘汰最近最少使用的键 volatile-lru -\> 只在设置了 TTL 的键中淘汰 LRU 键 allkeys-lfu -\> 淘汰使用频率最低的键 volatile-ttl -\> 优先淘汰剩余 TTL 最短的键 allkeys-random -\> 随机淘汰

```
# in redis.conf
maxmemory 256mb
maxmemory-policy allkeys-lru
```

如果 Redis 纯粹当缓存用，最常见的选择是 allkeys-lru 或 allkeys-lfu。

**3\. 事务**

Redis 事务让你把一组命令排队，然后原子地一次性执行。执行期间没有其他客户端能插入命令。

```
MULTI           → start transaction
SET balance 100
DECRBY balance 30
INCR txn_count
EXEC            → execute all at once
```

取消事务：

```
MULTI
SET key1 "value"
DISCARD         → cancel, nothing is executed
```

重要提示：Redis 事务不支持回滚。如果其中一条命令失败，其余命令照样执行。这是有意为之的设计，Redis 把速度放在复杂错误处理之前。

使用场景：在键之间转移数值、必须保持一致的成组更新。

**4\. 用 WATCH 实现乐观锁**

WATCH 让你在事务开始前监视某个键。如果在 EXEC 执行之前该键发生了变化，事务就会被中止。

```
WATCH balance

MULTI
DECRBY balance 50
EXEC            → returns nil if balance changed since WATCH
```

**5\. 管道（Pipelining）**

默认情况下，每条 Redis 命令都是发送后等待响应，一条一条来。管道让你把多条命令打包一次性发出，大幅减少往返时间。

不使用管道时：

```
SET a 1   → wait → SET b 2 → wait → SET c 3 → wait
```

在高延迟环境下，管道能把吞吐量提升 5 到 10 倍。大多数 Redis 客户端库都原生支持管道。

注意：与事务不同，管道不保证原子性。管道中的命令可能与其他客户端的命令交错执行。

**6\. Lua 脚本**

Redis 允许你用 EVAL 命令在服务端运行 Lua 脚本。整个脚本原子执行，期间不会有其他命令插入。

```
EVAL "return redis.call('SET', KEYS[1], ARGV[1])" 1 mykey myvalue
```

一个更实用的例子是带上限的原子自增：

```
local current = redis.call('GET', KEYS[1])
if tonumber(current) < tonumber(ARGV[1]) then
  return redis.call('INCR', KEYS[1])
else
  return current
end
```

你也可以先加载脚本，之后通过它的 SHA 哈希来调用：

```
SCRIPT LOAD "return redis.call('GET', KEYS[1])"
→ returns a SHA hash

EVALSHA <sha> 1 mykey
```

使用场景：限流逻辑、原子的检查并设置（check-and-set）、复杂的条件更新。

**7\. 发布/订阅**

Redis 内置了发布/订阅消息系统。发布者向频道发送消息，订阅者实时接收。

```
# Subscriber (client 1)
SUBSCRIBE news

# Publisher (client 2)
PUBLISH news "Redis 8.0 released!"

# Client 1 receives:
→ "Redis 8.0 released!"
```

你也可以按模式订阅：

```
PSUBSCRIBE news.*       → matches news.tech, news.sports, etc.
```

**发布/订阅的重要限制：**

- 消息不会持久化。如果订阅者离线，就会漏掉消息。
- 没有确认机制，也没有投递保证。
- 需要可靠消息传递时，请改用 Streams。

使用场景：实时通知、聊天系统、实时仪表盘、事件广播。

## Redis 的持久化与可靠性

由于 Redis 把数据放在内存中，一旦重启或崩溃，数据就会全部丢失——除非你把它落盘。Redis 提供了两种主要的持久化机制，两者可以同时使用。

**1\. RDB（Redis 数据库快照）**

RDB 会对整个数据集做一次时间点快照，保存到名为 dump.rdb 的二进制文件中。它通过 fork 主进程来完成这件事，所以快照在后台写入的同时，Redis 仍然照常处理请求。

```
# redis.conf — auto snapshot rules
save 900 1       → save if at least 1 key changed in 900 seconds
save 300 10      → save if at least 10 keys changed in 300 seconds
save 60 10000    → save if at least 10000 keys changed in 60 seconds
```

你也可以手动触发快照：

```
BGSAVE     → background save (non-blocking)
SAVE       → foreground save (blocks all commands until done)
LASTSAVE   → returns Unix timestamp of last successful save
```

**优点：**

- 文件非常紧凑，便于备份和传输
- 重启更快，加载一个二进制文件很迅速
- 正常运行期间对性能影响极小

**缺点：**

- 最后一次快照到崩溃之间的数据会丢失
- 对超大数据集来说，fork 可能很慢且占用大量内存

使用场景：备份、灾难恢复、在不同环境之间迁移数据集。

**2\. AOF（Append Only File，只追加文件）**

AOF 会把服务器收到的每一条写命令记录到名为 appendonly.aof 的文件中。重启时，Redis 重放这个文件来重建数据集。

```
# redis.conf
appendonly yes
appendfilename "appendonly.aof"
```

你通过 fsync 策略控制 Redis 把 AOF 刷盘的频率：

```
appendfsync always    → flush after every command (safest, slowest)
appendfsync everysec  → flush every second (good balance — default)
appendfsync no        → let the OS decide (fastest, least safe)
```

使用 everysec 时，崩溃最多丢失 1 秒的数据。

**AOF 重写**

随着时间推移，AOF 文件会变得很大。Redis 可以只保留重建当前状态所需的最少命令，从而压缩这个文件：

```
BGREWRITEAOF    → triggers background AOF rewrite
```

你也可以配置成自动重写：

```
# redis.conf
auto-aof-rewrite-percentage 100   → rewrite when AOF doubles in size
auto-aof-rewrite-min-size 64mb    → but only if it's at least 64MB
```

**优点：**

- 持久性远强于 RDB，最多只丢 1 秒数据
- 文件可读，便于检查和修复
- 只追加写入，不会因为写了一半而损坏

**缺点：**

- AOF 文件比 RDB 文件大
- 重启更慢，重放命令比加载快照耗时更长
- 写入开销略高

**3\. RDB + AOF 混合模式（推荐）**

从 Redis 4.0 起，你可以同时启用两种持久化方式。执行 AOF 重写时，Redis 会先把一份 RDB 快照写进 AOF 文件，之后再追加新的命令。这样你能同时得到：

- 从内嵌的 RDB 快照快速重启
- 从后续的 AOF 命令获得完整的持久性
```
# redis.conf
appendonly yes
aof-use-rdb-preamble yes    → enables hybrid mode
```

这是大多数生产系统的推荐配置。

**4\. 不做持久化（纯缓存模式）**

如果你只把 Redis 当缓存用，能接受数据丢失，那可以禁用所有持久化：

```
# redis.conf
save ""
appendonly no
```

由于 Redis 完全不写磁盘，这样能获得最高性能。

**5\. 数据恢复**

Redis 重启时，按以下优先级加载数据：

```
1. AOF file (if enabled) - preferred, more complete
2. RDB file (if no AOF) - fallback
3. Empty dataset (if neither exists)
```

你应该定期备份 dump.rdb 和 appendonly.aof 文件，升级前尤其要备份。

**关键配置汇总**

```
# redis.conf - recommended production setup
save 900 1
save 300 10
save 60 10000

appendonly yes
appendfsync everysec
aof-use-rdb-preamble yes
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb
```

## Redis 的可扩展性与架构

单个 Redis 实例在一定规模内工作良好。但要实现高可用、容错和水平扩展，Redis 提供了三层架构：复制（Replication）、哨兵（Sentinel）和集群（Cluster）。

**1\. 复制**

复制让一台 Redis 服务器（主节点）把数据拷贝到一台或多台服务器（副本）。副本会实时与主节点保持同步。

```
# On the replica - redis.conf
replicaof 192.168.1.1 6379

# Or at runtime
REPLICAOF 192.168.1.1 6379

# To stop replication
REPLICAOF NO ONE
```

**工作原理：**

副本连接到主节点

主节点向副本发送一份完整的 RDB 快照

从那时起，主节点把每条写命令实时推送给副本

如果连接断开，Redis 会使用部分重同步来追赶进度，不必重新全量传输

**关键特性：**

- 复制是异步的，主节点不会等待副本确认写入
- 副本默认只读
- 一个主节点可以有多个副本
- 副本本身也可以有自己的副本（级联/链式复制）

**优点：**

- 读扩展：把读流量分流到副本
- 数据冗余：副本充当热备份
- 把重负载操作（备份、分析）转移到副本上执行

**缺点：**

- 没有自动故障转移。主节点挂掉后，必须由人工或工具手动把某个副本提升为主节点

**2\. Redis Sentinel（哨兵）**

Sentinel 解决了故障转移的问题。它是一个独立进程，负责监控 Redis 主节点和副本，并在主节点宕机时自动把某个副本提升为主节点。

```
# sentinel.conf
sentinel monitor mymaster 192.168.1.1 6379 2
sentinel down-after-milliseconds mymaster 5000
sentinel failover-timeout mymaster 60000
sentinel parallel-syncs mymaster 1
```

上面的数字 2 表示至少要有 2 个 Sentinel 一致认为主节点已宕机，才会触发故障转移。这可以防止单个误判导致脑裂。

**故障转移的过程：**

Sentinel 检测到主节点不可达

等待 down-after-milliseconds 以确认它确实宕机

各个 Sentinel 投票，在它们之中选出一个领导者

领导者把数据最新的副本提升为主节点

其他副本被重新配置，改为跟随新的主节点

客户端收到新主节点地址的通知

**Sentinel 还提供服务发现：**

```
# Ask Sentinel for the current master address
SENTINEL get-master-addr-by-name mymaster
→ 192.168.1.2 6379   (could be a different IP after failover)
```

你的应用应该始终向 Sentinel 查询主节点地址，而不是硬编码。

**推荐部署：** 至少 3 个 Sentinel 实例，分布在不同机器上，以保证法定人数（quorum）投票的可靠性。

**Sentinel 不做的事：**

- 它不做数据分片
- 它不能扩展写能力
- 每套配置同一时刻只管理一个主节点

**3\. Redis Cluster（集群）**

Redis Cluster 提供水平分片能力。数据会自动分散到多个主节点上。它还内置了复制和故障转移，不需要额外的 Sentinel。

**分片如何工作——哈希槽：**

Redis Cluster 把键空间划分为 16,384 个哈希槽。每个键通过 CRC16 分配到某个槽：

slot = CRC16(key) % 16384

这些槽再分配到各个主节点上：

Node A -\> 槽 0 到 5460 Node B -\> 槽 5461 到 10922 Node C -\> 槽 10923 到 16383

每个主节点可以有一个或多个副本来实现容错。

**搭建集群：**

```
# Start each node with cluster mode enabled - redis.conf
cluster-enabled yes
cluster-config-file nodes.conf
cluster-node-timeout 5000

# Create the cluster (3 masters, 3 replicas)
redis-cli --cluster create \
  192.168.1.1:6379 \
  192.168.1.2:6379 \
  192.168.1.3:6379 \
  192.168.1.4:6379 \
  192.168.1.5:6379 \
  192.168.1.6:6379 \
  --cluster-replicas 1
```

**集群命令：**

CLUSTER INFO -\> 集群健康状况和状态 CLUSTER NODES -\> 所有节点及其槽范围 CLUSTER KEYSLOT mykey -\> 某个键映射到哪个槽

**哈希标签（控制键的分布）：**

默认情况下，每个键各自落到自己的槽上。如果你需要多个键落在同一个槽上（以便执行多键操作），就使用哈希标签：

SET {user:1}.name "Alice" SET {user:1}.age "30" → 两个键都按 "user:1" 计算哈希，落在同一个槽上

```
SET {user:1}.name "Alice"
SET {user:1}.age  "30"
→ both keys hash on "user:1", land on same slot
```

**集群中的自动故障转移：**

- 主节点宕机时，它的副本会被自动提升
- 如果宕机的主节点没有副本，那部分键空间就会变得不可用

**集群的限制：**

- 多键命令只有在所有键位于同一个槽时才能工作
- Lua 脚本只能操作同一个槽上的键
- 需要客户端提供稍复杂一些的支持

## Redis 的安全与运维

**1\. 认证**

默认情况下 Redis 没有认证。任何能连上端口的人都可以执行任意命令。生产环境务必开启认证。

**简单密码（requirepass）：**

```
# redis.conf
requirepass your_strong_password

# Client must authenticate before any command
AUTH your_strong_password
```

这种方式给所有客户端一个密码，并授予全部权限。可用，但对多用户环境来说粒度不够。

**2\. ACL——访问控制列表**

ACL 在 Redis 6.0 引入，让你创建多个用户，每个用户有自己的密码、允许执行的命令和可访问的键。

```
# Create a user
ACL SETUSER alice on >password123 ~cache:* +GET +SET

# Breakdown:
# on          → user is active
# >password123 → password
# ~cache:*    → can only access keys starting with cache:
# +GET +SET   → can only run GET and SET
```
```
# View all users
ACL LIST

# View current user
ACL WHOAMI

# Delete a user
ACL DELUSER alice

# Test what a user can do
ACL DRYRUN alice SET cache:key value
```

你也可以把 ACL 定义在文件中：

\# redis.conf aclfile /etc/redis/users.acl

**默认用户：** Redis 始终有一个 default 用户。生产环境中应该限制或禁用它：

ACL SETUSER default off

**常见的 ACL 模式：**

```
# Read-only user
ACL SETUSER readonly on >pass ~* +@read

# Cache-only user (specific key prefix)
ACL SETUSER cacheuser on >pass ~session:* +GET +SET +DEL +EXPIRE

# Admin user (full access)
ACL SETUSER admin on >adminpass ~* +@all
```

**3\. 网络安全**

**绑定到指定网卡：**

```
# redis.conf- only accept connections from localhost
bind 127.0.0.1

# Accept from localhost and one internal IP
bind 127.0.0.1 192.168.1.10
```

生产环境绝不要绑定到 0.0.0.0，除非前面有防火墙挡着。

**禁用危险命令：**

```
# redis.conf - rename a command to make it inaccessible
rename-command FLUSHALL ""
rename-command CONFIG   ""
rename-command DEBUG    ""
rename-command SHUTDOWN ""
```

把命令重命名为 "" 就等于彻底禁用它。

**保护模式：**

当既没有配置 bind 地址也没有设置密码时，Redis 会自动开启保护模式，阻止外部连接：

```
protected-mode yes   → default, blocks external access without auth
```

**4\. TLS / SSL**

Redis 6.0 及以上版本原生支持 TLS，用于加密客户端与服务器之间的连接。

```
# redis.conf
tls-port 6380
tls-cert-file /etc/redis/redis.crt
tls-key-file  /etc/redis/redis.key
tls-ca-cert-file /etc/redis/ca.crt
tls-auth-clients yes    → require client certificates
```
```
# Connect with TLS via redis-cli
redis-cli -p 6380 \
  --tls \
  --cert /etc/redis/client.crt \
  --key  /etc/redis/client.key \
  --cacert /etc/redis/ca.crt
```

只要 Redis 和应用不在同一台机器上，就应该使用 TLS，跨网络或跨云环境时尤其如此。

**5\. 内存管理**

Redis 活在内存里，所以内存管理至关重要。

**设置内存上限：**

```
# redis.conf
maxmemory 512mb
maxmemory-policy allkeys-lru
```

查看当前内存使用情况：

```
INFO memory

# Key fields:
# used_memory          → bytes used by data
# used_memory_human    → human readable
# mem_fragmentation_ratio → ratio of RSS to used memory (ideally ~1.0-1.5)
```

**内存碎片：**

随着时间推移，内存会产生碎片，Redis 占用的物理内存会超过实际数据所需。你可以用主动碎片整理来解决：

```
# redis.conf
activedefrag yes
active-defrag-ignore-bytes 100mb   → start defrag after 100mb fragmentation
active-defrag-threshold-lower 10   → start at 10% fragmentation
```

**按键分析内存：**

```
MEMORY USAGE mykey        → bytes used by a specific key
MEMORY DOCTOR             → Redis suggestions for memory issues
MEMORY STATS              → full breakdown of memory usage
```

**对象编码：**

对于小型数据结构，Redis 会自动使用紧凑的内部编码。比如一个小哈希会用 ziplist 而不是完整的哈希表。你可以调整这些阈值：

```
# redis.conf
hash-max-ziplist-entries 128
hash-max-ziplist-value   64
zset-max-ziplist-entries 128
zset-max-ziplist-value   64
```

把数据控制在这些阈值以内能节省可观的内存。

**6\. 监控**

**INFO 命令：**

INFO 是检查 Redis 健康状况的主要工具。它按若干分区返回统计数据：

```
INFO server       → version, uptime, OS
INFO clients      → connected clients, blocked clients
INFO memory       → memory usage and fragmentation
INFO stats        → commands processed, hits, misses
INFO replication  → master/replica status
INFO keyspace     → number of keys per database
INFO all          → everything at once
```

需要重点关注的指标：

```
connected_clients         → spikes may indicate connection leaks
used_memory               → make sure it stays under maxmemory
keyspace_hits/misses      → cache hit rate = hits / (hits + misses)
evicted_keys              → if rising, memory is too tight
rejected_connections      → maxclients limit being hit
instantaneous_ops_per_sec → current throughput
```

**SLOWLOG——找出慢命令：**

Redis 会记录执行时间超过阈值的命令：

```
# redis.conf
slowlog-log-slower-than 10000   → log commands slower than 10ms (in microseconds)
slowlog-max-len 128

# Query the slow log
SLOWLOG GET        → last 128 slow commands
SLOWLOG GET 10     → last 10
SLOWLOG LEN        → how many entries
SLOWLOG RESET      → clear the log
```

**MONITOR——实时命令流：**

```
MONITOR    → streams every command received by Redis in real time
```

这个命令只用于调试，在生产环境中会显著影响性能。

**CLIENT 相关命令：**

```
CLIENT LIST          → all connected clients with details
CLIENT GETNAME       → name of current connection
CLIENT SETNAME myapp → name your connection for easier debugging
CLIENT KILL ID 42    → kill a specific client connection
```

**7\. RedisInsight**

RedisInsight 是 Redis 官方的图形界面工具。它提供：

- 所有键和数据结构的可视化浏览器
- 实时的内存和性能曲线图
- 慢日志查看器
- 内置命令行
- 发布/订阅调试器
- 集群拓扑视图

它是免费的，可以在 [redis.io/redisinsight](https://x.com/Harry_The_Nerd/status/redis.io/redisinsight) 获取。强烈推荐用于开发和生产监控。

关键配置汇总

```
# redis.conf — production security and operations setup

# Auth
requirepass your_strong_password
aclfile /etc/redis/users.acl

# Network
bind 127.0.0.1
protected-mode yes
rename-command FLUSHALL ""
rename-command CONFIG   ""

# TLS
tls-port 6380
tls-cert-file /etc/redis/redis.crt
tls-key-file  /etc/redis/redis.key

# Memory
maxmemory 512mb
maxmemory-policy allkeys-lru
activedefrag yes

# Monitoring
slowlog-log-slower-than 10000
slowlog-max-len 128
```

## Redis 的实战模式

这才是 Redis 真正大放异彩的地方。下面这些模式经过全球生产系统的实战检验。

**1\. 缓存模式**

**Cache-Aside（懒加载）**

最常见的模式。应用先查缓存，未命中时从数据库加载，再写回缓存。

```
function getData(key):
    data = redis.GET(key)
    
    if data is null:
        data = database.query(key)
        redis.SET(key, data, EX 3600)
    
    return data
```
- 优点：只缓存真正被用到的数据
- 缺点：第一次请求总会打到数据库（冷启动）

**Write-Through（写穿透）**

每次写数据库的同时立即写缓存，缓存始终保持同步。

```
function saveData(key, value):
    database.save(key, value)
    redis.SET(key, value, EX 3600)
```
- 优点：缓存永远是新鲜的，没有冷启动
- 缺点：写入变慢，缓存里会塞满可能永远不会被读取的数据

**Write-Behind（回写）**

先立即写入缓存，再在后台异步刷入数据库。

```
function saveData(key, value):
    redis.SET(key, value)
    redis.LPUSH(write_queue, {key, value})   → background worker drains this
```
- 优点：写入极快
- 缺点：如果 Redis 在刷盘前崩溃，有丢数据的风险

**防止缓存击穿（Cache Stampede）**

当一个热点键过期时，成百上千的请求会同时打到数据库。用锁来解决：

```
function getData(key):
    data = redis.GET(key)
    if data: return data

    # Try to acquire lock
    lock = redis.SET(lock:key, 1, NX EX 10)
    
    if lock:
        data = database.query(key)
        redis.SET(key, data, EX 3600)
        redis.DEL(lock:key)
    else:
        sleep(50ms)
        return getData(key)    → retry
```

**2\. 会话管理**

在分布式系统中，Redis 是存储用户会话的标准选择。所有应用服务器共用一个 Redis 实例，因此任何一台服务器都能读到任何会话。

```
# On login
sessionId = generateUUID()
redis.HSET(session:sessionId, userId, "123", role, "admin", loginTime, now())
redis.EXPIRE(session:sessionId, 86400)   → 24 hours

# On each request
session = redis.HGETALL(session:sessionId)
if session is null: redirect to login

# Extend session on activity
redis.EXPIRE(session:sessionId, 86400)

# On logout
redis.DEL(session:sessionId)
```

**3\. 限流**

**固定窗口：**

统计每个用户在一个时间窗口内的请求数。简单，但在窗口边界处可能放过突发流量。

```
function isAllowed(userId):
    key = ratelimit:userId:currentMinute()
    count = redis.INCR(key)
    
    if count == 1:
        redis.EXPIRE(key, 60)
    
    return count <= 100    → allow max 100 requests per minute
```

**用有序集合实现滑动窗口：**

更精确。把每次请求的时间戳作为分值存进去，删掉过期的，再统计剩下的数量。

```
function isAllowed(userId):
    now = currentTimestampMs()
    windowStart = now - 60000   → last 60 seconds
    key = ratelimit:userId

    redis.MULTI
        ZREMRANGEBYSCORE(key, 0, windowStart)       → remove old requests
        ZADD(key, now, now)                          → add current request
        ZCARD(key)                                   → count requests in window
        EXPIRE(key, 60)
    results = redis.EXEC

    return results[2] <= 100    → allow max 100 per 60 seconds
```

**4\. 分布式锁**

当多台服务器需要独占地执行某个操作时，就要用分布式锁。

**用 SETNX 实现基础锁：**

```
# Acquire lock
lock = redis.SET(lock:resource, clientId, NX EX 30)
→ NX = only set if not exists
→ EX 30 = auto-release after 30 seconds

if lock is null: someone else holds it, retry or fail

# Do the critical work
processPayment()

# Release lock — only if we own it (Lua for atomicity)
redis.EVAL("
    if redis.call('GET', KEYS[1]) == ARGV[1] then
        return redis.call('DEL', KEYS[1])
    else
        return 0
    end
", 1, lock:resource, clientId)
```

**Redlock——多节点分布式锁：**

为了获得更强的保证，可以在多个相互独立的 Redis 实例上加锁。只有在多数节点（N/2 + 1）上都加锁成功，这把锁才有效：

```
nodes = [redis1, redis2, redis3, redis4, redis5]
acquired = 0
lockId = generateUUID()

for node in nodes:
    if node.SET(lock:resource, lockId, NX EX 30):
        acquired++

if acquired >= 3:    → majority
    # lock is valid, do work
else:
    # release on all nodes, retry
    for node in nodes:
        node.DEL(lock:resource)
```

**5\. 任务队列**

**用列表实现简单队列：**

```
# Producer - push jobs to the queue
redis.LPUSH(jobqueue, JSON.stringify({type: "email", to: "alice@example.com"}))

# Consumer - blocking pop, waits until a job arrives
job = redis.BRPOP(jobqueue, timeout=0)
process(job)
```

BRPOP 是阻塞的，不用轮询，因此效率更高。

**可靠队列——防止任务丢失：**

如果消费者在处理过程中崩溃，任务就丢了。用一个「处理中」列表来解决：

```
# Atomically move job from queue to processing list
job = redis.BRPOPLPUSH(jobqueue, processing, timeout=0)
process(job)
redis.LREM(processing, 1, job)    → remove after success

# Recovery worker — requeue stuck jobs
stuckJobs = redis.LRANGE(processing, 0, -1)
for job in stuckJobs:
    if isExpired(job):
        redis.LREM(processing, 1, job)
        redis.LPUSH(jobqueue, job)
```

**6\. 排行榜**

有序集合天生适合做排行榜，分值总是自动保持有序。

```
# Add or update a player score
redis.ZADD(leaderboard, 1500, "Alice")
redis.ZADD(leaderboard, 2300, "Bob")
redis.ZADD(leaderboard, 1900, "Carol")

# Top 10 players (highest score first)
redis.ZREVRANGE(leaderboard, 0, 9, WITHSCORES)
→ Bob 2300, Carol 1900, Alice 1500

# A player's rank (0-indexed)
redis.ZREVRANK(leaderboard, "Alice")   → 2

# A player's score
redis.ZSCORE(leaderboard, "Alice")    → 1500

# Increment score (after a game)
redis.ZINCRBY(leaderboard, 200, "Alice")   → 1700

# Players within a score range
redis.ZRANGEBYSCORE(leaderboard, 1000, 2000, WITHSCORES)
```

对于周榜或月榜，只要用带时间范围的键名即可：

leaderboard:2026:week22 leaderboard:2026:05

然后在不再需要时让它们自动过期。

**7\. 用发布/订阅做实时通知**

```
# Server - publish an event when something happens
redis.PUBLISH(notifications:user:123, JSON.stringify({
    type: "message",
    from: "Bob",
    text: "Hey!"
}))

# Client listener - subscribed and waiting
redis.SUBSCRIBE(notifications:user:123)
→ receives event instantly
```

适用于实时通知、聊天、实时仪表盘和协作功能。

**8\. 用有序集合实现自动补全**

把所有可能的补全项以分值 0 存入，再用字典序范围查询来查找匹配项。

```
# Index all terms
redis.ZADD(autocomplete, 0, "redis")
redis.ZADD(autocomplete, 0, "redis cluster")
redis.ZADD(autocomplete, 0, "redis sentinel")
redis.ZADD(autocomplete, 0, "relational database")

# Search for prefix "redis"
redis.ZRANGEBYLEX(autocomplete, "[redis", "[redis\xff")
→ "redis", "redis cluster", "redis sentinel"
```

**9\. 布隆过滤器（通过 RedisBloom）**

不用存储每一个元素，就能判断某个元素之前是否出现过。内存占用只是完整集合的极小一部分。

```
# Add items
redis.BF.ADD(seen_emails, "alice@example.com")

# Check before processing
exists = redis.BF.EXISTS(seen_emails, "alice@example.com")
→ 1 (probably seen) or 0 (definitely not seen)
```

## 现代 Redis

Redis 早已不只是一个简单的缓存。Redis Stack 打包了一系列强大的模块，把 Redis 变成一个多模型数据库，支持全文搜索、原生 JSON 存储、时序数据、概率型结构，甚至面向 AI 应用的向量相似度搜索。

**1\. Redis Stack**

Redis Stack 是官方发行版，把 Redis 核心与一组精选模块打包在一起：

RedisJSON -\> 原生 JSON 存储与查询 RediSearch -\> 全文搜索与二级索引 RedisTimeSeries -\> 时序数据的写入与查询 RedisBloom -\> 概率型数据结构 RedisGraph -\> 图数据库（2023 年已弃用）

你可以通过 Docker 安装 Redis Stack：

```
docker run -d --name redis-stack -p 6379:6379 -p 8001:8001 redis/redis-stack:latest
```

8001 端口暴露的是内置的图形界面 RedisInsight。

**2\. RedisJSON**

RedisJSON 让你在 Redis 内部原生地存储、更新和查询 JSON 文档，不必再把 JSON 序列化成字符串、每次读取时再反序列化。

**基本操作：**

```
# Store a JSON document
JSON.SET user:1 $ '{"name":"Alice","age":30,"address":{"city":"NYC","zip":"10001"}}'

# Get the full document
JSON.GET user:1 $
→ {"name":"Alice","age":30,"address":{"city":"NYC","zip":"10001"}}

# Get a nested field
JSON.GET user:1 $.address.city
→ "NYC"

# Update a specific field
JSON.SET user:1 $.age 31

# Increment a numeric field
JSON.NUMINCRBY user:1 $.age 1
→ 32

# Append to an array
JSON.SET product:1 $ '{"tags":["electronics"]}'
JSON.ARRAPPEND product:1 $.tags '"sale"' '"featured"'
→ ["electronics","sale","featured"]

# Delete a field
JSON.DEL user:1 $.address.zip

# Get the type of a field
JSON.TYPE user:1 $.age
→ integer
```

**JSONPath 语法：**

RedisJSON 使用 JSONPath 进行查询：

```
$           → root
$.field     → top-level field
$.a.b       → nested field
$[*]        → all array elements
$..field    → recursive search for field anywhere in document
$[0]        → first array element
```

RedisJSON 与 RediSearch 结合使用、对 JSON 字段建索引并查询时，威力尤其大。

**3\. RediSearch**

RediSearch 在 Redis 数据之上增加了全文搜索、二级索引、过滤和聚合能力。它对哈希和 JSON 文档都适用。

**创建索引：**

```
# Index on Hash fields
FT.CREATE idx:users ON HASH PREFIX 1 user:
    SCHEMA
        name TEXT WEIGHT 2.0
        age  NUMERIC SORTABLE
        city TAG

# Index on JSON fields
FT.CREATE idx:products ON JSON PREFIX 1 product:
    SCHEMA
        $.name    AS name    TEXT
        $.price   AS price   NUMERIC SORTABLE
        $.tags[*] AS tags    TAG
```

**搜索：**

```
# Full-text search
FT.SEARCH idx:users "Alice"

# Filter by numeric range
FT.SEARCH idx:users "@age:[25 35]"

# Filter by tag
FT.SEARCH idx:users "@city:{NYC}"

# Combined query
FT.SEARCH idx:users "@city:{NYC} @age:[25 35]"

# Fuzzy search (typo tolerance)
FT.SEARCH idx:users "%aliice%"

# Search with pagination
FT.SEARCH idx:users "Alice" LIMIT 0 10

# Return specific fields only
FT.SEARCH idx:users "Alice" RETURN 2 name age
```

**聚合：**

```
# Count users per city
FT.AGGREGATE idx:users "*"
    GROUPBY 1 @city
    REDUCE COUNT 0 AS total
    SORTBY 2 @total DESC

# Average age per city
FT.AGGREGATE idx:users "*"
    GROUPBY 1 @city
    REDUCE AVG 1 @age AS avg_age
```

**用 RediSearch 做自动补全：**

```
FT.SUGADD autocomplete "redis cluster" 1.0
FT.SUGADD autocomplete "redis sentinel" 1.0

FT.SUGGET autocomplete "redis" FUZZY MAX 5
→ "redis cluster", "redis sentinel"
```

4\. RedisTimeSeries

RedisTimeSeries 专为存储和查询带时间戳的数据而设计，比如各类指标、传感器读数和金融数据。

**基本操作：**

```
# Create a time series
TS.CREATE temperature:sensor1 RETENTION 86400000 LABELS location kitchen

# Add data points
TS.ADD temperature:sensor1 * 22.5    → * = current timestamp
TS.ADD temperature:sensor1 * 23.1
TS.ADD temperature:sensor1 1717000000000 21.8   → custom timestamp (ms)

# Get latest value
TS.GET temperature:sensor1

# Get range
TS.RANGE temperature:sensor1 1716000000000 1717000000000

# Aggregation over time
TS.RANGE temperature:sensor1 - + AGGREGATION avg 3600000
→ average temperature per hour
```

**用规则做降采样：**

```
# Create a downsampled series — hourly average
TS.CREATE temperature:sensor1:hourly

TS.CREATERULE temperature:sensor1 temperature:sensor1:hourly
    AGGREGATION avg 3600000
```

原始数据会自动压缩进降采样序列，既节省内存又保留了趋势。

**跨序列查询：**

```
# Query across all sensors by label
TS.MRANGE - + FILTER location=kitchen
```

**5\. RedisBloom**

RedisBloom 增加了四种概率型数据结构，用极小的内存占用解决一些常见问题。

**布隆过滤器（这个元素出现过吗？）**

```
BF.ADD emails "alice@example.com"
BF.ADD emails "bob@example.com"

BF.EXISTS emails "alice@example.com"   → 1 (probably yes)
BF.EXISTS emails "new@example.com"    → 0 (definitely no)
```

可能出现假阳性，但绝不会有假阴性。非常适合去重。

**布谷鸟过滤器，与布隆类似但支持删除：**

```
CF.ADD     visited "page:home"
CF.EXISTS  visited "page:home"   → 1
CF.DEL     visited "page:home"
CF.EXISTS  visited "page:home"   → 0
```

**Count-Min Sketch——频次估算：**

```
CMS.INCRBY events click 5 view 20 share 3

CMS.QUERY events click   → approximately 5
CMS.QUERY events view    → approximately 20
```

用于统计热门事件，而无需存储每一次发生记录。

**Top-K——追踪出现最频繁的 K 个元素：**

```
TOPK.ADD trending "redis" "kafka" "redis" "postgres" "redis"

TOPK.LIST trending
→ redis (3), kafka (1), postgres (1)
```

用于热门话题、访问量最高的页面、热销商品。

**6\. 把 Redis 当作向量数据库**

从 Redis 7.2 起，Redis Stack 通过 RediSearch 原生支持向量相似度搜索。这让 Redis 可以作为 AI 和机器学习应用的向量存储，用来保存嵌入向量并检索语义相近的条目。

**什么是向量？**

当你把文本、图像或音频输入嵌入模型（例如 OpenAI 的 text-embedding-ada-002）时，它会输出一个高维浮点数组，也就是向量。内容相似的数据，其向量在空间中的位置也彼此接近。向量搜索所做的，就是找出与查询向量最邻近的那些向量。

**创建向量索引：**

```
FT.CREATE idx:embeddings ON HASH PREFIX 1 doc:
    SCHEMA
        content TEXT
        embedding VECTOR HNSW 6
            TYPE FLOAT32
            DIM 1536
            DISTANCE_METRIC COSINE
```

HNSW（Hierarchical Navigable Small World，分层可导航小世界图）是用于快速向量检索的近似最近邻算法。

**存储带嵌入向量的文档：**

```
HSET doc:1 content "Redis is an in-memory database" embedding <float32 bytes>
```

**检索相似文档：**

```
FT.SEARCH idx:embeddings "*=>[KNN 5 @embedding $query_vector AS score]"
    PARAMS 2 query_vector <float32 bytes>
    SORTBY score
    RETURN 2 content score
    DIALECT 2
```

这会返回与查询向量语义最相近的 5 个文档。

**常见使用场景：**

- 语义搜索：按含义而不只是关键词来查找文档
- 推荐系统：找出相似的商品或内容
- RAG（检索增强生成）：为大模型提供相关上下文
- 图像相似度搜索
- 异常检测

**7\. Redis Cloud**

Redis Cloud 是 Redis Ltd. 提供的全托管云服务，帮你卸下自建 Redis 的运维负担。

**主要特性：**

- 自动故障转移和备份
- 多区域双活复制（跨区域无冲突的数据同步）
- 内置 Redis Stack 模块
- 拖动滑块即可扩容，无需手动管理集群
- 可在 AWS、Google Cloud 和 Azure 上使用

**双活（Active-Active）地理分布：**

与标准复制不同，双活模式允许多个区域同时写入。冲突由 CRDT（无冲突复制数据类型）自动解决。

```
Region US-East  → write SET user:1 "Alice"
Region EU-West  → write SET user:1 "Bob"
→ conflict resolved by last-write-wins or merge strategy
```

这非常适合需要在全球各地都获得低延迟写入的分布式应用。

**8\. Redis 7.x 与 8.x 的亮点**

**Redis 7.0：**

- Redis Functions：Lua 脚本的替代方案，重启后依然存在
- 多分片 AOF：降低 AOF 重写的开销
- 分片发布/订阅：在集群模式下也能原生工作的 pub/sub
```
# Redis Functions example
FUNCTION LOAD "#!lua name=mylib\nredis.register_function('myfunc', function(keys, args) return args[1] end)"
FCALL myfunc 0 "hello"   → "hello"
```

**Redis 7.2：**

- RediSearch 原生向量搜索
- LMPOP / ZMPOP——原子地从多个列表或有序集合中弹出元素

**Redis 8.0（2024）：**

- Redis Stack 模块并入 Redis 核心，无需单独安装
- 内存效率提升
- 集群性能增强

以上就是全部内容，各位，干杯！
