---
title: "AWS Certified Solutions Architect - Associate SAA-C03 Exam Tricks"
url: "https://x.com/Harry_The_Nerd/status/2066793644783513735"
category: "Engineering Articles"
date: "2026-06-16"
description: "Tips and tricks for passing the AWS SAA-C03 exam."
---

# AWS Certified Solutions Architect - Associate SAA-C03 Exam Tricks

> Tips and tricks for passing the AWS SAA-C03 exam.
>
> 原文：[https://x.com/Harry_The_Nerd/status/2066793644783513735](https://x.com/Harry_The_Nerd/status/2066793644783513735) · 2026-06-16

![Cover image](https://pbs.twimg.com/media/HK39XC8awAA7aE9.jpg)

I cleared my Certified Solutions Architect-Associate (SAA-C03) exam on the first attempt, so I thought I could help others with this article where I will list some of the most common patterns in these exams and what option must be chosen in every case. Prepare well for the exam, and use this article when you have to revise all the concepts in one go. I hope it’ll help you all. Cheers!

The content in this article is written in this format — (Keyword in the question — \> Correct Answer/Service)

Restrict access to some countries -\> **CloudFront ( Geo Restriction )**

Protection against SQL injection or cross-site scripting/ Protect your web app & API -\> **AWS WAF**

DDOS attacks -\> **AWS Shield**

Preserve the contents of an instance -\> **Hibernate mode in EC2**

Accessing DynamoDB or S3 from a private subnet within an AWS network -\> **VPC endpoint**

Credentials in a database -\> **AWS Secrets Manager**

Preventing accidental deletion of objects in S3 -\> **Enable** **Versioning + MFA**

File Systems in multiple EC2 instances/Hierarchical directory structure/ Linux File Server-\> **EFS**

Increase performance for read operations in RDS -\> **Read Replicas**

Querying data in a storage system -\> **S3 + Athena**

Handle requests with low latency for TCP -\> **Network Load Balancer(NLB)**

Path/Host based routing with HTTP/HTTPS Traffic -\> **Application Load Balancer (ALB)**

Hosting Static Content -\> **S3 static hosting**

Download/Upload data greater than 20GB -\> **S3 Transfer Acceleration**

Unpredictable object access pattern -\> **S3 Intelligent tiering**

Frequent Access and high performance -\> **S3 standard**

Infrequent access but fast retrieval -\> **S3 Standard -IA (S3 One Zone-IA if high availability isn’t mentioned + cost is low)**

Preserve for a long time + slow retrieval -\> **S3 Glacier Deep Archive**

Prevention of any type of modifications of objects -\> **S3 Object Lock**

Serverless, Scalable, and elastic architecture for API -\> **API Gateway + Lambda**

SMB file store + Gaming App (HPC workload) -\> **Amazon FSx**

Work time is less than 15 mins + less than 512MB of memory -\> **AWS lambda**

Real-time ingestion of streaming data (logs, IoT, clickstream, event data) -\> **Kinesis Data Streams**

Delivering streaming data to destinations like S3, Redshift -\> **Kinesis Data Firehose**

Real-time analyticson streaming data using SQL, Apache Flink -\> **Kinesis Data Analytics**

Decoupling apps/Asynchronous working -\> **SQS**

Duplicate messages in the queue -\> **Increase VisibilityTimeout**

Low Latency for HPC in EC2 instances -\> **Cluster Placement Group**

Migration of NFS or SMB server from on-prem to AWS -\> **AWS DataSync but you can also use AWS Storage Gateway (for hybrid cloud use cases)**

Encrypt data during transit -\> **SSL/TLS or Client Side Encryption**

HPC for Linux-\> **FSx for Lustre**

Windows File Server -\> **FSx for Windows**

Low Latency+Temporary storage for EC2 instances -\> **Instance store**

Global Audience with TCP-based connection/ Reduce latency for global users with TCP-based connection -\> **AWS Global Accelerator**

Complicated analytical queries using joins -\> **Redshift**

ETL (extract, transform, and load) jobs / Convert .csv files to parquet format-\> **AWS Glue**

SQL Database to be deployed across multiple AWS regions -\> **Aurora global database**

AWS will manage both data key & master key -\> **Server-side encryption with Amazon S3 Managed Keys (SSE-S3)**

AWS will manage the data key & the customer will manage the master key -\> **Server-side encryption with AWS KMS-manged keys ( SSE-KMS )**

The customer will manage both data key & master key → **Server-side encryption with customer-provided keys (SSE-C)**

EBS volume for millisecond latency -\> **Provisioned IOPS SSD ( io1)**

Identify users who have attached IAM policies/ Info about configurations -\> **AWS Config**

Restrict access to S3 from CloudFront -\> **Origin Access Identity ( OAI )**

AWS Storage gateway file gateway -\> **NFS or SMB**

Monitor and safeguard organizations from cyberattacks -\> **AWS GuardDuty**

Replicate S3 data from one region to another region -\> **S3 Cross-Region Replication**

Expected spike in traffic/ predictable load for certain days -\> **EC2 scheduled scaling or predictive scaling**

Scaling metric and set a target value ( like CPU 40% ) -\> **Target Tracking**

Redirect traffic from one region resources to another -\> **Route 53 geoproximity**

Redirect traffic based on location of user -\> **Route 53 Geolocation**

High availability (in almost all cases) -\> **Multi-AZ Deployment**

Vulnerability in your application -\> **AWS Inspector**

Centralized maintenance & governance system -\> **AWS Organizations**

Temporary ability for one service to access another service securely -\> **IAM Roles**

Instance level changes -\> **Security Groups**

Subnet level changes -\> **NACLs**

Low latency + static hosting -\> **Cloudfront and S3 static hosting**

Establish low-cost, immediate on-premises to aws connection without large bandwidth-\> **Site-to-Site VPN**

Dedicated private link to AWS from on-premises data center -\> **AWS Direct Connect**

Large-scale data migrations (10 TB — petabytes) -\> **AWS Snowball**

Transferring exabytes (100+ PB) of data -\> **AWS Snowmobile**

No SQL database with high performance/ Unclear about the database schema/Key Value based-\> **AWS DynamoDB**

Structured data with relationships, transactions, ACID compliance-\> **AWS RDS ( AWS Aurora if high performance is mentioned)**

Flexible JSON document storage, schema-less design (MongoDB) -\> **AWS DocumentDB**

Image & video analysis (face detection, object recognition, OCR) -\> **Amazon Rekognition**

Natural language processing (NLP) for sentiment analysis, key phrase extraction (what’s written in the text) -\> **Amazon Comprehend**

Text-to-speech conversion -\> **Amazon Polly**

Extract text & data from documents (OCR + table extraction) -\> **Amazon Textract**

That's all, folks..Cheers! All the best for the exam!
