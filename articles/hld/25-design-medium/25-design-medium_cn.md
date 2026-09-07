---
title: "Design Medium"
url: "https://x.com/Harry_The_Nerd/status/2084263186199638124"
category: "HLD"
date: "2026-08-03"
description: "System design for an open publishing and social journalism platform like Medium"
lang: "zh-CN"
---

# 设计 Medium

> 类似 Medium 的开放出版与社交新闻平台的系统设计
>
> 原文：[https://x.com/Harry_The_Nerd/status/2084263186199638124](https://x.com/Harry_The_Nerd/status/2084263186199638124) · 2026-08-03

![封面图](https://pbs.twimg.com/media/HOuWxFYaIAANN1Y.jpg)

## 1\. 需求

**功能性需求**

- 作者与读者的资料管理
- 创建、编辑、发布和删除文章（富文本、内嵌图片、代码块）
- 带自动保存的草稿管理
- 给文章鼓掌（每个用户对每篇文章最多 50 次）
- 文章下的嵌套评论
- 关注与取消关注作者
- 首页信息流（已关注作者的文章 + 基于话题的推荐）
- 每篇文章的标签和话题
- 搜索文章、作者和话题
- 阅读时长估算
- 通知（关注的作者发新文章、鼓掌、评论、新粉丝）

**不在范围内**

- 付费会员与付费墙文章
- Medium 合作伙伴计划（作者收益）
- 转发功能
- Publications（多作者协作专栏）
- 文章的音频朗读

**非功能性需求**

- 文章阅读高可用，这是最频繁的操作
- 信息流读取延迟低于 100ms
- 自动保存必须感觉是瞬时的，作者不应察觉到写入延迟
- 鼓掌数、信息流排序和推荐结果可以接受最终一致性
- 文章发布状态（草稿 vs 已发布）必须强一致
- 读写比约为 1000:1，重度优化读路径
- 用户绝不能在信息流中看到自己已经读过的文章

## 2\. 容量估算

- **日活用户（DAU）：** 100 万
- **每天发布的文章数：** DAU 的 1-5% = 每天约 1 万篇
- **每篇文章的文本内容：** 约 50 KB
- **每篇文章的图片：** 约 50% 的文章有 2 张图，每张 500 KB
- **每天的文本存储：** 1 万 x 50 KB = 每天约 500 MB（可忽略）
- **每天的图片存储：** 1 万 x 0.5 x 2 x 500 KB = 每天约 5 GB
- **每天总存储：** 约 5 GB，主要来自图片
- **任意时刻的活跃草稿：** 约为已发布文章的 10 倍 = 约 10 万篇正在编辑的草稿
- **自动保存写入：** 10 万篇活跃草稿 x 每 5 秒一次自动保存 = 写作高峰时段约每秒 2 万次写入
- **读写比：** 约 1000:1，一篇爆款文章写一次会被读上万次

主要挑战不是存储规模，而是自动保存的写入量和信息流个性化的质量。Medium 既是发布平台，也是内容发现平台，信息流体验决定了读者会不会每天回来。

## 3\. API 网关

所有客户端请求都经过 API 网关，由它处理认证、限流和路由。媒体上传（文章图片）在认证之后绕过网关，用预签名 S3 URL 由客户端直传 S3。API 网关从不处理二进制媒体负载数据。

Medium 有相当可观的未认证读流量。很多读者是通过 Google 搜索或分享链接进来的，并没有登录。API 网关会把未认证的文章读请求直接路由到文章读取路径，省去认证开销；而写操作（鼓掌、评论、关注、发布）则始终要求认证。

## 4\. 用户资料服务

Medium 有两类相互重叠的用户：发布内容的作者和消费内容的读者。大多数用户同时是两者。

**数据模型**

**Users 表（PostgreSQL）**

Users user\_id UUID，主键 username VARCHAR，唯一 email VARCHAR，唯一 display\_name VARCHAR bio TEXT profile\_pic\_url TEXT website\_url TEXT twitter\_handle VARCHAR follower\_count INTEGER following\_count INTEGER article\_count INTEGER total\_claps\_received INTEGER created\_at TIMESTAMP

follower\_count、following\_count、article\_count 和 total\_claps\_received 都是反范式计数器，由 Kafka 消费者异步更新。在这个量级上，读取时再从原始表里聚合会太慢。

**Follows 表（PostgreSQL）**

Follows follower\_id UUID，外键 -\> Users followee\_id UUID，外键 -\> Users created\_at TIMESTAMP PRIMARY KEY (follower\_id, followee\_id)

关注就是一次 INSERT，取关就是一次 DELETE。「用户 X 关注了哪些作者」就是 WHERE follower\_id = X。两列都建索引，两个方向的查询都很快。

**UserTopics 表（PostgreSQL）**

UserTopics user\_id UUID，外键 -\> Users topic VARCHAR interest\_score FLOAT（由推荐流水线更新）created\_at TIMESTAMP PRIMARY KEY (user\_id, topic)

显式的话题兴趣（用户手动关注某个话题）和推断出的兴趣（推荐流水线从阅读行为中推导）都存在这里。interest\_score 由机器学习流水线基于阅读历史信号定期更新。

**缓存策略**

作者资料缓存在 Redis 中，因为每次打开文章页都要取。热门作者（粉丝多、文章出现在大量信息流里的那些）从缓存中获益最大。用户的话题兴趣分数也按用户缓存，方便快速组装信息流。

## 5\. 文章服务

文章服务管理文章的完整生命周期，从创建草稿到编辑、发布和删除。

**文章存储架构**

Medium 的文章不是简单的文本字符串，而是包含标题、段落、图片、代码块、引言和内嵌链接的富文档。这种结构以 **JSON 文档**形式存在 MongoDB 中，而不是以裸文本存在 PostgreSQL 里。

文章内容选 MongoDB 而不是 PostgreSQL，原因是：

- 每篇文章的 schema 可以灵活变化（不同文章有不同的块结构）
- 高效存储和读取大型嵌套 JSON 文档
- 文章内容不需要做复杂 join
- 原生支持文档局部更新（改一个小节不必重写整篇文档）

PostgreSQL 存文章元数据（标题、slug、状态、作者、鼓掌数）。MongoDB 存文章内容（完整的富文本文档）。S3 存文章里引用的内嵌图片。

文章内容结构（MongoDB）

```json
{
  "article_id": "uuid",
  "blocks": [
    { "type": "heading", "level": 1, "text": "Why I Switched to Rust" },
    { "type": "paragraph", "text": "It started on a Tuesday morning..." },
    { "type": "image", "url": "https://cdn.medium.com/...", "caption": "My workspace" },
    { "type": "code", "language": "rust", "content": "fn main() { ... }" },
    { "type": "paragraph", "text": "The borrow checker felt restrictive at first..." }
  ],
  "updated_at": "2026-06-20T10:00:00Z"
}
```

这种基于块的结构，和 Notion、Substack 以及 Medium 自家编辑器（背后是类似格式）表示富内容的方式很像。

**文章元数据表（PostgreSQL）**

**Articles** article\_id UUID，主键 author\_id UUID，外键 -\> Users title VARCHAR subtitle VARCHAR cover\_image\_url TEXT slug VARCHAR，唯一（发布前为 null）status ENUM('draft', 'published', 'deleted') reading\_time INTEGER（分钟，发布时计算）clap\_count INTEGER（反范式总数）comment\_count INTEGER（反范式计数器）view\_count INTEGER（反范式计数器）published\_at TIMESTAMP（发布前为 null）created\_at TIMESTAMP updated\_at TIMESTAMP

标签与话题

**ArticleTags 表（PostgreSQL）**

ArticleTags article\_id UUID，外键 -\> Articles tag VARCHAR PRIMARY KEY (article\_id, tag)

标签由作者设置。话题是更宽泛的分类，可以是推断的也可以是手动选的。两者都会索引进 Elasticsearch 供搜索使用，也会被推荐流水线用于基于话题的信息流个性化。

**草稿自动保存流程**

作者打字时，Medium 每隔几秒就自动保存一次。10 万篇活跃草稿、每 5 秒保存一次，高峰时段就是每秒约 2 万次写入。把每次自动保存都直接写 MongoDB 会把数据库压垮。

**自动保存流程用 Redis 做写缓冲：**

作者打字，客户端对按键做防抖，每 3-5 秒发一次自动保存请求

文章服务立即把草稿内容写入 Redis：

Key: draft:{article\_id} Value: 序列化后的文章内容 JSON TTL: 7 天

立刻给客户端返回响应（Redis 写入是亚毫秒级的）

向 Kafka 发布自动保存事件：draft.updated

Kafka 消费者按草稿维度每 30 秒把草稿内容从 Redis 异步刷写到 MongoDB

作者从来不用等 MongoDB。Redis 吸收了全部自动保存写入。MongoDB 收到的是可持续速率下的批量更新。

缓存未命中时（Redis 已淘汰该草稿，作者 7 天后才回来），草稿会从 MongoDB 取回并重新写入 Redis。

**发布流程**

当作者点击「发布」：

文章服务从 Redis 读取最终草稿内容（缓存未命中则从 MongoDB 读）

从标题生成唯一的 URL slug（例如 "why-i-switched-to-rust-a1b2c3"）

计算阅读时长：word\_count / 200（成人平均阅读速度，单位是每分钟词数）

更新 Articles 表：status = 'published'，slug = generated\_slug，published\_at = NOW()

把最终内容写入 MongoDB（永久的已发布版本）

向 Kafka 发布 article.published 事件

Kafka 消费者：**搜索索引 Worker** 把文章索引进 Elasticsearch **信息流扩散 Worker** 把文章写入所有粉丝的信息流有序集合 **通知 Worker** 通知所有粉丝有新文章 **话题索引 Worker** 用新文章更新 Redis 中的话题有序集合

**图片上传流程**

**当作者在编辑器里插入图片时：**

客户端向文章服务申请预签名 S3 URL

客户端把图片直传 S3

图片处理 Worker 生成多种分辨率变体（信息流卡片用缩略图、正文用中等尺寸、展开查看用原图）

CDN 挡在 S3 前面，在全球范围内以高缓存命中率提供图片

**文章服务的瓶颈**

自动保存的 Redis 写缓冲是防止 MongoDB 被压垮的关键架构决策。发布流程量很小（每天 1 万次），没有瓶颈方面的顾虑。图片处理流水线和之前所有系统的模式一样，通过 Kafka Worker 水平扩展。

## 6\. 信息流服务

Medium 的信息流结合两类信号：用户关注的作者发的文章，以及基于话题的推荐。这比 Twitter 纯粹基于关注的信息流复杂，但比 Netflix 的多阶段推荐流水线简单。

**对所有用户都采用写时扩散**

不像 Instagram 和 Twitter 那样，拥有数百万粉丝的名人必须走读时扩散；Medium 最大的作者也不过几十万粉丝。对所有用户都用写时扩散完全可行，不需要混合模式。

一篇文章发布时，信息流扩散 Worker 会把它写进每个粉丝的信息流有序集合：

Key: feed:{user\_id} Member: article\_id Score: 加权相关性分数（时新度 + 关注加成 + 互动信号）TTL: 7 天

加权分数公式

Score = base\_engagement\_score + follow\_boost + recency\_decay

- base\_engagement\_score 反映文章发布以来收到的互动量（鼓掌、评论、阅读）
- follow\_boost 加给用户关注的作者所发的文章，保证关注内容有优先级，但又不至于完全霸屏
- recency\_decay 保证在互动量相同的情况下新文章分数更高，防止陈旧内容一直占据顶部

这种加权合并意味着信息流是关注内容和话题内容的混合，而不是严格的先后排序。一条高度相关的新鲜话题推荐可以排在某位关注作者的旧文章之前，这与 Medium 实际的信息流表现一致。

**基于话题的推荐**

推荐流水线定期运行，为每个用户计算基于话题的文章推荐，依据包括：

- **阅读历史信号：** 用户读完了的文章（completion\_pct 高）的话题，权重高于那些点开就跳出的文章
- **协同过滤：** 和该用户读了相同文章的人还读了 Y
- **话题图谱：** 相邻话题（自助成长和诗歌的读者有重合；后端工程和系统设计的读者有重合）
- **关注作者的话题：** 从用户关注的作者所写话题中推断兴趣

**排名靠前的话题推荐按用户写入 Redis：**

Key: recommendations:{user\_id} Value: \[article\_id\_1, article\_id\_2, ..., article\_id\_20\] TTL: 6 小时

组装信息流时，信息流服务把预计算好的关注流有序集合与话题推荐合并，套上加权评分，返回排名前 20 的文章 ID。

**用布隆过滤器排除已读文章**

用户绝不能在信息流中看到已经读过的文章。阅读历史服务为每个用户维护一个布隆过滤器：

BF.ADD seen:{user\_id} article\_id（用户打开文章时）BF.EXISTS seen:{user\_id} article\_id（插入信息流有序集合前）

任何文章在被插入用户的信息流有序集合或作为推荐返回之前，都会先做 BF.EXISTS 检查。如果已经读过就排除掉。没有假阴性意味着读过的文章绝不会再次出现。少量假阳性（偶尔把一篇没读过的文章排除掉）是可以接受的。

**信息流读取流程**

当用户打开 Medium 首页信息流：

信息流服务通过 ZREVRANGE feed:user\_123 0 49 从 feed:{user\_id} 有序集合取前 50 个文章 ID

与 recommendations:{user\_id} Redis 键中的话题推荐合并

用布隆过滤器排除已读文章

按加权分数对合并后的列表重新排序

通过 Redis MGET 填充前 20 篇文章的元数据（标题、副标题、封面图、作者、阅读时长、鼓掌数）

任何文章缓存未命中时，回退到 PostgreSQL 并回写 Redis

把渲染好的信息流卡片返回给客户端

**信息流服务的瓶颈**

对一位有 10 万粉丝的头部作者做写时扩散，意味着每发布一篇文章要写 10 万个有序集合。按每天 1 万篇文章、平均每位作者 500 粉丝算，就是每天 500 万次有序集合写入，用水平扩展的扩散 Worker 池完全能应付。信息流有序集合的 Redis 内存占用受 7 天 TTL 和每个信息流有序集合最多 200 篇文章的上限约束。

## 7\. 鼓掌服务

鼓掌服务处理 Medium 独有的互动机制。单个读者对一篇文章最多可以鼓掌 50 次。文章上显示的总鼓掌数是所有读者鼓掌次数的总和，而不是简单统计有多少人鼓过掌。

**Redis 结构**

**按用户维度的鼓掌数（用于执行上限）：**

Key: claps:{article\_id}:{user\_id} Value: 整数（1 到 50）TTL: 30 天

处理一次鼓掌前先执行 GET claps:{article\_id}:{user\_id}。如果值等于 50，就拒绝这次鼓掌（已达上限）。否则执行 INCR claps:{article\_id}:{user\_id}。

**总鼓掌数（在文章上展示）：**

Key: article:claps:{article\_id} Value: 整数（所有用户累计的总数）

每一次通过了单用户上限检查的鼓掌，也会执行 INCR article:claps:{article\_id}。展示总鼓掌数只需一次 GET article:claps:{article\_id}，读取时不需要 SUM 查询。

**完整鼓掌流程**

用户点击鼓掌按钮（可以快速连点多次）

客户端对点击做防抖并批量打包（每 1 秒发送一次累积的鼓掌次数）

鼓掌服务收到 { user\_id, article\_id, clap\_increment }

GET claps:{article\_id}:{user\_id} -\> 当前用户的鼓掌数

计算允许的增量：min(clap\_increment, 50 - current\_count)

如果允许的增量 \> 0：INCRBY claps:{article\_id}:{user\_id} allowed\_increment INCRBY article:claps:{article\_id} allowed\_increment 向 Kafka 发布 clap.created 事件

Kafka 消费者：**鼓掌持久化 Worker** upsert 到 PostgreSQL 的 Claps 表 **计数器同步 Worker** 定期把 Redis 中的总数同步到 Articles 表的 clap\_count **通知 Worker** 通知文章作者（做聚合，不是每次鼓掌发一条通知）

**Claps 表（PostgreSQL）**

Claps user\_id UUID，外键 -\> Users article\_id UUID，外键 -\> Articles clap\_count INTEGER（1 到 50）last\_clapped TIMESTAMP PRIMARY KEY (user\_id, article\_id)

在 (user\_id, article\_id) 上做 UPSERT，配合 clap\_count = clap\_count + allowed\_increment，一次操作就同时处理了首次鼓掌和后续鼓掌。

**鼓掌服务的瓶颈**

爆款文章上的鼓掌洪峰（成千上万读者同时鼓掌）完全由 Redis 的 INCRBY 操作吸收，它是原子的且亚毫秒级。Kafka 消费者负责异步持久化，洪峰期间不会给 PostgreSQL 带来任何写压力。Redis 总数与 PostgreSQL clap\_count 之间的定期对账能捕捉任何偏差。

## 8\. 评论服务

Medium 的评论是嵌套的，但层级比 Reddit 浅。相比深层评论串，Medium 更鼓励「回应」（以独立文章形式写的回复）。评论量与鼓掌相比要少得多。

数据模型

**Comments 表（PostgreSQL）**

Comments comment\_id UUID，主键 article\_id UUID，外键 -\> Articles parent\_id UUID，外键 -\> Comments（顶层评论为 null）user\_id UUID，外键 -\> Users content TEXT clap\_count INTEGER created\_at TIMESTAMP updated\_at TIMESTAMP

和 Reddit 相同的邻接表模式。parent\_id = NULL 表示文章下的顶层评论，parent\_id 非空表示对另一条评论的回复。顶层评论按鼓掌数排序（最受认可的评论排前面），同一串里的回复按时间顺序排列。

(article\_id, parent\_id) 上的复合索引让两个主要查询都很快：

- 顶层评论：WHERE article\_id = X AND parent\_id IS NULL ORDER BY clap\_count DESC LIMIT 20
- 某条评论的回复：WHERE parent\_id = comment\_123 ORDER BY created\_at ASC

热门文章的评论区以较短 TTL 缓存在 Redis 中。每日阅读量最高的那些文章从评论缓存中获益显著。

评论服务的瓶颈

以 Medium 的规模，评论量并不大，PostgreSQL 从容应付评论写入。主要顾虑是爆款文章的评论区会遭遇几千次同时读取，Redis 缓存顶层评论就能吸收掉。

## 9\. 搜索服务

Medium 的搜索覆盖三类实体：文章（按标题、内容和标签）、作者（按姓名和用户名）和话题。

存储引擎 - Elasticsearch

Elasticsearch 负责跨全部文章内容的全文搜索、部分匹配和相关性排序。Medium 的搜索主要是基于关键词和话题的，不像 Instagram 那样以互动量加权，但鼓掌数和阅读数会作为排序信号。

**文章文档：**

```json
{
  "article_id": "123",
  "title": "Why I Switched to Rust After 10 Years of Python",
  "subtitle": "A pragmatic engineer's perspective",
  "author_id": "456",
  "author_name": "harry_dev",
  "tags": ["rust", "python", "programming", "backend"],
  "content_preview": "It started on a Tuesday morning when my service crashed for the third time...",
  "reading_time": 8,
  "clap_count": 4200,
  "view_count": 18000,
  "published_at": "2026-06-20T10:00:00Z"
}
```

完整的文章内容不会索引进 Elasticsearch，只索引一段内容预览（前 500 个字符）和标签。对全文做索引会让索引体积极其庞大，而大多数搜索意图靠标题、标签和预览的匹配就能满足。

**作者文档：**

```json
{
  "user_id": "456",
  "username": "harry_dev",
  "display_name": "Harry Singh",
  "bio": "Backend engineer. Writing about distributed systems and Rust.",
  "follower_count": 12000,
  "article_count": 47
}
```

排序信号

Medium 的搜索排序综合了：

- **文本匹配分**（标题匹配权重高于标签匹配，标签匹配权重高于预览匹配）
- **鼓掌数**（同一查询下，更受认可的文章排更前）
- **时新度**（宽泛查询下新文章有排名加成）
- **作者粉丝数**（成名作者略微靠前）

保持 Elasticsearch 同步

搜索索引 Worker 消费 article.published 和 article.updated 两个 Kafka 主题。鼓掌数和浏览数的更新会批量攒起来定期同步到 Elasticsearch，而不是每次鼓掌都同步，因为搜索结果里的互动数不需要精确实时。

## 10\. 阅读历史服务

阅读历史服务追踪每个用户读过哪些文章、读得有多深。这些数据既喂给推荐流水线，也支撑用于信息流去重的布隆过滤器。

**数据模型**

**ReadingHistory 表（PostgreSQL，按 user\_id 分区）**

ReadingHistory user\_id UUID，外键 -\> Users article\_id UUID，外键 -\> Articles read\_at TIMESTAMP completion\_pct FLOAT（0.0 到 1.0，用户滚动到了多深）time\_spent\_sec INTEGER（实际阅读时长，单位秒）

completion\_pct 是最有价值的信号。一个读完了 8 分钟 Rust 文章 95% 内容的用户，对 Rust 的兴趣远高于一个点开就立刻跳出的用户。这种区分驱动了推荐流水线里的话题兴趣打分。

**写入流程**

每次文章浏览都会产生一条阅读事件。按 100 万 DAU、平均每人每天读 5 篇算，就是每天 500 万条阅读事件。写入量大，且绝不能阻塞阅读体验。

阅读事件由客户端异步发布到 Kafka：

用户打开文章 -\> 客户端发送 GET /articles/{slug} -\> 文章返回

客户端在后台追踪滚动位置和停留时间

文章关闭时或 30 秒后：客户端发送阅读事件 { user\_id, article\_id, completion\_pct, time\_spent\_sec }

阅读历史服务发布到 Kafka：[article.read](https://x.com/Harry_The_Nerd/status/article.read)

Kafka 消费者：**历史持久化 Worker** 写入 ReadingHistory 表 **布隆过滤器 Worker** 调用 BF.ADD seen:{user\_id} article\_id **浏览计数 Worker** 在 Redis 中执行 INCR article:views:{article\_id} **推荐信号 Worker** 根据 completion\_pct 更新 UserTopics 的兴趣分数

**阅读历史服务的瓶颈**

每天 500 万条阅读事件平均下来约每秒 58 条，远在 Kafka 的能力范围之内。布隆过滤器写入是亚毫秒级的 Redis 操作。PostgreSQL 按 user\_id 分区让按用户查历史保持很快。

## 11\. 通知服务

架构和之前所有系统相同。从各个互动服务经 Kafka 扇出，iOS 走 APNs，Android 走 FCM，邮件走 SES。

通知服务订阅：

- article.published -\> 向所有粉丝推送「Harry 发布了新文章」
- clap.created -\> 向文章作者推送聚合后的鼓掌通知（「Harry 和另外 42 人为你的文章鼓掌」）
- comment.created -\> 向文章作者和父评论作者推送评论通知
- follow.created -\> 「有人关注了你」通知

**通知聚合**

鼓掌通知必须做激进的聚合。一篇爆款文章一小时内收到 5000 次鼓掌，不能给作者发 5000 条推送。通知服务把 1 小时窗口内的鼓掌事件攒成一批，只发一条通知：「你的文章在过去一小时内收到了 5247 次鼓掌。」

关注作者的新文章通知则逐条投递，因为每一条都对应读者想知道的一个独立内容事件。

## 12\. 完整数据流总结

文章创建流程（草稿）：作者打字 -\> 客户端防抖 -\> 每 3-5 秒自动保存 -\> 文章服务 -\> Redis WRITE draft:{article\_id}（亚毫秒）-\> Kafka（draft.updated）-\> MongoDB 写入器（异步，每篇草稿每 30 秒刷写一次）图片上传流程：作者插入图片 -\> 文章服务 -\> 预签名 S3 URL 作者 -\> S3（直传）S3 -\> Kafka -\> 图片处理 Worker -\> S3（缩略图、中等尺寸、原图变体）CDN 缓存所有变体 发布流程：作者点击发布 -\> 文章服务 -\> 生成 slug，计算 reading\_time -\> UPDATE Articles: status = 'published', published\_at = NOW() -\> 把最终内容写入 MongoDB -\> Kafka（article.published）-\> 搜索 Worker -\> Elasticsearch 索引 -\> 信息流扩散 Worker -\> 对所有粉丝执行 Redis ZADD feed:{follower\_id} -\> 通知 Worker -\> 向所有粉丝走 APNs / FCM / SES -\> 话题索引 Worker -\> 更新 Redis 话题有序集合 信息流读取流程：用户打开 Medium -\> 信息流服务 -\> Redis ZREVRANGE feed:{user\_id} 0 49（关注类文章）-\> Redis GET recommendations:{user\_id}（话题类文章）-\> 对每个候选执行布隆过滤器 BF.EXISTS seen:{user\_id}（排除已读文章）-\> 加权分数合并与重排 -\> Redis MGET（填充前 20 篇的文章元数据）-\> CDN（封面图直接返回）文章阅读流程：用户打开文章 -\> 文章服务 -\> Redis（文章元数据缓存）-\> MongoDB（完整文章内容）-\> CDN（内嵌图片）-\> 客户端追踪滚动 + 停留时间 -\> 关闭时：向 Kafka 发布阅读事件（[article.read](https://x.com/Harry_The_Nerd/status/article.read)）-\> 历史持久化 Worker -\> ReadingHistory 表（PostgreSQL）-\> 布隆过滤器 Worker -\> BF.ADD seen:{user\_id} article\_id -\> 浏览计数 Worker -\> INCR article:views:{article\_id} -\> 推荐信号 Worker -\> 更新 UserTopics 兴趣分数 鼓掌流程：用户鼓掌 -\> 客户端防抖并批量打包 -\> 鼓掌服务 -\> GET claps:{article\_id}:{user\_id}（当前用户鼓掌数）-\> 计算允许的增量（上限 50）-\> INCRBY claps:{article\_id}:{user\_id} allowed\_increment -\> INCRBY article:claps:{article\_id} allowed\_increment -\> Kafka（clap.created）-\> 鼓掌持久化 Worker -\> UPSERT Claps 表（PostgreSQL）-\> 计数器同步 Worker -\> UPDATE Articles.clap\_count（定期批处理）-\> 通知 Worker -\> 向作者推送聚合后的鼓掌通知 搜索流程：用户搜索 "rust backend" -\> 搜索服务 -\> Redis（热门查询缓存）-\> Elasticsearch（标题 + 标签 + 预览匹配，按鼓掌数 + 时新度排序）-\> Redis MGET（填充结果的文章元数据）

## 13\. 韧性与容错

**Redis 自动保存缓冲**保证即使 MongoDB 暂时不可用，作者也绝不会丢失工作内容。草稿在 Redis 中以 7 天 TTL 存活，数据库恢复后 Kafka 消费者会把积压的 MongoDB 写入追上。

**用于信息流去重的布隆过滤器**没有假阴性，读过的文章绝不会再次出现。少量假阳性（偶尔排除一篇没读过的文章）完全可以接受，用户根本察觉不到。

**对所有用户都用写时扩散**意味着无论数据库状态如何，信息流读取始终很快。Redis 中的信息流有序集合是主读路径，只有在完全缓存未命中时才会打到 PostgreSQL。

**Kafka 的持久性**贯穿整条流水线，即使下游服务暂时不可用，也不会丢失任何文章发布事件、鼓掌事件或阅读事件。所有消费者恢复后都会从 Kafka 位点追上进度。

**Redis 鼓掌计数器加异步同步 PostgreSQL**吸收了爆款文章上的鼓掌洪峰，数据库不承受任何写压力。定期对账能发现 Redis 总数与 PostgreSQL clap\_count 之间的任何偏差。

**用 MongoDB 存文章内容**把内容存储与元数据存储解耦。MongoDB 变慢只影响文章正文的读取，不影响信息流组装、搜索结果或个人主页，这些都走 PostgreSQL 和 Redis。

**CDN 缓存**让所有文章图片都从全球边缘节点返回。文章图片极少变化（作者发布后不会再改图），所以缓存命中率非常高。

**Elasticsearch 的最终一致性**在文章索引上是可以接受的。刚发布的文章可能有几秒钟搜不到，因为搜索索引 Worker 还在处理那条 Kafka 事件。读者主要通过信息流而不是搜索发现新文章。

## 14\. 技术选型总结

**API 网关** -\> Kong / AWS API Gateway（认证、限流、未认证读请求路由）

**文章内容存储** -\> MongoDB（富文本 JSON 文档、灵活 schema、大型嵌套内容）

**文章元数据数据库** -\> PostgreSQL 分片 + 副本（标题、slug、状态、计数器，发布状态强一致）

**草稿自动保存缓冲** \-\> Redis（亚毫秒级自动保存写入，7 天 TTL，经 Kafka 异步刷写到 MongoDB）

**图片存储** \-\> AWS S3（文章内嵌图片，持久可靠，天然适配 CDN）

**CDN** -\> CloudFront / Akamai（文章图片的边缘缓存，缓存命中率高）

**解耦** -\> Apache Kafka（文章发布扇出、鼓掌持久化、阅读事件、通知触发）

**搜索引擎** -\> Elasticsearch（标题、标签、预览的全文搜索，鼓掌数加权排序）

**信息流存储** -\> 按用户的 Redis 有序集合（关注类与话题类文章的加权分数合并）

**话题推荐** -\> 按用户的 Redis（预计算的话题文章推荐，6 小时 TTL）

**已读去重** -\> 按用户的 Redis 布隆过滤器（从信息流中排除已读文章）

**鼓掌计数器（单用户上限）** -\> Redis INCRBY 配合 GET 上限检查（原子地执行 50 次上限）

**鼓掌计数器（文章总数）** -\> Redis INCRBY（累计总数，展示时一次 GET）

**鼓掌数据库** -\> PostgreSQL UPSERT（单用户鼓掌数的事实来源）

**阅读历史数据库** \-\> 按 user\_id 分区的 PostgreSQL（完成率、停留时长、推荐信号）

**用户资料缓存** -\> Redis（作者资料、粉丝数、话题兴趣分数）

**通知投递** -\> APNs + FCM + AWS SES（推送通知和邮件，鼓掌通知做激进聚合）

以上就是全部内容，各位……干杯！！
