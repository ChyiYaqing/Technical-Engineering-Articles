---
title: "Serialization, Deserialization, Authentication and Authorization"
url: "https://x.com/Harry_The_Nerd/status/2053487726033559704"
category: "Backend Engineering"
date: "2026-05-10"
description: "Backend fundamentals: serialization, auth, and authz."
lang: "zh-CN"
---

# 序列化、反序列化、认证与授权

> 后端基础：序列化、认证与授权。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2053487726033559704](https://x.com/Harry_The_Nerd/status/2053487726033559704) · 2026-05-10

![封面图](https://pbs.twimg.com/media/HH9jA11aEAE394e.jpg)

各位，欢迎来到本后端系列的第二篇！在第一篇里，我们聊了 HTTP 和路由相关的东西。现在，我们要学一些后端工程里更有意思、也更重要的能力。

打包与拆包数据的艺术

想象你要给外地的朋友寄一件东西。你没法把它瞬移过去，对吧？你得把它包好、装进箱子、寄出去，然后你朋友在另一头拆开。序列化和反序列化做的正是这件事，只不过对象是数据。

**什么是序列化？**

序列化是把内存里的对象（一个 Python 字典、一个 Java 类实例、一个 JavaScript 对象）转换成可以存储或传输的格式（例如 JSON、XML，或者二进制字节）。

假设你的后端内存里有一个 User 对象：python（举例）

user = User(id=42, name="Harry", role="admin", active=True)

要通过 HTTP 把它发出去，或者存到文件里，你就要序列化它：

json

{ "id": 42, "name": "Harry", "role": "admin", "active": true }

原本内容丰富的复杂对象，变成了一段扁平、可搬运的字符串。它已经「打包好上路了」。

**什么是反序列化？**

反序列化就是刚才那件事的逆过程，各位。把那段扁平的序列化格式拿过来，在内存里重建出原来的对象。

你朋友那边的服务器收到 JSON 字符串，然后这么干：

python

data = json.loads(raw\_json) user = User(\*\*data) Harry 又回来了

字符串重新活成了一个完全可用的对象。它被拆包并重新组装好了。

**那么，后端工程为什么这么在意这件事？**

因为后端就是天天在到处搬数据。你的系统每一天都在：

- 向移动端和浏览器返回 API 响应
- 向 Kafka、RabbitMQ 这类队列投递消息
- 在 Redis 里缓存对象
- 往数据库里写记录
- 通过 gRPC 或 REST 在微服务之间通信

这些事没有序列化一件都干不成。

主流格式

1\. JSON：Web 上使用最广的格式。人类可读、语言无关，到处都支持。

json

{ "product": "Laptop", "price": 999.99, "inStock": true }

优点：容易调试、通用支持。缺点：没有 schema 约束。

2\. XML：老将

更老、更啰嗦，但在企业系统和 SOAP API 里依然是王。

xml

<product\> <name\>Laptop</name\> <price\>999.99</price\> </product\>

3\. Protocol Buffers（Protobuf）：这里的快银（抱歉玩了个梗）

Google 的二进制格式。你先定义 schema（.proto 文件），它会生成紧凑而快速的二进制输出。在微服务架构里非常受欢迎。

proto

message Product { string name = 1; float price = 2; bool in\_stock = 3; }

优点：负载数据体积极小、速度飞快。缺点：不可读，而且需要先配好 schema。

4\. MessagePack：二进制版的 JSON

跟 JSON 很像，但用二进制编码。对于已经在用 JSON 语义的系统来说，是一次无痛的性能升级。

优点：比 JSON 快、更紧凑。缺点：不可读。

5\. Avro：schema 的严格执行者

Apache 出品，在 Kafka 和数据管道领域用得很多。schema 要么内嵌在数据里，要么存在注册中心。

优点：支持 schema 演进、非常适合流式处理。缺点：搭起来更复杂。

顺带一提，序列化和反序列化是每个后端系统里看不见的管道。它们太基础了，以至于大多数开发者都视其为理所当然，直到出事那天。一个字段改名，移动端就崩了；一个二进制格式，让调试变得一头雾水；一个不可信的负载数据，就能利用反序列化器发起攻击。**认证与授权** 说白了就是：「你是谁？」 vs 「你能干什么？」

想象你走进一家高档写字楼（或者你学校的大门）。前台的保安查你的证件，这一步是在确认你是谁。然后他对着一张名单看你被允许去哪几层，这一步是在决定你能干什么。

这个两步流程每天都在每一个后端系统里发生几十亿次。它有个名字：

**认证（AuthN）** 意思是：你是谁？

**授权（AuthZ）** 意思是：你被允许做什么？

它们听起来很像，实际上差别很大。

**认证：证明你的身份**

认证就是验证「你确实是你自称的那个人」的过程。后端不会因为你说一句「我是 Harry」就信你，它需要证据。

经典方案：用户名 + 密码

最老的一招。你把凭据发过去，后端拿它和存下来的（哈希过的）密码比对，然后给你一个会话或者 token。

http

POST /login { "email": "harry@gmail.com", "password": "secret123" }

服务器返回身份证明，也就是你从此随身携带的那个 token。

**这里有条黄金法则：** 永远不要明文存密码。一定要用慢哈希算法，比如 **bcrypt**、**Argon2** 或 **scrypt**。MD5 和 SHA1 不可接受，它们太快了，而快对密码哈希来说是坏事。

Python

（哈希一个密码） hashed = bcrypt.hashpw(password.encode(), bcrypt.gensalt()) （登录时校验） bcrypt.checkpw(entered\_password.encode(), hashed) - True 或 False

**Token 的难题：怎么保持登录状态**

HTTP 是**无状态**的。服务器在两次请求之间什么都不记得。那么登录之后，服务器怎么知道下一个请求还是你呢？

有两个经典答案：

会话（老派做法）

登录之后，服务器在自己的数据库或内存存储（比如 Redis）里创建一条会话记录，然后给你一个 session\_id cookie。

登录 -\> 服务器创建 Session { id: "abc123", userId: 10 } -\> 下发 cookie: session\_id=abc123 下次请求 -\> 浏览器自动带上 cookie -\> 服务器查 "abc123" -\> 找到 userId 10 -\> 搞定！

这种方式撤销起来很容易（把会话删掉就行），但服务器必须存状态，所以在多台服务器之间扩展得不够漂亮。

JWT —— JSON Web Token（现代做法）

服务器不在自己这边存任何东西，而是**给一个 token 签名**后交给客户端。客户端每次请求都把它带回来，服务器只要验签就行，不需要查数据库。

一个 JWT 长这样：

eyJhbGciOiJIUzI1NiJ9.eyJ1c2VySWQiOjQyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV\_adQssw5c

三部分，用点号分隔：

HEADER.PAYLOAD.SIGNATURE Header → { "alg": "HS256", "typ": "JWT" } Payload → { "userId": 100, "role": "admin", "exp": 1735689600 } Signature → HMAC(header + payload, secret\_key)

服务器用一个密钥给 token 签名。只要有人改动了负载数据，签名就对不上了，伪造立刻被发现。

请求 -\> Bearer eyJhbGci... 服务器 -\> 验签 -\> 解码负载数据 -\> userId: 100

无状态且易于扩展，跨微服务用起来非常顺。但过期之前很难撤销（得配个黑名单）。而且密钥一旦泄露，一切都完了。

**多因素认证（MFA）：信任，但要再验一次**

只靠密码太脆弱。MFA 会再加几层证明：

**你知道的东西：** 密码、PIN 码。**你拥有的东西：** OTP 应用（比如 Microsoft Authenticator）、短信验证码、硬件密钥。**你本身的特征：** 指纹、Face ID。

双因素认证（2FA）就是任取其中两项组合。就算你的密码泄露了，攻击者没有你的手机照样登不上。

**OAuth 2.0：「用 Google 登录」**

有时候你根本不想自己处理认证。**OAuth 2.0** 让你的用户通过一个可信的第三方（Google、GitHub、Apple）完成认证，你的后端拿到一个能证明其身份的 token。

用户点击「用 Google 登录」 Google 完成对用户的认证 Google 把一个授权码发给你的后端 你的后端用它换取访问 token 你就知道用户是谁了，而且全程没见过对方的密码

做认证的不是你，你只是把这件事**委托**给了 Google。这就是 OAuth 2.0 的核心思路，而 OpenID Connect（OIDC）建在它之上，专门用来处理身份。

**授权：你被允许做什么**

系统知道你是谁之后，还得决定你能访问什么。这就是授权，它既是安全问题，也同样是个设计问题。

**基于角色的访问控制（RBAC）**

最常见的模型。给用户分配角色，角色拥有权限。

角色： admin：可以读、写、删除任何东西 editor：可以读和写文章 viewer：只能读文章 用户 Harry -\> 角色: editor 用户 Dimps -\> 角色: viewer Harry 试图删除一篇文章 -\> 拒绝 Harry 试图编辑一篇文章 -\> 允许

清晰、好理解、好维护。对大多数应用都很够用。

**基于属性的访问控制（ABAC）**

更强大也更灵活。访问决策基于用户、资源和环境的各种属性，按策略求值得出。

python

\# Can this user access this document? def can\_access(user, document, action): return ( user.department == document.department and user.clearance\_level \>\= document.sensitivity and action in user.allowed\_actions and current\_time() within business\_hours() )

粒度极细，能处理复杂的条件规则。但更难推理，也更难审计。

**基于权限 / ACL（访问控制列表）**

不走角色这条路，而是让每个资源自己带一份「谁能对我做什么」的名单，就像 Unix 文件权限或 Google Docs 的共享设置。

文档 "Q4\_Report.pdf": Harry - 读、写 Dimps - 读 公开 - 无权访问

对单个资源的控制非常精确。但规模一上来就很难管（几百万资源 × 几千用户）。

**授权在代码里怎么落地**

实际工程中，授权校验会在多个层次上执行：

1\. 中间件 / 路由层

python

[@app](https://x.com/app).route("/admin/users") [@require\_role](https://x.com/require_role)("admin") # <\- Gate at the door def list\_users(): return User.query.all()

2\. 服务 / 业务逻辑层

python

def delete\_post(requesting\_user, post\_id): post = Post.get(post\_id) if [post.author](https://x.com/Harry_The_Nerd/status/post.author)\_id != requesting\_user.id and not requesting\_user.is\_admin: raise PermissionError("You can't delete someone else's post") post.delete()

3\. 数据库 / 查询层

python

\# Never return data the user shouldn't see, even accidentally posts = Post.query.filter\_by(author\_id=current\_user.id).all()

只在路由层校验、其余全靠信任，是个经典错误，会导致「不安全的直接对象引用」（IDOR）漏洞：用户猜到别人的资源 ID，就能随意访问。

最佳实践速览

**用 bcrypt/Argon2 哈希密码：** 慢哈希能挫败暴力破解。**用短有效期的 JWT + 刷新 token：** 万一 token 被偷，损失可控。**永远在资源层面做授权：** 别只查是不是登录了，要查是不是被允许。**给登录接口限流：** 防止暴力破解和撞库。**全站启用 HTTPS：** 明文 HTTP 里的 token 被偷易如反掌。**最小权限原则：** 只给用户真正需要的权限。**记录认证相关事件：** 登录失败、token 刷新、权限拒绝，全都值得审计。

一张图看全流程（请忽略我的字迹）

![](https://pbs.twimg.com/media/HH9zb4YasAEGemQ.jpg)

认证和授权是整个系统的守门人。这两块做错了，你的数据库表结构设计得多优雅、API 跑得多快都没意义了——用户数据泄露，公司要担责，信任荡然无存。

就这些啦，各位！
