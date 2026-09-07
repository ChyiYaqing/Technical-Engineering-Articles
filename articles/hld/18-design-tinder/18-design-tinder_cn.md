---
title: "Design Tinder"
url: "https://x.com/Harry_The_Nerd/status/2065790968448909572"
category: "HLD"
date: "2026-06-13"
description: "System design for a location-based matching app like Tinder."
lang: "zh-CN"
---

# 设计 Tinder

> 类 Tinder 的基于地理位置的匹配应用的系统设计。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2065790968448909572](https://x.com/Harry_The_Nerd/status/2065790968448909572) · 2026-06-13

![封面图](https://pbs.twimg.com/media/HKnLDVdaIAA3SM8.jpg)

## 高层设计：Tinder

## 1\. 需求

**功能性需求**

- 用户资料管理，比如照片、简介、年龄、性别、交往偏好
- 基于位置的用户发现，比如在可配置的半径内寻找潜在匹配对象
- 对资料卡右滑（喜欢）和左滑（跳过）
- 匹配检测。互相喜欢就触发一次匹配
- 仅限已匹配用户之间的实时聊天

**不在讨论范围**

- Tinder Gold / Boost（在发现流中付费推广资料）
- 视频通话
- 用于滑动排序的高级机器学习排序（提及但不做设计）
- 推送通知（与上一篇 Instagram 文章已覆盖的 APNs/FCM 架构相同）+ 我另有一篇专门讲通知系统的文章

**非功能性需求**

- 高可用：发现和聊天绝不能宕机
- 低延迟：打开 App 后资料卡必须在 100ms 内出现在屏幕上
- 位置精度：发现结果必须近乎实时地反映用户当前所在位置
- 不重复展示：用户绝不能再看到自己已经滑过的资料卡
- 匹配检测需要强一致性：匹配绝不能漏掉，也不能重复触发
- 资料更新和喜欢数可以接受最终一致性

## 2\. 容量估算

- **日活用户：** 1000 万
- **每日滑动次数：** 每人约 50 次 -\> 5 亿次滑动/天 -\> 峰值约每秒 6000 次滑动
- **匹配率：** 右滑中约 1-2% -\> 约 500 万次匹配/天
- **注册用户总数：** 约 5000 万
- **每位用户的照片数：** 约 5 张，每张约 1 MB -\> 每位用户 5 MB
- **媒体存储总量：** 5000 万 × 5 MB = 总计约 250 TB（资料只写一次，很少更新）
- **每日消息量：** 匹配上的用户每天交换约 10 条消息 -\> 5000 万条消息/天（量相对不大）

从这些数字能看出几个要点。每天 5 亿次的滑动量是系统中的主导负载。每一次滑动都必须被记录、检查是否构成匹配，并用来确保那张资料卡不再出现。媒体存储量可控，因为资料创建一次之后很少更新。聊天量出乎意料地低，因为只有匹配上的用户才能聊天，而匹配率很低。这意味着系统主要应该为发现和滑动这两条路径做优化，而不是为媒体分发或消息传递。

## 3\. API 网关

所有客户端请求都经由 API 网关进入，网关负责认证、限流和路由到正确的下游服务。这里的约束和其他所有设计一样：API 网关绝不能出现在二进制媒体负载数据的链路上。资料照片上传在认证之后就绕开网关，用预签名 S3 URL 由客户端直传 S3。网关只做轻量的请求编排。

## 4\. 用户资料服务

资料服务管理用户在 Tinder 上身份的一切：照片、简介、年龄、性别和发现偏好。

上传流程

用户上传资料照片时：

客户端向 API 网关发送上传请求，网关认证用户后转发给资料服务

资料服务生成一个**预签名 S3 URL** 并返回给客户端

客户端把照片直接传到 S3，后端从不接触二进制负载数据

S3 经由 Kafka 触发一个处理事件

**照片处理 Worker** 生成多个分辨率变体——用于发现卡片堆的缩略图、用于完整资料页的中等尺寸——并把所有变体存回 S3

CDN 挡在 S3 前面，向全球客户端分发资料照片，且命中率很高，因为资料照片很少变动

**数据模型**

**Users 表（PostgreSQL）**

Users user\_id UUID, primary key name VARCHAR bio TEXT age INTEGER gender ENUM('male', 'female', 'non\_binary') looking\_for ENUM('male', 'female', 'everyone') relationship\_type ENUM('casual', 'serious', 'friendship') min\_age\_pref INTEGER max\_age\_pref INTEGER radius\_km INTEGER created\_at TIMESTAMP last\_active TIMESTAMP

**Photos 表（PostgreSQL）**

Photos photo\_id UUID, primary key user\_id UUID, foreign key -\> Users s3\_url TEXT order\_index INTEGER uploaded\_at TIMESTAMP

照片单独存一张表，并带一个 order\_index，这样用户就能重排自己的照片顺序，而不必更新 Users 那一行。

**索引策略**

发现查询会在 gender、looking\_for、relationship\_type 和 age 上做过滤。在 (gender, looking\_for, relationship\_type, age) 上建一个复合索引，数据库就能一次索引扫描满足全部过滤条件，不必做全表扫描。这正是让发现查询在大规模下依然快的关键。

此外，被频繁访问的资料（出现在很多人发现卡堆里的用户）会连同完整元数据一起缓存在 Redis 中，这样每次资料填充请求就不必都打到数据库。

资料服务中的瓶颈

资料服务不是高吞吐的写路径，用户很少更新资料。需要关注的是读路径，因为在发现和信息填充过程中资料元数据会被不断读取。挡在 PostgreSQL 前面的 Redis 缓存吸收了这部分读负载的绝大部分。

## 5\. 位置服务

位置是 Tinder 发现系统的基础层。每一次发现查询都从「就近」开始——先找出我附近的用户，再施加其他过滤条件。

**用原始坐标的问题**

把用户位置存成原始的经纬度对（例如孟买是 19.0760, 72.8777），会让半径查询代价极高。要找出 50 公里内的所有用户，就得同时在两个浮点数列上做范围扫描，这既不好建索引，在大规模下又会慢到不可接受。

**Geohash**

Geohash 解决了这个问题：它把二维坐标转换成一个同时编码了经纬度的字符串。世界被递归地划分成矩形单元格构成的网格，每个单元格分配一个短字符串编码。关键性质在于，**地理位置相近的地点共享相同的 geohash 前缀**：

- te -\> 覆盖印度大部分地区的大区域
- te7u -\> 孟买地区
- te7u3 -\> 孟买的某个具体街区
- te7u3x -\> 几个街块

于是查找孟买附近的用户就变成了一次简单的前缀查询 —— WHERE geohash LIKE 'te7u%'，而不是复杂的二维半径计算。5-6 个字符的 geohash 精度对应约 5-10 公里的单元格，正好适合 Tinder 典型的发现半径。

**Redis GEO**

与其在关系型数据库里手工实现 geohash，位置服务直接使用 **Redis GEO**——这是 Redis 原生的数据类型，能存储坐标并开箱即用地支持半径查询。它底层内部就用了 geohash，把坐标存进一个以 geohash 值为分数的有序集合。

关键操作：

**GEOADD users:locations 72.8777 19.0760 "user\_123"** 把 user\_123 加到孟买坐标上 **GEORADIUS users:locations 72.8777 19.0760 50 km** 返回该坐标 50 公里内的所有用户

**位置更新**

Tinder 用户一直在移动：早上在家，中午在公司，晚上在健身房。只要客户端检测到明显位移（通常超过 500 米），或者用户打开 App，位置服务就会更新其在 Redis GEO 中的位置。这就是一次简单的 GEOADD 调用，原子地覆盖掉此前的位置。

位置更新也会异步写入 PostgreSQL 的 **Locations 表**用于审计和历史追溯，但发现的热路径永远只读 Redis GEO。

**位置服务中的瓶颈**

Redis GEO 在内存中运行，无论写入还是半径读取都极快。主要顾虑是内存。为 1000 万活跃用户存坐标是可控的（每个用户大约 100 字节，约 1 GB）。次要顾虑是位置更新频率。如果每个客户端每隔几秒就上报一次位置，写入量就会相当可观。这靠客户端侧的节流来缓解：只有当用户自上次更新以来移动超过 500 米，或者 App 被打开时，才发送位置更新。

## 6\. 发现服务

发现服务是 Tinder 的核心。用户打开 App 时，它必须在 100ms 内产出一堆 20 张资料卡，按距离、年龄、性别和交往偏好过滤，并排除掉所有已经看过的人。

**「已看过」问题**

Instagram 上每个人看到的都是同一条名人帖子，Tinder 则不同：发现卡堆是每个用户独有的，而且绝不能重复出现用户已经滑过的资料。这个排除检查是整个系统里技术上最有意思的需求。

朴素做法，比如在 PostgreSQL 的 Swipes 表上查 WHERE swiper\_id = X AND swiped\_id IN (candidate\_list)，在大规模下太慢了。一个在 Tinder 上待了一年的用户可能已经滑过几万份资料，每次发现请求都去和这段历史做关联，代价高得离谱。

**用布隆过滤器排除已看过的资料**

**布隆过滤器**是一种概率型数据结构，能极高效地回答一个问题：「我以前见过这个东西吗？」

它使用一个固定大小的位数组和多个哈希函数。当用户 A 滑过用户 B 时，用户 B 的 ID 会经过 3-4 个哈希函数，每个函数产出一个下标，数组中对应位置的比特被置为 1。要检查「用户 A 是否见过用户 B」，就用同样的哈希函数算出所有比特位置并逐一检查。如果所有比特都是 1，答案是「可能见过」；只要有一位是 0，答案就是「肯定没见过」。

关键性质：

- **没有假阴性：** 用户见过的资料一定会被排除，绝不会重复看到同一份资料。
- **有很小的假阳性率：** 偶尔会把用户其实没见过的资料也排除掉。在 Tinder 上这完全可以接受，偶尔跳过一个合格候选人，远好过再次展示一份已经被拒绝的资料。
- **极其节省内存：** 在 Redis Set 里存 10 万个用户 ID 可能要好几 MB，同样的数据用布隆过滤器只要几 KB。

Redis 通过 RedisBloom 模块原生支持布隆过滤器：

BF.ADD seen:user\_A user\_B -\> 记录用户 A 已经看过用户 B BF.EXISTS seen:user\_A user\_B -\> 1（可能见过）或 0（肯定没见过）

**完整的发现流水线**

当用户 A 在孟买打开 Tinder 时，会发生下面这些步骤：

**第 1 步（就近查询）-** 发现服务在 Redis GEO 上调用 GEORADIUS users:locations 72.8777 19.0760 50 km，返回 50 公里内所有用户 ID 的列表，候选人可能有几千个。

**第 2 步（布隆过滤器排除）-** 对每一个候选用户 ID，发现服务调用 BF.EXISTS seen:user\_A candidate\_id。任何返回 1 的候选人立刻从池子里剔除。这一步在数据库参与之前就大幅缩小了候选列表。

**第 3 步（偏好过滤）-** 剩下的候选人被交给一条 PostgreSQL 查询，利用复合索引按年龄、性别和交往类型过滤：

SELECT user\_id FROM Users WHERE user\_id IN (candidate\_list) AND gender = 'female' AND relationship\_type = 'serious' AND age BETWEEN 24 AND 30 LIMIT 50

建在 (gender, relationship\_type, age) 上的复合索引，让这条查询即便候选列表很大也依然极快。

**第 4 步（资料填充）-** 过滤后的候选列表（现在是 20-50 个用户 ID 这种可控规模）用一次 MGET 调用从 Redis 取出完整资料元数据来填充：

MGET profile:user\_1 profile:user\_2 ... profile:user\_50

MGET 用一次往返就取回全部记录。若某份资料缓存未命中，发现服务会回落到 PostgreSQL 取出缺失记录，再写回 Redis。

**第 5 步（返回响应）** 发现服务把 20 份填充完整的资料返回给客户端，包括姓名、年龄、简介、照片 URL（指向 CDN）、距离。客户端立刻渲染出卡片堆。

**发现服务中的瓶颈**

布隆过滤器和 Redis GEO 操作都是亚毫秒级的。PostgreSQL 的偏好过滤是唯一碰到磁盘存储的一步，复合索引让它保持很快。剩下的顾虑是候选池的规模：在孟买这样人口稠密的城市，一次 GEORADIUS 查询可能返回 50 公里内的 10 万个用户。把 10 万个 ID 逐个过布隆过滤器再塞进 SQL 的 IN 子句，在这个量级上仍然很快，但对于极端稠密的场景，可以动态缩小半径，或在进入 SQL 之前给候选池设上限。

## 7\. 滑动服务

滑动服务处理系统中量最大的写路径：每天 5 亿次滑动，峰值约每秒 6000 次。每一次滑动都必须被持久记录、用于更新布隆过滤器，并检查是否构成互相喜欢的匹配。

**滑动流程**

**第 1 步（接收滑动）-** 客户端通过 API 网关向滑动服务发送 POST /swipes，携带 { swiper\_id, swiped\_id, direction: 'left' | 'right' }。

**第 2 步（发布到 Kafka）-** 滑动服务立即向 Kafka 发布一条 swipe.created 事件。这把热路径与所有下游处理解耦，同时提供了持久性——即使下游服务临时变慢，也不会丢掉任何一次滑动。

**第 3 步（更新布隆过滤器）-** BF.ADD seen:user\_A user\_B，记录用户 A 现在已经看过用户 B。这确保用户 B 不会再出现在用户 A 的发现卡堆里。

**第 4 步（仅对右滑做匹配检测）** 如果滑动方向是右（喜欢），滑动服务会查询 Redis：

SISMEMBER likes:user\_B user\_A

这是在检查用户 B 之前是否喜欢过用户 A。用户 B 的待处理喜欢存在一个以其用户 ID 为键的 Redis Set 里。

- 如果 SISMEMBER 返回 1 -\> **检测到匹配**。滑动服务把这次匹配写入 Matches 数据库，向 Kafka 发布一条 match.created 事件，通知服务接收后通过 APNs/FCM 提醒双方用户。
- 如果 SISMEMBER 返回 0 -\> 暂时没有匹配。滑动服务把用户 A 加进用户 B 的待处理喜欢集合：SADD likes:user\_B user\_A。

**第 5 步（异步持久化）-** 一个 Kafka 消费者把所有滑动异步写入 PostgreSQL 的 **Swipes 表**（系统中每一次滑动的永久记录）。这次写入发生在热路径之外，不影响滑动延迟。

数据模型

**Swipes 表（PostgreSQL）**

Swipes swipe\_id UUID, primary key swiper\_id UUID, foreign key -\> Users swiped\_id UUID, foreign key -\> Users direction ENUM('left', 'right') swiped\_at TIMESTAMP

**Matches 表（PostgreSQL）**

Matches match\_id UUID, primary key user\_1\_id UUID, foreign key -\> Users user\_2\_id UUID, foreign key -\> Users matched\_at TIMESTAMP

滑动服务中的瓶颈

热路径，也就是更新布隆过滤器和查 Redis Set，完全在内存中完成，从容支撑每秒 6000 次操作。Kafka 吸收写入的突刺并为下游消费者削峰填谷。异步的 PostgreSQL 写入是唯一的磁盘操作，且发生在关键路径之外。Redis 内存方面的主要顾虑是待处理喜欢集合：一个极受欢迎的用户可能积累几十万条待处理喜欢。应对办法是给每个用户的喜欢集合设上限，并定期淘汰那些不太可能促成匹配的旧条目。

## 8\. 聊天服务

只有匹配上的用户才能互相聊天。聊天量相对较低，比如每天 500 万次匹配对应 5000 万条消息，但延迟要求很严格，消息必须给人实时的感觉。

协议选用 WebSocket

HTTP 是请求-响应式协议：客户端问，服务端答。而实时聊天需要服务端在没有请求的情况下主动把消息推给客户端。**WebSocket** 在客户端和服务端之间提供一条持久的双向连接，会话期间一直保持打开。连接一旦建立，服务端随时都能以亚毫秒级的开销把消息推给客户端。

**多服务器带来的问题**

WebSocket 连接是有状态的。一个用户连接在某一台具体的聊天服务器实例上。用户 A 可能连在孟买的聊天服务器 1 上，用户 B 可能连在聊天服务器 3 上。当用户 A 给用户 B 发消息时，聊天服务器 1 与用户 B 之间没有直接连接，它没法把消息推到用户 B 的 WebSocket 连接上，因为那条连接在聊天服务器 3 上。

解决方案是 **Redis 发布/订阅**。每台聊天服务器都会为其上每个活跃会话订阅一个 Redis 频道。当用户 A 发送消息时：

聊天服务器 1 收到消息

聊天服务器 1 把消息发布到 Redis 频道 chat:{match\_id}

订阅了 chat:{match\_id} 的聊天服务器 3 立刻收到消息

聊天服务器 3 把消息推到用户 B 的 WebSocket 连接上

不需要任何服务器之间的直接通信，Redis 充当了聊天服务器之间的消息总线。

**消息存储**

消息需要持久存储，好让用户能回滚查看历史会话。访问模式永远是「按时间顺序、分页地给我会话 X 的消息」。这正好非常契合 **Cassandra**。

**Messages 表（Cassandra）**

Messages match\_id UUID, partition key message\_id UUID, clustering key (ordered by timestamp) sender\_id UUID content TEXT sent\_at TIMESTAMP

按 match\_id 分区意味着一段会话的所有消息都存在同一个 Cassandra 节点上，让拉取会话历史极快。message\_id 这个聚簇键则在分区内按时间顺序排列消息。

对于活跃会话视图，也就是最近 50 条消息，消息还会缓存在一个 **Redis 有序集合**里：

Key: conversation:{match\_id} Member: message\_id Score: timestamp

ZREVRANGE conversation:match\_id 0 49 用一次 Redis 调用就能按顺序返回最新的 50 条消息。更早的消息在用户上滑时从 Cassandra 取。

**在线状态**

用户需要知道自己的匹配对象当前是否在线（那个绿点标识）。在线状态存在 Redis 中：

Key: presence:{user\_id} Value: { status: "online", last\_seen: timestamp } TTL: 30 seconds

App 打开期间，客户端每 15 秒向聊天服务发送一次**心跳**。每次心跳都把这个 Redis 键的 TTL 重置为 30 秒。如果用户关掉 App 或断网，就不再发心跳，30 秒后键自然过期，用户显示为离线。这样既不需要显式登出，也不需要轮询，还能优雅地处理意外断连。

**聊天服务中的瓶颈**

WebSocket 连接是长连接且有状态的，这意味着每台聊天服务器实例只能承载有限数量的并发连接，视内存情况通常是每台 1 万到 5 万条。在 1000 万日活的情况下，这需要数百台聊天服务器实例。一个**带粘性会话的负载均衡器**（或对用户 ID 做一致性哈希）能确保用户在短暂断连后总是重连回同一台聊天服务器实例，避免会话丢失。如果某段会话消息量极大，Redis 发布/订阅会成为瓶颈；但在约会应用里，会话就发生在两个人之间，实践中这不是问题。

## 9\. 完整数据流汇总

资料上传流程： 客户端 -\> API 网关（认证） -\> 资料服务 -\> 预签名 S3 URL 客户端 -\> S3（直传） S3 -\> Kafka -\> 照片处理 Worker -\> S3（缩略图 + 中等尺寸变体） 资料服务 -\> PostgreSQL（用户元数据） 资料服务 -\> Redis GEO（GEOADD 用户位置） 发现流程： 客户端打开 App -\> 发现服务 -\> Redis GEO GEORADIUS（半径内的附近用户 ID） -\> Redis 布隆过滤器 BF.EXISTS（排除已看过的用户） -\> PostgreSQL（通过复合索引按年龄、性别、交往类型过滤） -\> Redis MGET（填充资料元数据） -\> CDN（资料照片 URL 直接分发） -\> 100ms 内向客户端返回 20 份资料 滑动流程： 客户端右滑/左滑 -\> 滑动服务 -\> Kafka（swipe.created，保证持久性） -\> Redis BF.ADD seen:user\_A user\_B（更新布隆过滤器） -\> 若为右滑：Redis SISMEMBER likes:user\_B user\_A -\> 匹配成功：写入 Matches 数据库 + Kafka match.created -\> 通知服务 -\> APNs/FCM -\> 未匹配：Redis SADD likes:user\_B user\_A -\> Kafka 消费者 -\> PostgreSQL Swipes 表（异步持久化） 聊天流程： 客户端 -\> WebSocket 连接 -\> 聊天服务器 发送消息 -\> 聊天服务器 1 -\> Redis 发布/订阅 publish chat:{match\_id} -\> 聊天服务器 N（已订阅） -\> 通过 WebSocket 推给接收方 -\> Cassandra（消息持久化存储） -\> Redis 有序集合 ZADD conversation:{match\_id}（最近消息缓存） 在线状态流程： 客户端每 15 秒心跳 -\> 聊天服务 -\> Redis SET presence:{user\_id} TTL 30s -\> 断连后键过期 -\> 用户显示为离线

## 10\. 韧性与容错

**用 Redis GEO 存位置**意味着位置数据在内存中，速度极快，但 Redis 默认并不持久。位置数据同时也会异步写入 PostgreSQL，所以 Redis 故障时位置服务可以从持久存储重建。位置数据本身风险也不高，Redis 恢复期间有一小段时间位置略微陈旧是可以接受的。

**布隆过滤器的持久性**是个需要细究的问题。如果某个用户的 Redis 布隆过滤器丢失，在过滤器重建之前，他可能会短暂地重新看到已经滑过的资料。这是可以接受的，属于轻微的体验问题，而非数据完整性问题。缓存未命中时，可以用 PostgreSQL 中的永久滑动历史为该用户重建布隆过滤器。

滑动和上传流水线上处处存在的 **Kafka 解耦**，意味着每个下游消费者都能独立地故障和恢复而不丢数据。滑动记录永远不会丢，它们会一直待在 Kafka 里，直到消费者处理完。

聊天消息的 **Cassandra 复制**确保即便节点故障也不会丢消息。跨可用区的 3 副本复制因子是标准做法。

**WebSocket 重连**处理得很优雅。如果某台聊天服务器宕机，客户端会通过负载均衡器自动重连到一台新的服务器实例。新服务器订阅相关的 Redis 发布/订阅频道，并读取存放最近消息的 Redis 有序集合，从而在不丢数据的情况下重建状态。

**匹配检测的一致性-** 用于检查互相喜欢的 Redis Set 查询（SISMEMBER）是原子的。Redis 执行命令是单线程的，所以不会出现这样的竞态：用户 A 和用户 B 同时右滑，结果两边都没检测到匹配。总有一方先执行，把自己加进喜欢集合，另一方随后就能在那里找到它。

## 11\. 技术选型汇总

**API 网关** -\> Kong / AWS API Gateway（认证、限流、路由）

**对象存储** -\> AWS S3（不可变的资料照片，持久，与 CDN 原生集成）

**CDN** -\> CloudFront / Akamai（资料照片的边缘缓存，命中率高）

**解耦** -\> Apache Kafka（滑动持久性、照片处理扇出、匹配事件）

**位置存储** -\> Redis GEO（内存中的半径查询，实时位置更新）

**已看过资料的排除** -\> Redis 布隆过滤器（内存高效的已看过追踪，无假阴性）

**待处理喜欢存储** -\> Redis Set（O(1) 的互相喜欢检测，触发匹配）

**用户 / 资料数据库** -\> 带复合索引的 PostgreSQL（结构化数据，偏好过滤）

**滑动 / 匹配数据库** -\> PostgreSQL（永久的滑动与匹配历史）

**聊天协议** \-\> WebSocket（持久的双向连接，实时消息投递）

**聊天消息总线** -\> Redis 发布/订阅（WebSocket 服务器之间的跨服务器消息路由）

**消息存储** -\> Cassandra（高写入吞吐，按 match\_id 分区，按时间排序）

**最近消息缓存** -\> Redis 有序集合（亚毫秒级的最近会话拉取）

**在线状态** -\> Redis 加 TTL 与心跳（断连时自动判定离线）

以上就是全部内容，干杯！！
