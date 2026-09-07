---
title: "Design Google Drive"
url: "https://x.com/Harry_The_Nerd/status/2088619948113641872"
category: "HLD"
date: "2026-08-15"
description: "System design of a cloud storage service like Google Drive"
lang: "zh-CN"
---

# 设计 Google Drive

> 类似 Google Drive 的云存储服务的系统设计。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2088619948113641872](https://x.com/Harry_The_Nerd/status/2088619948113641872) · 2026-08-15

![封面图](https://pbs.twimg.com/media/HPnyaD0bYAA2Czg.jpg)

## 1\. 需求

**功能性需求**

- 文件上传与下载（任意文件类型：文档、图片、视频、音频、PDF、压缩包）
- 文件夹层级与组织
- 文件和文件夹的分享，支持基于角色的访问控制（查看者、评论者、编辑者、所有者）
- 文件版本管理（查看并恢复历史版本）
- 搜索文件和文件夹（按名称和按内容）
- 通知（分享提醒、文件更新通知）
- 回收站与恢复已删除文件
- 每个用户的存储配额管理

**不在范围内**

- Google Docs 的实时协同编辑（这是另一个系统，之前已经设计过）
- Google Photos（独立产品）
- 第三方应用集成
- 团队云端硬盘与共享云端硬盘（企业版功能）
- 离线同步客户端

**非功能性需求**

- 文件上传和下载要具备高可用性
- 上传可续传，中断的上传必须能接着传而不用从头再来
- 不能丢数据，每个上传的文件在硬件故障下都必须保持持久
- 文件元数据和权限要保证强一致性
- 搜索索引和通知可以接受最终一致性
- 存储去重，避免同一份文件被重复存多次
- 存储分层，旧版本自动迁移到更便宜的冷存储
- 权限变更必须立即生效，回收的访问权限要在几秒内失效

## 2\. 容量估算

- **日活用户（DAU）：** 1000 万
- **每个用户每天上传文件数：** 2 个
- **平均文件大小：** 500 KB
- **每日原始存储量：** 1000 万 x 2 x 500 KB = 10 TB/天
- **按 40% 去重率计算后：** 60% x 10 TB = 每天实际新增 6 TB
- **读写比：** 10:1，文件上传一次，之后会被多台设备和多个共享用户反复访问
- **存储分层：** 当前版本放 S3 Standard，超过 30 天的版本放 S3 Infrequent Access，超过 90 天的版本放 S3 Glacier
- **每个用户的配额：** 免费 15 GB，更多容量需付费

这个设计最大的挑战是三个维度上的正确性：能扛住网络故障的可续传上传、既省存储又不会在用户之间泄露信息的去重，以及权限被回收时能立即生效的权限校验。

## 3\. API 网关

所有客户端请求都从 API 网关进入，由网关负责认证、限流和路由。任何文件操作到达下游服务之前，API 网关都会先对请求做认证。

和前面的系统有一个重要区别：API 网关处理元数据操作（创建文件夹、重命名文件、分享文件），但绝不直接处理二进制文件负载数据。小文件上传走预签名 URL 模式，大文件上传走可续传的分片模式。API 网关只负责编排会话创建、跟踪上传状态，从不接触原始字节。

## 4\. 上传服务与块服务

小文件上传（小于 5 MB）

对小文件，采用标准的预签名 S3 URL 模式：

客户端通过 API 网关向上传服务发送 POST /api/v1/files/upload，带上文件元数据（名称、大小、MIME 类型）

上传服务生成一个预签名 S3 URL

客户端把文件直接上传到 S3

S3 向 Kafka 发出上传事件

文件组装服务接收该事件并触发去重

大文件上传（超过 5 MB）—— 可续传上传

大文件在客户端侧先切成分片再上传。这样就支持可续传：传输中断后可以从最后一个成功的分片继续。

**第 1 步 —— 创建上传会话**

POST /api/v1/files/upload?uploadType=resumable Body: { file\_name, file\_size, mime\_type, parent\_folder\_id }

上传服务创建一条上传会话记录并返回 session\_id：

UploadSessions session\_id UUID，主键 user\_id UUID，外键 -\> Users file\_name VARCHAR file\_size BIGINT mime\_type VARCHAR parent\_folder\_id UUID total\_chunks INTEGER chunk\_size INTEGER（默认 5 MB） status ENUM('in\_progress', 'completed', 'expired') expires\_at TIMESTAMP（自最后一次活动起 24 小时） created\_at TIMESTAMP

已上传分片的跟踪信息放在 Redis 里，便于快速读写：

Key：upload:{session\_id}:chunks Value：已完成分片索引的 SET TTL：24 小时

**第 2 步 —— 上传分片** 客户端把文件切成 5 MB 的分片，各自独立上传：

PUT /api/v1/files/upload/{session\_id}/chunk/{chunk\_index} Content-Range: bytes 0-5242879/10737418240 Body: 分片的原始字节

块服务收到每个分片后：

用该文件专属的加密密钥以 AES-256 加密这个分片

把加密后的分片上传到 S3 的路径：uploads/{session\_id}/{chunk\_index}

记录完成状态：SADD upload:{session\_id}:chunks {chunk\_index}

向客户端返回成功

**第 3 步 —— 中断后续传** 如果上传中断，客户端查询上传状态：

GET /api/v1/files/upload/{session\_id}/status Response: { completed\_chunks: \[0, 1, 2, 4\], total\_chunks: 10 }

客户端识别出缺失的分片（3、5、6、7、8、9），从第一个缺失的分片开始续传。已经传好的分片不需要重传。

**第 4 步 —— 完成上传** 所有分片都上传完之后，客户端发送：

POST /api/v1/files/upload/{session\_id}/complete

文件组装服务接收该请求，调用 S3 Multipart Upload Complete API 把分片拼接成最终文件。临时的分片对象被合并成一个 S3 对象。上传会话被标记为完成。

**加密**

每个文件落盘都是加密的。块服务使用**信封加密**：

- 每个文件生成一个唯一的数据加密密钥（DEK）
- 文件分片用这个 DEK 以 AES-256 加密
- DEK 本身再用 AWS KMS 托管的主密钥加密密钥（KEK）加密
- 数据库里只存加密后的 DEK，绝不存明文 DEK

文件被下载时，下载服务通过 KMS 解出 DEK，再用它解密文件分片，然后把内容返回给客户端。

**上传流水线的瓶颈**

块服务是主要瓶颈，因为每个文件的每个分片都要由它加密，而加密是 CPU 密集型的。缓解办法是基于 CPU 利用率通过 Kubernetes 自动扩缩容水平扩展块服务实例。Redis 里的上传会话状态读写很快，不构成问题。S3 分片上传原生支持分片拼接，也不会带来额外瓶颈。

## 5\. 去重服务

去重服务避免同一份文件内容在 S3 里被存多份，在这个规模下能省下可观的存储成本。

基于内容的去重：SHA-256 哈希

当文件组装服务完成分片拼接后：

去重服务对完整的组装文件计算 SHA-256 哈希

查询 S3Objects 表：

```sql
SELECT file_id, s3_url, reference_count FROM S3Objects
   WHERE content_hash = 'abc123...'
```

如果哈希已存在：说明这是重复文件。创建一条新的文件元数据记录指向已有的 S3 对象，把 reference\_count 加一，并从 S3 删掉临时组装出来的文件（内容早就存过了）。

如果哈希不存在：说明是新文件。把它从临时上传路径移到永久路径，并创建一条 reference\_count = 1 的 S3Objects 记录。

S3Objects 表（PostgreSQL）

S3Objects content\_hash VARCHAR(64)，主键（SHA-256 十六进制） s3\_url TEXT（永久 S3 路径） file\_size BIGINT reference\_count INTEGER created\_at TIMESTAMP

用引用计数保证安全删除

reference\_count 记录有多少个用户文件指向这个 S3 对象：

- 用户 A 上传文件 X：reference\_count = 1
- 用户 B 上传同一个文件 X（被去重）：reference\_count = 2
- 用户 A 删除自己的文件：reference\_count = 1（S3 对象保留）
- 用户 B 删除自己的文件：reference\_count = 0（可以安全删除 S3 对象）

引用计数的更新放在 PostgreSQL 事务里执行，避免竞态：

```sql
BEGIN;
UPDATE S3Objects SET reference_count = reference_count - 1
WHERE content_hash = 'abc123...'
RETURNING reference_count;
-- If reference_count = 0, publish s3.object.orphaned to Kafka
COMMIT;
```

S3 清理 Worker 消费 s3.object.orphaned 事件，删除真正的 S3 对象。

**安全与隐私**

去重绝不能在用户之间泄露信息。用户 B 不能因为自己上传同一个文件时瞬间完成，就推断出用户 A 也有这个文件。去重对用户完全透明。每个用户在自己的云端硬盘里看到的都是自己的文件，看不出底层存储被共用了。

**去重服务的瓶颈**

对大文件（比如 10 GB 的视频）做 SHA-256 哈希是 CPU 密集型的，要花好几秒。这一步在上传完成后异步执行，不会阻塞用户。哈希计算还可以分布式化：对每个分片独立算哈希，再把分片哈希合并成最终的文件哈希（Merkle 树的思路），这样能大幅缩短单线程哈希的耗时。

## 6\. 文件元数据服务

文件元数据服务管理文件和文件夹的所有元数据：名称、类型、大小、在文件夹层级中的位置，以及归属关系。

数据模型

**Files 表（PostgreSQL）**

Files file\_id UUID，主键 owner\_id UUID，外键 -\> Users file\_name VARCHAR file\_extension VARCHAR mime\_type VARCHAR file\_size BIGINT content\_hash VARCHAR(64)，外键 -\> S3Objects s3\_url TEXT is\_folder BOOLEAN is\_starred BOOLEAN is\_trashed BOOLEAN trashed\_at TIMESTAMP（未进回收站时为 null） version INTEGER（当前版本号） created\_at TIMESTAMP modified\_at TIMESTAMP last\_modified\_by UUID，外键 -\> Users

**文件夹层级 —— 多父节点邻接表**

Google Drive 用多父节点邻接表来建模文件夹层级。文件和文件夹存在同一张 Files 表里（用 is\_folder 区分）。父子关系存在另一张关联表里，从而支持同一个文件通过快捷方式同时出现在多个文件夹中：

**FileParents 表（PostgreSQL）**

FileParents file\_id UUID，外键 -\> Files parent\_id UUID，外键 -\> Files is\_shortcut BOOLEAN（为 true 表示这是快捷方式，不是原始位置） PRIMARY KEY (file\_id, parent\_id)

关键查询：

**列出文件夹 X 的全部内容（只往下一层）：**

```sql
SELECT f.* FROM Files f
JOIN FileParents fp ON f.file_id = fp.file_id
WHERE fp.parent_id = 'folder_x'
AND f.is_trashed = false
ORDER BY f.is_folder DESC, f.file_name ASC
```

**给出文件 X 的完整路径（面包屑导航）：**

递归查找：文件 -\> 父节点 -\> 父节点 -\> 父节点 -\> 根节点 通常只有 5-6 层深，有索引的查找很快

邻接表的权衡对 Google Drive 来说是可以接受的，原因是：

- 只往下一层的查询（列出文件夹内容）占了绝大多数查询
- 完整路径重建（面包屑）在实际中只需要往上走 5-6 层
- 移动文件只是一条 UPDATE FileParents SET parent\_id = new\_folder 的操作

缓存策略

热门文件夹（「我的云端硬盘」根目录、经常访问的项目文件夹）以较短的 TTL 缓存在 Redis 里。最近访问过的文件元数据按用户维度缓存。文件重命名、移动或删除时，通过 Kafka 事件让缓存失效。

文件元数据服务的瓶颈

对于大文件夹（几千个文件）的列表查询，如果索引没建好会很慢。在 (parent\_id, is\_trashed, is\_folder, file\_name) 上建复合索引可以高效覆盖最常见的列表查询。文件元数据服务是读多写少的，热门目录的 Redis 缓存能带来显著收益。

## 7\. 权限服务

权限服务管理所有文件和文件夹的基于角色的访问控制，是整个系统中安全性最关键的服务。

权限模型

FilePermissions file\_id UUID，外键 -\> Files user\_id UUID，外键 -\> Users（链接分享时为 null） permission ENUM('owner', 'editor', 'commenter', 'viewer') share\_type ENUM('specific\_user', 'link\_anyone', 'link\_org') created\_at TIMESTAMP granted\_by UUID，外键 -\> Users PRIMARY KEY (file\_id, user\_id)

**权限继承**

Google Drive 的权限沿文件夹层级向下继承。如果用户 A 对某个文件夹有编辑者权限，那他对该文件夹下的所有文件和子文件夹也都有编辑者权限。权限继承在查询时解析：沿着 FileParents 树向上走，直到找到一条权限记录为止。

继承解析的结果按「用户 + 文件」维度缓存在 Redis 里，避免反复遍历树：

Key：perm:{file\_id}:{user\_id} Value：ENUM('owner', 'editor', 'commenter', 'viewer', 'none') TTL：15 分钟

**权限回收**

当文件所有者回收某个用户的访问权限时：

从 PostgreSQL 的 FilePermissions 表里删除权限记录

删除 Redis 中的权限缓存：DEL perm:{file\_id}:{user\_id}

向 Kafka 发布 permission.revoked 事件

下载服务作废该用户对该文件的所有有效签名 URL

用户在几秒内失去访问权限

Google Drive 的签名 URL 之所以 TTL 很短（15-30 分钟），正是因为权限随时可能被回收。长 TTL 的签名 URL 在权限被回收之后依然能用，直到过期为止。

权限服务的瓶颈

每一次文件访问、下载和元数据操作都要做权限检查。Redis 权限缓存吸收了其中绝大部分。对层级很深的文件夹来说，继承解析的向上遍历是瓶颈；缓存解析后的最终权限（而不是原始的继承链）可以缓解这一点。

## 8\. 下载服务

下载服务负责生成文件访问用的签名 URL，同时处理小文件下载和大文件流式传输。

下载流程

客户端请求文件：GET /api/v1/files/{file\_id}/download

下载服务通过 Redis 缓存做权限检查

若有权限：从文件元数据服务取得文件元数据（S3 URL、加密密钥引用）

为该文件生成一个短时效的签名 CDN URL（TTL 15-30 分钟）

把签名 URL 返回给客户端

客户端用签名 URL 直接从 CDN 拉取文件

**CDN 层**

所有文件都通过位于 S3 前面的 CDN（CloudFront 或 Akamai）分发。热门的共享文件（比如一次产品发布时分享给一万名同事的 PDF）会缓存在 CDN 边缘节点上。第一次下载触发从 S3 回源填充缓存，之后所有下载都由 CDN 边缘节点直接提供，不再打到 S3。

对于大文件（视频、大型压缩包），CDN 支持**字节范围请求**，允许客户端只取文件的某一部分。这带来了：

- 视频拖动进度条时无需下载整个文件
- 可续传下载，中断后从断点继续
- 并行分片下载，提高传输速度

**小文件与大文件的差别处理**

- **小文件（小于 10 MB）：** 一次 CDN 请求，一个响应返回完整内容
- **大文件（超过 10 MB）：** 客户端发起字节范围请求，CDN 顺序或并行地返回各个分片

块服务按分片加密的做法天然契合字节范围请求。每个分片在上传时是独立加密的，所以也可以独立解密。

**下载服务的瓶颈**

CDN 吸收了绝大部分下载压力。下载服务本身只做两件事：生成签名 URL 和检查权限，都是亚毫秒级操作。真正的瓶颈是新上传的文件被同时分享给成千上万用户时的 CDN 回源填充，可以通过对预期高流量的文件提前预热 CDN 来缓解。

## 9\. 版本历史服务

文件被修改并重新上传时，Google Drive 会保留历史版本。用户可以查看并恢复任意历史版本。

数据模型

**FileVersions 表（PostgreSQL）**

FileVersions version\_id UUID，主键 file\_id UUID，外键 -\> Files version\_number INTEGER content\_hash VARCHAR(64)，外键 -\> S3Objects s3\_url TEXT file\_size BIGINT created\_at TIMESTAMP created\_by UUID，外键 -\> Users comment VARCHAR（可选的版本标签）

每次文件被重新上传，都会创建一条新的 FileVersions 记录指向新的 S3 对象，之前的版本记录仍然保留。Files 表的 version 列指向当前版本号。

**旧版本的存储分层**

版本存储采用分层策略来控制成本：

- **当前版本：** S3 Standard（访问快，全价）
- **最近 30 天内的版本：** S3 Infrequent Access（略慢，便宜 40%）
- **超过 30 天的版本：** S3 Glacier（取回需要几分钟，便宜 80%）

一个后台的**版本分层 Worker** 每晚运行一次，按版本的年龄把符合条件的版本迁移到更便宜的存储层级。实际的存储类别转换由 S3 对象生命周期策略自动完成。

**版本恢复**

用户恢复某个历史版本时：

版本历史服务从 FileVersions 表取出目标版本记录

如果该版本在 Glacier 里：发起取回请求（需要 1-5 分钟）

取回完成后：创建一条指向旧内容的新版本记录（不会删除任何已有版本）

把 Files 表的 version 更新为新的版本号

向 Kafka 发布 file.version\_restored 事件，用于搜索重新索引和通知

**去重与版本管理的相互作用**

如果用户上传了一个文件，改了内容，然后又改回原样重新上传，这次重传得到的 SHA-256 哈希和最初那份一样。去重机制会识别出这一点，新版本直接指向和原始版本相同的 S3 对象，不会产生重复存储。该 S3 对象的引用计数加一，以记录这个新增的版本引用。

## 10\. 配额服务

每个 Google Drive 用户都有存储配额。配额服务负责跟踪用量并强制执行限制。

配额跟踪

**UserQuotas 表（PostgreSQL）**

UserQuotas user\_id UUID，主键，外键 -\> Users quota\_bytes BIGINT（免费 15 GB = 16,106,127,360 字节） used\_bytes BIGINT last\_updated TIMESTAMP

used\_bytes 是反范式化的计数器，文件上传或删除时由 Kafka 消费者异步更新。它反映的是该用户名下所有文件当前版本的大小之和，与去重节省了多少无关。

配额校验

每次上传之前都要检查配额：

从 Redis 取 quota:{user\_id} 如果 used\_bytes + new\_file\_size \> quota\_bytes：以 507 Insufficient Storage 拒绝上传

配额余量按用户维度缓存在 Redis 里，TTL 较短，便于快速检查：

Key：quota:{user\_id} Value：{ quota\_bytes, used\_bytes } TTL：5 分钟

配额与去重

不论是否发生去重，每个用户的配额都按自己的文件计算：

- 用户 A 上传 500 MB 的文件：用户 A 的 used\_bytes += 500 MB
- 用户 B 上传同样的 500 MB 文件（被去重）：用户 B 的 used\_bytes += 500 MB
- Google 内部省下了这 500 MB 的 S3 存储成本，但两个用户各自被计费，是公平的

配额与删除

用户删除文件时，文件先进入回收站。配额不会立刻释放，要等到文件被永久删除（用户清空回收站，或 30 天后自动清理）：

文件被永久删除 -\> Kafka（file.permanently\_deleted）

配额服务消费者：UPDATE UserQuotas SET used\_bytes = used\_bytes - file\_size WHERE user\_id = X

Redis 配额缓存失效：DEL quota:{user\_id}

S3 对象的引用计数减一

## 11\. 搜索服务

Google Drive 的搜索覆盖文件名、文件夹名，以及所支持文件类型的全文内容。

**文本抽取流水线**

文件要能按内容被搜到，先得把文本抽出来。这一步在上传之后异步进行：

file.assembled 事件 -\> Kafka -\> 文本抽取 Worker -\> 根据 MIME 类型判断文件类型 -\> 调用对应的解析器：PDF -\> 用 Apache PDFBox 抽取文本 DOCX -\> 用 Apache POI 从 XML 中抽取文本 XLSX -\> 用 Apache POI 抽取单元格内容 图片 -\> 用 Tesseract 做 OCR（让白板照片也能被搜到） TXT -\> 直接索引，无需抽取 ZIP -\> 解压并递归解析内部内容 -\> 向 Kafka 发布 file.text\_extracted -\> 搜索索引 Worker 索引进 Elasticsearch

抽取出来的文本只索引进 Elasticsearch，不会永久存进数据库。如果搜索索引需要重建，就从 S3 里的原始文件重新跑一遍文本抽取。

Elasticsearch 文档模型

```json
{
  "file_id": "123",
  "owner_id": "456",
  "file_name": "Q3 Product Roadmap.pdf",
  "file_type": "application/pdf",
  "file_extension": "pdf",
  "file_size_bytes": 2048000,
  "content_text": "This document outlines the product strategy for Q3 2026...",
  "parent_folder_id": "789",
  "shared_with": ["user_101", "user_202", "user_303"],
  "is_starred": false,
  "is_trashed": false,
  "created_at": "2026-06-01T10:00:00Z",
  "modified_at": "2026-08-10T14:30:00Z",
  "last_modified_by": "user_456"
}
```

过滤器支持

云端硬盘里常见的过滤条件都能映射成 Elasticsearch 的结构化查询：

**与我共享的文件：**

{ "filter": { "term": { "shared\_with": "current\_user\_id" } } }

**上周修改过的文件：**

{ "filter": { "range": { "modified\_at": { "gte": "now-7d" } } } }

**大于 10 MB 的文件：**

{ "filter": { "range": { "file\_size\_bytes": { "gte": 10485760 } } } }

**PDF 类型的文件：**

{ "filter": { "term": { "file\_extension": "pdf" } } }

**带过滤条件的组合搜索：**

```json
{
  "query": {
    "bool": {
      "must": [
        { "match": { "content_text": "product strategy" } }
      ],
      "filter": [
        { "term": { "shared_with": "current_user_id" } },
        { "term": { "file_extension": "pdf" } },
        { "range": { "modified_at": { "gte": "now-7d" } } },
        { "term": { "is_trashed": false } }
      ]
    }
  }
}
```

安全过滤

每一条搜索查询都会自动带上一个安全过滤条件：

```json
{
  "filter": {
    "bool": {
      "should": [
        { "term": { "owner_id": "current_user_id" } },
        { "term": { "shared_with": "current_user_id" } }
      ]
    }
  }
}
```

用户绝不可能在搜索结果里看到自己无权访问的文件，哪怕他知道确切的文件名。Elasticsearch 文档中的 shared\_with 数组会在权限变更时由 Kafka 消费者更新。

保持 Elasticsearch 同步

搜索索引 Worker 消费多个 Kafka 主题：

- file.assembled -\> 上传后建立初始索引
- file.text\_extracted -\> 用抽取出的内容更新文档
- file.renamed -\> 更新 file\_name
- file.moved -\> 更新 parent\_folder\_id
- file.shared -\> 更新 shared\_with 数组
- file.trashed -\> 更新 is\_trashed 标志
- file.permanently\_deleted -\> 从索引中删除

## 12\. 通知服务

架构和前面所有系统一致。上游服务通过 Kafka 扇出，iOS 走 APNs，Android 走 FCM，邮件走 SES。

通知服务订阅：

- file.shared -\> 「Harry 与你共享了一个文件」邮件 + 推送通知
- file.permission\_changed -\> 「你对某个文件的访问权限发生了变化」通知
- file.comment\_added -\> 向文件所有者和被提及的用户发送评论通知
- file.version\_restored -\> 「某个文件已恢复到之前的版本」通知
- quota.warning -\> 用量超过 80% 和 95% 时发送「你的存储空间快用完了」邮件
- upload.completed -\> 大文件上传完成时发送「上传已完成」通知

## 13\. 完整数据流总览

小文件上传流程： 客户端 -\> API 网关（认证） -\> 上传服务 -\> 预签名 S3 URL 客户端 -\> S3（直传） S3 -\> Kafka（file.uploaded） -\> 去重服务： -\> 计算 SHA-256 哈希 -\> 查 S3Objects 表（重复还是新文件） -\> 重复则 INCR reference\_count -\> 新文件则创建新的 S3Objects 记录 -\> 文件元数据服务：创建 Files 记录 -\> 配额服务：UPDATE used\_bytes -\> 文本抽取 Worker -\> Kafka（file.text\_extracted） -\> 搜索索引 Worker -\> Elasticsearch 大文件上传流程（可续传）： 客户端 -\> POST /upload?uploadType=resumable -\> 上传服务 -\> session\_id 客户端 -\> PUT /upload/{session\_id}/chunk/{index}（每个分片重复一次） -\> 块服务：加密分片 -\> S3（uploads/{session\_id}/{index}） -\> Redis SADD upload:{session\_id}:chunks {index} 客户端 -\> POST /upload/{session\_id}/complete -\> 文件组装服务：S3 Multipart Complete -\> 去重服务（与小文件流程相同） 上传中断后的续传流程： 客户端重连 -\> GET /upload/{session\_id}/status -\> Redis SMEMBERS upload:{session\_id}:chunks -\> 已完成分片列表 -\> 客户端从第一个缺失的分片索引处继续上传 下载流程： 客户端 -\> GET /files/{file\_id}/download -\> 下载服务 -\> Redis 权限检查（perm:{file\_id}:{user\_id}） -\> 取出文件元数据 + S3 URL -\> 生成短时效的签名 CDN URL（TTL 15-30 分钟） -\> 客户端从 CDN 拉取（热门共享文件的缓存命中率约 90%） -\> CDN 未命中 -\> S3 -\> CDN 缓存 -\> 返回文件 权限回收流程： 所有者回收访问权限 -\> 权限服务 -\> 从 FilePermissions 删除（PostgreSQL） -\> DEL perm:{file\_id}:{user\_id}（Redis） -\> Kafka（permission.revoked） -\> 下载服务作废有效的签名 URL -\> 用户在几秒内失去访问权限 文件删除流程： 用户删除文件 -\> 文件元数据服务 -\> is\_trashed = true 用户清空回收站（或 30 天自动清理） -\> 触发硬删除 -\> 配额服务：used\_bytes 减少 -\> S3Objects：在 PostgreSQL 事务里把 reference\_count 减一 -\> 若 reference\_count = 0：Kafka（s3.object.orphaned） -\> S3 清理 Worker 删除 S3 对象 -\> 搜索索引 Worker：从 Elasticsearch 索引中删除 版本历史流程： 用户重新上传文件 -\> 创建新的 FileVersions 记录 -\> 保留之前的版本 -\> Files.version 加一 版本分层 Worker（每晚）： -\> 把超过 30 天的版本迁到 S3 Infrequent Access -\> 把超过 90 天的版本迁到 S3 Glacier 用户恢复版本： -\> 若在 Glacier：发起取回（1 到 5 分钟） -\> 创建指向旧内容的新版本记录 -\> 更新 Files.version 搜索流程： 客户端搜索「product strategy pdf」 -\> 搜索服务 -\> Elasticsearch bool 查询： must：match content\_text filter：owner 或 shared\_with、file\_extension、is\_trashed = false 安全过滤：owner\_id 或 shared\_with = 当前用户 -\> Redis MGET（为结果填充文件元数据）

## 14\. 韧性与容错

**基于 Redis 分片跟踪的可续传上传**确保网络故障时不会丢失上传进度。客户端始终清楚哪些分片已经成功，并从第一个失败处继续。上传会话在 24 小时无活动后过期，避免留下孤儿会话状态。

**带引用计数的内容寻址去重**既避免存储浪费，又避免过早删除 S3 对象。PostgreSQL 事务保证引用计数的更新是原子的。两个并发的删除操作不可能都把计数减到零、然后都去删同一个 S3 对象。

**短时效签名 URL（TTL 15 到 30 分钟）**保证即使显式的回收事件有延迟，权限回收也能在几分钟内生效。被回收权限的用户手上的签名 URL 最多在回收后 30 分钟就失效。

所有流水线上的 **Kafka 持久化**意味着即使下游服务崩溃，也不会丢失文件事件。文本抽取、搜索索引、配额更新和通知投递在恢复后都能从 Kafka 的偏移量重试。

**用 S3 Glacier 做存储分层**在不删除旧版本的前提下降低了长期存储成本。从 Glacier 取回要几分钟，但对查看版本历史来说是可以接受的。

所有文件下载的 **CDN 缓存**吸收了绝大部分下载流量。热门共享文件在首次回源填充之后，就完全由 CDN 边缘节点提供，不再回到 S3 源站。

**基于 AWS KMS 的信封加密**保证文件落盘加密，且每个文件有独立密钥。就算 S3 存储桶被攻破，没有加密密钥也读不出内容，而这些密钥单独由 KMS 托管。

**用 PostgreSQL 事务做引用计数**防止并发删除文件时出现竞态。S3 清理 Worker 只有在收到 Kafka 事件、确认 reference\_count 在一个已提交的事务里降到零之后，才会删除 S3 对象。

**短 TTL 的 Redis 配额缓存**让配额检查既快又足够准确。5 分钟的 TTL 意味着理论上用户在竞态条件下可能稍微超出配额一点，但 Kafka 消费者会在几分钟内把真实的 used\_bytes 对齐回来。

## 15\. 技术选型总结

**API 网关** -\> Kong / AWS API Gateway（认证、限流、元数据操作的路由）

**块服务** -\> 自研加密服务 + AWS S3 Multipart Upload（按文件密钥的 AES-256 加密，5 MB 分片上传）

**上传会话存储** -\> Redis（分片完成情况跟踪、可续传上传状态、24 小时 TTL）

**文件内容存储** -\> AWS S3（加密后的文件分片、内容寻址路径、用于分层的生命周期策略）

**CDN** -\> CloudFront / Akamai（所有文件下载的边缘缓存，支持字节范围请求）

**存储分层** -\> S3 Standard + S3 Infrequent Access + S3 Glacier（通过 S3 对象生命周期策略自动完成）

**去重** \-\> SHA-256 哈希 + 带引用计数的 PostgreSQL S3Objects 表

**文件元数据库** -\> PostgreSQL 分片 + 副本（Files、FileParents、FileVersions、FilePermissions，强一致性）

**文件夹层级** -\> FileParents 表中的多父节点邻接表（用多条父节点记录支持快捷方式）

**权限缓存** \-\> Redis，15 分钟 TTL + 基于 Kafka 的失效机制（亚毫秒级权限检查）

**配额存储** -\> PostgreSQL UserQuotas + 5 分钟 TTL 的 Redis 缓存（快速配额检查，计数器异步更新）

**加密密钥管理** -\> 采用信封加密的 AWS KMS（每个文件的 DEK 由主 KEK 加密）

**去重库** -\> PostgreSQL S3Objects 表（content\_hash 作主键、reference\_count、原子事务）

**搜索引擎** -\> Elasticsearch（文件名搜索、全文内容搜索、丰富的过滤支持）

**文本抽取** -\> Apache PDFBox + Apache POI + Tesseract OCR（通过 Kafka 的异步流水线）

**解耦** -\> Apache Kafka（上传事件、去重触发、权限变更、配额更新、搜索索引）

**版本分层 Worker** -\> 后台任务 + S3 生命周期策略（每晚把旧版本迁移到更便宜的存储）

**S3 清理 Worker** -\> Kafka 消费者（reference\_count 归零时删除孤儿 S3 对象）

**通知投递** -\> APNs + FCM + AWS SES（分享通知、配额告警、上传完成提醒）

就到这里啦……干杯！！

点赞、评论、分享、转发！！
