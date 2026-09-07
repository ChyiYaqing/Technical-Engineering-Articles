---
title: "Design Airbnb"
url: "https://x.com/Harry_The_Nerd/status/2079561027788611975"
category: "HLD"
date: "2026-07-21"
description: "System design for a hotel-booking platform like Airbnb."
lang: "zh-CN"
---

# 设计 Airbnb

> 类似 Airbnb 的住宿预订平台的系统设计。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2079561027788611975](https://x.com/Harry_The_Nerd/status/2079561027788611975) · 2026-07-21

![封面图](https://pbs.twimg.com/media/HM879HTbkAA16YF.jpg)

## 1\. 需求

**功能性需求**

- 房源管理（房东创建、更新、删除房源，含照片和描述）
- 按位置、日期区间、价格、评分和设施搜索房源
- 房源的地图视图
- 每个房源的可订状态与日历管理
- 预订与订单管理
- 支付处理（房客付款，房东收款）
- 评价（房客评价房源，房东评价房客）
- 通知（预订确认、入住提醒、评价提示）
- 房东和房客双方的资料管理

**不在范围内**

- Airbnb 体验（Experiences）
- 动态定价的机器学习引擎
- 身份验证与背景调查
- 超赞房东（Superhost）计划
- 纠纷仲裁系统

**非功能性需求**

- 搜索和房源浏览要具备高可用性
- 预订和可订状态要保证强一致性。同一房源在重叠日期上绝不能被重复预订
- 评价、推荐和通知可以接受最终一致性
- 搜索延迟低于 200ms，其中包含地理过滤和可订状态检查
- 支付必须恰好执行一次
- 日期区间的预订是原子的。一次预订要么完全成功，要么完全失败，不存在中间状态

## 2\. 容量估算

- **日活用户（DAU）：** 1000 万（Airbnb 的 DAU 低于社交平台，但用户意图强）
- **全球活跃房源总数：** 700 万个
- **每日搜索量：** 5000 万次（用户下单前会搜索多次）
- **每日预订量：** 约 50 万单（搜索到预订的转化率 1%）
- **每个房源平均照片数：** 20 张，每张 1 MB = 每个房源 20 MB
- **照片总存储：** 700 万房源 x 20 MB = 约 140 TB
- **可订状态记录：** 700 万房源 x 365 天 = 可订状态表约 25 亿行
- **峰值并发：** 抢订场景（旺季的热门房源）可能出现数百次同时发起的预订尝试

这个系统真正的难点不是存储或吞吐量，而是**并发条件下的正确性**。一个房源在圣诞周被重复预订，是灾难级的故障。预订与可订状态这一层的每一个架构决策，都是围绕防止这件事展开的。

## 3\. API 网关

所有客户端请求都从 API 网关进入，由网关负责认证、限流和路由。Airbnb 有两类完全不同的用户打到同一个网关上：房客（搜索和预订）和房东（管理房源与日历）。网关根据操作类型把请求路由到不同的服务集群。

媒体上传（房源照片）在认证之后绕开网关，使用预签名的 S3 URL 由客户端直传 S3。房源照片上传和 Instagram 的图片上传是同一个问题：体积很大的二进制负载数据，绝不能走 API 层。

## 4\. 房东与房客资料服务

Airbnb 是一个双边市场。房东和房客都是一等公民用户，各自的工作流和数据需求完全不同。

**数据模型**

**Users 表（PostgreSQL）**

Users user\_id UUID，主键 email VARCHAR，唯一 name VARCHAR profile\_pic\_url TEXT phone VARCHAR is\_host BOOLEAN host\_rating FLOAT guest\_rating FLOAT host\_review\_count INTEGER guest\_review\_count INTEGER joined\_at TIMESTAMP

一张 Users 表同时承载房东和房客。is\_host 标志表示该用户是否有在架房源。一个用户可以同时是房东和房客。很多 Airbnb 用户一边把自己的房子租出去，一边也在外面旅行订别人的房子。

host\_rating 和 guest\_rating 是反范式化的计数器，新评价提交时由 Kafka 消费者异步更新。房东关心房客评分（这个人靠不靠谱？），房客关心房东评分（这个房东回复及不及时？）。

**HostPayouts 表（PostgreSQL）**

HostPayouts payout\_id UUID，主键 host\_id UUID，外键 -\> Users bank\_account VARCHAR（加密） routing\_number VARCHAR（加密） payout\_currency VARCHAR created\_at TIMESTAMP

房东的银行账户信息要求强一致性，并且落盘加密。这些是与真实资金划转绑定的金融记录，不能接受最终一致性。

**缓存策略**

用户资料缓存在 Redis 里，TTL 设为中等长度。每次访问房源页都会读取房东资料，所以热门房东（拥有大量房源的超赞房东）从缓存中获益明显。房客资料是在发起预订时供房东查看用的，读取频率远低于房东资料。

## 5\. 房源服务与上传流水线

房源服务管理一个房源的方方面面：描述、设施、住宿守则、位置、定价和照片。

照片上传流程

和前面所有设计一样的预签名 URL 模式：

房东通过 API 网关认证，向房源服务申请预签名 S3 URL

房东把照片直接上传到 S3

S3 向 Kafka 发出上传事件

照片处理 Worker 生成多种分辨率变体（用于搜索结果的缩略图、用于房源网格的中等尺寸、用于房源页图库的全分辨率）

所有变体存放在 S3 的 media/{property\_id}/{variant}/ 路径下

CDN 位于 S3 前面，面向全球分发房源照片

数据模型

**Properties 表（PostgreSQL）**

Properties property\_id UUID，主键 host\_id UUID，外键 -\> Users title VARCHAR description TEXT property\_type ENUM('entire\_place', 'private\_room', 'shared\_room') accommodation ENUM('apartment', 'house', 'villa', 'unique') max\_guests INTEGER bedrooms INTEGER bathrooms INTEGER base\_price DECIMAL(10,2) currency VARCHAR latitude FLOAT longitude FLOAT geohash VARCHAR（精度 6，约 1km 网格） address TEXT city VARCHAR country VARCHAR rating FLOAT review\_count INTEGER is\_active BOOLEAN created\_at TIMESTAMP

geohash 是由 latitude 和 longitude 计算出来的列。精度为 6 的 geohash 对应大约 1km x 1km 的网格，正好适合房源搜索的半径查询。geohash 前缀相同的房源在地理上彼此靠近（和 Tinder 的做法一样）。

**PropertyAmenities 表（PostgreSQL）**

PropertyAmenities property\_id UUID，外键 -\> Properties amenity ENUM('wifi', 'pool', 'parking', 'kitchen', 'ac', 'pet\_friendly', ...) PRIMARY KEY (property\_id, amenity)

设施是按行存储的，而不是存成一个 JSON 数组。这样就能用标准 SQL 的 WHERE 子句在有索引的列上做高效过滤，比如「找出所有既有泳池又有停车位的房源」。

**PropertyPhotos 表（PostgreSQL）**

PropertyPhotos photo\_id UUID，主键 property\_id UUID，外键 -\> Properties s3\_url TEXT order\_index INTEGER uploaded\_at TIMESTAMP

**搜索索引**

新房源上架或更新时，房源服务向 Kafka 发布 property.updated 事件。搜索索引 Worker 消费该事件并更新 Elasticsearch 索引。房源元数据（包括 geohash、价格、评分、设施和可订状态摘要）全部索引进 Elasticsearch，以支撑快速搜索查询。

房源服务的瓶颈

房源服务是读多写少的，因为房源页被浏览的次数远多于被更新的次数。Redis 缓存热门房源的元数据。写入侧主要的隐患是大型房源管理公司同时批量更新数百个房源，这一点通过 API 网关层的限流来处理。

## 6\. 可订状态服务

可订状态服务是整个系统中对一致性要求最高的服务。它负责存储、检查和锁定房源在各个日期区间上的可订状态，并且绝不允许出现重复预订。

**日历表设计**

可订状态按「每房源每日期」一条记录来存。这叫做**日历表模式**：

PropertyAvailability

property\_id UUID，外键 -\> Properties date DATE status ENUM('available', 'booked', 'blocked') price DECIMAL(10,2) booking\_id UUID（available 或 blocked 时为 null） PRIMARY KEY (property\_id, date)

每一行代表一个房源在某一个日历日期上的状态。从 12 月 20 日住到 12 月 26 日的 7 晚行程会产生 7 行（每晚一行），它们都指向同一个 booking\_id。

blocked 状态用于房东在日历上手动封锁某些日期（自用、维修等）。

按日期存 price 使得动态定价成为可能：房源可以在周末或本地活动期间涨价，而不用改动基础价格。

可订状态检查查询

检查某房源在 12 月 20 日至 25 日（5 晚）是否可订：

```sql
SELECT COUNT(*) FROM PropertyAvailability
WHERE property_id = 'property_123'
AND date BETWEEN '2024-12-20' AND '2024-12-24'
AND status = 'available'
```

如果 COUNT 等于 5（也就是请求的晚数），说明该房源在请求的日期上完全可订。

(property\_id, date) 上的复合主键让这条查询走索引扫描而不是全表扫描，即使表里有 25 亿行也极快。

**防止重复预订 —— 悲观锁**

并发问题是这个设计中最难的部分。100 个用户同时查询同一个房源同一批日期的可订状态，全都看到「可订」，然后全都发起预订。如果没有并发控制机制，这 100 个预订可能全部成功，房源被严重超卖。

解法是用 PostgreSQL 的 SELECT FOR UPDATE 实现**悲观锁**：

```pgsql
BEGIN;

SELECT COUNT(*) FROM PropertyAvailability
WHERE property_id = 'property_123'
AND date BETWEEN '2024-12-20' AND '2024-12-24'
AND status = 'available'
FOR UPDATE;

-- If COUNT = 5 (fully available), proceed:
UPDATE PropertyAvailability
SET status = 'booked', booking_id = 'booking_456'
WHERE property_id = 'property_123'
AND date BETWEEN '2024-12-20' AND '2024-12-24';

COMMIT;
```

SELECT FOR UPDATE 在整个事务期间对所有匹配的可订状态行加上行级锁。任何其他事务想读取或更新这些行都会被阻塞，直到第一个事务提交或回滚。对任意房源的任意日期区间，只有一个预订能够成功。

这和 Amazon 用 Redis 分布式锁的做法不同。因为可订状态存在 PostgreSQL 里，而检查与更新必须是原子的，所以这里合适的工具是 PostgreSQL 原生的事务锁，而不是另外引入一个 Redis 锁。

带过期机制的两阶段预订

为了避免在用户还停留在支付页面时一直持有锁，Airbnb 采用两阶段预订模式：

**第一阶段 —— 临时占位（保留 10 分钟）**

对请求的日期加 PostgreSQL 锁

检查可订状态

如果可订，把状态置为 reserved，并写入一个 10 分钟后的 reserved\_until 时间戳

释放锁（事务提交）

用户进入支付页面

**第二阶段 —— 支付成功后确认**

支付服务向房客扣款

支付成功后，把状态从 reserved 更新为 booked

支付失败或超时时，由后台任务把过期的 reserved 行重置回 available

这样一来，用户慢慢输入信用卡号的那几分钟里锁不会被一直持有，同时在支付窗口期内依然能防止重复预订。

**更新后的 PropertyAvailability 表：**

PropertyAvailability property\_id UUID date DATE status ENUM('available', 'booked', 'blocked', 'reserved') price DECIMAL(10,2) booking\_id UUID reserved\_until TIMESTAMP（除非 status = 'reserved'，否则为 null） PRIMARY KEY (property\_id, date)

一个后台的**预订过期 Worker** 每分钟运行一次，重置过期的占位：

```sql
UPDATE PropertyAvailability
SET status = 'available', booking_id = NULL, reserved_until = NULL
WHERE status = 'reserved'
AND reserved_until < NOW();
```

可订状态的 Redis 缓存

对于搜索这一层，并不需要为每条搜索结果都去 PostgreSQL 查精确到日的可订状态。Redis 位图可以提供一个快速的近似可订信号：

Key：availability:{property\_id}:{year\_month} Value：位图，每一位代表一天（第 0 位 = 1 号，第 30 位 = 31 号） 1 = 可订，0 = 已订或已封锁

每个房源每个月一个 31 位的位图，只占 4 字节。检查一个日期区间的可订状态，就是对相关月份的位图做按位与运算，耗时不到 1 毫秒。

这个 Redis 位图是搜索过滤的快路径。带悲观锁的权威检查只在用户真正发起预订时才执行，而不是每次搜索都做。

**可订状态服务的瓶颈**

在预订高峰期，热门房源上的 PostgreSQL 行锁是主要瓶颈。跨年周果阿的某个房源，可能会遇到数百次同时发起的预订尝试。两阶段占位模式缓解了这一点：锁只在毫秒级的时间内被持有（完成可订检查和状态更新即可），而不是覆盖整个支付过程。Redis 位图快路径则保证搜索查询永远不会直接打到 PostgreSQL。

## 7\. 搜索服务

在我们设计过的九个系统里，Airbnb 的搜索查询是最复杂的。一次搜索请求同时结合了地理邻近度（像 Tinder）、日期区间可订过滤、价格与评分过滤（像 Amazon），以及相关性排序。

存储引擎 —— 带地理空间支持的 Elasticsearch

主搜索层由 Elasticsearch 承担。Tinder 用 Redis GEO 做邻近度查询，而 Airbnb 的搜索需要在一条查询里同时完成邻近度、丰富的属性过滤和排序。Elasticsearch 原生的 geo\_distance 过滤器能处理这些，不需要再单独搭一个地理存储。

**Elasticsearch 中的房源文档：**

```json
{
  "property_id": "123",
  "title": "Cozy 1BHK in South Goa",
  "property_type": "entire_place",
  "accommodation": "apartment",
  "max_guests": 2,
  "bedrooms": 1,
  "bathrooms": 1,
  "base_price": 3500,
  "currency": "INR",
  "location": {
    "lat": 15.2993,
    "lon": 74.1240
  },
  "city": "Goa",
  "country": "India",
  "rating": 4.8,
  "review_count": 124,
  "amenities": ["wifi", "pool", "parking", "kitchen"],
  "host_id": "456",
  "is_active": true,
  "available_months": ["2024-12", "2025-01"]
}
```

location 存成 Elasticsearch 的 geo\_point 类型，从而支持原生的半径过滤。available\_months 是一个粗粒度的可订信号 —— 房源在这些月份里至少还有一些空档。它用作精确日期区间检查之前的预过滤条件。

**搜索查询流程**

当用户搜索「果阿，12 月 20-25 日，2 位房客，每晚不超过 5000 卢比」时：

**第 1 步 —— Elasticsearch 查询**

```json
{
  "query": {
    "bool": {
      "filter": [
        { "geo_distance": { "distance": "50km", "location": { "lat": 15.2993, "lon": 74.1240 } } },
        { "term": { "available_months": "2024-12" } },
        { "range": { "base_price": { "lte": 5000 } } },
        { "term": { "is_active": true } },
        { "range": { "max_guests": { "gte": 2 } } }
      ]
    }
  },
  "sort": [
    { "rating": { "order": "desc" } },
    { "_geo_distance": { "location": { "lat": 15.2993, "lon": 74.1240 }, "order": "asc" } }
  ]
}
```

这一步返回一批符合粗粒度过滤条件的候选房源 ID，按评分和距离排序。

**第 2 步 —— Redis 位图可订过滤** Elasticsearch 返回的候选列表可能包含 500 个房源。在渲染之前，搜索服务会逐个检查这些房源的 Redis 可订位图，把在精确请求日期上不可订的房源剔除掉。这样就把候选集收敛成真正可订的房源，而不必访问 PostgreSQL。

**第 3 步 —— 房源元数据填充** 过滤后的候选列表通过 Redis MGET 批量取回完整的房源元数据（标题、照片、价格、评分）用于展示。缓存未命中时回落到 PostgreSQL。

**第 4 步 —— 地图数据生成** 对于地图视图，搜索服务返回所有候选房源的经纬度坐标，客户端据此在地图上渲染标记点。地图视图由同一个 Elasticsearch 地理查询驱动，只是返回坐标而不是排序后的结果列表。

**搜索结果排序**

Airbnb 的搜索排序结合了多种信号：

- **评分和评价数**（评分高、评价多的房源排得更靠前）
- **与搜索位置的距离**（评分相同时，更近的房源排得更靠前）
- **价格竞争力**（同一档次内，性价比更高的房源排得更靠前）
- **回复率**（回复及时的房东排得更靠前）
- **接单率**（经常拒单的房东排得更靠后）
- **超赞房东身份**（超赞房东获得排序加权）

**保持 Elasticsearch 同步**

搜索索引 Worker 消费 property.updated 和 property.created 这两个 Kafka 主题来同步房源变更。可订状态摘要（available\_months）由一个后台 Worker 每晚扫描 PropertyAvailability 表并更新 Elasticsearch 文档。评分和评价数则从评价服务定期同步过来。

**搜索的瓶颈**

每次搜索都要对 500 个候选房源做 Redis 位图可订检查，这是延迟上的主要隐患。用 MGET 把所有位图读取合并成一次往返可以缓解。只要索引配置得当（位置字段用 geo\_point 类型），跨 700 万房源文档的 Elasticsearch 地理查询是很快的。

## 8\. 预订服务

预订服务编排完整的预订流程，使用 Saga 模式在可订状态服务、支付服务和通知服务之间协调。

预订状态机

INITIATED -\> RESERVED（可订状态已占位，等待支付） -\> PAYMENT\_PROCESSING -\> CONFIRMED（支付成功，日期已锁定） -\> CANCELLED\_BY\_GUEST -\> CANCELLED\_BY\_HOST -\> COMPLETED（行程结束，触发评价提示） -\> EXPIRED（支付前占位超时）

完整的预订 Saga 流程

**第 1 步 —— 发起预订** 房客选好日期并点击「预订」。预订服务创建一条 INITIATED 状态的预订记录，并向 Kafka 发布 booking.initiated。

**第 2 步 —— 可订状态占位** 可订状态服务对请求日期加 PostgreSQL 行锁，检查可订状态，若可订则把状态置为 reserved 并设置 10 分钟 TTL。随后发布 availability.reserved 到 Kafka。

若不可订：发布 availability.failed -\> 预订服务把预订标记为 EXPIRED，并通知房客。

**第 3 步 —— 支付处理** 支付服务向房客的支付方式扣款。成功则发布 payment.completed；失败则发布 payment.failed：

- **补偿事务：** 可订状态服务把占位的日期重置回 available
- 预订服务把预订标记为 EXPIRED，通知房客，不产生扣款

**第 4 步 —— 预订确认** 预订服务把预订更新为 CONFIRMED，可订状态服务把日期从 reserved 更新为 booked。随后发布 booking.confirmed 到 Kafka。

**第 5 步 —— 通知** 通知服务向房客发送预订确认（邮件 + 推送），向房东发送新订单提醒（邮件 + 推送）。

**第 6 步 —— 房东收款** 支付服务在入住 24 小时后发起房东打款（这是 Airbnb 的标准打款时点）。款项要等到入住之后才放出，以防房东临时取消。

Bookings 表（PostgreSQL）

Bookings booking\_id UUID，主键 property\_id UUID，外键 -\> Properties guest\_id UUID，外键 -\> Users host\_id UUID，外键 -\> Users check\_in DATE check\_out DATE total\_nights INTEGER total\_price DECIMAL(10,2) service\_fee DECIMAL(10,2) host\_payout DECIMAL(10,2) status ENUM('initiated', 'reserved', 'confirmed', 'cancelled\_guest', 'cancelled\_host', 'completed', 'expired') idempotency\_key VARCHAR，唯一 created\_at TIMESTAMP updated\_at TIMESTAMP

idempotency\_key 防止网络重试造成重复预订。如果同一个房客对同一个房源、同一批日期发起了两次预订（双击或网络重试导致），第二次请求会直接返回第一次的结果，而不会创建重复的预订。

host\_payout 和 total\_price 分开存储，因为 Airbnb 向房客和房东双向收取服务费。房客支付 total\_price，Airbnb 留下服务费，房东拿到 host\_payout。

**预订服务的瓶颈**

热门房源上的 PostgreSQL 可订状态锁是主要瓶颈。10 分钟的占位 TTL 意味着锁只被持有毫秒级的时间，而不是几分钟。Saga 模式确保任何一步失败都会触发对应的补偿事务，不会让系统停留在不一致的状态里。

## 9\. 支付服务

架构模式和 Amazon 的支付服务一致，只是针对房东打款做了 Airbnb 特有的扩展。

房客支付流程

支付服务从 Kafka 接收 availability.reserved 事件

在 Redis 中检查幂等键，防止重复扣款

调用外部支付网关（按地区选择 Razorpay、Stripe 或 PayPal）

成功：发布 payment.completed，并写入 Payments 表

失败：发布 payment.failed，由 Saga 补偿事务释放占位

**房东打款流程**

房东收款不是即时的。Airbnb 会先扣住房客的付款，在房客入住 24 小时后才放给房东。这是为了在入住时房源出问题的情况下保护房客。

**打款时序：**

房客入住 -\> 启动 24 小时计时器

24 小时内无问题上报 -\> 打款服务向房东登记的账户发起银行转账

房客在 24 小时内上报问题 -\> 款项扣住，等待纠纷处理

**Payments 表（PostgreSQL）**

Payments payment\_id UUID，主键 booking\_id UUID，外键 -\> Bookings guest\_id UUID，外键 -\> Users host\_id UUID，外键 -\> Users gross\_amount DECIMAL(10,2) service\_fee DECIMAL(10,2) host\_payout DECIMAL(10,2) currency VARCHAR gateway VARCHAR gateway\_txn\_id VARCHAR status ENUM('pending', 'completed', 'failed', 'refunded', 'payout\_pending', 'payout\_completed') idempotency\_key VARCHAR，唯一 payout\_after TIMESTAMP（入住时间 + 24 小时） created\_at TIMESTAMP

支付服务的瓶颈

外部支付网关的延迟（500ms 到 2 秒）是主要隐患。做法是把支付处理通过 Kafka 完全异步化。房客立刻看到「处理中」状态，而不是同步等待网关返回。

## 10\. 评价服务

Airbnb 有一套独特的双向评价系统。行程结束后，房客和房东会同时收到互评提示。在双方都提交之前，或者在 14 天期限到达之前，任何一方的评价都不会公开。这样可以防止报复性评价 —— 一方看到对方的评价后再给出负面反馈。

**数据模型**

**Reviews 表（PostgreSQL）**

Reviews review\_id UUID，主键 booking\_id UUID，外键 -\> Bookings reviewer\_id UUID，外键 -\> Users reviewee\_id UUID，外键 -\> Users property\_id UUID（房东评房客时为 null） reviewer\_type ENUM('guest', 'host') rating INTEGER（1-5） content TEXT is\_published BOOLEAN created\_at TIMESTAMP UNIQUE (booking\_id, reviewer\_type)

is\_published 初始为 false。一个后台 Worker 在双方都提交后、或者 14 天到期后（以先到者为准）同时公开两条评价。这就在机制上落实了盲评策略。

Properties 表和 Users 表上的房源评分、房东/房客评分都是反范式化的计数器，在评价公开时由 Kafka 消费者异步更新。

**评价服务的瓶颈**

评价的量很小（每单一条，每天 50 万单，也就是每天最多 50 万条评价）。PostgreSQL 轻松扛得住。14 天的延迟公开机制也意味着评价流水线没有实时性压力。

## 11\. 通知服务

架构和前面所有系统一致。上游服务通过 Kafka 扇出，iOS 走 APNs，Android 走 FCM，邮件走 SES。

通知服务订阅：

- booking.confirmed -\> 向房客发预订确认 + 向房东发新订单提醒
- booking.cancelled\_guest -\> 向房客发取消确认 + 向房东发提醒
- booking.cancelled\_host -\> 向房客发房东取消提醒（附退款信息）
- payment.payout\_completed -\> 向房东发打款确认
- booking.completed -\> 向房客和房东双方发评价提示
- review.published -\> 向双方发送「你的评价已公开」通知
- availability.expiring -\> 向房客发「请尽快完成预订，占位将在 2 分钟后失效」的催促提示

通知服务的瓶颈

Airbnb 的通知量远低于 Twitter 或 Instagram，因为它是交易型的（预订事件）而不是社交型的（点赞、关注）。不需要做合并，因为每条通知都是独立且重要的事件。标准的 APNs/FCM 投递足以从容应对这个量级。

## 12\. 完整数据流总览

**房源上架流程（房东）：** 房东 -\> API 网关（认证） -\> 房源服务 -\> 预签名 S3 URL 房东 -\> S3（照片直传） S3 -\> Kafka -\> 照片处理 Worker -\> S3（缩略图、中等尺寸、全分辨率变体） 房源服务 -\> PostgreSQL Properties 表 房源服务 -\> PropertyAvailability 表（房东设置可订日期） 房源服务 -\> Kafka（property.created） -\> 搜索索引 Worker -\> Elasticsearch **搜索流程（房客）：** 房客搜索「果阿 12 月 20-25 日 2 位房客」 -\> 搜索服务 -\> Elasticsearch geo\_distance + 价格 + 设施过滤（粗筛） -\> Redis 位图 MGET（对所有候选做精确日期区间可订检查） -\> Redis MGET（为过滤后的结果填充房源元数据） -\> CDN（房源照片 URL 直接分发） -\> 返回地图坐标用于渲染地图视图 **预订流程（房客）：** 房客点击「预订」 -\> 预订服务 -\> 预订状态 INITIATED -\> Kafka（booking.initiated） -\> 可订状态服务 -\> 对请求日期执行 PostgreSQL SELECT FOR UPDATE -\> 日期可订：UPDATE status = 'reserved'，reserved\_until = now + 10 分钟 -\> Kafka（availability.reserved） -\> 支付服务 -\> 在 Redis 中做幂等检查 -\> 调用支付网关 -\> 成功：Kafka（payment.completed） -\> 可订状态服务：UPDATE status = 'booked' -\> 预订服务：预订状态 CONFIRMED -\> 通知服务：向房客发确认 + 向房东发提醒 -\> 失败：Kafka（payment.failed） -\> 可订状态服务：UPDATE status = 'available'（补偿事务） -\> 预订服务：预订状态 EXPIRED -\> 通知服务：向房客发失败提醒 **占位过期流程：** 过期 Worker 每 60 秒运行一次 -\> SELECT \* FROM PropertyAvailability WHERE status = 'reserved' AND reserved\_until < NOW() -\> UPDATE status = 'available'，booking\_id = NULL，reserved\_until = NULL -\> Kafka（reservation.expired） -\> 通知服务 -\> 通知房客 **房东打款流程：** 房客入住 -\> 打款服务启动 24 小时计时器 24 小时后无纠纷 -\> 向房东发起银行转账 -\> UPDATE Payments SET status = 'payout\_completed' -\> Kafka（payment.payout\_completed） -\> 通知服务 -\> 通知房东 **评价流程：** booking.completed -\> 通知服务 -\> 向房客和房东发评价提示 房客提交评价 -\> Reviews 表（is\_published = false） 房东提交评价 -\> Reviews 表（is\_published = false） 双方都已提交 或 已过 14 天 -\> 后台 Worker 置 is\_published = true -\> Kafka（review.published） -\> 评分更新 Worker -\> UPDATE Properties.rating、[Users.host](https://x.com/Harry_The_Nerd/status/Users.host)\_rating / guest\_rating -\> 搜索索引 Worker -\> UPDATE Elasticsearch 文档的 rating 字段 -\> 通知服务 -\> 向双方发送公开确认

## 13\. 韧性与容错

**PostgreSQL 悲观锁**（SELECT FOR UPDATE）是最核心的正确性保证。同一房源同一批日期上不可能有两个预订同时成功，因为数据库在行级别强制了串行化。无论并发请求有多少，都绕不过这个保证。

**带 TTL 的两阶段占位**确保锁只被持有毫秒级而不是分钟级。如果房客中途放弃支付页面，10 分钟的占位 TTL 会自动释放这个占位。过期 Worker 作为兜底，负责捡起 TTL 机制漏掉的占位。

**Saga 补偿事务**确保支付失败时一定会释放可订状态占位。系统不会卡在「日期被标记为 reserved，却没有对应的预订或支付」这种状态里。

整条预订流水线上的 **Kafka 持久化**意味着即使下游服务临时不可用，也不会丢事件。所有 saga 步骤都可以各自从 Kafka 的偏移量重试。

预订和支付上的**幂等键**保证在任何重试或网络故障场景下都不会产生重复预订和重复扣款。

可订状态搜索的 **Redis 位图快路径**意味着搜索查询永远不会持有 PostgreSQL 锁。权威锁只在真正发起预订时才用到，从而把数据库留给并发的预订事务。

房源照片的 **CDN 缓存**让图片从全球边缘节点分发。房源照片极少变化，所以缓存命中率非常高。

**盲评公开机制**在数据库层面通过 is\_published 标志加后台 Worker 落实。即使通知服务挂了，Worker 下一轮运行时评价最终还是会被公开。

**Elasticsearch 可订状态同步**是最终一致的，但对搜索来说这可以接受。某个房源被订满之后，可能还会在搜索结果里出现几分钟。Redis 位图检查以及真正下单时的 PostgreSQL 锁，保证了即便搜索层是最终一致的，也不会出现重复预订。

## 14\. 技术选型总结

**API 网关** \-\> Kong / AWS API Gateway（认证、限流、房东/房客流量路由）

**对象存储** \-\> AWS S3（房源照片，持久可靠，天然适配 CDN）

**CDN** -\> CloudFront / Akamai（房源照片的边缘缓存，照片极少变化，缓存命中率高）

**解耦** \-\> Apache Kafka（预订 saga 事件、房源更新、评价公开、打款触发）

**搜索引擎** -\> 带 geo\_point 的 Elasticsearch（在一条查询里同时做地理半径、日期可订、价格、评分、设施过滤）

**房源库** -\> PostgreSQL 分片 + 副本（房源元数据，定价和房东信息要求强一致性）

**可订状态库** -\> 带行级锁的 PostgreSQL（日历表模式，用 SELECT FOR UPDATE 防止重复预订）

**预订库** -\> PostgreSQL（预订状态机、幂等键、saga 编排）

**支付库** \-\> PostgreSQL（金融记录、房客扣款、房东打款、幂等性）

**评价库** \-\> PostgreSQL（双向盲评系统，延迟公开）

**可订状态快路径** -\> Redis 位图（每房源每月的可订状态，用按位与做日期区间检查，作为搜索预过滤）

**房源元数据缓存** -\> Redis MGET（批量填充搜索结果和房源页）

**用户资料缓存** \-\> Redis（房东和房客资料、订阅状态、回复率信号）

**占位过期** -\> 后台 Worker + PostgreSQL TTL 字段（自动释放被放弃的占位）

**打款调度** -\> Kafka 延迟事件 + 打款 Worker（入住后扣住 24 小时再给房东打款）

**通知投递** -\> APNs + FCM + AWS SES（交易型预订通知，无需合并）

就到这里啦……干杯！

点赞、评论、分享、转发！
