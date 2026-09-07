---
title: "Design Medium"
url: "https://x.com/Harry_The_Nerd/status/2084263186199638124"
category: "HLD"
date: "2026-08-03"
description: "System design for an open publishing and social journalism platform like Medium"
---

# Design Medium

> System design for an open publishing and social journalism platform like Medium
>
> 原文：[https://x.com/Harry_The_Nerd/status/2084263186199638124](https://x.com/Harry_The_Nerd/status/2084263186199638124) · 2026-08-03

![Cover image](https://pbs.twimg.com/media/HOuWxFYaIAANN1Y.jpg)

## 1\. Requirements

**Functional Requirements**

- Writer and reader profile management
- Create, edit, publish, and delete articles (rich text, embedded images, code blocks)
- Draft management with autosave
- Clap on articles (up to 50 claps per user per article)
- Threaded comments on articles
- Follow and unfollow writers
- Home feed (followed writers + topic-based recommendations)
- Tags and topics per article
- Search for articles, writers, and topics
- Reading time estimation
- Notifications (new article from followed writer, clap, comment, new follower)

**Out of Scope**

- Premium membership and paywalled articles
- Medium Partner Program (writer earnings)
- Repost feature
- Publications (multi-writer managed blogs)
- Audio narration of articles

**Non-Functional Requirements**

- High availability for article reading, the most frequent operation
- Low latency feed reads under 100ms
- Autosave must feel instantaneous, no perceptible write latency for writers
- Eventual consistency acceptable for clap counts, feed ranking, and recommendations
- Strong consistency required for article publish state (draft vs published)
- Read/write ratio of approximately 1000:1, heavily optimized for reads
- A user must never see an article they have already read in their feed

## 2\. Capacity Estimation

- **DAU:** 1 million
- **Articles published per day:** 1-5% of DAU = ~10,000 articles per day
- **Text content per article:** ~50 KB
- **Images per article:** ~50% of articles have 2 images at 500 KB each
- **Text storage per day:** 10,000 x 50 KB = ~500 MB/day (negligible)
- **Image storage per day:** 10,000 x 0.5 x 2 x 500 KB = ~5 GB/day
- **Total storage per day:** ~5 GB/day, dominated by images
- **Active drafts at any time:** ~10x published articles = ~100,000 drafts being edited
- **Autosave writes:** 100,000 active drafts x 1 autosave per 5 seconds = ~20,000 writes/second during peak writing hours
- **Read/write ratio:** ~1000:1, a viral article is read tens of thousands of times after being written once

The dominant challenge is not storage scale but autosave write volume and feed personalization quality. Medium is a content discovery platform as much as a publishing platform, and the feed experience determines whether readers return daily.

## 3\. API Gateway

All client requests enter through the API Gateway which handles authentication, rate limiting, and routing. Media uploads (article images) bypass the Gateway after authentication using presigned S3 URLs for direct client-to-S3 upload. The API Gateway never handles binary media payloads.

Medium has a meaningful unauthenticated read traffic pattern. Many readers arrive via Google search or shared links without being logged in. The API Gateway routes unauthenticated article reads directly to the Article Read path without authentication overhead, while write operations (clap, comment, follow, publish) always require authentication.

## 4\. User Profile Service

Medium has two overlapping user types: writers who publish content and readers who consume it. Most users are both simultaneously.

**Data Model**

**Users Table (PostgreSQL)**

Users user\_id UUID, primary key username VARCHAR, unique email VARCHAR, unique display\_name VARCHAR bio TEXT profile\_pic\_url TEXT website\_url TEXT twitter\_handle VARCHAR follower\_count INTEGER following\_count INTEGER article\_count INTEGER total\_claps\_received INTEGER created\_at TIMESTAMP

follower\_count, following\_count, article\_count, and total\_claps\_received are denormalized counters updated asynchronously via Kafka consumers. Aggregating these at read time from raw tables would be too slow at scale.

**Follows Table (PostgreSQL)**

Follows follower\_id UUID, foreign key -\> Users followee\_id UUID, foreign key -\> Users created\_at TIMESTAMP PRIMARY KEY (follower\_id, followee\_id)

A follow is an INSERT, an unfollow is a DELETE. "Give me all writers user X follows" is WHERE follower\_id = X. Indexed on both columns for fast lookups in both directions.

**UserTopics Table (PostgreSQL)**

UserTopics user\_id UUID, foreign key -\> Users topic VARCHAR interest\_score FLOAT (updated by recommendation pipeline) created\_at TIMESTAMP PRIMARY KEY (user\_id, topic)

Explicit topic interests (user manually follows a topic) and inferred interests (derived from reading behavior by the recommendation pipeline) both live here. The interest\_score is updated periodically by the ML pipeline based on reading history signals.

**Caching Strategy**

Writer profiles are cached in Redis since they are fetched on every article page view. Hot writers (those with many followers whose articles appear in many feeds) benefit most from caching. User topic interest scores are cached per user for fast feed assembly.

## 5\. Post Service

The Post Service manages the full article lifecycle from draft creation through editing, publishing, and deletion.

**Article Storage Architecture**

Medium articles are not simple text strings. They are rich documents with headings, paragraphs, images, code blocks, pull quotes, and embedded links. This structure is stored as a **JSON document** in MongoDB, not as raw text in PostgreSQL.

MongoDB is chosen over PostgreSQL for article content because:

- Flexible schema per article (different articles have different block structures)
- Efficient storage and retrieval of large nested JSON documents
- No need for complex joins on article content
- Native support for partial document updates (editing one section without rewriting the entire document)

PostgreSQL stores article metadata (title, slug, status, author, clap count). MongoDB stores article content (the full rich text document). S3 stores embedded images referenced within the article.

Article Content Structure (MongoDB)

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

This block-based structure is similar to how Notion, Substack, and Medium's own editor (backed by a similar format) represent rich content.

**Articles Metadata Table (PostgreSQL)**

**Articles** article\_id UUID, primary key author\_id UUID, foreign key -\> Users title VARCHAR subtitle VARCHAR cover\_image\_url TEXT slug VARCHAR, unique (null until published) status ENUM('draft', 'published', 'deleted') reading\_time INTEGER (minutes, computed on publish) clap\_count INTEGER (denormalized total) comment\_count INTEGER (denormalized counter) view\_count INTEGER (denormalized counter) published\_at TIMESTAMP (null until published) created\_at TIMESTAMP updated\_at TIMESTAMP

Tags and Topics

**ArticleTags Table (PostgreSQL)**

ArticleTags article\_id UUID, foreign key -\> Articles tag VARCHAR PRIMARY KEY (article\_id, tag)

Tags are set by the writer. Topics are broader categories inferred or manually selected. Both are indexed in Elasticsearch for search and used by the recommendation pipeline for topic-based feed personalization.

**Draft Autosave Flow**

Medium autosaves every few seconds while a writer is typing. At 100,000 active drafts with saves every 5 seconds, this is ~20,000 writes per second during peak hours. Writing every autosave directly to MongoDB would overwhelm the database.

**The autosave flow uses Redis as a write buffer:**

Writer types, client debounces keystrokes and sends autosave request every 3-5 seconds

Post Service writes draft content to Redis immediately:

Key: draft:{article\_id} Value: serialized article content JSON TTL: 7 days

Response returned to client instantly (sub-millisecond Redis write)

Autosave event published to Kafka: draft.updated

Kafka consumer flushes draft content from Redis to MongoDB asynchronously every 30 seconds per draft

The writer never waits for MongoDB. Redis absorbs all autosave writes. MongoDB receives batched updates at a sustainable rate.

On cache miss (Redis evicted the draft, writer returns after 7 days), the draft is fetched from MongoDB and re-populated in Redis.

**Publish Flow**

When a writer clicks Publish:

Post Service reads final draft content from Redis (or MongoDB on cache miss)

Generates unique URL slug from title (e.g. "why-i-switched-to-rust-a1b2c3")

Computes reading time: word\_count / 200 (average adult reading speed in wpm)

Updates Articles Table: status = 'published', slug = generated\_slug, published\_at = NOW()

Writes final content to MongoDB (permanent published version)

Publishes article.published event to Kafka

Kafka consumers:**Search Indexing Worker** indexes article into Elasticsearch **Feed Fanout Worker** writes article into followers' feed sorted sets **Notification Worker** notifies all followers of new article **Topic Index Worker** updates topic sorted sets in Redis with new article

**Image Upload Flow**

**When a writer inserts an image into the editor:**

Client requests presigned S3 URL from Post Service

Client uploads image directly to S3

Image Processing Worker generates multiple resolution variants (thumbnail for feed cards, medium for article body, full for expanded view)

CDN sits in front of S3, serving images globally with high cache hit rates

**Bottlenecks in the Post Service**

The autosave Redis write buffer is the key architectural decision that prevents MongoDB from being overwhelmed. The publish flow is low volume (10,000 per day) and has no bottleneck concerns. The image processing pipeline is the same pattern as all previous systems and scales horizontally via Kafka workers.

## 6\. Feed Service

Medium's feed combines two signals: articles from writers the user follows and topic-based recommendations. This is more complex than Twitter's purely follow-based feed but simpler than Netflix's multi-stage recommendation pipeline.

**Fan-out on Write for All Users**

Unlike Instagram and Twitter where celebrities with millions of followers require fan-out on read, Medium's largest writers have at most a few hundred thousand followers. Fan-out on write for all users is perfectly viable. No hybrid model needed.

When an article is published, the Feed Fanout Worker writes it into the feed sorted set of every follower:

Key: feed:{user\_id} Member: article\_id Score: weighted relevance score (recency + follow boost + engagement signals) TTL: 7 days

Weighted Score Formula

Score = base\_engagement\_score + follow\_boost + recency\_decay

- base\_engagement\_score reflects how much engagement (claps, comments, reads) the article has received since publishing
- follow\_boost is added for articles from writers the user follows, ensuring followed content gets priority but does not completely dominate
- recency\_decay ensures newer articles score higher than older ones with the same engagement, preventing stale content from staying at the top

This weighted merge means the feed is a blend of follow-based and topic-based content rather than a strict ordering. A highly relevant fresh topic recommendation can surface above a stale article from a followed writer, which matches Medium's actual feed behavior.

**Topic-Based Recommendations**

The recommendation pipeline runs periodically and computes topic-based article recommendations per user based on:

- **Reading history signals:** topics of articles the user read to completion (high completion\_pct) weighted more than articles they bounced from
- **Collaborative filtering:** users who read the same articles as this user also read Y
- **Topic graph:** adjacent topics (self-help and poetry share readers; backend engineering and system design share readers)
- **Followed writer topics:** infer interest from the topics of writers the user follows

**Top topic-based recommendations are written to Redis per user:**

Key: recommendations:{user\_id} Value: \[article\_id\_1, article\_id\_2, ..., article\_id\_20\] TTL: 6 hours

At feed assembly time, the Feed Service merges the precomputed follow feed sorted set with the topic recommendations, applies the weighted scoring, and returns the top 20 article IDs.

**Already-Read Exclusion with Bloom Filter**

A user must never see an article they have already read in their feed. The Reading History Service maintains a Bloom Filter per user:

BF.ADD seen:{user\_id} article\_id (when user opens an article) BF.EXISTS seen:{user\_id} article\_id (before inserting into feed sorted set)

Before any article is inserted into a user's feed sorted set or returned as a recommendation, BF.EXISTS is checked. If the article has been read, it is excluded. No false negatives means a read article is never shown again. The small false positive rate (occasionally excluding an unread article) is acceptable.

**Feed Read Flow**

When a user opens their Medium home feed:

Feed Service fetches top 50 article IDs from feed:{user\_id} sorted set via ZREVRANGE feed:user\_123 0 49

Merges with topic recommendations from recommendations:{user\_id} Redis key

Applies Bloom Filter exclusion for already-read articles

Re-ranks merged list by weighted score

Hydrates top 20 article IDs via Redis MGET for article metadata (title, subtitle, cover image, author, reading time, clap count)

On cache miss for any article, falls back to PostgreSQL and writes back to Redis

Returns rendered feed cards to client

**Bottlenecks in the Feed Service**

Fan-out on write for 100,000 followers of a top writer means writing to 100,000 sorted sets per article publish. At 10,000 articles per day with average 500 followers each, this is 5 million sorted set writes per day, which is very manageable with a horizontally scaled Fanout Worker pool. Redis memory for feed sorted sets is capped by the 7-day TTL and a maximum of 200 articles per feed sorted set.

## 7\. Clap Service

The Clap Service handles Medium's unique engagement mechanism. A single reader can clap up to 50 times on a single article. Total claps displayed on an article is the SUM of all readers' clap counts, not a simple count of unique clappers.

**Redis Structure**

**Per-user clap count (cap enforcement):**

Key: claps:{article\_id}:{user\_id} Value: integer (1 to 50) TTL: 30 days

Before processing a clap: GET claps:{article\_id}:{user\_id}. If value equals 50, reject the clap (cap reached). Otherwise INCR claps:{article\_id}:{user\_id}.

**Total clap count (display on article):**

Key: article:claps:{article\_id} Value: integer (running total across all users)

Every clap that passes the per-user cap check also increments INCR article:claps:{article\_id}. Reading total claps for display is a single GET article:claps:{article\_id}. No SUM query needed at read time.

**Full Clap Flow**

User taps clap button (can tap multiple times rapidly)

Client debounces taps and batches them (sends accumulated clap count every 1 second)

Clap Service receives { user\_id, article\_id, clap\_increment }

GET claps:{article\_id}:{user\_id} -\> current user clap count

Compute allowed increment: min(clap\_increment, 50 - current\_count)

If allowed increment \> 0:INCRBY claps:{article\_id}:{user\_id} allowed\_increment INCRBY article:claps:{article\_id} allowed\_increment Publish clap.created event to Kafka

Kafka consumers:**Clap Persistence Worker** upserts to Claps Table in PostgreSQL **Counter Sync Worker** periodically syncs Redis total to clap\_count on Articles Table **Notification Worker** notifies article author (coalesced, not one notification per clap)

**Claps Table (PostgreSQL)**

Claps user\_id UUID, foreign key -\> Users article\_id UUID, foreign key -\> Articles clap\_count INTEGER (1 to 50) last\_clapped TIMESTAMP PRIMARY KEY (user\_id, article\_id)

UPSERT on (user\_id, article\_id) with clap\_count = clap\_count + allowed\_increment handles both first clap and subsequent claps in a single operation.

**Bottlenecks in the Clap Service**

Clap bursts on viral articles (thousands of readers clapping simultaneously) are absorbed entirely by Redis INCRBY operations which are atomic and sub-millisecond. The Kafka consumer handles async persistence without any write pressure on PostgreSQL during the burst. Periodic reconciliation between Redis totals and PostgreSQL clap\_count catches any drift.

## 8\. Comment Service

Comments on Medium are threaded but shallower than Reddit. Medium encourages responses (separate articles written as replies) more than deep comment threads. Comments are low volume compared to claps.

Data Model

**Comments Table (PostgreSQL)**

Comments comment\_id UUID, primary key article\_id UUID, foreign key -\> Articles parent\_id UUID, foreign key -\> Comments (null for top-level) user\_id UUID, foreign key -\> Users content TEXT clap\_count INTEGER created\_at TIMESTAMP updated\_at TIMESTAMP

Same Adjacency List pattern as Reddit. parent\_id = NULL means top-level comment on the article. Non-null parent\_id means reply to another comment. Top-level comments sorted by clap count (most appreciated comments surface first). Replies sorted chronologically within a thread.

Composite index on (article\_id, parent\_id) makes both primary queries fast:

- Top-level comments: WHERE article\_id = X AND parent\_id IS NULL ORDER BY clap\_count DESC LIMIT 20
- Replies to a comment: WHERE parent\_id = comment\_123 ORDER BY created\_at ASC

Hot article comment sections are cached in Redis with a short TTL. The daily most-read articles benefit significantly from comment caching.

Bottlenecks in the Comment Service

Comment volume is low at Medium's scale. PostgreSQL handles comment writes comfortably. The main concern is comment sections on viral articles receiving thousands of simultaneous reads. Redis caching of top-level comments absorbs this.

## 9\. Search Service

Medium search covers three entities: articles (by title, content, and tags), writers (by name and username), and topics.

Storage Engine - Elasticsearch

Elasticsearch handles full-text search, partial matching, and relevance ranking across all article content. Medium search is primarily keyword and topic-based rather than engagement-weighted like Instagram, but clap count and read count are used as ranking signals.

**Article document:**

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

Full article content is not indexed in Elasticsearch., only a content preview (first 500 characters) and tags. Full content search would make the index enormous and most search intent is satisfied by title, tags, and preview matching.

**Writer document:**

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

Ranking Signals

Medium search ranking combines:

- **Text match score** (title match weighted higher than tag match, tag match weighted higher than preview match)
- **Clap count** (more appreciated articles rank higher for the same query)
- **Recency** (newer articles get a ranking boost for broad queries)
- **Author follower count** (established writers rank slightly higher)

Keeping Elasticsearch in Sync

The Search Indexing Worker consumes from article.published and article.updated Kafka topics. Clap count and view count updates are batched and synced to Elasticsearch periodically rather than on every clap since exact engagement counts in search results do not need to be real-time.

## 10\. Reading History Service

The Reading History Service tracks which articles each user has read and how deeply they read them. This data feeds the recommendation pipeline and powers the Bloom Filter for feed deduplication.

**Data Model**

**ReadingHistory Table (PostgreSQL, partitioned by user\_id)**

ReadingHistory user\_id UUID, foreign key -\> Users article\_id UUID, foreign key -\> Articles read\_at TIMESTAMP completion\_pct FLOAT (0.0 to 1.0, how far the user scrolled) time\_spent\_sec INTEGER (actual reading time in seconds)

completion\_pct is the most valuable signal. A user who reads 95% of an 8-minute article about Rust is far more interested in Rust than a user who opened and immediately bounced. This distinction drives topic interest scoring in the recommendation pipeline.

**Write Flow**

Every article view generates a read event. At 1 million DAU reading an average of 5 articles per day, this is 5 million read events per day. Writes are high volume and must not block the reading experience.

Read events are published to Kafka asynchronously from the client:

User opens article -\> client sends GET /articles/{slug} -\> article served

Client tracks scroll position and time spent in the background

On article close or after 30 seconds: client sends read event { user\_id, article\_id, completion\_pct, time\_spent\_sec }

Reading History Service publishes to Kafka: [article.read](https://x.com/Harry_The_Nerd/status/article.read)

Kafka consumers:**History Persistence Worker** writes to ReadingHistory Table **Bloom Filter Worker** calls BF.ADD seen:{user\_id} article\_id **View Counter Worker** increments INCR article:views:{article\_id} in Redis **Recommendation Signal Worker** updates UserTopics interest scores based on completion\_pct

**Bottlenecks in the Reading History Service**

5 million read events per day is ~58 per second on average, well within Kafka's capacity. The Bloom Filter writes are sub-millisecond Redis operations. PostgreSQL partition by user\_id keeps per-user history queries fast.

## 11\. Notification Service

Same architecture as all previous systems. Kafka fanout from all interaction services, APNs for iOS, FCM for Android, email via SES.

The Notification Service subscribes to:

- article.published -\> "Harry published a new article" to all followers
- clap.created -\> coalesced clap notification to article author ("Harry and 42 others clapped for your article")
- comment.created -\> comment notification to article author and parent comment author
- follow.created -\> "Someone followed you" notification

**Notification Coalescing**

Clap notifications must be coalesced aggressively. A viral article receiving 5,000 claps in an hour should not send 5,000 push notifications to the author. The Notification Service batches clap events within a 1-hour window and sends a single notification: "Your article received 5,247 claps in the last hour."

New article notifications from followed writers are delivered individually since each represents a distinct content event the reader wants to know about.

## 12\. Full Data Flow Summary

Article Creation Flow (Draft): Writer types -\> client debounces -\> autosave every 3-5 seconds -\> Post Service -\> Redis WRITE draft:{article\_id} (sub-millisecond) -\> Kafka (draft.updated) -\> MongoDB Writer (async, flushes every 30 seconds per draft) Image Upload Flow: Writer inserts image -\> Post Service -\> presigned S3 URL Writer -\> S3 (direct upload) S3 -\> Kafka -\> Image Processing Worker -\> S3 (thumbnail, medium, full variants) CDN caches all variants Publish Flow: Writer clicks Publish -\> Post Service -\> Generate slug, compute reading\_time -\> UPDATE Articles: status = 'published', published\_at = NOW() -\> Write final content to MongoDB -\> Kafka (article.published) -\> Search Worker -\> Elasticsearch index -\> Feed Fanout Worker -\> Redis ZADD feed:{follower\_id} for all followers -\> Notification Worker -\> APNs / FCM / SES to all followers -\> Topic Index Worker -\> Redis topic sorted sets updated Feed Read Flow: User opens Medium -\> Feed Service -\> Redis ZREVRANGE feed:{user\_id} 0 49 (follow-based articles) -\> Redis GET recommendations:{user\_id} (topic-based articles) -\> Bloom Filter BF.EXISTS seen:{user\_id} for each candidate (exclude read articles) -\> Weighted score merge and re-rank -\> Redis MGET (hydrate article metadata for top 20) -\> CDN (cover images served directly) Article Read Flow: User opens article -\> Post Service -\> Redis (article metadata cache) -\> MongoDB (full article content) -\> CDN (embedded images) -\> Client tracks scroll + time spent -\> On close: read event published to Kafka ([article.read](https://x.com/Harry_The_Nerd/status/article.read)) -\> History Persistence Worker -\> ReadingHistory Table (PostgreSQL) -\> Bloom Filter Worker -\> BF.ADD seen:{user\_id} article\_id -\> View Counter Worker -\> INCR article:views:{article\_id} -\> Recommendation Signal Worker -\> UPDATE UserTopics interest scores Clap Flow: User claps -\> client debounces and batches -\> Clap Service -\> GET claps:{article\_id}:{user\_id} (current user clap count) -\> Compute allowed increment (cap at 50) -\> INCRBY claps:{article\_id}:{user\_id} allowed\_increment -\> INCRBY article:claps:{article\_id} allowed\_increment -\> Kafka (clap.created) -\> Clap Persistence Worker -\> UPSERT Claps Table (PostgreSQL) -\> Counter Sync Worker -\> UPDATE Articles.clap\_count (periodic batch) -\> Notification Worker -\> coalesced clap notification to author Search Flow: User searches "rust backend" -\> Search Service -\> Redis (hot query cache) -\> Elasticsearch (title + tag + preview match, ranked by claps + recency) -\> Redis MGET (hydrate article metadata for results)

## 13\. Resilience and Fault Tolerance

**Redis autosave buffer** ensures writers never lose work even if MongoDB is temporarily unavailable. Drafts survive in Redis with a 7-day TTL. The Kafka consumer catches up on MongoDB writes when the database recovers.

**Bloom Filter for feed deduplication** has no false negatives. A read article is never shown again. The small false positive rate (occasionally excluding an unread article) is completely acceptable and users never notice.

**Fan-out on write for all users** means feed reads are always fast regardless of database health. The feed sorted set in Redis is the primary read path. PostgreSQL is only hit on full cache misses.

**Kafka durability** throughout the pipeline means no article publish event, clap event, or read event is lost even if downstream services are temporarily unavailable. All consumers catch up from Kafka offsets on recovery.

**Redis clap counters with async PostgreSQL sync** absorb clap bursts on viral articles without any write pressure on the database. Periodic reconciliation catches any drift between Redis totals and PostgreSQL clap\_count.

**MongoDB for article content** decouples content storage from metadata storage. A MongoDB slowdown affects article body reads but not feed assembly, search results, or profile pages which all use PostgreSQL and Redis.

**CDN caching** for all article images means embedded media is served from edge nodes globally. Article images change rarely (writers do not edit images after publishing) so cache hit rates are very high.

**Elasticsearch eventual consistency** for article indexing is acceptable. A newly published article may not appear in search results for a few seconds while the Search Indexing Worker processes the Kafka event. Readers discover new articles primarily through their feed, not through search.

## 14\. Technology Choices Summary

**API Gateway** -\> Kong / AWS API Gateway (Auth, rate limiting, unauthenticated read routing)

**Article Content Store** -\> MongoDB (Rich text JSON documents, flexible schema, large nested content)

**Article Metadata DB** -\> PostgreSQL sharded + replicas (Title, slug, status, counters, strong consistency for publish state)

**Draft Autosave Buffer** \-\> Redis (Sub-millisecond autosave writes, 7-day TTL, async flush to MongoDB via Kafka)

**Image Storage** \-\> AWS S3 (Embedded article images, durable, CDN-native)

**CDN** -\> CloudFront / Akamai (Edge caching for article images, high cache hit rate)

**Decoupling** -\> Apache Kafka (Article publish fanout, clap persistence, read events, notification triggers)

**Search Engine** -\> Elasticsearch (Full-text search on title, tags, preview, clap-weighted ranking)

**Feed Store** -\> Redis Sorted Sets per user (Weighted score merge of follow-based and topic-based articles)

**Topic Recommendations** -\> Redis per user (Precomputed topic-based article recommendations, 6-hour TTL)

**Read Deduplication** -\> Redis Bloom Filter per user (Already-read article exclusion from feed)

**Clap Counter (per-user cap)** -\> Redis INCRBY with GET cap check (Atomic 50-clap limit enforcement)

**Clap Counter (article total)** -\> Redis INCRBY (Running total, single GET for display)

**Claps DB** -\> PostgreSQL UPSERT (Source of truth for per-user clap counts)

**Reading History DB** \-\> PostgreSQL partitioned by user\_id (Completion rate, time spent, recommendation signals)

**User Profile Cache** -\> Redis (Writer profiles, follower counts, topic interest scores)

**Notification Delivery** -\> APNs + FCM + AWS SES (Push notifications and email, aggressive clap coalescing)

That's all, folks...Cheers!!
