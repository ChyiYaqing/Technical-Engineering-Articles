---
title: "Design an Authentication System"
url: "https://x.com/Harry_The_Nerd/status/2048758919770816850"
category: "HLD"
date: "2026-04-27"
description: "Designing a scalable authentication and session management system."
lang: "zh-CN"
---

# 设计一个认证系统

> 设计一个可扩展的认证与会话管理系统。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2048758919770816850](https://x.com/Harry_The_Nerd/status/2048758919770816850) · 2026-04-27

![封面图](https://pbs.twimg.com/media/HG6k-YKaUAACvCZ.jpg)

认证（authentication）解决的是系统怎么确认用户是谁。授权（authorization）解决的是他能做什么。这份设计覆盖完整的认证链路——注册、登录、会话管理、token 刷新、OTP 和 OAuth……以及如何在大规模下把它做得足够安全。

**功能需求**

1\. 注册 -\> 安全地保存凭据 2. 登录 -\> 校验凭据、签发 token 3. 会话管理 -\> 用 token 维持登录状态 4. token 刷新 -\> 不用重新登录就续期会话 5. 忘记密码 -\> OTP 验证、重置凭据 6. 登出 -\> 让会话失效

**密码存储：哈希与加盐**

密码绝不能明文存储。如果数据库被攻破，攻击者直接看到 "password123"，那游戏就结束了。解法是哈希。

但光有哈希还有个漏洞。两个用户如果用了同一个密码，算出来的哈希就是一样的。手里有彩虹表（预先算好的常见密码哈希表）的攻击者可以瞬间反推出原文。

加盐能解决这个问题。给每个用户生成一个唯一的随机字符串，拼在密码后面再做哈希：

password = "password123" salt = "xK92p"（每个用户唯一，随机）stored\_hash = bcrypt("password123" + "xK92p") 用户 1：bcrypt("password123" + "xK92p") = "a3f9c2..." 用户 2：bcrypt("password123" + "m7n1q") = "z8p4r1..."

同样的密码，哈希完全不同。彩虹表失效。盐和哈希存在一起，它不是秘密，作用只是让每个哈希都独一无二。用 bcrypt 或 Argon2，两者都内建了加盐，而且被刻意设计得计算很慢，让暴力破解变得不切实际。

**JWT —— JSON Web Token**

登录之后，服务端会签发一个 JWT，这样用户就不必每次请求都带上密码。一个 JWT 由三部分组成：

header.payload.signature header -\> 算法（HS256）payload -\> { userID: "123", role: "admin", exp: 1714000000 } signature -\> hash(header + payload + secret\_key) 证明 token 没被篡改过

JWT 是无状态的，也就是说服务端什么都不存。每个请求都带着 token，服务端用数学方法校验签名，再读取 payload。零数据库查询。每次请求的校验都在亚毫秒级完成。

**刷新 token 模式**

JWT 有一个弱点：一旦签发，在过期之前它都是有效的，被偷走的 token 没法作废。把过期时间设得很短（15 分钟）能限制损失，但用户会被频繁踢下线。解法是用两个 token：

登录 -\> 签发两个 token：Access Token -\> JWT，15 分钟过期 无状态，靠签名校验 不存数据库 Refresh Token -\> 不透明的随机字符串，30 天有效 存在数据库 + Redis 缓存里 每个请求都用 Access Token（快，不查库）Access Token 过期后：-\> 客户端发来 Refresh Token -\> 服务端对着数据库校验 -\> 签发新的 Access Token 登出时：-\> 从数据库删掉 Refresh Token -\> 被偷的 Access Token 最多 15 分钟后失效 -\> Refresh Token 已死 -\> 无法续期

两全其美：短命的 access token 保证安全，refresh token 保证体验。

**OTP —— 忘记密码流程**

用户输入邮箱 -\> 生成 6 位 OTP -\> 把哈希后的 OTP 存进 OTP 表，TTL 10 分钟 -\> 通过通知服务下发（短信或邮件）用户提交 OTP：-\> 校验哈希 -\> 有效且未过期 -\> 允许重置密码 无效？-\> 尝试次数加一 -\> 失败 3 次 -\> 封锁，强制重新发送

OTP 永远以哈希形式存储，绝不存明文。限制尝试次数可以防住暴力破解（6 位 OTP 一共只有 100 万种可能）。

**OAuth 2.0 —— 用 Google 登录**

OAuth 让你的应用把认证委托给一个可信的提供方——Google、GitHub、Apple。四个参与方：用户、你的应用、认证服务器（Google），以及资源服务器（Google 的 API）。

用户点击「用 Google 登录」-\> 你的应用重定向到 Google 认证服务器 -\> Google 向用户请求授权 -\> 用户同意 -\> Google 向你的应用签发 Authorization Code -\> 你的应用拿 code + secret key 换取 Access Token -\> 拉取用户资料（姓名、邮箱、头像）-\> 用户登录完成

之所以要有 Authorization Code 这一步，是因为重定向发生在浏览器里，任何人都看得见。而这个 code 离开你应用的 secret key 就没有用，所以被截获也无害。OAuth 回答的是「谁来为这个用户背书」的问题，JWT 负责的是 token 的格式。两者是配合关系。

**数据层**

三张数据库表加一个缓存

用户表（PostgreSQL）—— userID、email、password\_hash、salt、role、created\_at。只在注册和登录时查询。绝不存明文密码或 token。

Refresh Token 表（PostgreSQL）—— tokenID、userID、refresh\_token、expires\_at、device\_info、created\_at。每个活跃会话一行。登出时删除。

OTP 表（PostgreSQL）—— otpID、userID、otp\_hash、expires\_at、attempts、created\_at。行的生命周期很短，过期后自动删除。

Redis 缓存 —— refresh token 查询（比数据库快）、用户会话数据（角色、邮箱）、用于限流的登录失败次数计数器。

Access Token 在任何地方都不存储。JWT 是自校验的，这正是它的意义所在。

**非功能需求**

**延迟**

每个请求上的 JWT 校验纯粹是数学运算。不查库、不查缓存，亚毫秒级。refresh token 的查询优先走 Redis。只有登录这一条链路会碰 PostgreSQL，而相对于整体请求量，登录发生的频率很低。

**可扩展性**

无状态的 JWT 意味着任何一个认证服务实例都能校验任何一个 token。服务器之间不需要共享会话状态。水平扩展的协调开销为零。少量需要共享的状态（refresh token、限流计数器）交给 Redis。

**安全性**

密码用 bcrypt + 唯一盐做哈希。Access Token 短命（15 分钟）。Refresh Token 以哈希形式存进数据库。OTP 做哈希、限时、限次。登录尝试通过 Redis 计数器限流。OAuth 把密码这份责任完全委托给可信的提供方。以上就是全部内容，各位！
