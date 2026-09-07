---
title: "Design Leetcode"
url: "https://x.com/Harry_The_Nerd/status/2083591314067927258"
category: "HLD"
date: "2026-08-01"
description: "System design for a programming platform like Leetcode"
lang: "zh-CN"
---

# 设计 Leetcode

> 类似 Leetcode 的编程平台的系统设计
>
> 原文：[https://x.com/Harry_The_Nerd/status/2083591314067927258](https://x.com/Harry_The_Nerd/status/2083591314067927258) · 2026-08-01

![封面图](https://pbs.twimg.com/media/HOZkJvvbEAArI73.jpg)

## 1\. 需求

**功能性需求**

- 题库目录（列表、按难度/分类/公司标签筛选、分页）
- 支持多语言的代码编辑器
- 用可见测试用例运行代码（Run Code）
- 代码提交，并用隐藏测试用例判题
- 竞赛管理（限时提交、实时排行榜）
- 用户主页（提交历史、已解题目、评分、连续打卡、竞赛历史）
- 每道题的讨论区（评论与题解）

**不在范围内**

- 会员订阅管理
- 模拟面试
- 学习计划
- 视频讲解
- 公司专属题单
- Judge0 内部实现（当作黑盒处理）

**非功能性需求**

- 执行任意代码时要有强隔离，用户代码绝不能影响宿主基础设施
- Run Code 低延迟，1-2 秒内返回结果
- Submit 延迟可接受，2-5 秒内出判题结论
- 竞赛期间排行榜实时更新
- 题库目录和编辑器高可用性，读多且对延迟敏感
- 承受竞赛期间约每秒 170 次提交的写入尖峰
- 提交处理恰好一次，同一次提交绝不能被判题两次

## 2\. 容量估算

- **日活用户（DAU）：** 500 万
- **在线题目数：** 4000
- **提交量：** 约 20% 的 DAU 至少提交一次 = 每天 100 万次提交
- **Run Code 量：** 约为提交量的 5 倍 = 每天 500 万次运行
- **单次提交的代码体积：** 约 10 KB
- **每天的提交存储：** 100 万 x 10 KB = 每天约 10 GB
- **测试用例：** 平均每题 50 个测试用例 x 4000 题 = S3 中 20 万个测试用例文件
- **竞赛参与人数：** 每场大型竞赛约 5 万并发用户
- **竞赛提交速率：** 5 万用户每 5 分钟提交一次 = 峰值约每秒 170 次提交
- **排行榜规模：** Redis 有序集合中每场竞赛 5 万条记录

系统层面最主要的挑战是代码执行隔离和竞赛写入尖峰。存储量相比媒体类平台并不算大。最难的问题是安全地运行不受信任的任意代码，并近乎实时地返回判题结论。

## 3\. API 网关

所有客户端请求都经过 API 网关，由它处理认证、限流和路由。LeetCode 的 API 网关在竞赛期间还有一项重要职责：**竞赛专属限流**。用户不应该能靠刷提交来干扰判题队列。按用户维度限制「每道题每 30 秒只能提交一次」，既能防止队列被淹没，又不影响正常使用。

## 4\. 用户主页服务

负责管理用户身份、提交历史、已解题目追踪、竞赛参与历史和评分。

数据模型

**Users 表（PostgreSQL）**

Users user\_id UUID，主键 username VARCHAR，唯一 email VARCHAR，唯一 password\_hash VARCHAR profile\_pic\_url TEXT rating INTEGER（初始 1500，Elo 风格）max\_rating INTEGER problems\_solved INTEGER easy\_solved INTEGER medium\_solved INTEGER hard\_solved INTEGER submission\_count INTEGER acceptance\_rate FLOAT streak\_days INTEGER last\_active TIMESTAMP created\_at TIMESTAMP

problems\_solved、easy\_solved、medium\_solved、hard\_solved、submission\_count 和 acceptance\_rate 都是反范式的计数器，判题完成后由 Kafka 消费者异步更新。如果每次加载主页都去扫描 Submissions 表统计已解题目数，会太慢。

**ContestHistory 表（PostgreSQL）**

ContestHistory history\_id UUID，主键 user\_id UUID，外键 -\> Users contest\_id UUID，外键 -\> Contests rank INTEGER problems\_solved INTEGER total\_penalty INTEGER rating\_change INTEGER attended\_at TIMESTAMP

**SolvedProblems 表（PostgreSQL）**

SolvedProblems user\_id UUID，外键 -\> Users problem\_id UUID，外键 -\> Problems solved\_at TIMESTAMP language VARCHAR PRIMARY KEY (user\_id, problem\_id)

(user\_id, problem\_id) 上的唯一主键保证一道题只会被标记为已解一次，无论用户对这题有多少次通过的提交。

**缓存策略**

用户主页数据缓存在 Redis 中，TTL 较短。主页是 LeetCode 上访问最频繁的页面之一。每个用户的已解题目集合也会缓存，因为每次加载题目列表页都要检查它，好在做过的题上打绿色对勾。

## 5\. 题库目录服务

题库目录服务是整个系统中读压力最大的服务。用户不停地浏览题目（筛选、排序、搜索），频率远高于提交代码。

数据模型

**Problems 表（PostgreSQL）**

Problems problem\_id UUID，主键 problem\_number INTEGER，唯一 title VARCHAR slug VARCHAR，唯一 description TEXT difficulty ENUM('easy', 'medium', 'hard') acceptance\_rate FLOAT submission\_count INTEGER accepted\_count INTEGER is\_premium BOOLEAN is\_active BOOLEAN created\_at TIMESTAMP

**ProblemTopics 表（PostgreSQL）**

ProblemTopics problem\_id UUID，外键 -\> Problems topic ENUM('array', 'string', 'dp', 'graph', 'tree', 'binary\_search', ...) PRIMARY KEY (problem\_id, topic)

**ProblemCompanies 表（PostgreSQL）**

ProblemCompanies problem\_id UUID，外键 -\> Problems company VARCHAR frequency ENUM('very\_high', 'high', 'medium', 'low') PRIMARY KEY (problem\_id, company)

**TestCases 表（S3 + PostgreSQL 引用）**

TestCases testcase\_id UUID，主键 problem\_id UUID，外键 -\> Problems input\_s3\_url TEXT output\_s3\_url TEXT is\_visible BOOLEAN（为 true 表示展示给用户的示例用例）order\_index INTEGER

测试用例内容以文件形式存放在 S3。TestCases 表只存引用（S3 URL），不存实际内容。大输入（比如 10 万个节点的图论题）可能有好几 MB，把它们当作文本存进 PostgreSQL 会毫无必要地撑爆数据库。

**分页与筛选**

题目列表请求的形式如下：

GET /v1/problems?page=1&limit=50&difficulty=medium&topic=dp&company=google&status=unsolved

(difficulty, is\_active, is\_premium) 上的复合索引覆盖了最常见的筛选组合。话题和公司筛选走关联表，外键上建有索引。

对于分页后的题目列表，整个结果集都以较短 TTL 缓存在 Redis 中，因为题目列表很少变化（新题添加得不频繁）：

Key: problems:list:{filter\_hash}:{page} Value: 序列化后的题目元数据列表 TTL: 10 分钟

filter\_hash 是筛选参数的哈希值，不同的筛选组合会落到不同的缓存键上。

**测试用例缓存**

热门题目（每日一题、竞赛题、NeetCode 150、Blind 75）的测试用例会预加载到 Redis：

Key: testcases:{problem\_id} Value: \[{input: "...", expected\_output: "..."}, ...\] TTL: 热门题 24 小时，冷门题 1 小时

缓存未命中时，判题 Worker 从 S3 拉取测试用例并回填 Redis。同一道题的后续提交就直接命中 Redis。冷门题目（很少有人做的偏门难题）按需从 S3 拉取。

**目录服务的瓶颈**

题目列表读多写少，且极少变化。在列表级别做 Redis 缓存能吸收绝大部分目录读流量。主要顾虑是题目更新（修正描述、新增测试用例）时的缓存失效，一条 problem.updated Kafka 事件会触发所有包含该题的列表缓存页失效。

## 6\. 代码执行流水线

这是我们设计过的九个系统里最独特的服务。用户提交的代码必须在共享基础设施上被安全、正确、快速地执行。

安全隔离栈

在自己的服务器上运行任意用户代码，是一个系统能做的最危险的操作之一。每次执行都要套上多层安全栈：

**第 1 层 - 非 root 进程：** 代码在容器内以非特权用户身份运行，没有 sudo 权限。

**第 2 层 - seccomp 系统调用过滤：** 这是 Linux 内核特性，用来限制进程能发起哪些系统调用。危险的系统调用在内核层就被拦掉：

- 屏蔽 fork（防止 fork 炸弹）
- 屏蔽 socket（在内核层阻断网络连接）
- 屏蔽 exec（防止派生子进程）
- 屏蔽 ptrace（防止调试其他进程）

**第 3 层 - 无网络接口：** 容器完全没有网络连通性，出站入站都没有。代码调不到外部 API，也没法把数据传出去。

**第 4 层 - CPU 和内存硬限制：**

\--memory="256m" 最多 256MB 内存 --cpus="0.5" 最多占用半个 CPU 核

如果提交的代码超出限制，容器会被杀掉，返回 Memory Limit Exceeded。

**第 5 层 - 墙钟超时：** 一个硬性的执行时间上限（视题目而定，通常 2-5 秒）。死循环会被杀掉，返回 Time Limit Exceeded。

**第 6 层 - 只读文件系统：** 容器文件系统只读，只有一个 /tmp 临时目录可写，且有容量上限。代码写不了系统目录，也没法把磁盘占满。

**第 7 层 - gVisor 沙箱（可选，最高安全级别）：** Google 的 gVisor 是一个沙箱化的容器运行时，它在用户态拦截所有系统调用，而不是透传给宿主内核。这样能彻底消除容器逃逸风险，达到最高安全级别。

两条独立的流水线：Run Code vs Submit

Run Code 和 Submit 的需求根本不同，不能共用同一条流水线。

**Run Code 流水线（轻量、交互式）：**

- 用户在编辑器里点「Run」时触发
- 只跑 3-5 个可见的示例测试用例
- 测试用例硬编码在 Redis 的题目元数据缓存里，不需要访问 S3
- 用户期望 1-2 秒内出结果
- 独立的 Kafka 主题：[code.run](https://x.com/Harry_The_Nerd/status/code.run)
- 专用的 Run Worker，资源限制更宽松、容器启动更快
- 结果通过 WebSocket 返回

**Submit 流水线（重量级、权威判定）：**

- 用户点「Submit」时触发
- 跑所有隐藏测试用例（每题 50-100 个以上）
- 测试用例从 Redis 缓存（热门题）或 S3（冷门题）获取
- 用户期望 2-5 秒内出结果
- 独立的 Kafka 主题：code.submitted
- 专用的判题 Worker，套完整安全栈
- 完整判题结论（Accepted、Wrong Answer、TLE、MLE、Runtime Error）+ 运行时长（ms）+ 内存占用（KB）
- 结果通过 WebSocket 返回

**完整的 Submit 流程**

**第 1 步 - 接收提交** 用户点击 Submit。客户端通过 API 网关向代码服务发送 POST /submissions，请求体为 { user\_id, problem\_id, language, source\_code, contest\_id（若在竞赛中） }。

**第 2 步 - 服务端校验** 代码服务校验：

- 语言是否受支持
- 源码大小是否在限制内（最大 50 KB）
- 若在竞赛中：提交时间戳是否落在竞赛时间窗内（服务端校验，不是客户端）
- 用户是否被限流（每道题每 30 秒一次提交）

**第 3 步 - 持久化提交记录** 在 Submissions 表中以 PENDING 状态创建提交记录，提交 ID 立即返回给客户端。

**第 4 步 - 发布 Kafka 事件** 向 Kafka 发布 code.submitted 事件，携带提交 ID、题目 ID、语言和源码的 S3 URL。

**第 5 步 - 建立 WebSocket 连接** 客户端建立 WebSocket 连接，通过 Redis Pub/Sub 订阅 submission:{submission\_id} 频道。用户在等待时看到加载动画。

**第 6 步 - 判题 Worker 接单** 判题 Worker 消费 Kafka 事件，然后拉取：

- 从 S3 拉源码
- 从 Redis（缓存命中）或 S3（缓存未命中，随后回填 Redis）拉测试用例

**第 7 步 - 启动容器** 判题 Worker 启动一个对应语言的 Docker 容器，并套上完整安全栈。容器镜像是预热好的（不是每次提交都重新拉取），以尽量降低启动延迟。

**第 8 步 - 编译** 对于编译型语言（C++、Java），先编译源码。编译出错则立即终止，不跑任何测试用例，返回 Compile Error。

**第 9 步 - 执行测试用例** 测试用例顺序执行。一旦首次失败（Wrong Answer、TLE、MLE、Runtime Error），执行就停止，并报告失败的那个用例。全部通过则判定为 Accepted。

**第 10 步 - 销毁容器** 执行完成后立即销毁容器。两次提交之间不残留任何状态。

**第 11 步 - 发布结果** 判题 Worker 向 Kafka 发布 submission.judged 事件，携带完整判题详情。

**第 12 步 - 结果送达用户** Kafka 消费者把结果发布到 Redis Pub/Sub 频道 submission:{submission\_id}。订阅了该频道的 WebSocket 服务器收到结果，推送到用户已建立的 WebSocket 连接上。加载动画消失，判题结论出现。

**第 13 步 - 异步更新数据库** Kafka 消费者把最终结论写入 Submissions 表。如果是 Accepted，则发布 problem.solved 事件，触发：

- 用户主页计数器更新（problems\_solved、acceptance\_rate）
- 向 SolvedProblems 表插入记录
- 若在竞赛中：更新排行榜

**WebSocket 多服务器问题**

WebSocket 连接是有状态的。连在 WebSocket 服务器 1 上的用户，收不到服务器 3 推来的消息。解法是 Redis Pub/Sub：

判题 Worker -\> Kafka（submission.judged）Kafka 消费者 -\> Redis PUBLISH submission:{submission\_id} {verdict} WebSocket 服务器（已订阅 submission:{submission\_id}）-\> 推送到用户连接

每台 WebSocket 服务器都会订阅本机上所有活跃提交对应的 Redis Pub/Sub 频道。判题结论一进入 Redis Pub/Sub，正确的那台服务器实例就会收到并推给用户。不需要任何服务器之间的直接通信。

Submissions 表（PostgreSQL）

Submissions submission\_id UUID，主键 user\_id UUID，外键 -\> Users problem\_id UUID，外键 -\> Problems contest\_id UUID（不在竞赛中则为 null）language VARCHAR source\_code\_url TEXT（S3 URL）status ENUM('pending', 'running', 'accepted', 'wrong\_answer', 'time\_limit\_exceeded', 'memory\_limit\_exceeded', 'runtime\_error', 'compile\_error') runtime\_ms INTEGER memory\_kb INTEGER test\_cases\_passed INTEGER total\_test\_cases INTEGER submitted\_at TIMESTAMP judged\_at TIMESTAMP

源码存在 S3，不存数据库。把代码当文本存进 PostgreSQL 会撑大数据库并拖慢它。Submissions 表只存 S3 引用 URL。

**代码执行流水线的瓶颈**

容器启动延迟是首要瓶颈。从零拉一个 Docker 镜像要花好几秒。缓解办法是在每个判题 Worker 节点上预热一批各语言的容器池。提交到来时立即使用一个预热好的容器，同时在后台再启动一个补上空缺。容器启动延迟就从秒级降到了毫秒级。

第二个瓶颈是竞赛期间判题 Worker 的自动扩缩容。每秒 170 次提交时，Worker 池必须水平扩展。Kafka 把提交速率和处理速率解耦——提交先在 Kafka 里排队，Worker 按自己的节奏处理。Kubernetes 的水平 Pod 自动扩缩容根据 Kafka 消费滞后量增加判题 Worker Pod。

## 7\. 竞赛服务

竞赛服务负责管理竞赛生命周期、题单和实时排行榜。

数据模型

**Contests 表（PostgreSQL）**

Contests contest\_id UUID，主键 title VARCHAR start\_time TIMESTAMP end\_time TIMESTAMP duration\_mins INTEGER status ENUM('upcoming', 'active', 'ended') participant\_count INTEGER created\_at TIMESTAMP

**ContestProblems 表（PostgreSQL）**

ContestProblems contest\_id UUID，外键 -\> Contests problem\_id UUID，外键 -\> Problems position INTEGER（竞赛中的第 1、2、3、4 题）points INTEGER PRIMARY KEY (contest\_id, problem\_id)

**ContestParticipants 表（PostgreSQL）**

ContestParticipants contest\_id UUID，外键 -\> Contests user\_id UUID，外键 -\> Users registered\_at TIMESTAMP final\_rank INTEGER（竞赛结束前为 null）PRIMARY KEY (contest\_id, user\_id)

**竞赛开始 - 应对读尖峰**

竞赛开始时，5 万用户同时请求竞赛题目。这是一次协同式读尖峰，类似 Netflix 新一季上线的场景。

缓解手段是主动预热缓存，而不是被动扩容：

竞赛开始前 30 分钟，竞赛服务把所有竞赛题目及其可见测试用例预加载到 Redis

竞赛开始时 5 万用户全部打到 Redis，尖峰完全不会传到 PostgreSQL

竞赛前根据报名人数提前扩容额外的 API 服务器 Pod

Key: contest:{contest\_id}:problems Value: 完整题目数据，含描述和可见测试用例 TTL: 竞赛时长 + 1 小时

用 Redis 有序集合实现实时排行榜

竞赛排行榜由 Redis 有序集合驱动：

Key: leaderboard:{contest\_id} Member: user\_id Score: (problems\_solved \* 10000) - total\_penalty\_minutes

把解题数乘以 10000，保证解出更多题的人一定排在「题少但更快」的人前面。罚时分钟数（每次错误提交罚 5 分钟）在同一档位内做减法。

**分数操作：**

ZINCRBY leaderboard:contest\_123 10000 user\_A（解出一题）ZINCRBY leaderboard:contest\_123 -5 user\_A（答案错误罚时）ZREVRANGE leaderboard:contest\_123 0 49（前 50 名用户）ZREVRANK leaderboard:contest\_123 user\_A（用户当前排名）

所有操作都是 O(log N)，即使有 5 万参赛者也极快。

竞赛期间的提交流程

当用户 A 在竞赛开始 47 分钟时解出第 3 题：

代码服务收到带 contest\_id 的提交

服务端校验：submitted\_at <\= contest.end\_time（在服务端强制执行，不是客户端）

判题 Worker 照常处理提交

判定为 Accepted 后：向 Kafka 发布带 contest\_id 的 submission.judged 事件

竞赛排行榜消费者接到事件：检查该用户是否已经解出过这道题（在 Redis 中做幂等性检查）如果尚未解出：ZINCRBY leaderboard:contest\_123 10000 user\_A 记录这次通过之前累积的错误提交罚时 ZINCRBY leaderboard:contest\_123 -{penalty\_minutes} user\_A

用户通过 WebSocket 看到第 3 题上出现绿色对勾

排行榜更新在几秒内对所有参赛者可见

**错误提交罚时追踪**

Key: contest:{contest\_id}:penalties:{user\_id}:{problem\_id} Value: wrong\_answer\_count TTL: 竞赛时长 + 1 小时

当一次提交被判为 Wrong Answer 时：INCR contest:123:penalties:user\_A:problem\_3

当这道题最终被解出时，把罚时分钟数 = wrong\_answer\_count \* 5 应用到排行榜分数上。

**竞赛结束流程**

竞赛结束时：

发布 contest.ended 事件到 Kafka

竞赛服务在 PostgreSQL 中把竞赛状态更新为 ended

从 Redis 中取最终排行榜快照，持久化到 PostgreSQL 的 ContestHistory 表

**评分计算 Worker** 根据最终排名和赛前评分，为所有参赛者计算新的 Elo 风格评分

在 PostgreSQL 的 Users 表中更新用户评分

Redis 排行榜键的 TTL 设为 7 天（供查看历史成绩），到期后淘汰

通知服务向所有参赛者发送竞赛结果摘要

用户主页服务更新竞赛历史、徽章和评分变化展示

**竞赛服务的瓶颈**

**排行榜的 Redis 有序集合**能轻松扛住每秒 170 次分数更新，因为每次 ZINCRBY 都是 O(log N)。竞赛开始时的读尖峰靠主动预热缓存解决。竞赛结束后的评分计算是批处理作业，它那几分钟的延迟可以接受。主要风险是竞赛期间 Redis 的可用性——如果 Redis 在比赛中途挂了，排行榜就用不了了。缓解办法是用 Redis Sentinel 或带自动故障转移的 Redis Cluster。

## 8\. 讨论服务

每道题都有一个讨论区，用户在里面发题解、提问和评论。这在架构上和 Reddit 的评论系统很像。

数据模型

**Posts 表（PostgreSQL）**

Posts post\_id UUID，主键 problem\_id UUID，外键 -\> Problems user\_id UUID，外键 -\> Users title VARCHAR content TEXT post\_type ENUM('solution', 'question', 'discussion') language VARCHAR（非题解帖为 null）upvotes INTEGER view\_count INTEGER created\_at TIMESTAMP

**Comments 表（PostgreSQL）**

Comments comment\_id UUID，主键 post\_id UUID，外键 -\> Posts parent\_id UUID，外键 -\> Comments（顶层评论为 null）user\_id UUID，外键 -\> Users content TEXT upvotes INTEGER created\_at TIMESTAMP

嵌套评论采用和 Reddit 相同的邻接表（Adjacency List）模式。parent\_id = NULL 表示帖子下的顶层评论，parent\_id 非空表示对另一条评论的回复。配合深度限制的懒加载可以避免无边界的递归拉取。

热门题目的热帖和高赞评论会以较短 TTL 缓存在 Redis。每日一题的讨论区流量极大，从缓存中获益尤其明显。

## 9\. 完整数据流总结

题库目录读取流程：客户端 -\> API 网关 -\> 目录服务 -\> Redis（以筛选哈希为键的分页列表缓存）-\> PostgreSQL（缓存未命中时用复合索引查询）-\> Redis MGET（填充题目元数据）-\> Redis（按用户取已解题目，用于绿色对勾）Run Code 流程：客户端点 Run -\> 代码服务 -\> 校验输入（大小、语言、限流）-\> 发布到 Kafka（[code.run](https://x.com/Harry_The_Nerd/status/code.run) 主题）-\> 建立 WebSocket 连接（订阅 run:{run\_id} Redis 频道）-\> Run Worker 接单 -\> 启动轻量容器 -\> 从 Redis 元数据缓存取可见测试用例（硬编码，不走 S3）-\> 用 3-5 个可见测试用例执行代码 -\> 销毁容器 -\> 把结果发布到 Redis Pub/Sub（run:{run\_id}）-\> WebSocket 服务器把结果推给用户（总共 1-2 秒）Submit 流程：客户端点 Submit -\> 代码服务 -\> 校验输入 + 限流检查 + 竞赛时间检查（服务端）-\> 在 PostgreSQL 中创建 Submission 记录（PENDING）-\> 上传源码到 S3 -\> 发布到 Kafka（code.submitted 主题）-\> 建立 WebSocket 连接（订阅 submission:{submission\_id} Redis 频道）-\> 判题 Worker 接单 -\> 从 S3 取源码 -\> 从 Redis（热门）或 S3（冷门，随后回填 Redis）取测试用例 -\> 启动对应语言的容器并套上完整安全栈 -\> 编译（若为编译型语言）-\> 失败则 Compile Error -\> 顺序运行所有测试用例 -\> 首次失败即停止 -\> 销毁容器 -\> 向 Kafka 发布 submission.judged -\> Kafka 消费者 -\> Redis PUBLISH submission:{submission\_id} {verdict} -\> WebSocket 服务器把判题结论推给用户（总共 2-5 秒）-\> 异步 Kafka 消费者： -\> 更新 Submissions 表（PostgreSQL） -\> 若 Accepted：插入 SolvedProblems，更新用户计数器 -\> 若为竞赛提交：更新 Redis 中的排行榜有序集合 竞赛流程：开始前 30 分钟 -\> 竞赛服务把所有竞赛题目预热进 Redis 缓存 竞赛开始 -\> 5 万用户打到 Redis（数据库零压力）用户提交 -\> 代码服务（带 contest\_id）-\> 走同一条 Submit 流水线 判定 Accepted -\> 竞赛排行榜消费者 -\> 幂等性检查（这道题是否已解出？）-\> ZINCRBY leaderboard:{contest\_id} 10000 user\_id -\> 应用罚时分钟：ZINCRBY leaderboard:{contest\_id} -{penalties} user\_id -\> 用户通过 WebSocket 看到绿色对勾 竞赛结束 -\> contest.ended 事件 -\> Kafka -\> 从 Redis 取排行榜快照 -\> 写入 PostgreSQL ContestHistory -\> 评分计算 Worker -\> 更新用户评分 -\> 通知服务 -\> 向所有参赛者发送结果邮件 -\> Redis 排行榜 TTL 设为 7 天 讨论流程：客户端加载题目讨论区 -\> 讨论服务 -\> Redis（热门题目的热帖缓存）-\> PostgreSQL（top posts WHERE problem\_id = X ORDER BY upvotes DESC LIMIT 20）-\> 懒加载评论（先顶层，回复按需拉取）

## 10\. 韧性与容错

**Kafka 提交队列**吸收竞赛写入尖峰。每秒 170 次提交时，队列深度在峰值期间增长，判题 Worker 按自己的节奏处理。即使 Worker 暂时过载，也不会丢失任何提交。Kubernetes 自动扩缩容根据 Kafka 消费滞后量增加判题 Worker Pod。

**判题 Worker 节点上的预热容器池**消除了容器启动延迟。每个 Worker 节点上按语言维护一批空闲容器，用掉的容器在后台销毁并补充。

**用 Redis Pub/Sub 投递判题结论**确保结果能到达正确的 WebSocket 服务器，无论用户连在哪一台上。从判题结论路由的角度看，WebSocket 服务器是无状态的。

**主动预热竞赛缓存**消除了竞赛开始时的读尖峰。5 万用户同时打到 Redis，数据库毫无压力。

**竞赛计分的幂等性检查**保证一次通过的提交不会被重复计分，即使 Kafka 消费者重试了该事件也一样。一旦某道题在某场竞赛中被标记为该用户已解，同一事件的后续处理就是空操作。

**seccomp + 无网络 + 资源限制**为代码执行提供纵深防御。即使某一层隔离被绕过，其他层仍能防止宿主系统受损。

**PostgreSQL 只读副本**用于题库目录和用户主页，保证读多的浏览体验高可用。主节点故障不影响题库目录的读取。

**带自动故障转移的 Redis Cluster** 用于竞赛排行榜，保证即使某个 Redis 节点在比赛中途故障，排行榜仍然可用。

## 11\. 技术选型总结

**API 网关** -\> Kong / AWS API Gateway（认证、限流、按用户的提交限流）

**题库目录数据库** -\> PostgreSQL 分片 + 副本（题目元数据、用于筛选的复合索引）

**题目列表缓存** -\> Redis（分页筛选列表，短 TTL，题目更新时失效）

**测试用例存储** -\> AWS S3（隐藏测试用例文件，由判题 Worker 拉取）

**测试用例缓存** -\> Redis（热门题用例预加载，冷门题首次拉取后缓存）

**提交数据库** -\> PostgreSQL（提交记录、判题历史、竞赛提交追踪）

**源码存储** -\> AWS S3（提交的源码文件，在 Submissions 表中以 S3 URL 引用）

**Run Code 队列** -\> Apache Kafka（[code.run](https://x.com/Harry_The_Nerd/status/code.run) 主题，独立的轻量流水线）

**Submit 队列** \-\> Apache Kafka（code.submitted 主题，完整判题流水线）

**Run Worker** -\> 带轻量安全栈的 Docker 容器（只跑可见测试用例，1-2 秒延迟）

**判题 Worker** \-\> 带完整 seccomp + gVisor 安全栈的 Docker 容器（跑所有隐藏测试用例，2-5 秒延迟）

**容器编排** -\> Kubernetes，基于 Kafka 消费滞后量做水平 Pod 自动扩缩容

**判题结论投递** -\> Redis Pub/Sub + WebSocket（跨多服务器把结论路由到正确的客户端连接）

**竞赛排行榜** -\> Redis 有序集合（ZINCRBY 实时更新分数，ZREVRANGE 取排名）

**罚时追踪** -\> Redis 计数器，按用户 x 题目 x 竞赛维度（错误提交次数用于计算罚时）

**竞赛题目缓存** -\> Redis（竞赛开始前 30 分钟预热，吸收 5 万并发读）

**用户主页缓存** -\> Redis（已解题目、评分、主页计数器）

**讨论数据库** \-\> PostgreSQL 邻接表（嵌套评论，按深度懒加载）

**评分计算** -\> 由 contest.ended Kafka 事件触发的批处理 Worker（Elo 风格评分更新）

**通知投递** -\> APNs + FCM + AWS SES（判题结论通知、竞赛结果、评价提示）

以上就是全部内容，各位……干杯！
