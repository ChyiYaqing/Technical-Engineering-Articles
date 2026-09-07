---
title: "Design Twitter"
url: "https://x.com/Harry_The_Nerd/status/2068309801638195531"
category: "HLD"
date: "2026-06-20"
description: "System design for a microblogging platform like Twitter."
lang: "zh-CN"
---

# 设计 Twitter

> 类 Twitter 的微博客平台的系统设计。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2068309801638195531](https://x.com/Harry_The_Nerd/status/2068309801638195531) · 2026-06-20

![封面图](https://pbs.twimg.com/media/HLNXU0LboAADIct.jpg)

## 高层设计：Twitter

## 1\. 需求

**功能性需求**

- 用户资料管理（姓名、用户名、头像、横幅图、简介、出生日期、粉丝、关注）
- 发推（最多 280 字符的文本、图片、视频）
- 点赞、回复、转推、关注、取消关注
- 主页信息流（算法排序和时间倒序两种模式）
- 趋势页（全球热门话题标签和话题）
- 搜索推文、用户和话题标签
- 通知（点赞、转推、回复、关注、提及）

**不在范围内**

- 私信（与 Tinder 采用相同的 WebSocket 架构）
- Twitter Spaces（实时音频，属于另一套实时系统）
- Twitter Blue / 认证相关功能
- 广告定向系统
- 数据分析流水线

**非功能性需求**

- 信息流、发推和媒体分发都要具备高可用性
- 信息流读取延迟低于 100ms
- 能扛住全球性事件期间（世界杯进球、突发新闻）的集中写入峰值
- 面对被数百万人同时读取的爆款推文要有热点键韧性
- 点赞数、转推数和信息流排序可以接受最终一致性
- 关注/取消关注关系以及推文是否存在需要强一致性

## 2\. 容量估算

- **日活用户（DAU）：** 2 亿
- **每日原创推文量：** 1.5 亿（大致相当于 75% 的活跃用户各发一条，其余用户发得更多）
- **推文构成：** 60% 纯文本，30% 带图片，10% 带视频
- **文本存储：** 60% x 1.5 亿 x 300 字节 = 约 27 GB/天（可忽略不计）
- **图片存储：** 30% x 1.5 亿 x 1 MB = 约 45 TB/天
- **视频存储：** 10% x 1.5 亿 x 10 MB = 约 150 TB/天
- **每日总存储量：** 约 200 TB/天，主要来自视频
- **读写比：** 大约 3:1，比 Instagram 或 Reddit 要均衡得多，因为 Twitter 的用户群参与度极高、非常活跃

3:1 这个读写比是这里最重要的数字。Instagram 是名人生产、数百万人被动消费；Twitter 不同，它的用户一直在转推、回复和点赞。这意味着写入路径必须和读取路径一样健壮、一样可扩展——这个约束在我们设计过的其他系统里并不以同样的形式存在。

## 3\. API 网关

所有客户端请求都从 API 网关进入，由它负责认证、限流和路由。这里适用和前面每一篇设计相同的约束：API 网关绝不能出现在二进制媒体负载数据的传输路径上。图片和视频上传在完成认证后就绕开网关，通过预签名 S3 URL 由客户端直传 S3。

和 Instagram 或 Reddit 相比，Twitter 的 API 网关多了一层顾虑：**吸收写入峰值**。在重大全球性事件期间，数百万用户会同时冲击发推接口。网关层面的限流是第一道防线，把过量流量在到达下游服务之前就丢弃掉。

## 4\. 用户服务

用户服务管理所有资料数据，包括姓名、用户名、头像、横幅图、简介、出生日期、粉丝数和关注数。

**头像与横幅图上传**

同样是预签名 URL 的模式：

客户端通过 API 网关完成认证，向用户服务申请一个预签名 S3 URL

客户端直接上传到 S3

图片处理 Worker 生成缩略图和全分辨率变体

CDN 位于 S3 之前，面向全球分发资料相关媒体；由于头像很少变动，缓存命中率很高

数据模型

**Users 表（PostgreSQL）**

Users user\_id UUID，主键 username VARCHAR，唯一 display\_name VARCHAR bio TEXT profile\_pic\_url TEXT banner\_url TEXT date\_of\_birth DATE follower\_count INTEGER following\_count INTEGER tweet\_count INTEGER verified BOOLEAN created\_at TIMESTAMP

follower\_count、following\_count 和 tweet\_count 都是反范式化的计数器，通过 Kafka 消费者异步更新。在 Twitter 这个量级上，每次加载资料页都去查 Follows 表来统计粉丝数，会慢得灾难性。

**Follows 表（PostgreSQL）**

Follows follower\_id UUID，外键 -\> Users followee\_id UUID，外键 -\> Users created\_at TIMESTAMP PRIMARY KEY (follower\_id, followee\_id)

关注就是一次 INSERT，取消关注就是一次 DELETE。「列出用户 X 关注的所有人」就是 WHERE follower\_id = X。两列都建索引，两个方向的查找都很快。至于「推荐关注」「共同好友」这类二度关系，用 Neo4j 或 AWS Neptune 这样的图数据库更合适——跨数百万节点的图遍历在图数据库里是原生能力，在关系型 join 中却代价高昂。

**用户服务的瓶颈**

用户资料的读取远多于写入。Redis 缓存热门资料，包括认证账号和名人账号——它们的资料每天要被拉取数百万次。剩下的流量由 PostgreSQL 只读副本承接。在 Twitter 的规模下，Follows 表会膨胀到数十亿行，因此按 follower\_id 分片，让单个分片保持在可控大小。

## 5\. 推文服务与上传流水线

发推流程

**纯文本推文：**

客户端通过 API 网关向推文服务发送 POST /tweets，携带内容

推文服务写入 Tweets 库

发布 tweet.created 事件到 Kafka

Kafka 消费者扇出到信息流、搜索和通知系统

**带媒体的推文：**

客户端向推文服务申请一个预签名 S3 URL

客户端把媒体直接上传到 S3

S3 向 Kafka 触发一条 media.uploaded 事件

转码服务接手这条事件并进行处理：图片——生成缩略图、中等分辨率和全分辨率变体，做 WebP 压缩；视频——生成多档画质变体（360p、480p、720p、1080p），切分成 HLS 分片以支持自适应码流，并生成一张预览缩略图

所有变体存入 S3，路径为 media/{tweet\_id}/{variant}/

转码服务向 Kafka 发布 tweet.transcoded 事件

三个消费者各自独立地消费这条事件：**元数据 Worker** 把推文元数据写入 Tweets 库，并填充 Redis 中的推文元数据缓存；**搜索索引 Worker** 把推文索引进 Elasticsearch；**信息流扇出 Worker** 开始把这条推文写入粉丝们的信息流 sorted set

数据模型

**Tweets 表（PostgreSQL）**

Tweets tweet\_id UUID，主键 user\_id UUID，外键 -\> Users content VARCHAR(280) media\_url TEXT（纯文本推文为 null） media\_type ENUM('none', 'image', 'video') reply\_to\_id UUID，外键 -\> Tweets（原创推文为 null） like\_count INTEGER retweet\_count INTEGER reply\_count INTEGER view\_count INTEGER posted\_at TIMESTAMP

like\_count、retweet\_count、reply\_count 和 view\_count 全都是异步更新的反范式化计数器。单次互动的真实来源存在各自独立的表里（Likes、Retweets、Replies），但计数值维护在 Tweet 行上，以便快速读取。

**Retweets 表（PostgreSQL）**

Retweets retweet\_id UUID，主键 original\_tweet\_id UUID，外键 -\> Tweets retweeter\_id UUID，外键 -\> Users retweeted\_at TIMESTAMP

一次转推只是对已有推文的一个引用，不存储任何新内容。原推文行上的 retweet\_count 由 Kafka 消费者异步递增。用户 A 转推一条内容时，它扇出给的是用户 A 的粉丝，而不是原作者的粉丝。原作者只会收到一条通知。

**Likes 表（PostgreSQL）**

Likes like\_id UUID，主键 tweet\_id UUID，外键 -\> Tweets user\_id UUID，外键 -\> Users liked\_at TIMESTAMP UNIQUE (tweet\_id, user\_id)

UNIQUE 约束在数据库层面强制保证点赞的幂等性。Tweet 行上的点赞数是快速读取路径，Likes 表才是唯一真实来源。

**写入峰值的处理**

全球性事件期间，数百万用户同时发推。在这种负载下把每条推文同步写入 PostgreSQL，会把数据库压垮。缓解办法是在 Redis 中放一个写缓冲：

推文创建时立即写入 Redis

发布 tweet.created 事件到 Kafka

Kafka 消费者在热路径之外异步写入 PostgreSQL

Redis 写缓冲吸收峰值，PostgreSQL 以可持续的速率写入

点赞数和转推数遵循同样的模式：先在 Redis 中递增，再通过 Kafka 异步刷回 PostgreSQL。

**推文服务的瓶颈**

带媒体的推文，瓶颈在转码服务，靠 Worker 池的水平扩展来解决。纯文本推文的瓶颈就是上面说的写入峰值问题，靠 Redis 写缓冲和 Kafka 解耦来缓解。

## 6\. 信息流服务

Twitter 的信息流有两种模式，它们在基础设施层面的运作方式截然不同：算法信息流和时间倒序信息流。

**算法信息流**

算法信息流按相关性、互动度和关系亲密度对推文排序。这种排序在读取时计算代价很高，因此通过写时扇出预先算好。

**扇出策略：**

- **普通用户（粉丝数低于阈值）：** 写时扇出。用户 A 发推时，信息流扇出 Worker 立刻把这条推文写进 A 的所有粉丝的信息流 sorted set。
- **名人用户（粉丝数高于阈值，比如 100 万）：** 读时扇出。他们的推文存放在 Redis 的名人推文缓存里。读取信息流时，信息流服务取出最新的名人推文，与预计算好的信息流合并。

**Redis Sorted Set 结构：**

Key: feed:{user\_id}:algorithmic Member: tweet\_id Score: 相关性分数（时效性、互动信号和关系亲密度的组合）

ZREVRANGE feed:user\_123:algorithmic 0 19 瞬间返回排名前 20 的推文 ID。

**时间倒序信息流**

时间倒序信息流是纯粹的时间逆序排列，把所关注账号的最新推文排在最前面。因为不需要排序打分，读取时计算是可行的，而且彻底避开了高粉丝账号的写时扇出难题。

读取时：

从 Redis 中取出该用户关注的账号列表

查询 PostgreSQL：SELECT tweet\_id FROM Tweets WHERE user\_id IN (followed\_accounts) ORDER BY posted\_at DESC LIMIT 20

有了 (user\_id, posted\_at) 上的复合索引，即使关注了几千个账号，这条查询依然很快

结果按用户缓存在 Redis 里，设置较短 TTL：feed:{user\_id}:chronological

重复打开时间倒序信息流时命中的是 Redis，而不是 PostgreSQL。

**信息流数据补全**

两种信息流模式返回的都是推文 ID。完整的推文内容通过 Redis MGET 补全：

MGET tweet:101 tweet:202 tweet:303 ... tweet:2020

一次往返，20 条推文的元数据全部取回。缓存未命中时回退到 PostgreSQL 并回写 Redis。

**热点键问题**

当一位拥有 5000 万粉丝的名人发出一条爆款推文时，这一条推文的缓存键每分钟会被读取数百万次。标准的 Redis 缓存在这里会吃力，因为所有读取都打在同一个 Redis 节点的同一个键上。

解决方案是**多层缓存（L1/L2/L3）**：

- **L1 - 本地内存缓存**，位于每台应用服务器上（在 Java 里可以用 Caffeine 这样的 LRU 缓存）。每台应用服务器把爆款推文缓存在自己的内存里。零网络跳数，读取在亚微秒级完成。
- **L2 - Redis**，承载那些很热、但还没热到值得在每台服务器上做 L1 缓存的推文
- **L3 - PostgreSQL**，作为唯一真实来源，只在完全缓存未命中时才被访问

一条被 5000 万用户阅读的爆款推文，同时由数千台应用服务器的本地缓存提供。只有当某台服务器本地缓存是冷的时候才会访问 Redis。热点键问题就此彻底消失。

**信息流服务的瓶颈**

当中等热度的账号发推量很大时，算法信息流的扇出 Worker 是首要瓶颈。混合模型缓解了名人账号带来的这一问题。读取时合并名人推文的步骤会带来额外延迟，延迟大小与用户关注的名人账号数量成正比；把合并结果以较短 TTL 缓存起来即可缓解。

## 7\. 趋势服务

趋势页展示的是「此刻全世界正在谈论什么」，接近实时更新。这是 Twitter 独有的东西。趋势并不是简单地取推文总量最多的话题标签。#FIFA 历史累计有几十亿条推文，但它不该永远挂在趋势榜上。趋势意味着**当下活跃度的骤增**，是相对历史基线而言的。一个话题标签上榜，是因为它在最近一小时的使用量显著高于过去 24 小时的平均使用量。

**用 Redis Sorted Set 实现滑动窗口计数器**

对从进来的推文中提取出的每一个话题标签，趋势服务维护一个 Redis Sorted Set：

Key: hashtag:FIFA Member: tweet\_id（唯一的事件标识） Score: 该推文的 Unix 时间戳

要统计 #FIFA 在最近一小时被使用了多少次：

ZCOUNT hashtag:FIFA (now - 3600) now

这是一个 O(log N) 操作，即使集合里有数百万条记录也极快。窗口之外的旧条目会被周期性清理：

ZREMRANGEBYSCORE hashtag:FIFA 0 (now - 3600)

这就是一个**滑动窗口计数器**。窗口随时间自动向前推移，计数始终保持新鲜，无需任何手动重置。

趋势计算 Worker

趋势计算 Worker 每 1 到 5 分钟运行一次，它会：

用 ZCOUNT 遍历所有活跃的话题标签键，按近期推文量取出排名靠前的标签

为每个话题标签计算 24 小时的基线推文量

按骤增比例（相对基线的百分比涨幅，而不是绝对量）对话题标签排序

把排名前 50 的趋势话题写入一个 Redis 键：trending:global，并设置较短 TTL

客户端每次加载页面时读取 trending:global，一次 Redis GET 就拿到预先算好的列表。读取时不发生任何计算。

**话题标签提取流水线**

推文创建时，**话题标签提取 Worker** 从 tweet.created 这个 Kafka topic 消费，解析推文内容中的话题标签，对每个标签执行：

ZADD hashtag:{tag} <timestamp\> <tweet\_id\>

这一步通过 Kafka 与发推的热路径解耦。发推不必等待话题标签提取完成。

**个性化趋势**

Twitter 还会展示个性化趋势，即与你的位置和兴趣相关的话题。这由同一个 Worker 计算，但按用户的位置（国家、城市）和兴趣图谱（关注的账号、互动过的话题）做了过滤。个性化趋势结果按用户缓存，TTL 更长一些，因为它们的变化频率低于全球趋势。

**趋势服务的瓶颈**

话题标签提取 Worker 每天要处理 1.5 亿条推文，大约每秒 1700 条。每条推文还可能包含多个标签。在这个速率下，Worker 池必须水平扩展才跟得上。Redis Sorted Set 的每次操作都在亚毫秒级，不构成问题。每隔几分钟运行一次的趋势计算 Worker 是批处理作业，不影响实时性能。

## 8\. 搜索服务

Twitter 搜索覆盖三类实体：推文（按内容和话题标签）、用户（按用户名和显示名）以及作为一等可搜索对象的话题标签。

**存储引擎 - Elasticsearch**

Elasticsearch 负责这三类实体上的全文搜索、部分匹配和相关性排序。Twitter 搜索把文本匹配得分和互动信号结合起来。对同一条查询，高赞推文会排在无人问津的推文前面。

**Elasticsearch 文档**

**推文文档：**

```
{
  "tweet_id": "123",
  "content": "Just shipped a new feature using Rust and WebAssembly",
  "hashtags": ["rust", "webassembly", "programming"],
  "author": "harry_dev",
  "author_follower_count": 15000,
  "like_count": 3200,
  "retweet_count": 890,
  "posted_at": "2026-06-20T10:00:00Z"
}
```

**用户文档：**

```
{
  "user_id": "456",
  "username": "harry_dev",
  "display_name": "Harry The Nerd",
  "bio": "Backend engineer. Writing about distributed systems.",
  "follower_count": 15000,
  "verified": false
}
```

点赞数、转推数和粉丝数充当排序信号。在用户搜索中，认证账号排名更靠前。推文中的话题标签会被索引成独立的词项，所以搜索 #rust 能返回所有包含该标签的推文。

**保持 Elasticsearch 同步**

搜索索引 Worker 消费 tweet.created 和 tweet.transcoded 这两个 Kafka topic。互动计数的更新则是攒批后周期性同步到 Elasticsearch，而不是每次互动都同步，因为搜索结果中的精确计数不需要实时。

热门搜索查询以较短 TTL 缓存在 Redis 中，以降低 Elasticsearch 的负载。

**搜索的瓶颈**

和之前所有设计的模式一样。针对热门查询的 Redis 缓存在请求到达 Elasticsearch 之前就过滤掉了大部分重复流量。剩下的查询量靠水平扩展 Elasticsearch 集群来消化。

## 9\. 互动服务

互动服务负责点赞、回复、转推和收藏。除发推本身之外，这些是系统中写入量最高的操作。

点赞流程

客户端发送 POST /likes，携带 { user\_id, tweet\_id }

幂等性检查：在 Redis 中执行 SISMEMBER likes:tweet\_123 user\_456。若已点过赞，则忽略。

SADD likes:tweet\_123 user\_456，在 Redis 中记录这次点赞

INCR tweet:tweet\_123:like\_count，在 Redis 中更新计数器

发布 like.created 事件到 Kafka

Kafka 消费者：**点赞持久化 Worker** 异步写入 PostgreSQL 的 Likes 表；**计数同步 Worker** 周期性地把 Redis 中的点赞数同步到 PostgreSQL 中 Tweets 行的 like\_count 列；**通知 Worker** 通过 APNs/FCM 向推文作者推送通知

**转推流程**

客户端发送 POST /retweets，携带 { user\_id, tweet\_id }

在 Redis 中做幂等性检查，一个用户对同一条推文只能转推一次

通过 Kafka 异步写入 Retweets 表

在 Redis 中递增原推文的 retweet\_count

**信息流扇出 Worker** 把被转推的推文扇出到转推者的粉丝的信息流 sorted set 中，而不是原作者的粉丝

向原作者发送通知

**回复流程**

回复本质上是 reply\_to\_id 非空的推文。回复通过与原创推文相同的推文服务流程创建，只是额外设置了 reply\_to\_id 字段。原推文的 reply\_count 通过 Kafka 异步递增。

**互动服务的瓶颈**

点赞是量最大的操作。一条爆款推文每秒可能收到数千个赞。Redis 的幂等性检查和计数器递增完全在内存中吸收了这些压力。Kafka 负责持久性，并把热路径和 PostgreSQL 写入解耦开来。

## 10\. 通知服务

架构与 Instagram 相同。从各个互动服务经 Kafka 扇出，iOS 用 APNs、Android 用 FCM，并做通知合并以防打扰。用「Harry 和另外 4999 人赞了你的推文」代替 5000 条单独的推送。

通知服务订阅：

- like.created topic
- retweet.created topic
- reply.created topic
- follow.created topic
- mention.created topic（由话题标签提取 Worker 从推文内容中解析出来）

设备 token 按用户存放在一个简单的 Redis hash 里。应用内通知存放在 PostgreSQL 的 Notifications 表中，应用打开时通过 WebSocket 下发。

## 11\. 完整数据流总结

**发推流程（文本）：** 客户端 -\> API 网关（认证） -\> 推文服务 -\> Redis 写缓冲（吸收峰值） -\> Kafka（tweet.created） -\> 元数据 Worker -\> Tweets 库 + Redis 元数据缓存 -\> 搜索 Worker -\> Elasticsearch -\> 信息流扇出 Worker -\> Redis 信息流 sorted set（普通用户的粉丝） -\> 名人推文缓存（作者为名人时） -\> 话题标签 Worker -\> ZADD hashtag:{tag} <timestamp\> <tweet\_id\> -\> 通知 Worker -\> 解析出的提及 -\> APNs / FCM **发推流程（媒体）：** 客户端 -\> API 网关（认证） -\> 推文服务 -\> 预签名 S3 URL 客户端 -\> S3（直传） S3 -\> Kafka -\> 转码服务 -\> S3（所有变体） 转码服务 -\> Kafka（tweet.transcoded） -\> 元数据 Worker -\> Tweets 库 + Redis 缓存 -\> 搜索 Worker -\> Elasticsearch -\> 信息流扇出 Worker -\> Redis 信息流 sorted set **算法信息流读取流程：** 客户端 -\> 信息流服务 -\> Redis ZREVRANGE feed:{user\_id}:algorithmic 0 19 -\> Redis GET 名人推文（对关注了名人的用户在读取时合并） -\> Redis MGET tweet:1 tweet:2 ... tweet:20（补全元数据） -\> L1 本地缓存（爆款推文直接由应用服务器内存提供） -\> CDN（媒体 URL 直接由 CDN 提供） **时间倒序信息流读取流程：** 客户端 -\> 信息流服务 -\> Redis GET feed:{user\_id}:chronological（缓存命中） -\> PostgreSQL：SELECT tweet\_id FROM Tweets WHERE user\_id IN (...) ORDER BY posted\_at DESC LIMIT 20 -\> Redis MGET（补全元数据） -\> 结果以较短 TTL 回写 Redis **点赞流程：** 客户端点赞推文 -\> 互动服务 -\> Redis SISMEMBER likes:tweet\_id user\_id（幂等性） -\> Redis INCR tweet:tweet\_id:like\_count -\> Kafka（like.created） -\> 点赞持久化 Worker -\> PostgreSQL Likes 表 -\> 计数同步 Worker -\> PostgreSQL [Tweets.like](https://x.com/Harry_The_Nerd/status/Tweets.like)\_count（周期性批量） -\> 通知 Worker -\> APNs / FCM（已合并） **趋势流程：** tweet.created -\> Kafka -\> 话题标签提取 Worker -\> ZADD hashtag:{tag} <timestamp\> <tweet\_id\> 趋势计算 Worker（每 1-5 分钟） -\> ZCOUNT hashtag:{tag} (now-3600) now（最近一小时计数） -\> 与 24 小时基线对比 -\> ZREMRANGEBYSCORE（淘汰旧条目） -\> SET trending:global \[前 50 个话题标签\] 并设置 TTL 客户端 -\> GET trending:global（一次 Redis 读取） **搜索流程：** 客户端 -\> 搜索服务 -\> Redis（热门查询缓存） -\> Elasticsearch（缓存未命中时） -\> Redis MGET（为结果补全推文元数据）

## 12\. 韧性与容错

发推和互动路径上的 **Redis 写缓冲** 吸收了全球性事件期间的惊群式写入峰值。PostgreSQL 从不直面原始峰值，而是以可持续的速率从 Kafka 队列中消费写入。

针对爆款推文的**多层缓存（L1/L2/L3）** 消除了热点键问题。应用服务器的本地缓存以零网络跳数承接了爆款内容绝大部分的读取。Redis 负责温热内容。只有完全缓存未命中时才会访问 PostgreSQL。

整条流水线上的 **Kafka 持久性** 意味着即使下游服务暂时不可用，也不会丢失任何一条推文、点赞或转推。所有消费者都可以落后，恢复后再追上来，不会丢数据。

点赞和转推的**双层幂等性**：热路径上的 Redis Set 检查，加上作为兜底的 PostgreSQL UNIQUE 约束。即便在重试的情况下，用户也绝不会对同一条推文点两次赞。

趋势用的 **Redis 滑动窗口计数器** 是自维护的。旧条目通过 ZREMRANGEBYSCORE 自动过期。趋势计算 Worker 是无状态的批处理作业，随时可以重启而不丢数据。

所有媒体的 **CDN 缓存** 意味着头像、推文图片和视频分片都由全球边缘节点提供。只有缓存未命中时才会触及源站基础设施。

所有表都配 **PostgreSQL 只读副本**，确保读取不存在单点故障。写入走主节点，读取分散到各个副本上。

**时间倒序信息流兜底**：如果算法信息流的预计算流水线出现滞后，用户随时可以切换到时间倒序模式——它在读取时直接从 PostgreSQL 计算，与扇出流水线完全无关。

## 13\. 技术选型总结

**API 网关** -\> Kong / AWS API Gateway（认证、限流、写入峰值削峰）

**对象存储** -\> AWS S3（不可变的媒体文件，持久可靠，对 CDN 友好）

**CDN** -\> CloudFront / Akamai（推文媒体和头像的边缘缓存）

**转码** -\> 通过 Kafka 驱动的水平扩展 Worker 池（图片变体、HLS 视频切片）

**解耦** -\> Apache Kafka（推文扇出、点赞持久性、趋势提取、搜索索引）

**搜索引擎** -\> Elasticsearch（全文搜索、话题标签索引、按互动加权的排序）

**用户 / 推文 / 点赞数据库** \-\> PostgreSQL 分片 + 副本（关系型结构、强一致性、复合索引）

**算法信息流存储** \-\> 每用户一个 Redis Sorted Set（预计算的排序信息流，亚毫秒级读取）

**时间倒序信息流缓存** -\> 带短 TTL 的 Redis（缓存读取时计算出的信息流）

**推文元数据缓存** -\> Redis MGET（一次往返批量补全推文内容）

**热门推文缓存** -\> L1 本地内存（Caffeine LRU） + L2 Redis（多层热点键缓解）

**写缓冲** -\> Redis（在 Kafka 和 PostgreSQL 之前吸收惊群式写入峰值）

**趋势存储** -\> 带滑动窗口的 Redis Sorted Set（用 ZCOUNT 做时间窗口内的话题标签频次统计）

**互动幂等性** -\> Redis Set + PostgreSQL UNIQUE 约束（双层重复防护）

**关注关系图** -\> PostgreSQL + Neo4j / AWS Neptune（简单关注关系用 SQL，推荐关注用图遍历） 以上就是全部内容，各位……干杯！！
