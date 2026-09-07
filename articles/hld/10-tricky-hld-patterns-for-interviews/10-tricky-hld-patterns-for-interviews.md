---
title: "Tricky HLD Patterns for interviews"
url: "https://x.com/Harry_The_Nerd/status/2056739488114856226"
category: "HLD"
date: "2026-05-19"
description: "Commonly missed high-level design patterns for interviews."
---

# Tricky HLD Patterns for interviews

> Commonly missed high-level design patterns for interviews.
>
> 原文：[https://x.com/Harry_The_Nerd/status/2056739488114856226](https://x.com/Harry_The_Nerd/status/2056739488114856226) · 2026-05-19

![Cover image](https://pbs.twimg.com/media/HIr8CK1awAAcpqq.jpg)

## 7 Important Backend & Distributed System Design Patterns

Modern backend systems are no longer built as single monolithic applications running on one machine. Today’s applications serve millions of users, process enormous amounts of data, and must remain reliable even during failures, traffic spikes, and network instability. To achieve this level of scalability and resilience, engineers rely on proven architectural and distributed system patterns. These patterns are used daily in companies like Netflix, Amazon, Uber, Google, and Meta.

This article explores seven essential backend and distributed system patterns:

**Write-Behind (Write-Back) Cache**

**Thundering Herd Problem**

**Distributed Transactions & Two-Phase Commit (2PC)**

**Materialized View Pattern**

**Bulk Processing Pattern**

**Read Replica Architecture**

**Global Rate Limiting**

## 1\. Write-Behind (Write-Back) Cache

**Overview**

The Write-Behind Cache pattern improves write performance by temporarily storing writes in a fast cache layer before asynchronously persisting them to the database.

So, instead of every write operation directly hitting the database, the application writes data into a cache such as Redis or Memcached. A background worker later flushes those changes to the database in batches.

This reduces database write pressure and improves latency.

Work flow:

Client sends a write request.

Data is written into the cache immediately.

User receives a success response quickly.

Background workers periodically persist data into the database.

Databases are often the bottleneck in high-write systems. Writing every request synchronously can increase latency and overload the database.

Write-behind caching solves this by:

- Reducing database calls
- Batching writes together
- Improving response times
- Handling traffic bursts more efficiently

**Real-World Example**

Social media platforms use this technique for metrics such as:

- Likes
- View counts
- Post impressions
- Analytics counters

For example, Instagram does not synchronously update the database every time someone views a story. Instead, counters are accumulated in memory and flushed periodically.

**Advantages**

- Extremely fast write performance
- Reduced database load
- Better throughput
- Efficient batching

**Drawbacks**

- Risk of data loss if cache crashes before persistence
- Eventual consistency instead of immediate consistency
- Requires retry and recovery mechanisms

**Best Use Cases**

- Analytics systems
- Counters and metrics
- Logging systems
- High-frequency event ingestion

## 2\. Thundering Herd Problem

## Overview

The Thundering Herd Problem occurs when many clients simultaneously request the same resource after a cache miss or service recovery. Instead of one request rebuilding the cache, thousands of requests hit the backend at the same time, overwhelming the database or service.

**Example Scenario**

Imagine an e-commerce application where product details are cached for 30 minutes.

When the cache expires:

- Thousands of users request the same product
- All requests bypass the cache
- Every request hits the database
- Database CPU spikes dramatically
- System latency increases

This sudden flood of requests is called a “thundering herd.”

**Common Causes**

- Cache expiration
- Service restart
- Traffic spikes
- Sudden reconnect storms

**Solutions**

1\. Request Coalescing: Allow only one request to rebuild the cache while others wait.

2\. Distributed Locks: Use Redis locks or ZooKeeper locks to ensure only one process performs expensive recomputation.

3\. Randomized Cache Expiry: Avoid all cache entries expiring simultaneously.

4\. Stale-While-Revalidate: Serve stale data temporarily while asynchronously refreshing the cache.

5\. Rate Limiting: Protect downstream systems from overload.

**Real-World Example**

During flash sales on Amazon or Flipkart, cache invalidation can cause massive bursts of database traffic if not carefully handled.

Advantages of Solving It Properly

- Improved system stability
- Reduced database overload
- Better user experience
- Higher availability during traffic spikes

## 3\. Distributed Transactions & Two-Phase Commit (2PC)

**Overview** In distributed systems, a single business transaction may involve multiple services and databases.

For example: Order Service + Payment Service + Inventory Service

If one service succeeds and another fails, the system may become inconsistent.

Distributed transactions solve this problem by ensuring all participating services either:

- Commit successfully together
- Roll back together

One of the most well-known approaches is the Two-Phase Commit (2PC) protocol.

**How Two-Phase Commit Works**

Phase 1: Prepare Phase

The coordinator asks all participants whether they are ready to commit.

Each participant:

- Validates the transaction
- Locks required resources
- Responds with YES or NO

Phase 2: Commit Phase

If all participants respond YES:

- Coordinator sends COMMIT
- Every participant permanently applies changes

If any participant responds NO:

- Coordinator sends ROLLBACK
- All changes are reverted

## Example

Consider a banking transfer:

Debit money from Account A

Credit money into Account B

Both operations must succeed together.

If debit succeeds but credit fails, money disappears from the system, which is an unacceptable state.

2PC guarantees atomicity across distributed services.

Advantages

- Strong consistency
- Atomic transactions across services
- Reliable state management

Drawbacks

- High latency
- Blocking behavior
- Reduced scalability
- Coordinator becomes a bottleneck
- Difficult recovery scenarios

Modern Alternatives

Because 2PC can be expensive and slow, many modern systems prefer:

- Saga Pattern
- Event-driven architecture
- Compensation transactions
- Eventual consistency

**Real-World Usage:** Traditional banking systems and financial platforms commonly use distributed transaction mechanisms because consistency is critical.

## 4\. Materialized View Pattern

## Overview

A Materialized View is a precomputed and stored query result used to optimize expensive read operations. Instead of executing complex joins and aggregations repeatedly, the system calculates results beforehand and stores them. This significantly improves read performance.

Complex analytical queries can become very expensive in large-scale systems.

For example:

- Dashboard analytics
- Sales reports
- User statistics
- Recommendation summaries

Running those queries in real time can overload the database.

**Work Flow**

Raw data is stored normally.

Background jobs aggregate and process data.

Precomputed results are stored separately.

Applications query the materialized view directly.

**Real-World Example :** YouTube analytics dashboards do not calculate total views, watch time, and engagement metrics in real time for every page load.

Instead, metrics are periodically aggregated and stored.

**Advantages**

- Extremely fast reads
- Reduced query complexity
- Lower database CPU usage
- Better dashboard performance

**Drawbacks**

- Data may become stale
- Requires synchronization logic
- Additional storage cost
- Complexity in refresh mechanisms

**Common Refresh Strategies**

Scheduled Refresh

Refresh every few minutes or hours.

Incremental Refresh

Only update changed data.

Event-Driven Refresh

Update materialized views whenever events occur.

**Best Use Cases**

- Analytics dashboards
- Reporting systems
- Recommendation engines
- Data warehousing

## 5\. Bulk Processing Pattern

## Overview

Bulk processing combines multiple small operations into larger batches. Instead of processing requests individually, the system groups them together to improve efficiency. This pattern is heavily used in distributed systems because network calls, disk operations, and database writes are expensive.

Instead of inserting 10,000 rows one-by-one:

- System groups rows into batches
- Inserts them together
- Reduces network overhead
- Improves throughput dramatically

Every database operation has overhead:

- Connection handling
- Disk synchronization
- Transaction setup
- Network latency

Batching reduces these repeated costs.

**Real-World Examples**

**Email Systems:** Bulk email services send messages in batches instead of individually.

**Kafka Consumers:** Kafka consumers process messages in batches to maximize throughput.

**Payment Systems:** Banks process settlement transactions in groups.

**Advantages**

- Better throughput
- Lower infrastructure cost
- Reduced network calls
- Improved CPU and disk utilization

**Drawbacks**

- Increased latency for individual items
- Partial batch failure complexity
- Harder retry handling
- Requires batch sizing optimization

**Best Use Cases**

- ETL pipelines
- Data ingestion systems
- Notification services
- Logging systems
- Analytics pipelines

## 6\. Read Replica Architecture

## Overview

As applications grow, database read traffic often becomes much larger than write traffic. A single database server may struggle to handle:

- User queries
- Search operations
- Dashboard reads
- Analytics requests

Read replicas solve this problem by creating multiple copies of the primary database.

## Architecture

Primary Database

Handles:

- INSERT
- UPDATE
- DELETE

Read Replicas

Handle:

- SELECT queries
- Reporting
- Search operations
- Analytics reads

Data is asynchronously replicated from the primary database to replicas.

**Benefits**

Improved Scalability

Read traffic is distributed across multiple servers.

Reduced Primary Load

The primary database focuses on writes.

Better Availability

If one replica fails, others continue serving traffic.

Faster Query Performance

Heavy read queries no longer compete with write operations.

**Challenges**

Replication Lag

Replicas may not immediately reflect the latest writes.

Eventual Consistency

Users may temporarily see stale data.

Failover Complexity

Promoting replicas during outages can be complicated.

**Real-World Example**

Social media platforms often use read replicas for:

- Feed generation
- Profile viewing
- Search functionality
- Recommendation systems

Meanwhile, writes still go to the primary database.

**Best Use Cases**

- Read-heavy applications
- Analytics systems
- Content platforms
- Large-scale APIs

## 7\. Global Rate Limiting

## Overview

Rate limiting controls how many requests a user or client can make within a specific time window. Without rate limiting, systems become vulnerable to:

- Abuse
- DDoS attacks
- Traffic spikes
- Resource exhaustion

Global rate limiting becomes especially important in distributed architectures where traffic is served by multiple application servers.

**Failure of Local Rate Limiting:**

If each server independently tracks request counts:

- Limits become inconsistent
- Users can bypass restrictions by hitting different servers

Global rate limiting solves this using centralized storage such as Redis.

**Common Algorithms**

Token Bucket: Tokens refill at a fixed rate. Requests consume tokens.

Leaky Bucket: Requests are processed at a constant rate.

Sliding Window: Tracks requests within a moving time window.

Fixed Window Counter: Simpler but less accurate approach.

**Real-World Examples**

API Gateways

Platforms like Stripe and Twitter limit API requests.

Login Systems

Prevent brute-force attacks.

Payment Systems

Protect expensive financial operations.

AI Platforms

Control usage costs and prevent overload.

**Advantages**

- Protects infrastructure
- Improves reliability
- Prevents abuse
- Enables fair resource allocation
- Helps control operational costs

**Drawbacks**

- Additional infrastructure complexity
- Incorrect limits can hurt user experience
- Distributed synchronization challenges

**Common tools used for rate limiting:**

- Redis
- NGINX
- Envoy Proxy
- API Gateways
- Cloudflare

These seven patterns represent some of the most practical and widely used techniques in modern backend engineering.

Each pattern solves a specific category of problems:

Write-Behind Cache: Faster writes Thundering Herd : Protection + Traffic spike + stability Distributed Transactions: Strong consistency Materialized Views: Faster reads Bulk Processing: Higher throughput Read Replicas: Read scalability Global Rate Limiting: System protection

Once you understand these patterns deeply, you begin more like a systems architect.

That's all, folks..Cheers!
