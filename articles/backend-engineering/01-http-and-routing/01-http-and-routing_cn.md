---
title: "HTTP & Routing"
url: "https://x.com/Harry_The_Nerd/status/2053366145995178087"
category: "Backend Engineering"
date: "2026-05-10"
description: "Core concepts of HTTP and routing in backend systems."
lang: "zh-CN"
---

# HTTP 与路由

> 后端系统中 HTTP 与路由的核心概念。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2053366145995178087](https://x.com/Harry_The_Nerd/status/2053366145995178087) · 2026-05-10

![封面图](https://pbs.twimg.com/media/HH50enYbMAA2Ef4.jpg)

我打算从这里开始写这个后端系列，和系统设计系列并行。想用尽量简单的方式分享我的学习心得。希望大家读起来既有意思，也有收获。

**后端到底是个什么东西？** 当你打开一个 App 或网站，看到的按钮、文字、图片……那是前端。可是数据从哪来？谁在校验你的密码？谁在算你的银行余额？那就是后端。

后端就是跑在服务器上的一切：代码、数据库，以及用户永远不会直接看到的业务逻辑。它是界面背后的引擎。而它和浏览器对话用的语言，就是 **HTTP**。

**什么是 HTTP？**

HTTP 是超文本传输协议（HyperText Transfer Protocol）的缩写。它是浏览器和服务器之间约定好的语言。你每加载一个页面、提交一次表单、调用一次 API，背后都在发生一场 HTTP 对话。你发出一个**请求**，服务器回你一个**响应**。

HTTP 响应

浏览器发出请求之后，服务器会回复。每一个 HTTP 响应都是同样的三段式结构（状态行—响应头—响应体）：

**1\. 状态行：** 响应的第一行。一眼就能看出事情是顺利还是出岔子了。

HTTP/1.1 200 OK version code reason

**2\. 响应头：** 也就是元数据。告诉浏览器该怎么处理这个响应。

Content-Type: application/json Cache-Control: max-age=3600 Set-Cookie: session=abc123; HttpOnly

**3\. 响应体：** 真正的内容。JSON、HTML、一张图片、一个文件。有些响应根本没有响应体。

**状态码**

那个三位数字是任何开发者第一眼要看的东西。它们分成五大类：

- **1xx == 信息性响应。** 请求已收到，继续。
- **2xx == 成功。** 成了，太好了
- **3xx == 重定向。** 去别的地方看看。
- **4xx == 客户端错误。** 是你出错了。
- **5xx == 服务端错误。** 服务器挂了。

我列几个常见的

**200 OK** 表示成功了，数据给你。**201 Created** 表示你提交了内容，并且创建成功了。**301 Moved Permanently** 表示这个 URL 永久废弃了，浏览器会缓存这次重定向。**304 Not Modified** 表示自上次以来什么都没变，用你的缓存副本吧。**400 Bad Request** 表示你的请求格式有问题：少了字段、JSON 坏了等等。**401 Unauthorized** 表示你还没登录，先去认证吧，聪明人。**403 Forbidden** 表示你登录了，但没有权限。登录了也没用。**404 Not Found** 表示这儿什么都没有。**429 Too Many Requests** 表示你被限流了，慢点儿。**500 Internal Server Error** 表示服务端有东西炸了。**503 Service Unavailable** 表示服务器宕了或者过载了，稍后重试。

**那些重要的响应头**

**Content-Type：** 响应体的格式。没有它，浏览器没法解析。

Content-Type: application/json Content-Type: text/html; charset=utf-8

**Cache-Control** 表示这份副本能留多久。

Cache-Control: max-age=3600（缓存 1 小时） Cache-Control: no-store（永不缓存，敏感数据） Cache-Control: private（只在这个浏览器里缓存，CDN 不许存）

**Set-Cookie**：把这个 cookie 存下来，以后每次请求这个域名都带上。

Set-Cookie: session=abc123; HttpOnly; Secure; SameSite=Strict

HttpOnly 表示 JS 读不到它。Secure 表示只走 HTTPS。SameSite=Strict 表示跨站时绝不发送。

**ETag：** 响应内容的指纹。浏览器下次会把它带回来。如果对得上，服务器直接返回 304，不用把整份内容再发一遍，省带宽。

**Retry-After**：通常和 429 或 503 搭配。告诉你等多少秒再重试。

**CORS：跨域资源共享**

先来讲讲那个经典的夜店比喻

你的浏览器是个想进夜店的人，服务器是夜店，CORS 就是门口对着宾客名单查人的保安。

当浏览器向另一个域名发请求时，保安就上场了：「这家服务器把你写进名单了吗？」服务器通过一个响应头来回答是或否，也就是 Access-Control-Allow-Origin。如果你的源在名单上，你就能进；如果不在，浏览器就会拦掉这个响应，你会收到一个 CORS 错误。

一个「源」（origin）是协议 + 域名 + 端口的组合。[https://app.com](https://app.com/) 和 [https://api.app.com](https://api.app.com/) 虽然共用一个域，但属于不同的源。

**为什么需要 CORS？**

如果没有 CORS，任何网站都能悄悄用你自己的 cookie 从别的站点偷走你的数据。具体是这样的：

你登录了 [bank.com](https://x.com/Harry_The_Nerd/status/bank.com)，浏览器保存了你的会话 cookie。

你在新标签页里访问了 [evil-site.com](https://x.com/Harry_The_Nerd/status/evil-site.com)。

恶意站点的 JavaScript 悄悄执行：fetch('[https://bank.com/my-account](https://bank.com/my-account)')

浏览器自动把你的银行 cookie 附在这个请求上。

银行看到一个有效会话，于是返回了你的账户数据。

恶意站点读到了这个响应。你的余额、账号、交易记录，全没了。

CORS 拦住的就是第 6 步。请求照样发到银行，银行照样返回响应。但浏览器会在把响应交给 JavaScript 之前把它截下来，然后问：「[bank.com](https://x.com/Harry_The_Nerd/status/bank.com) 允许 [evil-site.com](https://x.com/Harry_The_Nerd/status/evil-site.com) 读这份数据吗？」没有 Access-Control-Allow-Origin 头？浏览器就默默丢掉响应。恶意站点拿到的是一个 CORS 错误，而不是你的数据。

**简单请求 vs 预检请求**

不是所有请求都一视同仁。浏览器把它们分成两类。

**简单请求**直接发给服务器，不会事先多问。GET、HEAD，以及带普通请求头的基础 POST 都算。浏览器直接发出去，然后在响应回来时检查 CORS 头。

**预检请求**会先做一次「我到底能不能干这件事」的确认。在发真正的请求之前，浏览器会自动先发一个 OPTIONS 请求：

OPTIONS /api/data HTTP/1.1 Origin: [https://app.com](https://app.com/) Access-Control-Request-Method: PUT Access-Control-Request-Headers: Authorization, Content-Type

它其实在问：「我想发一个带 Authorization 头的 PUT 请求，允许吗？」服务器回复它的许可范围，浏览器核对之后，只有获批才会发出真正的请求。

**为什么要有预检？** GET 只是读数据，就算请求过去了，JS 没有许可也读不到响应，直接试一下是安全的。但 DELETE /account 是真的删掉了东西，CORS 还没来得及检查，损害就已经造成了。所以浏览器要先申请许可，再扣扳机。

Access-Control-Max-Age 头会缓存预检的批准结果，这样浏览器就不用每次请求都重新确认一遍：

Access-Control-Max-Age: 86400 → 24 小时内有效

还有一件大多数开发者会忘的事：**CORS 纯粹是浏览器的机制。** curl、Node、Python、Postman 都完全不管 CORS。它存在的唯一目的，是保护真实的网页用户。

**内容协商与压缩**

**核心思想**

浏览器和服务器不只是对话，它们还会协商。数据发出去之前，双方先就格式、语言以及响应体的压缩方式达成一致。浏览器声明自己能处理什么，服务器挑一个最合适的。

它们要协商的四件事

**1\. 内容类型，也就是「什么格式」**

浏览器发送：

Accept: application/json, text/html, \*/\*

\*/\* 这个通配符的意思是「什么都行」。你还可以用权重因子表达偏好：

Accept: application/json;q=1.0, text/xml;q=0.8, \*/\*;q=0.5

服务器挑一个它支持的、q 值最高的，然后用 Content-Type 确认。如果一个都匹配不上：406 Not Acceptable。

**2\. 编码，也就是「响应体怎么压缩」**

浏览器发送：

Accept-Encoding: gzip, deflate, br, zstd

服务器压缩之后告诉你它用了哪种：

Content-Encoding: gzip

常用的压缩算法有三种。**gzip** 是老牌可靠选手，全平台支持，通常能把 JSON 压小约 77%。**Brotli（br）** 是 Google 的现代算法，大约能小 82%，只在 HTTPS 上可用。**Zstandard（zstd）** 是 Facebook 后来推出的，压缩率相近但更快。解压过程对你的 JavaScript 完全透明，你调用 response.json()，拿到的就是干净的数据。

**3\. 语言，也就是「哪种人类语言」**

Accept-Language: en-IN, en;q=0.9, hi;q=0.8

同一个 URL 就是靠这个给不同用户返回不同语言的内容。服务器用 Content-Language: en 来回应。

**4\. 字符编码。** 今天基本没什么意义了。UTF-8 赢了，服务器一律发 charset=utf-8。

当服务器启用内容协商时，它必须告诉 CDN 和缓存：响应内容会随请求头的不同而变化。

Vary: Accept-Encoding, Accept-Language

没有这一行，CDN 可能会缓存某个响应的 gzip 版本，然后把它发给一个只认纯文本的浏览器，那结果就是一堆乱码。Vary 告诉各级缓存：「针对不同的请求头组合，分开存不同的副本。」

一次完整的真实交互

请求：

GET /api/products HTTP/1.1 Accept: application/json Accept-Encoding: gzip, br Accept-Language: en-IN, en;q=0.9

响应：

HTTP/1.1 200 OK Content-Type: application/json; charset=utf-8 Content-Encoding: br Content-Language: en Vary: Accept-Encoding, Accept-Language Cache-Control: max-age=300

一次往返之内：格式谈妥、压缩谈妥、语言谈妥、缓存定为 5 分钟，而且 CDN 现在也知道要按编码和语言分开存副本。这就是内容协商完整发挥作用的样子。**路由：什么是路由？**

我们前面学的那些 HTTP 方法（GET、POST、PUT、DELETE）表达的是请求的**意图**，比如我想取点数据，或者改点已有的东西。教科书式的定义是：路由是把「HTTP 方法 + URL 路径」这个组合映射到某个特定的服务端处理器（一段指令或业务逻辑）的过程。所以说白了，路由讲的是路径，是「去哪儿」这件事。用更通俗的话说就是：「你到底想访问哪个目的地？」

这里有个唯一性的讲究。服务器会把 HTTP 方法和路由拼起来，形成一个唯一的键。举个例子，发往 \`/api/users\` 的 \`GET\` 请求和发往 \`/api/user\` 的 \`POST\` 请求永远不会冲突（哪怕它们实现的是完全不同的逻辑）。

**路由的类型** 路由分为两类： 1. **静态路由**：就像我们刚看到的 \`/api/users\`。没有任何可变参数，就是一个固定不变的 URL。 2. **动态路由：** 顾名思义，这类路由里含有变量，目标服务器会把它们当作数据来用。这些变量用冒号表示，比如 \`/api/users/:id\`。如果客户端请求 \`/api/users/30\`，服务器就把 "30" 提取出来当作 ID，去取那个特定用户的数据。

**路径参数 vs 查询参数**

向服务器传数据时，你有两种做法：**路径参数（也叫路由参数）：** 直接写在路径上的参数/变量（比如 \`/api/users/30\` 里的 \`30\`）。它们是具体值的占位符，帮服务器精确定位到某个唯一的条目。**查询参数：** 你的 \`GET\` 请求是没有「请求体」的。查询参数被追加在 URL 末尾，为如何处理所请求的数据提供额外指示，本质上是路径之后的一串键值对。语法长这样：\`/resource?key=value\`。用途是过滤、排序、搜索，或者对结果分页。**嵌套路由**

后端开发中的嵌套路由，指的是在一个路由内部再定义 API 端点，形成层级化的父子结构（例如 /users/:userId/posts/;postId），用来刻画资源之间的关系。它把子资源和父资源关联起来，从而提升代码组织性和语义清晰度。**路由版本化与废弃** 路由版本化让 API 能够同时支持多个版本（例如 /v1/、/v2/），从而在演进的同时不破坏已有客户端；具体实现一般走 URL 路径、请求头或查询参数。废弃告警则用来通知客户端某个 API 版本已经过时、即将下线，通常在最终移除之前通过 Deprecation 和 Sunset 这两个 HTTP 头来传达。\[[1](https://docs.fiserv.dev/public/docs/versioning), [2](https://medium.com/@instatunnel/api-versioning-vulnerabilities-the-deprecated-endpoints-still-accepting-requests-3b53631dfad6), [3](https://www.xmatters.com/blog/api-versioning-strategies), [4](https://www.gravitee.io/blog/api-versioning-best-practices), [5](https://openrest.krotscheck.net/api-fundamentals/api-versioning/)\]

**兜底路由（Catch-All Routes）** 兜底路由是一种动态路由，它能匹配多个或未知的 URL 路径，省得你把每条路由都单独定义一遍。当没有任何具体路由匹配上时，它就充当兜底的处理器。兜底路由常用于 404 页面、嵌套动态路由，以及前端 SPA 的路由。比如 Express/React Router 里的 \* 或者 Next.js 里的 \[...slug\]。这篇就到这里啦，各位！
