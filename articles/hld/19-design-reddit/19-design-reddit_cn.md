---
title: "Design Reddit"
url: "https://x.com/Harry_The_Nerd/status/2067615247259811850"
category: "HLD"
date: "2026-06-18"
description: "System design for a forum/link-aggregation platform like Reddit."
lang: "zh-CN"
---

# 设计 Reddit

> 类 Reddit 的论坛/链接聚合平台的系统设计。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2067615247259811850](https://x.com/Harry_The_Nerd/status/2067615247259811850) · 2026-06-18

![封面图](https://pbs.twimg.com/media/HKnLjBZbAAAPv4o.jpg)

## 高层设计：Reddit

## **1\. 需求**

**功能性需求**

- 创建帖子（文本、图片、视频）
- 对帖子和评论投赞成票、反对票
- 树状嵌套评论
- 加入 subreddit、关注用户
- 基于已加入 subreddit 的主页信息流
- subreddit 信息流，支持多种排序模式：Hot、New、Top、Controversial
- 搜索帖子、subreddit 和用户
- 创建 subreddit

不在范围内

- 聊天（与 Tinder 采用相同的 WebSocket 架构）
- Reddit 的奖励和金币体系
- 内容审核工具
- Reddit 广告

**非功能性需求**

- 信息流、投票和媒体分发都要具备高可用性
- 信息流读取延迟低于 100ms
- 即使流量突增，投票也绝不能丢失
- 热度分数必须随时间衰减，反映最近的活跃度
- 投票计数和信息流排序可以接受最终一致性
- 投票幂等性需要强一致性（同一个用户不能对同一篇帖子投两次票）

## 2\. 容量估算

- **日活用户（DAU）：** 5000 万
- **每日发帖量：** 1000 万（占 DAU 的 20%）
- **平均帖子大小：** 约 2 MB，这是文本（100 KB）、图片（500 KB 到 1 MB）和视频（最多 10 MB）的加权平均值
- **每日帖子存储量：** 1000 万 x 2 MB = 20 TB/天
- **每日投票量：** 5000 万 DAU x 每人 5 票 = 2.5 亿票/天 = 峰值约 3000 票/秒，遇到爆款内容时还要高得多
- **每日评论量：** 大约 1 亿条（平均每个 DAU 发 2 条评论）

写入负载的大头是每秒 3000 次的投票，而不是发帖。每一次投票都可能触发一次热度分数重算。这是投票系统和信息流系统的首要设计约束。每天 20 TB 的媒体存储量不算小，但用 S3 加 CDN 缓存完全能扛住。

## 3\. API 网关

所有客户端请求都从 API 网关进入，由它负责认证、限流和路由。这里适用和前面每一篇设计相同的约束：API 网关绝不能出现在二进制媒体负载数据的传输路径上。图片和视频上传在完成认证后就绕开网关，通过预签名 S3 URL 由客户端直传 S3。

## 4\. 帖子服务与上传流水线

**上传流程**

当用户创建一篇带媒体的帖子时：

客户端通过 API 网关完成认证，向帖子服务申请一个预签名 S3 URL

客户端用这个预签名 URL 把媒体直接上传到 S3

S3 发出上传事件，经由 Kafka 触发转码服务

转码服务处理媒体：图片——生成缩略图、中等分辨率和全分辨率三种变体，做压缩并转换为 WebP；视频——生成多档画质变体（360p、480p、720p、1080p），切分成 HLS 分片以支持自适应码流，并生成一帧缩略图

所有变体都按确定性路径写回 S3：media/{post\_id}/{variant}/

转码完成后向 Kafka 发布一条 post.transcoded 事件

两个消费者各自独立地消费这条事件：**元数据 Worker** 把帖子元数据写入 Posts 库，并填充 Redis 中的帖子元数据缓存；**搜索索引 Worker** 把帖子索引进 Elasticsearch

对于纯文本帖子，第 1 到第 6 步全部跳过。帖子服务直接写入 Posts 库，并向 Kafka 发布一条 post.created 事件。

**数据模型**

**Posts 表（PostgreSQL）**

Posts post\_id UUID，主键 subreddit\_id UUID，外键 -\> Subreddits user\_id UUID，外键 -\> Users title VARCHAR content TEXT（媒体帖为 null） media\_url TEXT（文本帖为 null） media\_type ENUM('text', 'image', 'video') upvotes INTEGER downvotes INTEGER score INTEGER hot\_score FLOAT posted\_at TIMESTAMP

upvotes、downvotes、score 和 hot\_score 都是反范式化的计数器，由 Kafka 消费者异步维护。这样读取时就不必执行昂贵的聚合查询。

**Subreddits 表（PostgreSQL）**

Subreddits subreddit\_id UUID，主键 name VARCHAR，唯一 description TEXT member\_count INTEGER created\_by UUID，外键 -\> Users created\_at TIMESTAMP

**SubredditMembers 表（PostgreSQL）**

SubredditMembers user\_id UUID，外键 -\> Users subreddit\_id UUID，外键 -\> Subreddits joined\_at TIMESTAMP PRIMARY KEY (user\_id, subreddit\_id)

这是一张多对多关联表。加入 subreddit 就是一次 INSERT，退出就是一次 DELETE。「列出用户 X 加入的所有 subreddit」只是一条简单的 WHERE user\_id = X 查询。

**帖子服务的瓶颈**

帖子服务本身是无状态的，可以水平扩展。媒体帖的瓶颈在转码服务，因为生成多档视频画质变体非常吃 CPU。解决办法是把转码服务做成可水平扩展的 Worker 池，从 Kafka topic 里拉取任务。文本帖不存在瓶颈。

## 5\. 投票服务

投票是系统里写入量最高的路径，平均每秒 3000 票，爆款帖子上还会出现极端峰值。投票系统必须在突发流量下不丢数据、防止重复投票，并保持热度分数及时更新。

完整投票流程

**第 1 步 - 接收投票** 客户端通过 API 网关向投票服务发送 POST /votes，携带 { user\_id, post\_id, direction: 'up' | 'down' }。

**第 2 步 - 在 Redis 中做幂等性检查** 在任何处理之前，投票服务先用一个 Redis Set 检查该用户是否已经给这篇帖子投过票：

SGET vote:user\_123:post\_456

如果用户此前已经投过相同方向的票，请求被忽略。如果用户在切换投票方向（赞成改反对），则计算差值并处理。这个幂等性检查完全在 Redis 的热路径上完成，不碰数据库。

**第 3 步 - 在 Redis 中递增投票计数器** 投票计数器在 Redis 中立即更新：

INCR post:post\_456:upvotes

客户端从 Redis 读取投票数，而不是从数据库读。这样爆款帖子的票数能做到近实时更新，同时不给数据库带来任何读压力。

**第 4 步 - 发布到 Kafka** 更新完 Redis 之后，投票服务立刻向 Kafka 发布一条 vote.created 事件。这一步把热路径和所有下游处理解耦开来，并提供持久性保证。即使下游服务一时变慢，也不会有任何一票丢失。

**第 5 步 - Kafka 消费者** 三个消费者各自独立处理这条事件：

**投票持久化 Worker** 异步地把投票写入 PostgreSQL 的 Votes 表。这是每一票的永久记录。

**分数重算 Worker** 重新计算这篇帖子的热度分数。Reddit 的热度算法是净赞成票数和发帖后经过时长的函数。较新但赞成票较少的帖子可以排在较旧但赞成票更多的帖子前面，因为时间衰减因子对「年龄」的惩罚非常激进。重算后的分数写回 Redis 的 Sorted Set：

ZADD subreddit:programming:hot <new\_hot\_score\> post\_456

subreddit 的 Hot 信息流就此即时更新，读取时无需任何计算。

**争议度分数 Worker** 单独计算争议度分数。当一篇帖子总票数很高、而赞成票和反对票几乎持平时，它就是有争议的。这个分数写入另一个独立的 sorted set：

ZADD subreddit:programming:controversial <controversial\_score\> post\_456

**第 6 步 - 周期性对账** Redis 投票计数器是快速读取路径，PostgreSQL 才是数据的唯一真实来源。一个后台作业周期性地把 Redis 计数器和数据库做对账，捕捉 Redis 重启或事件丢失造成的偏差。

**Votes 表（PostgreSQL）**

Votes vote\_id UUID，主键 user\_id UUID，外键 -\> Users post\_id UUID，外键 -\> Posts direction ENUM('up', 'down') voted\_at TIMESTAMP UNIQUE (user\_id, post\_id)

(user\_id, post\_id) 上的 UNIQUE 约束在数据库层面强制保证投票幂等性，作为兜底手段——即便热路径上的 Redis 检查已经处理了这件事。

投票服务的瓶颈

Redis 操作在亚毫秒级完成，从容应对投票的峰值吞吐量。Kafka 吸收爆款帖子带来的突发流量。如果单篇帖子每秒收到数千票，分数重算 Worker 可能变成瓶颈，因为每一票都会触发一次重算。缓解办法是在 Worker 内做重算的防抖：不再每收到一条投票事件就重算一次，而是把同一篇帖子在一小段时间窗口（比如 5 秒）内的投票事件攒成一批，每个窗口只重算一次。这样在爆款帖子上把重算开销降低了好几个数量级，同时分数依然足够新鲜。

## 6\. 信息流服务

Reddit 的信息流是基于 subreddit 的，而不是基于关注关系的。一个用户加入了 50 个 subreddit，主页信息流就展示这 50 个 subreddit 中最优质的帖子，按所选排序模式排列。

每个 subreddit、每种排序模式一个 Redis Sorted Set

每个 subreddit 维护四个 Redis Sorted Set，每种排序模式一个：

subreddit:{subreddit\_id}:hot 按 hot\_score 打分（赞成票 + 时间衰减） subreddit:{subreddit\_id}:new 按 posted\_at 时间戳打分 subreddit:{subreddit\_id}:top 按净赞成票数打分 subreddit:{subreddit\_id}:controversial 按争议度分数打分

这些集合由 Kafka 消费者随着投票涌入和新帖创建而实时更新。在正常路径下，信息流读取始终由 Redis 提供，绝不直接访问 Posts 库。

**扇出策略**

Reddit 采用和 Instagram 类似的混合扇出模型，只不过判断依据是 subreddit 的规模，而不是用户的粉丝数：

**写时扇出（小型 subreddit）：** 当一篇帖子发在小型 subreddit（成员数低于某个阈值，比如 10 万）时，信息流扇出 Worker 把这篇帖子直接写进所有成员的主页信息流 sorted set。每个成员的主页信息流就是一个 Redis Sorted Set：

Key: feed:{user\_id}:home Member: post\_id Score: hot\_score

**读时扇出（大型 subreddit）：** 对于 r/programming 或 r/worldnews 这类拥有数百万成员的大型 subreddit，写时扇出不现实。取而代之的做法是：在收到信息流请求时直接读取它们的 subreddit sorted set，再和用户预计算好的主页信息流合并。r/worldnews 里出现一篇爆款帖子，不会触发对数百万个主页信息流 sorted set 的写入。

主页信息流读取流程

用户打开 Reddit 时：

信息流服务从 Redis 的 SubredditMembers 缓存中取出该用户加入的 subreddit 列表

小型 subreddit：从预计算好的 feed:{user\_id}:home sorted set 中读取

大型 subreddit：直接从 subreddit:{subreddit\_id}:hot 中读取热门帖子

跨所有 subreddit 合并并重新排序结果

向客户端返回排名前 25 的帖子 ID

信息流服务通过 Redis MGET post:1 post:2 ... post:25 补全帖子数据

任何一篇帖子若缓存未命中，就回退到 Posts 库读取，并回写 Redis

客户端渲染信息流，展示完整的帖子元数据，媒体 URL 指向 CDN

**subreddit 信息流读取流程**

当用户带着某个排序模式打开一个具体的 subreddit 时：

信息流服务直接从 subreddit:{subreddit\_id}:{sort\_mode} sorted set 中读取

ZREVRANGE subreddit:programming:hot 0 24 瞬间返回排名前 25 的帖子 ID

帖子元数据通过 Redis MGET 补全

媒体由 CDN 提供

信息流服务的瓶颈

读取时针对大型 subreddit 的合并步骤会带来额外延迟，延迟大小与用户关注的大型 subreddit 数量成正比。缓解办法是限制每次请求读取的大型 subreddit 数量，并按用户把合并结果缓存在 Redis 里，设置一个较短的 TTL（30 到 60 秒）。不活跃用户的主页信息流 sorted set 会从 Redis 中淘汰，下次登录时再从 Posts 库重建。

## 7\. 评论服务

树状嵌套评论是 Reddit 最有特色的数据结构难题。一条评论可以有回复，回复还可以有回复，深度不限。每条评论都有自己的赞成/反对票分数。

邻接表数据模型

评论树在 PostgreSQL 中用邻接表（Adjacency List）模式存储。每条评论都保存一个指向父节点的引用：

**Comments** comment\_id UUID，主键 post\_id UUID，外键 -\> Posts parent\_id UUID，外键 -\> Comments（顶层评论为 NULL） user\_id UUID，外键 -\> Users content TEXT upvotes INTEGER downvotes INTEGER score INTEGER created\_at TIMESTAMP

parent\_id 为 NULL 表示这条评论是对帖子的直接回复。parent\_id 非空则表示它是对另一条评论的回复。仅凭这一列就能表达整棵树的结构，不需要图数据库。

在 (post\_id, parent\_id) 上建一个复合索引，就能让两条主要查询都很快：

```
Top-level comments for a post, sorted by score
SELECT * FROM Comments
WHERE post_id = X AND parent_id IS NULL
ORDER BY score DESC
LIMIT 20

Replies to a specific comment, sorted by score
SELECT * FROM Comments
WHERE parent_id = comment_123
ORDER BY score DESC
LIMIT 10
```

**懒加载**

Reddit 不会在一次请求里加载整棵评论树。一篇热门帖子的完整评论树可能有几十万条评论。它采用的是懒加载：

按分数拉取排名前 20 的顶层评论

对每条顶层评论，拉取排名前 3 的回复

更深的层级显示一个「加载更多回复」按钮，点击后触发单独的请求

不论评论树有多深，首屏加载都能保持很快。

评论投票流程

评论投票走的是和帖子投票完全相同的 Kafka 流水线。每条评论都有自己的分数计数器，由分数重算 Worker 维护。一个评论串内部的排序在每批投票处理完之后异步重算。

热门帖子的热门评论会以较短 TTL 缓存在 Redis 中，避免对同一评论串反复查询 PostgreSQL。

评论服务的瓶颈

邻接表模式配合懒加载，逐层拉取的效果很好。瓶颈出现在客户端试图一次性加载整棵树时的深度递归拉取。防范办法是在 API 层强制分页和深度限制。评论的写入量远低于投票，PostgreSQL 从容应对 Reddit 这个量级的评论写入。

## 8\. 搜索服务

Reddit 搜索覆盖三类实体：帖子（按标题和正文）、subreddit（按名称和描述）以及用户（按用户名）。

**存储引擎 - Elasticsearch**

Elasticsearch 负责这三类实体上的全文搜索、部分匹配和相关性排序。Reddit 的搜索主要是基于关键词的，不像 Instagram 那样以互动加权，但 subreddit 规模和帖子分数仍然会作为排序信号使用。

Elasticsearch 文档

**帖子文档：**

```
{
  "post_id": "123",
  "title": "Why Rust is taking over systems programming",
  "content": "A deep dive into memory safety...",
  "subreddit": "programming",
  "subreddit_id": "456",
  "author": "harry_dev",
  "score": 15000,
  "posted_at": "2026-06-10T10:00:00Z"
}
```

**subreddit 文档：**

```
{
  "subreddit_id": "456",
  "name": "programming",
  "description": "Computer programming discussion",
  "member_count": 5000000
}
```

帖子分数和 subreddit 成员数充当排序信号。对同一条搜索查询，大型 subreddit 中高赞成票的帖子会排在小型 subreddit 中低分帖子的前面。

保持 Elasticsearch 同步

搜索索引 Worker 消费 post.transcoded 和 post.created 这两个 Kafka topic 来处理新帖。分数更新则是批量攒起来周期性同步到 Elasticsearch，而不是每一票都同步，因为搜索结果中的精确分数并不需要实时。

热门搜索查询以较短 TTL 缓存在 Redis 中，以降低 Elasticsearch 在热门搜索上的负载。

**搜索的瓶颈**

和 Instagram、Spotify 的情况一样。针对热门查询的 Redis 缓存在请求到达 Elasticsearch 之前就过滤掉了大部分重复流量。剩下的查询量靠水平扩展 Elasticsearch 集群来消化。

## 9\. 完整数据流总结

**发帖上传流程：** 客户端 -\> API 网关（认证） -\> 帖子服务 -\> 预签名 S3 URL 客户端 -\> S3（直传） S3 -\> Kafka -\> 转码服务 -\> S3（所有变体） 转码服务 -\> Kafka（post.transcoded） -\> 元数据 Worker -\> Posts 库 + Redis 缓存 -\> 搜索 Worker -\> Elasticsearch -\> 信息流扇出 Worker -\> Redis 主页信息流 sorted set（小型 subreddit） -\> subreddit sorted set（所有 subreddit） **投票流程：** 客户端为帖子投赞成票 -\> 投票服务 -\> Redis 幂等性检查（SGET vote:user:post） -\> Redis INCR post:post\_id:upvotes -\> Kafka（vote.created） -\> 投票持久化 Worker -\> PostgreSQL Votes 表 -\> 分数重算 Worker -\> ZADD subreddit:id:hot <new\_score\> post\_id -\> 争议度分数 Worker -\> ZADD subreddit:id:controversial <score\> post\_id **主页信息流读取流程：** 客户端打开 Reddit -\> 信息流服务 -\> Redis：取出用户加入的 subreddit -\> Redis ZREVRANGE feed:{user\_id}:home（小型 subreddit 的帖子） -\> Redis ZREVRANGE subreddit:{id}:hot（大型 subreddit 的帖子，读时合并） -\> Redis MGET post:1 post:2 ... post:25（补全元数据） -\> CDN（媒体 URL 直接由 CDN 提供） subreddit 信息流读取流程：客户端打开 r/programming -\> 信息流服务 -\> Redis ZREVRANGE subreddit:programming:hot 0 24 -\> Redis MGET（补全帖子元数据） -\> CDN（媒体） **评论流程：** 客户端加载帖子 -\> 评论服务 -\> PostgreSQL：顶层评论 WHERE post\_id = X AND parent\_id IS NULL ORDER BY score DESC LIMIT 20 -\> PostgreSQL：每条顶层评论取前 3 条回复 -\> 更深层级按需懒加载 **搜索流程：** 客户端搜索 "rust programming" -\> 搜索服务 -\> Redis（热门查询缓存） -\> Elasticsearch（缓存未命中时） -\> Redis MGET（为结果补全帖子元数据）

## 10\. 韧性与容错

**信息流用 Redis Sorted Set** 意味着正常路径下的信息流读取从不触碰 Posts 库。即使 PostgreSQL 变慢或短暂不可用，用户依然能从 Redis 看到自己的信息流。新帖子会暂时不再出现，但已有的信息流内容不受影响。

**Kafka 的持久性** 保证即使在流量高峰期也不会丢失任何一票。投票会一直留在 Kafka 中，直到分数重算 Worker 处理完为止。一篇每小时产生 10 万票的爆款帖子不会压垮系统，因为 Kafka 吸收了这波突发，Worker 按自己的节奏消费。

**投票防抖** 避免分数重算 Worker 在爆款帖子上被压垮。把投票事件按 5 秒窗口攒批，能把重算开销降低好几个数量级，同时分数的新鲜度依然可以接受。

**投票的双层幂等性**：热路径上的 Redis 检查，加上作为兜底的 PostgreSQL UNIQUE 约束。即便在重试或网络故障的情况下，用户也绝不会对同一篇帖子投两次票。

**邻接表加懒加载** 避免评论拉取变得无边界。在 API 层强制深度限制和分页，可以保护 PostgreSQL 免受失控的递归查询之害。

**Cassandra 复制** 用于所有高写入量的存储，确保节点故障时不丢数据。

**CDN 缓存** 用于媒体，头像、帖子图片和视频分片都由全球边缘节点提供，缓存命中时源站完全不参与。

## 11\. 技术选型总结

**API 网关** -\> Kong / AWS API Gateway（认证、限流、路由）

**对象存储** -\> AWS S3（不可变的媒体文件，持久可靠，对 CDN 友好）

**CDN** -\> CloudFront / Akamai（帖子媒体和头像的边缘缓存）

**转码** -\> 通过 Kafka 驱动的水平扩展 Worker 池（图片变体、HLS 视频切片）

**解耦** -\> Apache Kafka（投票持久性、发帖扇出、分数重算、搜索索引）

**搜索引擎** \-\> Elasticsearch（全文搜索、部分匹配、按分数加权的排序）

**帖子 / 用户 / subreddit 数据库** -\> PostgreSQL 分片 + 副本（关系型结构、强一致性、复合索引）

**信息流存储** -\> 每个 subreddit、每种排序模式一个 Redis Sorted Set（亚毫秒级的排序信息流读取）

**投票计数器** \-\> Redis INCR（近实时票数，且不给数据库压力）

**投票幂等性** \-\> Redis Set + PostgreSQL UNIQUE 约束（双层重复防护）

**主页信息流缓存** -\> 每用户一个 Redis Sorted Set（小型 subreddit 扇出的预计算主页信息流）

**帖子元数据缓存** \-\> Redis MGET（一次往返批量补全帖子元数据）

**评论存储** -\> PostgreSQL 邻接表（树状嵌套评论，按深度懒加载）

**分析存储** \-\> Cassandra / ClickHouse（高写入量、时序投票数据）

以上就是全部内容，各位……干杯！！！
