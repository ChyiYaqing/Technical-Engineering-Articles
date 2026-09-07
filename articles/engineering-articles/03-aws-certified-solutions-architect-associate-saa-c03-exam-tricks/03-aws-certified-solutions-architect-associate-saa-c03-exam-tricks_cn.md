---
title: "AWS Certified Solutions Architect - Associate SAA-C03 Exam Tricks"
url: "https://x.com/Harry_The_Nerd/status/2066793644783513735"
category: "Engineering Articles"
date: "2026-06-16"
description: "Tips and tricks for passing the AWS SAA-C03 exam."
lang: "zh-CN"
---

# AWS Certified Solutions Architect - Associate SAA-C03 考试技巧

> 通过 AWS SAA-C03 考试的技巧与窍门。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2066793644783513735](https://x.com/Harry_The_Nerd/status/2066793644783513735) · 2026-06-16

![封面图](https://pbs.twimg.com/media/HK39XC8awAA7aE9.jpg)

我第一次参加 Certified Solutions Architect-Associate（SAA-C03）考试就通过了，所以想写这篇文章帮帮其他人。我会列出这类考试中最常见的一些出题套路，以及每种情况下该选哪个答案。请认真备考，等你需要把所有概念一口气过一遍的时候，再拿这篇文章来复习。希望它能帮到大家。加油！

本文内容采用这样的格式 —— （题干中的关键词 —— \> 正确答案／服务）

限制某些国家的访问 -\> **CloudFront（Geo Restriction）**

防护 SQL 注入或跨站脚本／保护你的 Web 应用和 API -\> **AWS WAF**

DDoS 攻击 -\> **AWS Shield**

保留实例中的内容 -\> **EC2 的 Hibernate 模式**

在 AWS 网络内部从私有子网访问 DynamoDB 或 S3 -\> **VPC endpoint**

数据库凭据 -\> **AWS Secrets Manager**

防止误删 S3 中的对象 -\> **启用** **Versioning + MFA**

多个 EC2 实例共享文件系统／层级目录结构／Linux 文件服务器 -\> **EFS**

提升 RDS 读操作的性能 -\> **Read Replicas**

在存储系统中查询数据 -\> **S3 + Athena**

以低延迟处理 TCP 请求 -\> **Network Load Balancer（NLB）**

基于路径／主机名路由 HTTP/HTTPS 流量 -\> **Application Load Balancer（ALB）**

托管静态内容 -\> **S3 static hosting**

下载／上传超过 20GB 的数据 -\> **S3 Transfer Acceleration**

对象访问模式不可预测 -\> **S3 Intelligent tiering**

频繁访问且要求高性能 -\> **S3 standard**

访问不频繁但要求快速取回 -\> **S3 Standard-IA（如果题目没提高可用性且成本要低，则选 S3 One Zone-IA）**

长期保存 + 取回慢 -\> **S3 Glacier Deep Archive**

阻止对象被任何形式修改 -\> **S3 Object Lock**

无服务器、可扩展、弹性的 API 架构 -\> **API Gateway + Lambda**

SMB 文件存储 + 游戏应用（HPC 工作负载）-\> **Amazon FSx**

运行时间少于 15 分钟 + 内存小于 512MB -\> **AWS Lambda**

实时摄取流式数据（日志、IoT、点击流、事件数据）-\> **Kinesis Data Streams**

把流式数据投递到 S3、Redshift 等目的地 -\> **Kinesis Data Firehose**

用 SQL、Apache Flink 对流式数据做实时分析 -\> **Kinesis Data Analytics**

应用解耦／异步处理 -\> **SQS**

队列中出现重复消息 -\> **调大 VisibilityTimeout**

EC2 实例上 HPC 场景要求低延迟 -\> **Cluster Placement Group**

把 NFS 或 SMB 服务器从本地迁移到 AWS -\> **AWS DataSync，也可以用 AWS Storage Gateway（适用于混合云场景）**

加密传输中的数据 -\> **SSL/TLS 或客户端加密**

Linux 上的 HPC -\> **FSx for Lustre**

Windows 文件服务器 -\> **FSx for Windows**

EC2 实例需要低延迟 + 临时存储 -\> **Instance store**

面向全球用户的 TCP 连接／降低全球用户的 TCP 连接延迟 -\> **AWS Global Accelerator**

带 join 的复杂分析查询 -\> **Redshift**

ETL（抽取、转换、加载）作业／把 .csv 文件转成 parquet 格式 -\> **AWS Glue**

SQL 数据库需要跨多个 AWS 区域部署 -\> **Aurora global database**

AWS 同时管理数据密钥和主密钥 -\> **Server-side encryption with Amazon S3 Managed Keys（SSE-S3）**

AWS 管理数据密钥、客户管理主密钥 -\> **Server-side encryption with AWS KMS-managed keys（SSE-KMS）**

客户同时管理数据密钥和主密钥 → **Server-side encryption with customer-provided keys（SSE-C）**

要求毫秒级延迟的 EBS 卷 -\> **Provisioned IOPS SSD（io1）**

查出哪些用户挂载了 IAM 策略／查看配置信息 -\> **AWS Config**

限制只能从 CloudFront 访问 S3 -\> **Origin Access Identity（OAI）**

AWS Storage Gateway 的 file gateway -\> **NFS 或 SMB**

监控并保护组织免受网络攻击 -\> **AWS GuardDuty**

把 S3 数据从一个区域复制到另一个区域 -\> **S3 Cross-Region Replication**

预期的流量高峰／特定日期的可预测负载 -\> **EC2 scheduled scaling 或 predictive scaling**

设定扩缩指标和目标值（比如 CPU 40%）-\> **Target Tracking**

把流量从一个区域的资源导向另一个区域 -\> **Route 53 geoproximity**

根据用户位置转发流量 -\> **Route 53 Geolocation**

高可用性（几乎所有情况）-\> **Multi-AZ Deployment**

应用中的漏洞 -\> **AWS Inspector**

集中化的维护与治理体系 -\> **AWS Organizations**

让一个服务临时且安全地访问另一个服务 -\> **IAM Roles**

实例级别的变更 -\> **Security Groups**

子网级别的变更 -\> **NACLs**

低延迟 + 静态托管 -\> **CloudFront 和 S3 static hosting**

在不需要大带宽的前提下，快速建立低成本的本地到 AWS 连接 -\> **Site-to-Site VPN**

从本地数据中心到 AWS 的专用私有链路 -\> **AWS Direct Connect**

大规模数据迁移（10 TB 到 PB 级）-\> **AWS Snowball**

传输 EB 级（100+ PB）数据 -\> **AWS Snowmobile**

高性能 NoSQL 数据库／数据库 schema 不确定／基于键值 -\> **AWS DynamoDB**

有关联关系的结构化数据、事务、符合 ACID -\> **AWS RDS（如果提到高性能则选 AWS Aurora）**

灵活的 JSON 文档存储、无 schema 设计（MongoDB）-\> **AWS DocumentDB**

图像和视频分析（人脸检测、物体识别、OCR）-\> **Amazon Rekognition**

用自然语言处理（NLP）做情感分析、关键短语抽取（分析文本里写了什么）-\> **Amazon Comprehend**

文本转语音 -\> **Amazon Polly**

从文档中提取文本和数据（OCR + 表格抽取）-\> **Amazon Textract**

就这些了，各位……加油！祝考试顺利！
