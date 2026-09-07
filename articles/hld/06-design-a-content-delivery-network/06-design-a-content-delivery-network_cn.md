---
title: "Design a Content Delivery Network"
url: "https://x.com/Harry_The_Nerd/status/2046578404758295032"
category: "HLD"
date: "2026-04-21"
description: "CDN architecture, caching strategies, and edge delivery."
lang: "zh-CN"
---

# 设计一个内容分发网络

> CDN 架构、缓存策略与边缘分发。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2046578404758295032](https://x.com/Harry_The_Nerd/status/2046578404758295032) · 2026-04-21

![封面图](https://pbs.twimg.com/media/HGbk260aAAAzCpC.jpg)

CDN 不只是又一个缓存玩意儿，它是一张地理上分布的服务器网络，把内容在物理上搬到离用户更近的地方。从孟买请求纽约的源站，大约要 200 毫秒；同样的请求打到孟买的边缘节点，大约只要 5 毫秒。这就是 CDN 要解决的问题。下面是完整拆解。

**功能需求** 1. 把高频请求的静态内容存放在地理上靠近用户的地方 2. 从最近的边缘节点提供内容，而不是从源站 3. 缓存未命中时回源拉取，并缓存下来供后续请求使用 4. 缓存失效，也就是在源站更新时保证内容是新鲜的

**什么该缓存，什么不该**

该缓存的

对所有用户都一样的静态内容，比如 HTML、CSS、JS、图片、视频、字体、PDF。这些缓存起来是安全的，因为响应不会因用户而异。

绝对不要缓存下面这些 1. 用户私有数据，比如账户余额、私信、观看清单。把它发给错误的用户就是隐私事故。2. 实时数据，比如板球直播比分、股票价格、打车轨迹。等缓存好了它已经过期了。3. POST/PUT/DELETE 请求，任何写入或修改数据的操作。永远不要缓存写请求。4. 个性化页面，你的 Amazon / Pinterest 首页跟别人的都不一样。

一个好记的判断法：对所有人响应相同，行，缓存它；因人而异或每秒都在变，不行，绝不缓存。

**地理路由（GeoDNS）**

当孟买的用户输入 [netflix.com](https://x.com/Harry_The_Nerd/status/netflix.com) 时，比普通 DNS 更聪明的东西开始工作了。GeoDNS 检测用户的 IP，判断出他的位置，然后返回最近那个边缘节点的 IP，而不是源站的。

孟买的用户输入 [netflix.com](https://x.com/Harry_The_Nerd/status/netflix.com)

↓

GeoDNS 检测到 → 用户在孟买

↓

返回孟买边缘节点的 IP

↓

用户以约 5 毫秒（而不是约 200 毫秒）拿到内容

普通 DNS 给所有人返回同一个 IP，GeoDNS 则根据地理位置返回不同的 IP。这是 CDN 的核心魔法，其余一切都建立在它之上。

**三层架构**

大多数人把 CDN 设计成两层：边缘节点和源站。而 Cloudflare、Akamai 这类生产级 CDN 用的是三层。

没有中间层的话，全球每个边缘节点的每一次缓存未命中都会直接打到源站：

孟买缓存未命中 -\> 打到源站

新加坡缓存未命中 -\> 打到源站

迪拜缓存未命中 -\> 打到源站

伦敦缓存未命中 -\> 打到源站

解决办法是区域缓存（Origin Shield，源站盾）。它是边缘节点和源站之间的中间层。边缘节点的缓存未命中先打到区域缓存，只有当区域缓存也未命中时，才会有一个请求发往源站：

边缘节点（新加坡、孟买、迪拜、伦敦）→ 区域缓存（亚洲）→ 源站

源站被打中一次，而不是成千上万次。后续每一次边缘未命中都由区域缓存来响应。

完整的请求流程：

孟买的用户

↓

GeoDNS → 孟买边缘节点

↓ 缓存命中 → 返回内容

↓ 缓存未命中

区域缓存（亚洲）

↓ 缓存命中 → 返回内容 + 在边缘缓存一份

↓ 缓存未命中

源站服务器 → 返回内容 + 在区域缓存一份 + 在边缘缓存一份

**缓存失效：保持内容新鲜**

TTL 是主要机制——每个缓存对象都有一个过期时间。过期之后，边缘节点会从源站重新拉取。但光靠 TTL 应付不了紧急更新。

**清除（Purge）**

明确告诉每一个边缘节点立刻删掉某个文件。见效即时，但代价不小，因为你要把清除命令发往全球成千上万个边缘节点。只在关键修复时使用。

**版本化**

不做失效，而是改文件名：

style.css?v=1 -\> 旧版本，仍在缓存里，自然过期

style.css?v=2 -\> 新版本，第一次请求时拉取最新内容

不需要清除。旧版本靠 TTL 过期，新版本立刻就是新鲜的。这是大多数前端工程师在生产中采用的做法。

**Cache-Control 响应头**

源站给每个文件附上指令，明确告诉 CDN 和浏览器该如何缓存它：

[netflix.com/logo.png](https://x.com/Harry_The_Nerd/status/netflix.com/logo.png)

Cache-Control: public, max-age=31536000

到处都缓存，缓存 1 年

[netflix.com/user/watchlist](https://x.com/Harry_The_Nerd/status/netflix.com/user/watchlist)

Cache-Control: private, no-store

任何地方都不缓存（私有内容）

[netflix.com/trending.json](https://x.com/Harry_The_Nerd/status/netflix.com/trending.json)

Cache-Control: public, s-maxage=300

CDN 缓存 5 分钟 ✅

[netflix.com/checkout](https://x.com/Harry_The_Nerd/status/netflix.com/checkout)

Cache-Control: no-store

从不缓存，永远是最新的 ✅

关键指令：max-age 以秒为单位设置 TTL；no-store 表示永不缓存；private 表示只在浏览器缓存，不在 CDN 缓存；s-maxage 专门针对 CDN 覆盖 max-age，让浏览器和 CDN 可以有不同的 TTL。

**接下来是数据层**

每一层的存储都体现了速度与容量的权衡：

边缘节点 -\> Redis（内存型，每节点约 100GB）。快得飞起，容量有限，只放那座城市最热的内容。

区域缓存 -\> SSD 存储（每节点约 10TB）。磁盘够快，容量大得多，存放整个大洲的温数据。

源站 -\> S3（容量无限）。最原始的唯一可信来源，每一份内容都永久存放在这里。

元数据库 -\> Cassandra。存放 TTL 规则、缓存策略，以及每个内容对象的缓存状态。边缘节点查询它来确定每个文件该缓存多久。

**非功能需求**

**延迟**

CDN 存在的全部意义就是降低延迟。在每个主要城市部署边缘节点，意味着用户几乎总能命中本地那份 Redis 速度的缓存。三层架构则意味着源站几乎不会被碰到。

**可扩展性**

CDN 通过在更多城市部署更多边缘节点来扩展。每个边缘节点彼此独立，之间不需要任何协调。区域缓存吸收了那些本来会打到源站的负载，让边缘节点的水平扩展变得简单直接。

**可用性**

如果某个边缘节点挂了，GeoDNS 会自动把流量改路由到次近的边缘节点。如果区域缓存挂了，边缘节点会直接回退到源站，虽然慢一些，但仍然可用。系统在每一层都能优雅降级。就这些了……干杯！
