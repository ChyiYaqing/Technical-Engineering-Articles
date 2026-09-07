---
title: "Validation, Transformation, Controllers, Middlewares"
url: "https://x.com/Harry_The_Nerd/status/2053882356071911853"
category: "Backend Engineering"
date: "2026-05-11"
description: "Request handling patterns: validation, transformation, controllers, middleware."
lang: "zh-CN"
---

# 校验、转换、控制器与中间件

> 请求处理模式：校验、转换、控制器、中间件。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2053882356071911853](https://x.com/Harry_The_Nerd/status/2053882356071911853) · 2026-05-11

![封面图](https://pbs.twimg.com/media/HICvLyCbYAAsW_Z.jpg)

这一篇我们来聊聊任何后端系统里都很重要的几个核心构件

**校验**

校验是你的第一道防线。它保证进入系统的数据在被任何业务逻辑碰到之前，结构完整、字段齐全、取值在可接受范围内。

校验大致存在于两个层面：

**schema 校验：** 检查结构和类型相关的东西。这个字段是字符串吗？这个数字在范围内吗？这个邮箱地址语法上合法吗？像 Zod（TypeScript）、Joi（Node.js）、Pydantic（Python）或 Jakarta Bean Validation（Java）这些库，让你可以声明一个 schema，然后以声明式的方式做校验。

**业务校验：** 它依赖上下文。这个用户名被占用了吗？这个用户的余额够完成这笔交易吗？这类校验往往需要查数据库，应该待在服务层深处，而不是最外层。

一个常见错误是把两者混在一起：在 schema 校验器里塞一次数据库调用，或者把业务规则零零散散撒在控制器里。要把它们分开。schema 校验应该是廉价且无状态的；业务校验则是领域逻辑中经过深思熟虑的一部分。

校验失败时应该大声、尽早地报出来，并给出清晰的错误信息。丢回一个含糊的 400 Bad Request、却不告诉客户端哪里错了，纯属浪费大家时间。要把所有违规项都收集起来（不只是第一个），并用统一的结构返回。

**转换**

原始的输入数据很少正好是你系统想要的形状，原始的输出数据也很少正好是客户端想要的形状。转换就是用来弥合这些落差的。

在**输入侧**，你可能需要：

- 去掉字符串首尾的空白
- 把邮箱统一转成小写
- 把日期字符串解析成真正的 Date 对象
- 把逗号分隔的字符串转成数组

在**输出侧**，你可能需要：

- 在发给客户端之前剥掉内部字段（比如 passwordHash）
- 把数据库里的 snake\_case 列名改成 camelCase，以便返回 JSON
- 把嵌套的数据库行拍平成更简单的 DTO
- 把时间戳格式化成 ISO 8601 字符串

关键在于：转换应该是显式的，而不是不知不觉发生的。当你直接把数据库实体丢进响应里时，你其实做了一次隐式转换，而且经常会泄露内部细节。为输入和输出定义清晰的 **DTO（数据传输对象）**。NestJS 里的 class-transformer、Django REST Framework 里的序列化器，都是把这件事正规化的工具。

一个干净的分层是这样的：原始请求 -\> 校验并转换后的 DTO -\> 领域对象 -\> 转换后的响应 DTO -\> 原始响应。

**控制器**

控制器是应用 HTTP 层的入口。它的职责很窄：接收请求、委派给服务、返回响应。就这些。

控制器里不应该有业务逻辑，不应该直连数据库，也不应该去判断数据意味着什么。它只负责编排：调用正确的服务方法、传入正确的参数、把结果映射成 HTTP 响应。

// Thin controller async createUser(req, res) { const dto = await validate(CreateUserDto, req.body); const user = await this.userService.create(dto); res.status(201).json(toUserResponse(user)); }

臃肿的控制器是一种坏味道。当你的控制器开始引入 repository、写条件逻辑，或者直接编排多个服务时，它就变得没法测试，也违反了单一职责原则。

控制器还负责 HTTP 层面的事情：状态码、响应头、内容协商。服务不该知道、也不该关心自己是被 HTTP 调用的，它同样可能被一条 CLI 命令或者一个任务队列调用。

**服务**

服务是业务逻辑的所在地，是应用的心脏，是把「你的系统究竟做什么」编码下来的那一层。

一个服务方法要回答的是这类问题：「用户下单时会发生什么？」「退款怎么处理？」「账号停用有哪些规则？」答案都在这里（不在控制器里，也不在 repository 里）。

服务负责：

- 编排多次 repository 调用
- 落实业务规则与不变量
- 触发副作用（发邮件、发布事件）
- 管理事务

// Service handles business logic async placeOrder(userId, items) { const user = await this.userRepo.findById(userId); if (!user.isActive) throw new ForbiddenError("Account suspended"); const order = Order.create(user, items); await [this.orderRepo.save](https://x.com/Harry_The_Nerd/status/this.orderRepo.save)(order); await this.emailService.sendOrderConfirmation(order); return order; }

服务应该是无状态的。不要把某个请求范围内的数据存在服务实例上。服务通常是单例，而单例上的状态就是一个随时会爆的并发 bug。

要把服务单独拿出来测试。既然它的依赖是注入进来的，也不碰 HTTP，写单元测试就既干净又快。

**Repository**

Repository 抽象了持久化层。它的契约很简单：进去的是领域对象，出来的也是领域对象，中间把所有数据库实现细节都藏起来。

为什么这很重要：你的业务逻辑不该关心你用的是 PostgreSQL、MongoDB 还是一个平文件。如果你的服务里写着裸 SQL，或者串着某个 ORM 专属的查询构造器，那它就和数据库耦合上了。换数据库，甚至只是想单独做测试，都会变得很痛苦。

Repository 的接口是以领域为中心的：

interface UserRepository { findById(id: string): Promise<User | null\>; findByEmail(email: string): Promise<User | null\>; save(user: User): Promise<void\>; delete(id: string): Promise<void\>; }

实现里去对付 ORM 或裸 SQL，应用的其余部分只跟接口打交道。

Repository 里不该有业务逻辑。getUsersWhoSignedUpInTheLastWeekAndHaveNotMadeAPurchase 是一次查询，不是一条业务规则，它属于 repository。但「拿这些用户来干什么」的决策，属于服务。

一个常见的坑：把 repository 做得太泛（只有一个 find(criteria) 方法），或者太具体（每个用例界面配一个方法）。目标是让方法自然对应到领域操作上。

中间件

中间件位于请求管道中，在控制器之前（或之后）执行。它是横切关注点的正确归宿，也就是那些适用于很多路由、却不专属于任何一条路由的逻辑。

中间件常见的职责：

- **认证**：校验 JWT 或会话 token，把用户挂到请求上
- **日志**：记录请求方法、路径、耗时、状态码
- **限流**：按客户端统计并节流请求
- **CORS**：为跨域请求设置正确的响应头
- **注入请求 ID**：给每个请求打上唯一 ID，便于追踪
- **请求体解析**：在到达控制器之前把 JSON 或表单数据解析好

中间件应该可组合，并且顺序要刻意安排。认证中间件必须跑在授权中间件之前；日志中间件多半想把所有东西都包在里面。顺序很关键，而且应当是显式的。

不要把业务逻辑塞进中间件。中间件不了解你的领域，它只了解请求和响应。如果你发现自己在中间件里判断「这个用户是不是管理员」来把守某个资源层面的决策，那就走过头了——那是授权逻辑，应该待在服务里，或者一个专门的 guard/策略层里。教你一个我自己的小窍门：中间件负责修改或丰富请求/响应管道；服务负责决定拿这个被丰富过的请求做什么。

**请求上下文**

请求上下文是一种机制，让你在整个调用栈里携带单次请求的状态，而不必把它当参数一层层往下传。

设想一下：认证中间件识别出了调用方用户；服务需要知道调用方是谁才能落实权限；审计日志需要知道谁做了什么。如果没有请求上下文，你就得把 currentUser 作为参数传给链路上的每一个函数……又乱又脆。

换个做法：把数据挂在一个作用域仅限单次请求生命周期的上下文对象上：

// Middleware attaches it app.use((req, res, next) =\> { const user = verifyToken(req.headers.authorization); req.context = { userId: [user.id](https://x.com/Harry_The_Nerd/status/user.id), role: user.role, requestId: uuid() }; next(); }); // Service accesses it via injection async deletePost(postId) { const { userId } = this.requestContext.get(); const post = await this.postRepo.findById(postId); if (post.authorId !== userId) throw new ForbiddenError(); // ... }

在 Node.js 里，AsyncLocalStorage 能让这件事跨越异步边界也安全可用，完全不用手动传递。在 NestJS 这样的框架里，一个 REQUEST 作用域的 provider 就能搞定。在 Java/Spring 里，ThreadLocal 干这活儿已经干了几十年。

哪些东西该放进请求上下文：已认证的用户、请求 ID、租户 ID（多租户应用里）、语言区域，以及分布式追踪的 trace 上下文。

哪些不该放：任何可以从领域模型推导出来、或者可以延迟计算出来的东西。别把应用状态硬塞进请求上下文。

这一切是怎么拼起来的

请求的生命周期，分层清晰（我尽力了 :p）：

![](https://pbs.twimg.com/media/HIDaKS4agAAS5eJ.jpg)

请求上下文纵向贯穿所有层，携带着身份和追踪数据。转换发生在两端：进来时净化输入，出去时塑形响应。每一层都清楚自己的职责，也只干自己的职责。

当这些关注点被清晰分离，系统就变得可预测了。出问题时你知道该去哪儿找；需求变了你知道该往哪儿加逻辑；写测试时你知道该 mock 什么。

就这些啦，各位，干杯！
