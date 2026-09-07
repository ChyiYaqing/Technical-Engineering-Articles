---
title: "System Design Of A Proximity Service"
url: "https://x.com/Harry_The_Nerd/status/2091192716059459825"
category: "Engineering Articles"
date: "2026-08-22"
description: "Designing a proximity service for discovering nearby apps"
lang: "zh-CN"
---

# 附近服务（Proximity Service）的系统设计

> 设计一个用于发现附近内容的附近服务
>
> 原文：[https://x.com/Harry_The_Nerd/status/2091192716059459825](https://x.com/Harry_The_Nerd/status/2091192716059459825) · 2026-08-22

![封面图](https://pbs.twimg.com/media/HQRHXcobcAAghP1.jpg)

## 1\. 需求

**功能性需求**

- 在给定半径内搜索商户（纬度、经度，半径最大 20 km）
- 返回按距离排序的结果
- 商户的增删改查（新增、编辑、删除条目）
- 商户详情页（名称、地址、类目、评分、照片、营业时间）
- 支持类目筛选（餐厅、咖啡馆、ATM、药店）

**范围之外**

- 实时位置追踪（Uber 司机的移动）
- 路径规划与导航（Google Maps 路线）
- 评论与评分
- 广告与推广条目
- 个性化与推荐

**非功能性需求**

- 搜索延迟低于 100ms
- 高可用性，搜索绝不能挂
- 读写比约 50:1，为读做了大量优化
- 位置精度在几百米以内
- 常见搜索模式的缓存命中率高于 80%
- 新增商户条目可接受最终一致性（几秒内出现）
- 商户删除需要强一致性（立即从结果中移除）

## 2\. 容量估算

- **日活用户（DAU）：** 1 亿
- **每用户每天搜索次数：** 5
- **搜索总 QPS：** 100M x 5 / 86,400 = 约 5000 QPS
- **全球商户总数：** 约 2 亿
- **写 QPS：** 2 亿商户中有 5% 会定期更新 = 每天 1000 万次更新 / 86,400 = 约 100 QPS
- **读写比：** 5000:100 = 50:1
- **商户表大小：** 2 亿行 x 每行 200 字节 = 核心数据约 40 GB
- **PostGIS 空间索引大小：** 额外约 10-20 GB
- **内存总占用：** 约 60 GB，在现代数据库服务器的缓冲池里绰绰有余

这个系统是压倒性的读密集型。2 亿商户的空间索引能放进内存，意味着 PostGIS 的半径查询在热路径上永远不会碰磁盘。再加上按 geohash 分桶缓存搜索结果，PostGIS 的负载被进一步压缩到只剩缓存未命中的部分。

## 3\. API 设计

搜索 API

GET /v1/search?lat=19.0760&long=72.8777&radius=5&category=restaurant&limit=20 响应：

```
{
  "businesses": [
    {
      "business_id": "biz_123",
      "name": "Cafe Madras",
      "address": "12 SV Road, Bandra, Mumbai",
      "category": "restaurant",
      "distance_km": 0.8,
      "rating": 4.3,
      "photo_url": "https://cdn.proximity.com/..."
    }
  ],
  "total": 47,
  "returned": 20
}
```

商户增删改查 API

POST /v1/businesses（创建新条目） GET /v1/businesses/{id}（获取商户详情） PUT /v1/businesses/{id}（更新商户） DELETE /v1/businesses/{id}（删除商户）

## 4\. 高层服务划分

这个附近服务刻意保持精简。它是一个平台组件，而不是一个完整产品，三个服务就覆盖了整个设计：

**搜索服务（Search Service）** 从客户端接收纬度、经度、半径和类目，编排完整的搜索流程，处理缓存，计算距离，返回排好序的结果。

**位置服务（Location Service）** 只负责对 PostGIS 做纯粹的地理空间查询。它唯一的工作就是返回给定半径内的商户 ID。它对缓存和排序一无所知。

**商户服务（Business Service）** 处理商户主的增删改查操作，在 PostgreSQL 中管理商户元数据，并在更新时触发缓存失效。

## 5\. 地理空间存储 —— PostGIS

为什么选 PostGIS 而不是别的方案

附近查询有好几种地理空间技术可选：

**Geohash** 把坐标转换成字符串前缀。作为缓存 key 和 CDN 模式很快，但它用的是矩形格子而不是真正的圆形，在格子边界处会有误差。

**Redis GEO** 是纯内存的，速度极快，但在 2 亿商户的规模下成本很高，而且缺乏持久化保证。Redis GEO 对 Tinder 来说是正确选择，因为用户一直在移动、位置更新极其频繁。而商户很少挪窝，所以持久性和精度更重要。

**四叉树（Quadtree）** 根据数据密度动态划分空间，对分布不均的数据非常合适，但实现和维护都复杂。

**PostGIS** 是 PostgreSQL 的空间扩展，提供了原生地理数据类型和 R 树空间索引。它通过 ST\_DWithin 支持真正的圆形半径查询，精确、持久，并且可以通过 PostgreSQL 只读副本做水平扩展。

对于一个查找静态商户的附近服务来说，PostGIS 在精度、简洁性和运维熟悉度上都胜出。

商户表（PostgreSQL + PostGIS）

Businesses

```pgsql
business_id     UUID, primary key
name            VARCHAR
address         TEXT
category        VARCHAR (restaurant, cafe, atm, pharmacy, etc)
lat             FLOAT
long            FLOAT
location        GEOGRAPHY(POINT, 4326) (PostGIS geometry column)
geohash         VARCHAR(6) (precision 6, ~1km x 1km cell, computed from lat/long)
rating          FLOAT
is_active       BOOLEAN
created_at      TIMESTAMP
updated_at      TIMESTAMP
```

location 列以 PostGIS 的 GEOGRAPHY 类型存储坐标。在这一列上建了空间索引：

```pgsql
CREATE INDEX idx_businesses_location ON Businesses USING GIST(location);
```

GIST 索引是一种 R 树空间索引。PostGIS 用它在计算精确距离之前高效地缩小候选集，这让半径查询即使面对 2 亿行也依然很快。

geohash 列（精度 6，格子大约 1 km x 1 km）是一个计算列，用作搜索结果的缓存 key。它单独建了索引，便于快速查找缓存 key：

CREATE INDEX idx\_businesses\_geohash ON Businesses(geohash);

```pgsql
CREATE INDEX idx_businesses_location ON Businesses USING GIST(location);
```

PostGIS 半径查询

```pgsql
SELECT
    business_id,
    name,
    address,
    category,
    rating,
    lat,
    long,
    ST_Distance(
        location::geography,
        ST_MakePoint(72.8777, 19.0760)::geography
    ) AS distance_meters
FROM Businesses
WHERE
    ST_DWithin(
        location::geography,
        ST_MakePoint(72.8777, 19.0760)::geography,
        5000  -- 5 km in meters
    )
    AND category = 'restaurant'
    AND is_active = true
ORDER BY distance_meters ASC
LIMIT 50;
```

ST\_DWithin 借助 GIST 空间索引高效地找出候选集。ST\_Distance 再为每个候选计算精确距离。这个组合很快——空间索引在计算精确距离之前就把 2 亿行里的绝大部分过滤掉了。

PostGIS 的扩展

PostGIS 的读由 PostgreSQL 只读副本承担。全部 5000 读 QPS 都打到只读副本。写 QPS（100）打到主节点。每个只读副本上的空间索引都能放进内存，所以热路径查询永远不碰磁盘。

在极端规模下，Businesses 表可以按地理位置分片——亚洲一个分片、欧洲一个、美洲一个——因为附近查询总是区域性的，从不跨洲。

## 6\. 缓存策略

50:1 的读写比让缓存变得不可或缺。两层缓存分别处理搜索流程的不同部分。

**第一层 —— 搜索结果缓存（按 geohash 分桶）**

原始的经纬度坐标不能直接当缓存 key——同一个街区里位置略有差异的两个用户会产生不同的缓存 key，双双未命中。解决办法是按 **精度 6 的 geohash** 给搜索分桶。

一个精度 6 的 geohash 格子大约 1.2 km x 0.6 km。落在同一格子里的两个用户会拿到同一份缓存的搜索结果。在很多用户搜索同一片街区的密集城区，这能大幅提高缓存命中率。

Key: search:{geohash\_p6}:{radius\_km}:{category} Value: 装着 business\_id 的 Redis SET TTL: 10 分钟

例如：

Key: search:te7u3x:5:restaurant Value: {biz\_123, biz\_456, biz\_789, biz\_101, ...} TTL: 10 分钟

值用 Redis SET 而不是普通列表，是因为缓存失效时需要精准地增删单个商户 ID，而不必让整个 key 失效。

**第二层 —— 商户对象缓存**

搜索结果返回的是商户 ID。完整的商户详情从另一个商户对象缓存中补齐：

Key: business:{business\_id} Value: { name, address, category, rating, hours, photo\_url, lat, long } TTL: 24 小时

当一次搜索返回 20 个商户 ID 时，这 20 个商户对象通过一次 Redis MGET 调用全部取回：

MGET business:biz\_123 business:biz\_456 business:biz\_789 ...

一次往返，20 条记录同时取回。任何一个商户未命中缓存时，回落到 PostgreSQL 查询，并回写 Redis。

**支持多种半径的缓存 key 设计**

从同一个 geohash 格子出发，搜 5 km 和搜 10 km 会得到不同的结果集。因此半径也要放进缓存 key：

search:te7u3x:5:restaurant（5 km 搜索） search:te7u3x:10:restaurant（10 km 搜索） search:te7u3x:20:restaurant（20 km 搜索） search:te7u3x:5:cafe（不同类目）

每一种组合都有自己的缓存条目。

## 7\. 完整搜索流程

当用户搜索「孟买 Bandra 附近 5 km 内的咖啡店」时：

**第 1 步 —— 收到请求**

GET /v1/search?lat=19.0544&long=72.8375&radius=5&category=cafe&limit=20

搜索服务通过 API 网关收到请求。

**第 2 步 —— 计算 geohash 缓存 key** 搜索服务根据用户坐标计算精度 6 的 geohash：

geohash(19.0544, 72.8375, precision=6) -\> "te7u3x"

缓存 key：search:te7u3x:5:cafe

**第 3 步 —— 检查 Redis 缓存**

SMEMBERS search:te7u3x:5:cafe

**缓存命中：** Redis 返回一个装着商户 ID 的 SET。直接跳到第 5 步。

**缓存未命中：** 继续第 4 步。

**第 4 步 —— PostGIS 查询（仅缓存未命中时）** 搜索服务带着坐标、半径和类目调用位置服务。

位置服务在只读副本上执行 PostGIS 查询：

```pgsql
SELECT business_id, lat, long
FROM Businesses
WHERE ST_DWithin(location::geography, ST_MakePoint(72.8375, 19.0544)::geography, 5000)
AND category = 'cafe'
AND is_active = true
LIMIT 50;
```

结果写回 Redis：

SADD search:te7u3x:5:cafe biz\_123 biz\_456 biz\_789 ... EXPIRE search:te7u3x:5:cafe 600

**第 5 步 —— 补齐商户对象** 通过 Redis MGET 为返回的所有商户 ID 取回完整详情：

MGET business:biz\_123 business:biz\_456 business:biz\_789 ...

其中任何单个商户未命中缓存时，从 PostgreSQL 取回并回填 Redis。

**第 6 步 —— 距离计算与排序** 对每个商户，用用户坐标和商户坐标（来自商户对象缓存）计算 Haversine 距离：

```python
def haversine(user_lat, user_long, biz_lat, biz_long):
    R = 6371
    dlat = radians(biz_lat - user_lat)
    dlong = radians(biz_long - user_long)
    a = sin(dlat/2)**2 + cos(radians(user_lat)) * cos(radians(biz_lat)) * sin(dlong/2)**2
    return R * 2 * asin(sqrt(a))
```

按距离升序排序。返回前 20 条。

距离在应用层计算，因为它是相对于每个用户的精确坐标而言的。哪怕两个用户在同一个 geohash 格子里，位置也略有差别，到同一个商户的距离并不相同。对缓存结果来说，在 Redis 或 PostGIS 里预先排序是不可行的。

**第 7 步 —— 响应** 返回排好序的商户列表，包含名称、地址、类目、距离、评分和照片 URL。

总延迟拆解：

- Redis 缓存命中：约 5ms（SMEMBERS + MGET）
- Redis 未命中 + PostGIS：约 30-50ms（只读副本上的 PostGIS 空间查询）
- 应用层距离排序：约 1ms
- 端到端总计：命中和未命中两条路径都在 100ms 以内

## 8\. 商户服务 —— 写流程

新增商户条目

当商户主提交一个新条目时：

商户服务校验输入（坐标合法、必填字段齐全、类目在允许的枚举内）

根据经纬度计算 geohash：geohash(lat, long, precision=6)

INSERT 进 PostgreSQL 的 Businesses 表：

```pgsql
INSERT INTO Businesses (business_id, name, address, category, lat, long,
       location, geohash, rating, is_active, created_at)
   VALUES (gen_random_uuid(), 'Cafe Madras', '12 SV Road...', 'restaurant',
       19.0544, 72.8375, ST_MakePoint(72.8375, 19.0544)::geography,
       'te7u3x', 0.0, true, NOW());
```

向 Kafka 发布 business.created 事件

缓存更新 Worker 消费该事件：SET business:{business\_id} {完整商户对象}，TTL 24 小时 SADD search:{geohash}:\*:{category} business\_id（仅针对已存在的缓存 key）

商户在几秒内即可被搜到

先写数据库，再更新缓存。这可以避免商户在数据库记录尚不存在时就出现在搜索结果里的破碎体验。

商户更新（非位置变更）

当商户更新名称、营业时间或照片，但坐标不变时：

UPDATE PostgreSQL 的 Businesses 表

向 Kafka 发布 business.updated 事件

缓存失效 Worker：DEL business:{business\_id}（让商户对象缓存失效） 搜索结果缓存 key 不失效（该商户 ID 在搜索结果中依然有效） 商户对象缓存在下次取回时重新填充

**商户位置更新（地址变更）**

当商户搬到新地址、坐标随之改变时：

计算旧 geohash（来自现有数据库记录）和新 geohash（来自新坐标）

UPDATE Businesses 表，写入新的 lat、long、location、geohash

向 Kafka 发布 business.location\_updated 事件，同时带上新旧 geohash

缓存失效 Worker：扫描匹配 search:{old\_geohash}:\* 的 Redis key 对每个匹配到的 key 执行 SREM search:{old\_geohash}:{radius}:{category} business\_id DEL business:{business\_id}（让商户对象缓存失效） 对每个已存在的新 geohash 缓存 key 执行 SADD search:{new\_geohash}:{radius}:{category} business\_id 商户对象缓存在下次取回时重新填充

对单个商户 ID 做精准的 SREM，而不是删掉整个缓存 key，可以避免高流量 geohash 格子上的缓存击穿（cache stampede）。

**商户删除**

当一个商户被删除时：

UPDATE Businesses SET is\_active = false（软删除）

向 Kafka 发布 business.deleted 事件

缓存失效 Worker：扫描匹配 search:{geohash}:\* 的 Redis key 从所有匹配的搜索缓存 key 中 SREM 掉该商户 ID DEL business:{business\_id}（删除商户对象缓存）

商户立即从搜索结果中消失（下一次搜索拿到的是已被 SADD 清理过的集合）

用软删除而不是硬删除，既保留了记录以备审计，也让误删的商户有可能恢复。硬删除由后台清理任务在 30 天后执行。

## 9\. 数据模型汇总

**Businesses 表（PostgreSQL + PostGIS）**

Businesses

```pgsql
business_id     UUID, primary key
name            VARCHAR
address         TEXT
category        VARCHAR
lat             FLOAT
long            FLOAT
location        GEOGRAPHY(POINT, 4326) [GIST spatial index]
geohash         VARCHAR(6) [B-tree index]
rating          FLOAT
phone           VARCHAR
website         VARCHAR
hours           JSONB (opening hours per day)
is_active       BOOLEAN
created_at      TIMESTAMP
updated_at      TIMESTAMP
```

**BusinessPhotos 表（PostgreSQL）**

BusinessPhotos

```pgsql
photo_id        UUID, primary key
business_id     UUID, foreign key -> Businesses
s3_url          TEXT
order_index     INTEGER
uploaded_at     TIMESTAMP
```

照片存在 S3，通过 CDN 分发。照片 URL 已包含在商户对象缓存里，所以搜索时不需要再单独去取照片。

## 10\. 完整数据流汇总

搜索流程（缓存命中）：客户端 -\> API 网关 -\> 搜索服务 -\> 根据经纬度计算 geohash（精度 6）-\> Redis SMEMBERS search:{geohash}:{radius}:{category}（缓存命中）-\> Redis MGET business:biz\_1 business:biz\_2 ...（补齐对象）-\> 应用层 Haversine 距离计算 -\> 按距离升序排序，返回前 20 条 -\> 总延迟：约 5ms 搜索流程（缓存未命中）：客户端 -\> API 网关 -\> 搜索服务 -\> 根据经纬度计算 geohash -\> Redis SMEMBERS（缓存未命中）-\> 位置服务 -\> 在只读副本上执行 PostGIS ST\_DWithin 查询 -\> Redis SADD search:{geohash}:{radius}:{category} {business\_ids}（填充缓存）-\> Redis MGET（补齐商户对象）-\> 应用层距离排序 -\> 返回前 20 条 -\> 总延迟：约 30-50ms 新增商户条目流程：商户主 -\> API 网关 -\> 商户服务 -\> 校验输入 -\> 计算 geohash -\> INSERT 进 PostgreSQL Businesses 表 -\> Kafka（business.created）-\> 缓存更新 Worker： -\> SET business:{business\_id} {object} TTL 24h -\> SADD search:{geohash}:\*:{category} business\_id（仅已存在的 key）-\> 商户在几秒内可被搜到 商户位置更新流程：商户主 -\> API 网关 -\> 商户服务 -\> 从 PostgreSQL 取旧 geohash -\> 根据新坐标计算新 geohash -\> UPDATE PostgreSQL（新的 lat、long、location、geohash）-\> Kafka（business.location\_updated，带新旧 geohash）-\> 缓存失效 Worker： -\> 从所有 search:{old\_geohash}:\* key 中 SREM（精准移除）-\> DEL business:{business\_id} -\> 向所有 search:{new\_geohash}:\* key 执行 SADD（仅已存在的 key） 商户删除流程：商户主 -\> API 网关 -\> 商户服务 -\> UPDATE is\_active = false -\> Kafka（business.deleted）-\> 缓存失效 Worker： -\> 从所有 search:{geohash}:\* key 中 SREM -\> DEL business:{business\_id} -\> 商户立即从搜索结果中消失

## 11\. 韧性与容错

**PostGIS 只读副本** 保证即使主库故障，搜索查询也能继续。全部 5000 读 QPS 都走副本。主库故障只影响写操作（100 QPS），这些写会在 Kafka 里排队，直到主库恢复或某个副本被提升为主。

**Redis 缓存** 意味着绝大多数搜索查询根本到不了 PostGIS。PostGIS 变慢只会拖累缓存未命中的性能，缓存命中路径（占 80% 以上流量）不受影响。

**Kafka 的持久性** 覆盖所有商户写事件，意味着即使缓存失效 Worker 崩溃，缓存失效和更新操作也不会丢失。Worker 恢复后从 Kafka 的 offset 重放即可。

**软删除** 让误删的商户可以恢复。硬删除只在 30 天宽限期之后，由后台清理任务执行。

**用 SREM 做精准缓存失效** 避免了高流量 geohash 格子上的缓存击穿。从缓存结果集中移除一个商户，不会让同一格子里另外几百个商户的结果一起失效。

**数据库地理分片**（亚洲、欧洲、美洲分片）在区域之间提供了故障隔离。某个区域的数据库出问题，不会影响其他区域的附近搜索。

**缓存上较短的 PostGIS 查询 TTL（10 分钟）** 意味着陈旧的搜索结果（新增但还没进缓存的商户）最多 10 分钟内就会自我纠正，不需要任何显式失效操作。

## 12\. 技术选型汇总

**API 网关** -\> Kong / AWS API Gateway（认证、限流、路由）

**地理空间数据库** \-\> PostgreSQL + PostGIS 扩展（R 树 GIST 空间索引，用 ST\_DWithin 做真正的圆形半径查询，2 亿商户行可放进内存）

**搜索结果缓存** -\> 每个 geohash 格子一个 Redis SET（用 SMEMBERS 快速读取，用 SADD/SREM 做精准失效而不引发击穿）

**商户对象缓存** -\> Redis MGET（一次往返批量补齐完整商户详情）

**距离排序** -\> 应用层 Haversine 计算（距离因用户而异，无法预排序，对 20-50 条结果做内存排序只需微秒级）

**Geohash 缓存 key** -\> 精度 6 的 geohash（1.2km x 0.6km 格子，相比原始坐标大幅提升缓存命中率）

**照片存储** -\> AWS S3 + CDN（商户照片从边缘节点分发，因为照片极少变动所以缓存命中率很高）

**写路径解耦** \-\> Apache Kafka（商户的创建/更新/删除事件，缓存失效 Worker，可靠投递）

**数据库扩展** -\> PostgreSQL 只读副本（5000 读 QPS 分摊到各副本，100 QPS 的写打到主库）

**地理分片** \-\> 区域化的 PostgreSQL 集群（亚洲、欧洲、美洲，区域间故障隔离）

以上就是全部内容，干杯！！

点赞、分享、评论、转发！
