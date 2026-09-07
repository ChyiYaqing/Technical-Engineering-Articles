---
title: "Design Netflix"
url: "https://x.com/Harry_The_Nerd/status/2071256571028648283"
category: "HLD"
date: "2026-06-28"
description: "System design for a video streaming platform like Netflix."
lang: "zh-CN"
---

# 设计 Netflix

> 类似 Netflix 的视频流媒体平台的系统设计。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2071256571028648283](https://x.com/Harry_The_Nerd/status/2071256571028648283) · 2026-06-28

![封面图](https://pbs.twimg.com/media/HLaBVqJakAAd-NX.jpg)

## 高层设计：Netflix

## 1\. 需求

**功能性需求**

- 内容接入与转码流水线（从制片方源文件到可播放流）
- 播放电影、剧集和纪录片
- 内容搜索
- 片单（Watchlist）管理
- 继续观看（恢复播放进度）
- 推荐服务
- 指标采集与数据聚合
- 通知（新内容上线提醒、继续观看提示）

**不在范围内**

- 直播
- 离线下载（架构上与流媒体类似，只是预先拉取好分片）
- 计费与订阅管理
- 内容授权与 DRM 内部机制
- 单账号下的多用户档案管理
- 制片方协作工具

**非功能性需求**

- 超高可用性，播放过程中绝不能中断
- 起播延迟低，首帧要在 2 秒内出现
- 自适应码率流媒体，网络变差时画质平滑降级，绝不卡顿缓冲
- 重磅新片上线不能有冷启动，上线日之前要主动预热缓存
- PB 级存储，承载 15,000 部作品，每部 1200 个变体
- 推荐、片单和指标可以接受最终一致性
- 订阅权益和 DRM 授权要求强一致性

## 2\. 容量估算

- **日活用户（DAU）：** 1 亿
- **人均每日观看时长：** 150 分钟（大约 3 集剧或 1 部电影）
- **每日总观看分钟数：** 1 亿 x 150 分钟 = 每天 150 亿分钟
- **平均码率：** 各清晰度综合下来约 10 Mbps
- **每日传输数据总量：** 150 亿分钟 x 60 秒 x 10 Mbps / 8 = 每天约 11 PB
- **Netflix 片库：** 约 15,000 部作品，平均每部 90 分钟
- **每部作品的变体数：** 约 1200 个（分辨率、编解码器、HDR 格式、音频格式的组合）
- **每部作品每个变体的存储：** 10 Mbps x 90 分钟 x 60 秒 / 8 = 约 6.75 GB
- **每部作品的存储：** 6.75 GB x 1200 个变体 = 每部约 8 TB
- **片库总存储：** 8 TB x 15,000 部 = **总计约 120 PB**
- **每部作品每个变体的分片数：** 90 分钟 x 每分钟 6 个分片 = 每个变体约 540 个分片
- **存储对象总数：** 540 x 1200 x 15,000 = **S3 中约 97 亿个对象**

这里最主要的挑战不像 Twitter 或 Amazon 那样是写入吞吐量，而是每天向 1 亿并发观众提供 11 PB 数据，同时做到起播低于 2 秒、零卡顿。高峰时段，Netflix 占据了全球互联网流量的大约 15%。仅这一条事实，就把整个架构推向了以边缘为先的内容分发方案——Open Connect。

## 3\. API 网关

所有客户端请求都从 API 网关进入，由网关负责认证、限流和路由。和之前的系统不同，Netflix 的 API 网关多了一项职责：**区域路由**。Netflix 服务遍及各大洲的用户，会把播放请求路由到最近的流媒体基础设施集群。内容授权也因地区而异，所以网关会在请求到达流媒体服务之前先执行基于地理位置的访问控制。

API 网关绝不参与视频分片的分发。分片完全由 Open Connect Appliance 提供，热路径上没有任何后端 API 的参与。

## 4\. 内容接入与转码流水线

这是我们设计过的七个系统里最复杂的转码流水线。一部 2 小时的电影，以 100GB 的原始母版文件形式到达，必须变成大约 864,000 个独立的分片文件，用户才可能开始播放。

**母版文件交付**

制片方通过 AWS Direct Connect（专用高带宽光纤链路）把原始母版文件交给 Netflix；文件特别大时，也会用物理硬盘寄送到 Netflix 的数据中心。原始母版一到达就立即存入 S3，作为永久存档。

转码流水线的各个阶段

**阶段 1 - 预处理器** 预处理器接收原始母版文件，执行三项任务：

**GOP 切分（Group of Pictures，图像组）：** 视频被切成一个个 GOP，每个通常 2 秒长。GOP 是一段自包含、可独立编码的视频帧单元。一部 2 小时的电影大约产生 3600 个 GOP。这一步切分是实现大规模并行化的关键洞察：每个 GOP 都能在不了解其他 GOP 的情况下独立编码。

**DAG 生成：** 预处理器为所有编码任务构建一张有向无环图（DAG）。DAG 中的每个节点是一个「GOP x 变体」组合。一部 2 小时的电影会在 DAG 中产生 3600 个 GOP x 1200 个变体 = **432 万个编码任务**。

**音频与字幕分离：** 音轨（立体声、5.1、Dolby Atmos）和字幕文件从视频流中分离出来，各自走独立的编码流水线处理。

**阶段 2 - DAG 调度器** DAG 调度器接收任务图，把它拆成一个个独立任务，投入由 Kafka 支撑的分布式任务队列。任务的优先级经过精心安排：每个变体的头几个 GOP 会被优先编码，确保整部作品在完全编码完成之前就能开始播放。用户可以在第 2、3 集还在编码时就开始看新剧的第 1 集。

**阶段 3 - 资源管理器** 资源管理器维护三个队列：

- **任务队列：** 来自 DAG 调度器的所有待处理编码任务
- **Worker 队列：** 所有可用、随时能接任务的编码 Worker
- **运行中队列：** 所有正在执行的任务及其分配到的 Worker

资源管理器持续把任务匹配给可用的 Worker，并根据队列深度扩缩 Worker 池。在内容接入高峰期，Netflix 会为转码跑上数千台 EC2 实例。

**阶段 4 - 任务 Worker** 每个任务 Worker 领取一个「GOP x 变体」任务，独立完成编码。Worker 按变体类型分工：

- **视频编码 Worker：** 运行在 GPU 加速实例上的 H.264、H.265、AV1 编码器
- **音频编码 Worker：** AAC 立体声、Dolby Digital 5.1、Dolby Atmos 编码器
- **字幕 Worker：** 把字幕文件转换成所有支持语言的定时文本格式

在 GPU Worker 上，把单个 GOP 编成单个变体只需毫秒级时间。数千个 Worker 并行运行，一部 2 小时的电影在全部 1200 个变体上的编码可以在数小时内完成，而顺序执行则需要数周。

**阶段 5 - 输出与存储** 编码后的分片按一个确定性的路径写入 S3：

content/{content\_id}/{variant\_id}/chunk\_{gop\_index}.m4s

这种确定性命名意味着任何服务都能直接拼出任意变体任意分片的 URL，无需查数据库。

**阶段 6 - 完成后的 Kafka 扇出** 当一部作品的所有分片都编码完成，转码服务向 Kafka 发布 content.transcoded 事件。三个消费者会消费它：

- **元数据 Worker** 把内容元数据写入 Content 库和 Redis 缓存
- **搜索索引 Worker** 把该作品索引进 Elasticsearch
- **MPD 生成 Worker** 为所有设备档位生成 DASH 清单文件并存入 S3

**产出的变体**

Netflix 会把每部作品编码成大约 1200 个变体，覆盖：

- **分辨率：** 360p、480p、720p、1080p、4K
- **编解码器：** H.264（设备兼容性最广）、H.265/HEVC（压缩率比 H.264 好 50%）、AV1（Netflix 偏好的编解码器，比 H.265 再好 30%，但编码更慢）
- **HDR 格式：** SDR（标准动态范围）、HDR10、Dolby Vision
- **音频格式：** AAC 立体声、Dolby Digital 5.1、Dolby Atmos
- **语言：** 每部作品有多条音轨和多条字幕轨

**转码流水线的瓶颈**

编码步骤本身就是瓶颈。AV1 编码明显慢于 H.264 或 H.265，每个 GOP 需要更多 GPU 时间。Netflix 的缓解办法是把 AV1 编码的优先级排在 H.264 之后。一部作品在接入后数小时内就能以 H.264 形式可播放，AV1 变体则在接下来的几天里于后台慢慢完成。

## 5\. Open Connect —— Netflix 自建的 CDN

在视频分片分发上，Netflix 并不主要依赖 CloudFront 或 Akamai 这类第三方 CDN，而是自建了一套名为 **Open Connect** 的 CDN，由数千台 **Open Connect Appliance（OCA）**组成——这些是 Netflix 自有的物理服务器，直接部署在全球各地的 ISP 数据中心和互联网交换点内部。

Open Connect 与标准 CDN 的区别

**标准 CDN（CloudFront/Akamai）：**

- 缓存填充是被动触发的，发生在第一个用户请求时（缓存未命中 -\> 回源 -\> 填充缓存 -\> 提供服务）
- 流量要在 CDN 边缘节点与 ISP 之间穿越公共互联网
- Netflix 无法控制什么内容在什么时候被缓存到哪里

**Open Connect：**

- Netflix 每晚在非高峰时段根据预测的区域热度，主动把内容推送到 OCA 上。在任何用户发起请求之前，热门内容就已经躺在 OCA 里了。
- 流量直接从 ISP 网络内部的 OCA 流向用户家里的路由器，完全不经过公共互联网
- Netflix 依据细粒度的区域观看数据，精确控制每台设备上放哪些内容
- ISP 也受益，因为流量留在自己网络内部，降低了他们的中转成本

**内容投放策略**

Netflix 的内容投放算法每晚决定把哪些作品推到哪些 OCA 上，依据包括：

- 区域观看历史（这家 ISP 的用户群最爱看什么？）
- 即将上线的排期（热门剧的新一集会在上线前预先就位）
- 每台 OCA 的可用存储（OCA 的 SSD 容量有限，通常 100-200 TB）
- 作品热度衰减（不再流行的老内容会被淘汰，腾出空间）

**为新片主动预热缓存**

假设《怪奇物语》第 5 季定在周五上线，Netflix 会从周三晚上就开始把所有剧集分片推送到全球的 OCA 上。到周五早上，全世界每一台主要 OCA 都已经缓存好了内容。周五傍晚数百万用户同时点下播放键时，每一个请求都是缓存命中。没有冷启动，没有缓存填充风暴，没有惊群效应。

对于那些不值得占用 OCA 存储的长尾内容（小众纪录片、区域受众很少的外语片），Netflix 会回退到传统 CDN，或直接由 S3 源站提供。

**Open Connect 的瓶颈**

OCA 的存储容量是有限的。Netflix 全部 120 PB 的片库不可能都放在 OCA 上，只能放热门的那一部分。内容投放算法必须持续优化「什么内容缓存到哪里」。存储被填满的 OCA 必须淘汰不那么热门的内容，为即将上线的新片腾地方，而这套淘汰策略直接影响该 ISP 用户的缓存命中率和播放质量。

## 6\. 流媒体服务

流媒体服务负责管理播放会话、认证用户、生成签名 URL、选择合适的变体，并记录播放进度以支持继续观看。

**完整的起播流程**

**第 1 步 - 播放请求** 用户点击播放。客户端通过 API 网关向流媒体服务发送 POST /playback/start，携带 { user\_id, content\_id, device\_type, supported\_codecs, screen\_resolution, audio\_capabilities }。

**第 2 步 - 并行的权益校验** 三项检查同时进行：

- 用户的订阅是否有效？（用户资料服务 -\> Redis 缓存）
- 这部作品在用户所在区域是否有授权？（内容授权库）
- 用户设备是否通过 DRM 认证？（Android/Chrome 用 Widevine，iOS/Safari 用 FairPlay，Windows 用 PlayReady）

任一检查失败，播放请求立即被拒绝。

**第 3 步 - 变体选择** 根据客户端上报的能力，流媒体服务挑选合适的变体集合：

- 一台 2015 年的 Android 手机只拿到 H.264 变体，且封顶 1080p
- 一台支持 Dolby Vision 的 4K 电视拿到完整变体集，包含 AV1 和 HDR
- Mac 上的浏览器则视浏览器支持情况拿到 H.264 或 H.265

**第 4 步 - 生成个性化 MPD 清单** 流媒体服务为这个特定用户和设备生成一份个性化的 MPEG-DASH MPD（Media Presentation Description，媒体呈现描述）清单。MPD 中包含：

- 所有可用的变体表示（Representation）及其码率和分辨率
- 与用户会话绑定、TTL 很短的签名分片 URL 模板
- 分片时长与序号信息
- 音轨与字幕轨选项

MPD 按会话签名。嵌在 MPD 里的分片 URL 同样带签名，TTL 为几小时。取消订阅的用户没法拿一份旧 MPD 继续看。

**第 5 步 - 签发 DRM 授权** 与此同时，流媒体服务调用 DRM 授权服务，签发一把与用户已认证会话绑定的解密密钥。存放在 S3 和 OCA 上的所有分片都是加密的。客户端手里有 MPD 和签名 URL，但没有 DRM 授权就无法解密分片。这是内容保护的核心机制。

**第 6 步 - 创建播放会话**

PlaybackSessions session\_id UUID，主键 user\_id UUID，外键 -\> Users content\_id UUID，外键 -\> Content device\_id UUID started\_at TIMESTAMP last\_chunk\_index INTEGER last\_position\_ms BIGINT quality\_selected VARCHAR updated\_at TIMESTAMP

last\_chunk\_index 和 last\_position\_ms 由客户端每 30 秒更新一次。这正是「继续观看」背后的机制。下次播放时，流媒体服务读取这条记录，生成一份从上次位置开始的 MPD。

**第 7 步 - 交给 ABR 算法** 客户端拿到 MPD，开始从最近的 OCA 拉取分片。运行在客户端上的 ABR（Adaptive Bitrate，自适应码率）算法负责所有画质决策：

- 从一个保守的中等画质变体起播
- 测量每个分片的下载速度
- 监控缓冲健康度（已缓冲的视频秒数）
- 带宽允许时切到更高的变体，带宽变差时切到更低的变体
- 切换是无缝的：下一个分片来自另一个变体的 URL，播放不中断

从点击播放到首帧出现的总耗时：网络良好时低于 2 秒。

**流媒体服务的瓶颈**

流媒体服务是无状态的，可以水平扩展。会话初始化中计算量最大的部分是 MPD 生成，因为它要为可能多达数千个分片生成个性化的签名 URL。缓解办法是按设备档位缓存 MPD 基础模板，每个会话只做签名这一步的个性化，而不是从头重新生成整份清单。

## 7\. 内容分发 —— MPEG-DASH 与 ABR

**MPEG-DASH 与 HLS 对比**

Netflix 用的是 MPEG-DASH（Dynamic Adaptive Streaming over HTTP），而不是 HLS。主要区别：

- **HLS** 使用 .m3u8 播放列表和 .ts 分片，由 Apple 开发，在 iOS 和 Safari 上原生支持
- **DASH** 使用 .mpd 清单文件和 .m4s 分片（分片式 MP4），与编解码器无关，更适合 Netflix 的多编解码器策略

实际上 Netflix 用的是 **CMAF（Common Media Application Format，通用媒体应用格式）**分片，它同时兼容 DASH 和 HLS 客户端，这样同一套编码分片就能服务所有设备类型。清单格式因客户端而异，但底层的分片文件是共用的。

MPD 结构

```xml
<MPD type="static" mediaPresentationDuration="PT2H">
  <Period>
    <AdaptationSet mimeType="video/mp4" codecs="avc1">
      <Representation id="v1" bandwidth="800000" width="640" height="360">
        <SegmentTemplate media="video_360p_$Number$.m4s" duration="2"/>
      </Representation>
      <Representation id="v2" bandwidth="3000000" width="1280" height="720">
        <SegmentTemplate media="video_720p_$Number$.m4s" duration="2"/>
      </Representation>
      <Representation id="v3" bandwidth="16000000" width="3840" height="2160">
        <SegmentTemplate media="video_4k_$Number$.m4s" duration="2"/>
      </Representation>
    </AdaptationSet>
    <AdaptationSet mimeType="audio/mp4">
      <Representation id="a1" bandwidth="128000" audioSamplingRate="48000">
        <SegmentTemplate media="audio_aac_$Number$.m4s" duration="2"/>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>
```

客户端上的 ABR 算法读取这份清单，根据当前网络状况和缓冲状态，动态决定每个分片要请求哪一个 Representation。

## 8\. 用户资料服务

管理所有用户数据，包括认证凭据、订阅状态、观看历史、片单和播放会话。

**数据模型**

**Users 表（PostgreSQL）**

Users user\_id UUID，主键 email VARCHAR，唯一 name VARCHAR profile\_pic\_url TEXT subscription\_tier ENUM('standard', 'premium', '4k') subscription\_end TIMESTAMP created\_at TIMESTAMP

**Watchlist 表（PostgreSQL）**

Watchlist user\_id UUID，外键 -\> Users content\_id UUID，外键 -\> Content added\_at TIMESTAMP PRIMARY KEY (user\_id, content\_id)

简单的按键访问。「把用户 X 的所有片单项给我」就是 WHERE user\_id = X ORDER BY added\_at DESC。片单变动不频繁，所以按用户缓存在 Redis 中，TTL 较短。

**WatchHistory 表（PostgreSQL）**

WatchHistory user\_id UUID，外键 -\> Users content\_id UUID，外键 -\> Content watched\_at TIMESTAMP completion\_pct FLOAT episode\_id UUID（电影时为 null）

观看历史只追加，写入量很大。每一次完成的观看会话都会写一条记录。这张表也是推荐服务的主要输入。它按 user\_id 分区，以便快速拉取单个用户的历史。

**用户资料服务的瓶颈**

每一次起播都要检查订阅状态。Redis 按用户缓存订阅状态，TTL 为几分钟——足够频繁，能及时捕捉到取消订阅；又足够稀疏，不至于把 PostgreSQL 打爆。观看历史写入量大，通过 Kafka 消费者批量落库，而不是每个会话事件都同步写入。

## 9\. 指标与数据聚合服务

Netflix 从每个客户端收集海量的播放事件流。每一次播放、暂停、拖动、画质切换、缓冲事件、错误和播放完成都会产生一个事件。1 亿日活用户各看 150 分钟，事件量极其庞大。

采集的事件类型

- **播放事件：** 播放、暂停、恢复、拖动、看完、中途放弃
- **画质事件：** ABR 画质切换、缓冲欠载、起播耗时
- **错误事件：** 播放失败、DRM 错误、网络超时
- **互动事件：** 加入片单、搜索查询、点击推荐
- **设备事件：** 设备类型、系统版本、App 版本、网络类型

**接入架构**

所有客户端事件实时发布到 **Kafka**。Kafka topic 按事件类型和用户 ID 分区，以支持并行处理。

两条独立的消费流水线处理这个事件流：

**实时流水线（Apache Flink）** 在数秒内处理事件，用于：

- 实时运维看板（各区域的卡顿率、各设备类型的错误率、CDN 缓存命中率）
- 异常检测（孟买地区缓冲事件突然飙升 = 该 ISP 的 OCA 出了问题）
- A/B 测试指标（新的 ABR 算法是否降低了起播耗时？）
- 实时个性化信号（用户刚看完一集 -\> 触发下一集的通知）

结果落到 Redis，供实时看板消费和运维告警使用。

**批处理流水线（Apache Spark）** 按小时和按天批量处理事件，用于：

- 推荐模型的训练数据
- 内容采购决策（「东南亚增长最快的题材是什么？」）
- 业务分析（剧集完播率、集内的流失点、追剧模式）
- OCA 内容投放优化（每个 ISP 区域最常看的是哪些作品？）

结果以 Parquet 格式落到 **S3 上的数据湖**，用于长期存储和机器学习模型训练。

**面向推荐的按用户聚合**

指标服务在 Redis 中维护按用户聚合的信号：

Key: user\_signals:{user\_id} Value: { genres\_watched: {drama: 45, thriller: 30, comedy: 15, ...}, avg\_session\_duration: 87, preferred\_watch\_time: "evening", completion\_rate: 0.78, recently\_watched: \[content\_id\_1, content\_id\_2, ...\] }

这些聚合结果由 Flink 消费者近实时更新，供推荐服务消费。

**按内容聚合**

Key: content\_signals:{content\_id} Value: { total\_watches: 4500000, completion\_rate: 0.82, avg\_rating: 4.3, rewatch\_rate: 0.12, watchlist\_adds: 890000 }

高完播率和高重看率是很强的信号，说明这部作品值得更积极地推荐，并在 OCA 投放上优先安排。

**指标服务的瓶颈**

Kafka 的接入不是瓶颈，因为 Kafka 本来就是为这种规模设计的。真正需要关注的是 Flink 实时流水线的运维，它必须以低延迟处理每秒数百万条事件。Spark 批处理流水线按计划运行，对它服务的那些场景来说延迟是可以接受的。

## 10\. 推荐服务

推荐服务为每个用户产出一份个性化的作品排序列表。推荐的下发必须很快，因为 App 一打开，它们就要出现在 Netflix 首页上。

两阶段架构

第 **1 阶段 - 候选集生成** 从 15,000 部作品中，用以下方法收敛到与该用户相关的约 500 个候选：

- **协同过滤：** 与该用户看过相同作品的人还看了 Y -\> Y 成为候选
- **基于内容的过滤：** 用户看了很多犯罪惊悚片 -\> 其他犯罪惊悚片成为候选
- **热度信号：** 用户所在区域的热门作品始终是候选

这一阶段作为批处理作业周期性运行（每隔几小时一次），为每个用户产出一份候选集并存入 Redis。

**第 2 阶段 - 神经网络排序** 这 500 个候选由一个神经排序模型打分排序，模型综合考虑：

- 来自指标服务的用户题材偏好
- 一天中的时段（工作日晚上和周末下午，用户看的内容不一样）
- 设备类型（手机用户更偏好时长短的内容）
- 内容新鲜度（新上线的作品获得排序加权）
- 社交信号（在口味相近的用户群中受欢迎的作品）

排名前 20 的推荐结果按用户写入 Redis：

Key: recommendations:{user\_id} Value: \[content\_id\_1, content\_id\_2, ..., content\_id\_20\] TTL: 6 小时

下发一次推荐就是一次 Redis GET。整条机器学习流水线的复杂度，都藏在这一次缓存读取背后。

推荐流水线按计划运行，对大多数用户通常每 6 小时一次，对高活跃用户则更频繁。当用户看完一部作品（强信号事件）时，来自指标服务的 Kafka 事件流会立即触发一次推荐刷新，而不必等到下一次计划运行。

**推荐服务的瓶颈**

为每个用户在 15,000 个候选上运行神经排序模型，计算开销很大，同时为 1 亿用户跑一遍并不现实。应对办法是错峰计算：并非所有用户都需要在同一时刻拿到新鲜推荐。正在观看的用户不需要更新首页推荐。流水线会依据历史使用模式，优先为那些即将打开 App 的用户计算。

## 11\. 搜索服务

Netflix 的搜索覆盖作品标题、题材、演员、导演，以及「治愈系电影」「真实事件改编」这类描述性词语。

存储引擎 - Elasticsearch

Elasticsearch 负责在 15,000 份内容文档上做全文搜索和相关性排序。相比 Amazon 的 3.5 亿商品，这个目录很小，因此 Elasticsearch 集群的规模需求不高。

**内容文档：**

```json
{
  "content_id": "123",
  "title": "Stranger Things",
  "type": "series",
  "genres": ["sci-fi", "horror", "drama"],
  "cast": ["Millie Bobby Brown", "Finn Wolfhard"],
  "director": ["The Duffer Brothers"],
  "description": "When a young boy disappears...",
  "maturity_rating": "TV-14",
  "release_year": 2016,
  "total_watches": 4500000,
  "completion_rate": 0.82,
  "available_regions": ["US", "IN", "GB"]
}
```

**排序信号**

Netflix 的搜索排序把文本相关性与以下因素结合：

- **热度：** 总观看次数与完播率
- **个性化：** 用户常看的题材下的作品排得更靠前
- **区域可用性：** 在用户所在区域不可用的作品被完全排除
- **时效性：** 对宽泛的查询，新上线作品获得排序加权

**保持 Elasticsearch 同步**

搜索索引 Worker 从 content.transcoded topic 消费新作品，从 content.updated topic 消费元数据变更。热度信号（观看次数、完播率）则由指标服务的批处理流水线周期性同步，而非实时同步。

## 12\. 通知服务

架构与前面所有系统相同。上游服务经由 Kafka 扇出，iOS 用 APNs，Android 用 FCM，邮件走 SES。

通知服务订阅：

- content.released -\> 「《怪奇物语》新剧集已上线」推送通知
- session.completed -\> 「你看完了第 3 集，第 4 集已准备好」提示
- watchlist.available -\> 「你片单里的一部作品刚刚在你所在区域上线」
- recommendation.refresh -\> 周期性的「我们觉得你会喜欢这部」召回通知

Netflix 的通知经过严格节流。通知太多会导致用户卸载 App。通知服务会按用户做频次封顶：无论触发了多少条件，任何用户每天收到的推广类通知都不超过一条。

## 13\. 完整数据流总结

**内容接入流程：** 制片方 -\> AWS Direct Connect / 物理硬盘 -\> S3（原始母版）S3 -\> 预处理器（GOP 切分、DAG 生成、音频分离）DAG 调度器 -\> 任务队列（Kafka）资源管理器 -\> 任务 Worker（数千台 GPU EC2 实例）任务 Worker -\> S3（每个变体每个 GOP 的编码分片）所有分片完成 -\> Kafka（content.transcoded）-\> 元数据 Worker -\> Content 库 + Redis 缓存 -\> 搜索索引 Worker -\> Elasticsearch -\> MPD 生成 Worker -\> S3（按设备档位的清单文件）Open Connect 每晚的任务 -\> 从 S3 拉取热门内容 -\> 存入 ISP 内的 OCA **起播流程：** 用户点击播放 -\> API 网关 -\> 流媒体服务 -\> 用户资料服务（订阅校验）\[并行\] -\> 内容授权库（区域校验）\[并行\] -\> DRM 服务（设备认证校验）\[并行\] -\> 根据设备能力选择变体 -\> 生成带签名分片 URL 的个性化 MPD 清单 -\> 签发 DRM 授权 -\> 在 PostgreSQL 中创建 PlaybackSession 记录 -\> 把 MPD 返回给客户端 **分片分发流程：** 客户端 ABR 算法读取 MPD 客户端请求分片 -\> 用户所在 ISP 内的 Open Connect Appliance（缓存命中率约 95%）OCA 缓存未命中 -\> Netflix CDN / S3 源站 -\> OCA 缓存下来 -\> 提供分片 ABR 根据每个分片的带宽测量结果切换画质变体 **指标接入流程：** 客户端事件（播放、暂停、缓冲、画质切换）-\> Kafka -\> Apache Flink（实时）-\> Redis（实时看板、异常检测）-\> Apache Spark（批处理）-\> S3 数据湖（ML 训练、业务分析）-\> 按用户的信号聚合 -\> Redis（推荐输入） **推荐流程：** 指标服务 -\> Kafka -\> 推荐流水线 第 1 阶段（候选集生成，每 6 小时批量运行）-\> 协同过滤 + 基于内容的过滤 -\> 每个用户 500 个候选 第 2 阶段（神经排序）-\> 用 Redis 中的用户信号给 500 个候选打分排序 -\> 取前 20 -\> 写入 Redis recommendations:{user\_id}，TTL 6 小时 客户端打开 App -\> GET recommendations:{user\_id} -\> 一次 Redis 读取 **搜索流程：** 客户端搜索 "crime thriller" -\> 搜索服务 -\> Redis（热门查询缓存）-\> Elasticsearch（按用户题材偏好 + 热度做个性化排序）-\> Redis MGET（为结果补齐内容元数据） **继续观看流程：** 客户端每 30 秒发一次心跳 -\> 流媒体服务 -\> UPDATE PlaybackSessions SET last\_position\_ms = X WHERE session\_id = Y 用户回来 -\> 流媒体服务读取 PlaybackSessions -\> 生成从 last\_chunk\_index 开始的 MPD -\> 播放从精确位置恢复

## 14\. 韧性与容错

**Open Connect 的多 OCA 冗余**意味着，如果某个 ISP 内的一台 OCA 宕机，请求会回退到下一台最近的 OCA 或 Netflix 的 CDN 兜底。用户可能会经历短暂的画质下降，但播放不会中断。

**主动预热缓存**消除了所有重磅新片的冷启动问题。上线当天的惊群完全被 OCA 的缓存命中吸收掉，根本到不了 S3 源站。

**客户端侧的 ABR 算法韧性**意味着播放客户端能自动、持续地适应变差的网络状况，无需服务端参与。画质会平滑降级，但只要网络没有彻底断开，播放就不会卡顿缓冲。

转码流水线中的 **Kafka 持久化**意味着 Worker 崩溃不会丢失工作。任务会从 Kafka 重新入队，由另一个 Worker 接手。DAG 调度器跟踪已完成的任务，只把未完成的重新入队。

**基于 DAG 的转码并行化**意味着单个 Worker 故障只影响它正在处理的那些「GOP x 变体」任务，其他任务不受影响照常进行。资源管理器检测到故障 Worker 后，把它的任务释放回任务队列，重新分配给健康的 Worker。

**播放会话在 PostgreSQL 中的持久化**确保「继续观看」能挺过 App 崩溃、设备切换和后端重启。最后已知的播放位置始终是持久的。

**两阶段推荐流水线**意味着神经排序阶段失败时，可以回退到候选集生成的结果，而不是什么推荐都不显示。降级的推荐总好过空白的首页。

**每一层读路径上的 Redis 缓存**（订阅状态、内容元数据、推荐、片单）意味着在正常路径下大多数读操作根本不会碰到 PostgreSQL。PostgreSQL 故障会让体验逐步降级，而不是造成彻底不可用。

**指标流水线的独立性**意味着 Flink 或 Spark 流水线故障不会影响播放、搜索或任何面向用户的服务。指标是为了运维洞察而采集的，不在播放的关键路径上。

## 15\. 技术选型总结

**API 网关** -\> Kong / AWS API Gateway（认证、限流、区域路由、基于地理位置的访问控制）

**母版文件存储** \-\> AWS S3（制片方原始母版，永久存档，转码的源头）

**转码算力** \-\> 由 DAG 调度器 + Kafka 任务队列驱动的 GPU 加速 EC2 实例（数千个 Worker 并行执行 GOP x 变体编码）

**编码分片存储** -\> AWS S3（每部作品 864,000 个对象，确定性 URL 结构）

**内容分发** -\> 部署在 ISP 数据中心内的 Netflix Open Connect Appliance（主动预热缓存，热门内容零公共互联网穿越）

**流媒体协议** -\> 搭配 CMAF 分片的 MPEG-DASH（自适应码率，多编解码器，同时兼容 HLS 与 DASH 客户端）

**DRM** -\> Widevine + FairPlay + PlayReady（按会话的解密密钥，与设备绑定的授权）

**内容数据库** \-\> 分片 + 副本的 PostgreSQL（作品元数据、授权、区域可用性）

**用户资料数据库** -\> PostgreSQL（订阅状态、观看历史、片单、播放会话）

**播放会话存储** -\> PostgreSQL + Redis 缓存（继续观看、恢复位置、订阅权益缓存）

**搜索引擎** -\> Elasticsearch（全文搜索、个性化排序、题材与演员索引）

**推荐存储** \-\> Redis（预计算好的每用户前 20 名，首页加载时一次 GET）

**实时指标流水线** -\> Apache Kafka + Apache Flink（亚秒级事件处理、实时看板、异常检测）

**批处理指标流水线** -\> Apache Spark + S3 数据湖（ML 训练数据、业务分析、OCA 投放优化）

**用户信号缓存** -\> Redis（用于排序的每用户题材偏好、完播率、观看模式）

**通知投递** -\> APNs + FCM + AWS SES（带按用户频次封顶的推送通知与邮件）

以上就是全部内容，各位，干杯！！
