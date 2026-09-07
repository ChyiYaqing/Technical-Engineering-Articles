---
title: "Design Amazon"
url: "https://x.com/Harry_The_Nerd/status/2070543086258979132"
category: "HLD"
date: "2026-06-26"
description: "System design for an e-commerce platform like Amazon."
lang: "zh-CN"
---

# 设计 Amazon

> 类似 Amazon 的电商平台的系统设计。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2070543086258979132](https://x.com/Harry_The_Nerd/status/2070543086258979132) · 2026-06-26

![封面图](https://pbs.twimg.com/media/HLaSmhSbwAA0eKN.jpg)

## 高层设计：Amazon

## 1\. 需求

功能性需求

- 用户资料与卖家资料管理
- 商品目录（浏览并查看商品详情）
- 商品搜索
- 购物车管理（添加、删除、修改数量）
- 下单与订单追踪
- 支付处理
- 库存管理
- 仓储与履约流水线
- 商品评价与评分
- 通知（订单确认、发货更新、送达）

不在范围内

- 机器学习推荐流水线
- 客服聊天机器人
- Amazon Prime / 订阅管理
- Amazon 广告与推广位
- 退货与退款流程
- 仓储机器人与实体履约的内部细节

非功能性需求

- 商品目录、搜索、订单服务都要具备高可用性
- 价格、库存数量、支付状态要保证强一致性
- 商品描述、评价、推荐可以接受最终一致性
- 任何流量条件下都不能超卖，包括秒杀
- 支付必须恰好执行一次，用户绝不能被重复扣款
- 商品搜索和目录读取的延迟要低于 100ms
- 订单履约流水线必须能通过补偿事务优雅地处理局部失败

## 2\. 容量估算

- **日活用户（DAU）：** 5000 万
- **每日订单数：** 转化率 2%-3% -\> 每天 100 万到 150 万单
- **目录中的商品数：** 3.5 亿
- **平均商品图片：** 每个商品 5 张图，每张 1 MB -\> 每个商品 5 MB
- **图片总存储：** 3.5 亿 x 5 MB -\> 约 1.75 PB（写入一次，极少更新）
- **演示视频：** 约 10% 的商品有演示视频，每个 50 MB -\> 额外约 1.75 PB
- **商品元数据：** 每个商品约 10 KB -\> 3.5 亿 x 10 KB -\> 元数据约 3.5 TB
- **每日订单数据：** 100 万单 x 每条订单记录 10 KB -\> 每天约 10 GB（可忽略）
- **峰值流量：** 秒杀会在库存上制造极端的写入尖峰，单个商品可达每秒 10 万次请求

存储上的主要压力来自 PB 级的商品媒体文件，它们写入一次、极少更新。而系统层面最大的挑战是秒杀条件下的库存一致性，而不是原始存储量。社交平台最难的问题是读取时的信息流生成，Amazon 不一样：它最难的问题是集中购买尖峰下的写入一致性。

## 3\. API 网关

所有客户端请求都从 API 网关进入，由网关负责认证、限流和路由。和前面每一个设计一样的约束在这里依然成立：API 网关绝不出现在二进制媒体负载数据的传输路径上。卖家上传商品图片和视频时，认证之后就绕开网关，使用预签名的 S3 URL 由客户端直传 S3。

相比社交平台，Amazon 的 API 网关多了一项职责：**请求分类**。读请求（浏览、搜索、查看商品页）和写请求（下单、更新库存、处理支付）在扩展性和一致性上的要求完全不同。网关把它们路由到不同的服务集群，各自采用独立的扩容策略，确保秒杀带来的写入尖峰不会拖慢其他用户的浏览体验。

## 4\. 用户与卖家资料服务

用户资料服务

管理客户数据，包括姓名、收货地址、支付方式、订单历史和心愿单。

**Users 表（PostgreSQL）** Users user\_id UUID，主键 email VARCHAR，唯一 name VARCHAR phone VARCHAR created\_at TIMESTAMP

**Addresses 表（PostgreSQL）** Addresses address\_id UUID，主键 user\_id UUID，外键 -\> Users line\_1 VARCHAR line\_2 VARCHAR city VARCHAR state VARCHAR country VARCHAR pincode VARCHAR is\_default BOOLEAN

用户可以有多个收货地址。默认地址会打上标记，方便快速结账。用户资料缓存在 Redis 里，因为每次下单和购物车结账都会读到它。

**卖家资料服务**

管理卖家数据，包括企业名称、GST 与税务信息、用于打款的银行账户信息，以及卖家评分。

**Sellers 表（PostgreSQL）**

Sellers seller\_id UUID，主键 business\_name VARCHAR email VARCHAR，唯一 phone VARCHAR gstin VARCHAR bank\_account\_id UUID rating FLOAT total\_reviews INTEGER created\_at TIMESTAMP

卖家资料要求强一致性，因为它和金融交易与法律合规绑定在一起。卖家的银行账户或税务信息不能接受最终一致性。卖家评分是反范式化的计数器，由 Kafka 消费者从评价服务异步更新。

## 5\. 商品目录服务与上传流水线

商品目录服务是整个系统中读压力最大的服务。每一次浏览、搜索、商品页访问都会打到它上面。3.5 亿商品、5000 万日活用户、每个会话浏览多个商品，读负载非常巨大。

**卖家上传流程**

卖家上架新商品时：

卖家通过 API 网关认证，向目录服务申请预签名 S3 URL

卖家把商品图片和演示视频直接上传到 S3

S3 向 Kafka 发出上传事件

**转码服务**消费这些事件并处理：图片：生成缩略图（用于搜索结果）、中等尺寸（用于商品网格）和全分辨率（用于商品页放大）三种变体，并做 WebP 压缩 视频：生成多种清晰度变体（360p、720p、1080p），切成 HLS 分片以支持自适应码流，并生成一张预览缩略图

所有变体存放在 S3 的 media/{product\_id}/{variant}/ 路径下

转码服务向 Kafka 发布 product.transcoded 事件

两个消费者各自独立地消费该事件：**元数据 Worker** 把商品元数据写入 Products 库，并填充 Redis 里的商品元数据缓存 **搜索索引 Worker** 把商品索引进 Elasticsearch

**数据模型**

**Products 表（PostgreSQL）**

Products product\_id UUID，主键 seller\_id UUID，外键 -\> Sellers name VARCHAR description TEXT category VARCHAR subcategory VARCHAR brand VARCHAR price DECIMAL(10,2) mrp DECIMAL(10,2) media\_urls JSONB（每种变体对应的 S3/CDN URL 数组） attributes JSONB（颜色、尺寸、重量、尺寸规格等） rating FLOAT review\_count INTEGER is\_active BOOLEAN created\_at TIMESTAMP updated\_at TIMESTAMP

价格直接存在 Products 表上，并且是强一致的。卖家改价会立即触发一次 PostgreSQL 写入、一次 Redis 缓存失效，以及通过 Kafka 发往 Elasticsearch 的重新索引事件。价格陈旧的时间窗口要尽可能压到最短，因为展示错误价格既是法律风险也是信任风险。

attributes 用 JSONB 存储，因为商品属性因品类而异，差别极大。笔记本电脑有内存、处理器、屏幕尺寸；衬衫有颜色、尺码、面料。JSONB 允许每个品类有灵活的 schema，不必为每种商品类型单独建表。

商品页读取路径

用户打开商品页时，需要组装以下数据：

**商品元数据**（名称、描述、价格、属性）-\> 先查 Redis 缓存，未命中再查 PostgreSQL

**媒体 URL**（图片和视频）-\> 由 CDN 直接提供，后端不参与

**库存数量** -\> 库存服务（Redis 提供实时数量）

**卖家信息**（企业名称、评分）-\> 卖家资料服务（Redis 缓存）

**评价**（评分最高的评价）-\> 评价服务（PostgreSQL，缓存在 Redis）

这五次拉取并行进行。所有并行请求都返回后，商品页一次性组装并渲染。这就是 **scatter-gather（分散-聚合）**模式。

**缓存策略**

Amazon 的商品目录遵循极端的幂律分布。头部 1% 的商品带来了大约 90% 的流量。热门商品（畅销品、趋势商品、参加秒杀的商品）会以更长的 TTL 被积极缓存在 Redis 中。冷门商品（流量稀少的长尾商品）直接从 PostgreSQL 读取，不承担缓存开销。

对于秒杀期间最火爆的商品，采用多层缓存：

- **L1** -\> 每台应用服务器上的本地内存缓存（Caffeine LRU）
- **L2** -\> Redis
- **L3** -\> PostgreSQL

**目录服务的瓶颈**

对于带媒体的新商品上架来说，转码服务是瓶颈，靠水平扩展的 Worker 池来解决。读路径上的瓶颈是 3.5 亿商品所需的 Redis 内存，因此有选择地缓存热门商品、而不是缓存全量目录，是必须的做法。

## 6\. 搜索服务

Amazon 的搜索是整个系统里商业价值最关键的服务。用户找不到商品就买不了商品。搜索的延迟和相关性直接决定营收。

**存储引擎 - Elasticsearch**

Elasticsearch 负责在 3.5 亿商品文档上做全文搜索、分面筛选和相关性排序。

**商品文档：**

{ "product\_id": "123", "name": "Sony WH-1000XM5 Wireless Headphones", "description": "Industry leading noise cancellation...", "category": "Electronics", "subcategory": "Headphones", "brand": "Sony", "price": 29990, "rating": 4.5, "review\_count": 12000, "prime\_eligible": true, "in\_stock": true, "seller\_id": "456", "attributes": { "color": "Black", "connectivity": "Bluetooth", "battery\_life": "30 hours" } }

**排序信号**

Amazon 的搜索排序综合了文本相关性之外的多种信号：

- **文本匹配分** -\> 查询词与商品名称、描述、品牌的匹配程度
- **销售速度** -\> 在该查询下卖得好的商品排得更靠前
- **评分与评价数** -\> 评分高、评价多的商品排得更靠前
- **价格竞争力** -\> 同品类内价格有竞争力的商品排得更靠前
- **Prime 资格** -\> 对 Prime 用户，符合 Prime 的商品获得排序加权
- **是否有货** -\> 缺货商品降权

**自动补全**

用户在搜索框输入 "son" 时，自动补全展示 "sony headphones"、"sony tv"、"sony camera"。这由一个 Redis Sorted Set 支撑：

Key: autocomplete:son Member: "sony headphones" Score: 搜索频次（用户搜索该词的次数）

ZREVRANGE autocomplete:son 0 4 会立刻返回排名前 5 的补全建议。这个 Sorted Set 由后台 Worker 定期分析搜索日志来更新。

**分面筛选**

Amazon 搜索支持按价格区间、品牌、评分、Prime 资格以及品类专属属性（电视的屏幕尺寸、鞋类的鞋码）筛选。这些筛选条件作为 Elasticsearch 的查询后置过滤器，作用在已索引字段上。Elasticsearch 里 (category, price, rating, in\_stock) 的复合索引让带筛选的查询即使跨 3.5 亿文档也很快。

**保持 Elasticsearch 同步**

搜索索引 Worker 消费 product.transcoded 和 product.updated 两个 Kafka topic。价格变更会立即同步到 Elasticsearch，因为搜索结果里的陈旧价格会造成信任问题。评分和评价数的更新则批量、周期性地同步。

**搜索的瓶颈**

3.5 亿文档下的 Elasticsearch 性能需要一个大集群，并为索引缓存准备大量内存。热门搜索词（"iphone"、"laptop"、"headphones"）以较短的 TTL 缓存在 Redis 中。自动补全完全由 Redis 提供，不涉及 Elasticsearch。

## 7\. 购物车服务

购物车服务管理用户在结账之前加入购物车的商品。购物车是临时的、频繁更新的，读写都必须快。

**存储 - Redis 为主，PostgreSQL 兜底**

购物车数据主要存在 Redis，因为购物车操作（加商品、删商品、改数量、看购物车）极其频繁且对延迟敏感。购物车一慢，购物体验就毁了。

Key: cart:{user\_id} Value: Hash { product\_id: quantity, product\_id: quantity, ... } TTL: 30 天（购物车跨会话保留）

Redis Hash 操作：

- 加商品：HSET cart:user\_123 product\_456 2
- 删商品：HDEL cart:user\_123 product\_456
- 看购物车：HGETALL cart:user\_123
- 改数量：HSET cart:user\_123 product\_456 3

**全都是 O(1) 操作。结账时读取购物车只需一次 HGETALL 调用。**

购物车数据同时通过 Kafka 消费者异步持久化到 PostgreSQL。如果 Redis 挂了，下次会话时从 PostgreSQL 重建购物车。用户可能会丢失 Redis 故障前最近加入的几件商品，考虑到购物车本身的临时性，这是可以接受的。

**结账时的价格与库存校验**

用户进入结账流程时，购物车服务会校验车里的每一件商品：

从目录服务取当前价格 -\> 与购物车中记录的价格比对。若价格变了，先通知用户再继续。

向库存服务检查每件商品是否有货。若有任何一件缺货，先通知用户再继续。

这个校验发生在结账时刻，而不是加入购物车时刻，因为从用户加入商品到真正购买之间，价格和库存一直在变。

**购物车服务的瓶颈**

在 Amazon 这个量级上，Redis 处理购物车操作毫不费力。瓶颈在结账校验环节，因为它要为购物车里的每件商品并行调用目录服务和库存服务。装了 20 件以上商品的大购物车需要 20 次并行库存查询。缓解办法是用 scatter-gather 模式把所有检查同时并行发出，并对购物车容量设一个合理上限。

## 8\. 库存服务

库存服务是全系统对一致性要求最高的服务。超卖是严重的运营事故；少卖（因为数量陈旧而拦下有效购买）则直接损失营收。

**存储 - Redis + PostgreSQL**

库存数量存在 Redis 中，以支持快速的实时读写。PostgreSQL 是事实来源，异步更新。

Key: inventory:{product\_id} Value: available\_count（整数）

对于由多个卖家销售或分布在多个仓库的商品，库存按卖家、按仓库分别跟踪：

Key: inventory:{product\_id}:{seller\_id}:{warehouse\_id} Value: available\_count

**预留流程**

用户下单时，库存先被预留（而不是直接扣减），直到支付确认：

**预留：** 在分布式锁保护下执行 DECRBY inventory:product\_id 1

在 **Reservations 表**中记录这笔预留，并带上 TTL（15 分钟）

支付成功 -\> 预留转为已确认的销售，PostgreSQL 中永久扣减库存

支付失败 -\> 释放预留，执行 INCRBY inventory:product\_id 1

用户放弃结账 -\> 预留的 TTL 到期，库存自动释放

这种「先预留、后确认」的两阶段模式既避免了超卖，又不会在支付确认之前就永久扣掉库存。

**库存的分布式锁**

SET lock:inventory:product\_id <lock\_id\> NX EX 5

NX 表示仅当键不存在时才设置（原子获取）。EX 5 表示锁 5 秒后过期（防止持有者崩溃导致死锁）。同一时刻只有一个服务实例能持有该锁。预留记录写完后立刻释放锁。

**秒杀场景下基于队列的削峰**

秒杀期间，单个商品的库存要承受每秒 10 万次请求。与其让所有请求同时争抢分布式锁，不如把它们推进一个 Kafka topic：

flash\_sale\_orders:{product\_id}

由单个**库存 Worker** 从该 topic 顺序消费订单，逐个检查并扣减库存。用户立即收到「您的订单正在处理中」的响应，成功或失败再通过通知服务异步告知。这样在秒杀期间就完全消除了锁竞争。

**Reservations 表（PostgreSQL）**

**Reservations** reservation\_id UUID，主键 order\_id UUID，外键 -\> Orders product\_id UUID，外键 -\> Products seller\_id UUID，外键 -\> Sellers quantity INTEGER status ENUM('reserved', 'confirmed', 'released') expires\_at TIMESTAMP created\_at TIMESTAMP

**库存服务的瓶颈**

在非秒杀场景下，高需求商品的瓶颈是分布式锁。锁的获取与释放在 Redis 中都是微秒级，所以正常流量下实际吞吐量非常高。秒杀场景则用 Kafka 队列模式处理，彻底消除锁竞争。

## 9\. 订单服务与 Saga 模式

订单服务用 **Saga 模式**编排整个购买流程，处理分布式事务。下一个订单要牵动多个相互独立的服务，每一个都可能失败。Saga 模式确保任一步骤失败时都会触发补偿事务撤销已完成的步骤，让系统回到一致状态。

**完整 Saga 流程**

**第 1 步（订单创建）-** 用户点击「下单」。订单服务创建一条 PENDING 状态的订单记录，并向 Kafka 发布 order.created。

**第 2 步（库存预留）-** 库存服务消费 order.created，尝试预留库存。

- 成功 -\> 发布 inventory.reserved
- 失败 -\> 发布 inventory.failed -\> 订单服务把订单标记为 FAILED，通知用户

**第 3 步（支付处理）-** 支付服务消费 inventory.reserved，向用户的支付方式扣款。

- 成功 -\> 发布 payment.completed
- 失败 -\> 发布 payment.failed -\> **补偿事务：** 库存服务释放预留 -\> 订单服务把订单标记为 FAILED，通知用户，不产生扣款

**第 4 步（订单确认）-** 订单服务消费 payment.completed，把订单标记为 CONFIRMED，并发布 order.confirmed。

**第 5 步（通知仓库）-** 仓储服务消费 order.confirmed，创建拣货打包任务，发布 warehouse.notified。

**第 6 步（通知用户）-** 通知服务消费 order.confirmed 和 warehouse.notified，发送订单确认邮件和推送通知。

**第 7 步（物流更新）-** 仓库处理订单的过程中会发布 order.shipped 和 order.out\_for\_delivery 事件。通知服务在每一步发送更新。

**Orders 表（PostgreSQL）**

Orders order\_id UUID，主键 user\_id UUID，外键 -\> Users status ENUM('pending', 'confirmed', 'shipped', 'delivered', 'failed', 'cancelled') total\_amount DECIMAL(10,2) shipping\_address JSONB placed\_at TIMESTAMP updated\_at TIMESTAMP

**OrderItems 表（PostgreSQL）**

OrderItems order\_item\_id UUID，主键 order\_id UUID，外键 -\> Orders product\_id UUID，外键 -\> Products seller\_id UUID，外键 -\> Sellers quantity INTEGER unit\_price DECIMAL(10,2)

单价在下单时就固化保存，而不是从 Products 表引用。这样即使商品价格后来变了，订单历史也永远反映用户实际支付的金额。

**订单服务的瓶颈**

订单服务本身是无状态的，可以水平扩展。瓶颈在于高订单量下的 Saga 编排。Kafka 提供的持久化和解耦能力让 Saga 模式具备韧性：每一步都可以独立重试，无需重启整个流程。所有 Kafka 消费者上的幂等键则防止消息重试导致的重复处理。

## 10\. 支付服务

支付服务是全系统在财务上最关键的服务。给用户重复扣款是严重的信任事故，漏收一笔钱则是营收损失。两者都必须杜绝。

**幂等键**

每个支付请求都带一个由订单服务生成的唯一幂等键（通常就是 order\_id）。如果支付服务收到相同的幂等键两次（因为网络重试），它会返回第一次尝试的结果，而不会再次扣款。幂等键存在 Redis 中，TTL 为 24 小时。

**支付网关集成**

支付服务对接外部支付网关（Razorpay、Stripe、银行网络）。流程如下：

支付服务从 Kafka 收到 inventory.reserved 事件

带上金额、用户支付方式和幂等键调用外部支付网关

网关返回成功或失败

支付服务把结果写入 Payments 表

向 Kafka 发布 payment.completed 或 payment.failed

Payments 表（PostgreSQL）

Payments payment\_id UUID，主键 order\_id UUID，外键 -\> Orders user\_id UUID，外键 -\> Users amount DECIMAL(10,2) currency VARCHAR gateway VARCHAR（razorpay、stripe 等） gateway\_txn\_id VARCHAR status ENUM('pending', 'processing', 'completed', 'failed', 'refunded') idempotency\_key VARCHAR，唯一 created\_at TIMESTAMP updated\_at TIMESTAMP

支付服务的瓶颈

外部支付网关既是瓶颈也是延迟来源。网关调用通常耗时 500ms 到 2 秒。应对办法是通过 Kafka 让支付处理完全异步化，用户在结账时永远不必同步等待网关返回。

## 11\. 评价与评分服务

评价是最终一致的，而且只写一次（用户只有在确认购买之后才能评价某个商品）。

**Reviews 表（PostgreSQL）**

Reviews review\_id UUID，主键 product\_id UUID，外键 -\> Products user\_id UUID，外键 -\> Users order\_id UUID，外键 -\> Orders rating INTEGER (1-5) title VARCHAR content TEXT created\_at TIMESTAMP UNIQUE (product\_id, user\_id)

(product\_id, user\_id) 上的 UNIQUE 约束保证一个用户对一个商品只能评价一次。order\_id 外键则保证只有经过验证的购买者才能留下评价。

Products 表上的商品评分和评价数是反范式化的计数器，每当有新评价提交时，由 Kafka 消费者异步更新。热门商品的评价缓存在 Redis 中。搜索索引 Worker 会周期性地把更新后的评分同步到 Elasticsearch。

## 12\. 通知服务

架构与 Instagram、Twitter 相同。所有服务通过 Kafka 扇出，iOS 用 APNs，Android 用 FCM，邮件走 SES（Simple Email Service）。

通知服务订阅：

- order.confirmed -\> 订单确认邮件 + 推送通知
- warehouse.notified -\> 「您的订单正在备货」通知
- order.shipped -\> 带运单号的发货确认
- order.out\_for\_delivery -\> 「今日送达」通知
- order.delivered -\> 送达确认 + 评价提醒
- payment.failed -\> 支付失败通知，附重试指引
- inventory.failed -\> 缺货通知

社交平台上通知合并很关键（把 500 个点赞打包成一条通知），Amazon 不同：它的通知是事务性的，必须逐条送达。订单生命周期中的每一个事件都值得一条独立的通知。

## 13\. 完整数据流总结

商品上架流程（卖家）：卖家 -\> API 网关（认证）-\> 目录服务 -\> 预签名 S3 URL 卖家 -\> S3（直传图片和视频）S3 -\> Kafka -\> 转码服务 -\> S3（所有变体）转码服务 -\> Kafka（product.transcoded）-\> 元数据 Worker -\> Products 库 + Redis 缓存 -\> 搜索 Worker -\> Elasticsearch 商品页读取流程：客户端 -\> API 网关 -\> 目录服务 -\> Redis（商品元数据缓存）\[并行\] -\> 库存服务 -\> Redis（库存数量）\[并行\] -\> 卖家服务 -\> Redis（卖家信息缓存）\[并行\] -\> 评价服务 -\> Redis（热门评价缓存）\[并行\] -\> CDN（媒体 URL 直接提供）\[并行\] -\> scatter-gather 组装出完整商品页 搜索流程：客户端 -\> 搜索服务 -\> Redis（前缀对应的自动补全 sorted set）-\> Redis（热门查询缓存）-\> Elasticsearch（缓存未命中时执行带分面的完整搜索）-\> Redis MGET（为结果补齐商品元数据） 购物车流程：客户端 -\> 购物车服务 -\> Redis HSET/HDEL/HGETALL cart:{user\_id} -\> Kafka -\> PostgreSQL（异步持久化购物车）结账 -\> 购物车服务 -\> 库存服务（对所有商品并行检查库存）-\> 目录服务（对所有商品并行校验价格） 订单与 Saga 流程：客户端下单 -\> 订单服务 -\> 订单 PENDING -\> Kafka（order.created）-\> 库存服务：预留库存 -\> Kafka（inventory.reserved / inventory.failed）-\> 支付服务：向用户扣款 -\> Kafka（payment.completed / payment.failed）-\> 失败时：库存服务释放预留（补偿事务）-\> 订单服务：订单 CONFIRMED -\> Kafka（order.confirmed）-\> 仓储服务：拣货打包任务 -\> Kafka（warehouse.notified）-\> 通知服务：订单确认 -\> APNs / FCM / SES 秒杀流程：10 万用户同时点击购买 -\> 订单服务 -\> Kafka topic flash\_sale\_orders:{product\_id} -\> 库存 Worker 顺序逐个处理 -\> 无锁竞争，成功或失败异步通知用户 支付流程：支付服务 <\- Kafka（inventory.reserved）-\> 在 Redis 中检查幂等键 -\> 调用支付网关（Razorpay / Stripe）-\> 写入 Payments 表（PostgreSQL）-\> Kafka（payment.completed / payment.failed）

## 14\. 韧性与容错

**Saga 补偿事务**确保任何局部的订单状态都不会变成永久状态。支付失败一定会释放库存预留；库存失败一定会阻止支付发起。只要 Kafka 是持久可靠的，系统就永远不会卡在不一致状态。

**Kafka 的持久化**贯穿整条订单流水线，意味着即使下游服务暂时不可用，也不会丢失任何订单事件。Saga 的每一步都能从自己的 Kafka offset 独立重试，不必重启整个流程。

支付的**幂等键**在任何重试或网络故障场景下都能防止重复扣款。同一个 order\_id 永远不可能产生两笔成功的扣款。

**两阶段库存预留**防止超卖。下单时先预留库存，支付成功后才确认。被放弃的预留通过 TTL 自动过期。

库存上**带 TTL 的分布式锁**防止并发购买时的竞态条件。5 秒的 TTL 保证即使锁持有者崩溃，锁也一定会被释放，从而避免死锁。

秒杀场景下**通过 Kafka 做基于队列的削峰**，彻底消除锁竞争。无论请求尖峰多高，库存 Worker 都以可持续的速率从队列中顺序处理购买请求。

热门商品的**多层缓存（L1/L2/L3）**在读侧消除了秒杀期间的热点 key 问题。最热门的商品，其元数据直接从应用服务器的本地内存提供。

**购物车从 Redis 持久化到 PostgreSQL** 保证购物车数据能在 Redis 故障中存活。用户可能丢掉最近加入的几件商品，但绝不会丢掉整个购物车。

所有商品媒体的 **CDN 缓存**意味着图片和视频由全球边缘节点提供，缓存命中时源站完全不参与。

所有服务的 **PostgreSQL 只读副本**确保整个系统的读路径上没有单点故障。

## 15\. 技术选型总结

**API 网关** \-\> Kong / AWS API Gateway（认证、限流、读写流量隔离）

**对象存储** -\> AWS S3（商品图片与视频，高持久性，天然对接 CDN）

**CDN** -\> CloudFront / Akamai（商品媒体的边缘缓存，缓存命中率高）

**转码** \-\> 经 Kafka 驱动的水平扩展 Worker 池（图片变体、HLS 视频切片）

**解耦** -\> Apache Kafka（订单 Saga 编排、库存事件、搜索索引、通知）

**搜索引擎** -\> Elasticsearch（3.5 亿商品上的全文搜索、分面筛选与排序）

**自动补全** -\> Redis Sorted Set（基于前缀的查询建议，亚毫秒级读取）

**商品目录数据库** \-\> 分片 + 副本的 PostgreSQL（结构化商品元数据，价格强一致）

**订单 / 支付数据库** \-\> PostgreSQL（ACID 保证、财务记账、订单状态机）

**库存存储** -\> Redis + PostgreSQL（Redis 存实时数量，PostgreSQL 为事实来源）

**分布式锁** -\> 带 TTL 的 Redis SETNX（库存预留，防止超卖）

**秒杀队列** \-\> Apache Kafka（基于队列的削峰，消除锁竞争）

**购物车存储** -\> Redis Hash + PostgreSQL 兜底（亚毫秒级购物车操作，异步持久化）

**商品元数据缓存** \-\> Redis + L1 本地内存（多层缓解秒杀时的热点 key）

**幂等键存储** -\> 带 TTL 的 Redis（防止支付重试造成重复扣款）

**通知投递** -\> APNs + FCM + AWS SES（推送通知与事务性邮件）

**评价数据库** -\> PostgreSQL（强制验证购买，每用户每商品的 UNIQUE 约束）

以上就是全部内容，各位，干杯！
