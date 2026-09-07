---
title: "A Complete Guide For Claude Certified Architect Foundations (CCA-F) Exam"
url: "https://x.com/Harry_The_Nerd/status/2069050669487809004"
category: "Engineering Articles"
date: "2026-06-22"
description: "Guide covering the Claude Certified Architect Foundations exam."
lang: "zh-CN"
---

# Claude Certified Architect Foundations (CCA-F) 考试完全指南

> 覆盖 Claude Certified Architect Foundations 考试的指南。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2069050669487809004](https://x.com/Harry_The_Nerd/status/2069050669487809004) · 2026-06-22

![封面图](https://pbs.twimg.com/media/HJVXhp1boAAWU-x.jpg)

终极 Claude Certified Architect 指南（我写的）

用 Claude 构建生产级应用，需要跳出对话式聊天提示词的思维，把大语言模型当成确定性引擎来对待。我尝试把官方的五个考试领域拆解成清晰、技术化、易于理解的蓝图。

## Domain 1：Agentic 架构与编排

**1\. 吃透 Agentic Loop 和停止原因**

任何自主智能体的核心都是 **Agentic Loop**。Claude 不会随机决定什么时候停止说话、什么时候调用工具。相反，Anthropic Messages API 通过明确的、确定性的字符串信号来通信，这个信号叫 **stop\_reason**。

\[ 你的应用 \] -\> 发送 payload -\> \[ Claude 引擎 \] ▲ │ │ ▼ └─────────── 判断 stop\_reason ────────┘ • tool\_use：运行本地代码 • end\_turn：交付最终答案

架构师必须让应用外壳在每次 API 响应的信封上检查这个 stop\_reason 字段：

- **tool\_use**：这个信号表示 Claude 已经停止生成对话文本，正在等你去运行一段代码。模型会给出确切的工具名和对应的 JSON 参数。你的应用必须拦下它，执行本地代码，再把结果喂回去。
- **end\_turn**：这个信号确认 Claude 已经走完推理路径，给出了最终答案。此时循环可以安全结束。

**2\. 多智能体模式与 Hub-and-Spoke 设计**

对于复杂应用，把整个项目历史塞进单个提示词会造成严重的**上下文污染**。标准的设计模式是 **Hub-and-Spoke 架构**，也叫**编排者-工作者模式（Orchestrator-Worker Pattern）**。

在这种布局里，一个主 **Orchestrator Agent** 充当管理者。它接收用户请求，拆成一个个独立任务，再派生出高度聚焦的 **Worker 子智能体**去隔离执行这些任务。

要让系统保持快速且节省 token，就用**「按需知情」的上下文沙箱**策略。绝对不要把主对话历史往下复制给工作者子智能体。而是只抽取该工作者需要的那几个变量，用一个干净的迷你 payload 传过去，等它返回结构化答案的瞬间就销毁这个子智能体的沙箱上下文窗口。

**3\. 用 SDK 生命周期钩子加固流水线**

在企业环境里运行的 AI 智能体必须被硬边界圈住。**生命周期钩子（Lifecycle Hooks）**是写在你本地 SDK（比如 TypeScript 或 Python）里的程序化代码块，它们运行在 Claude 的概率式推理之外。

- **执行前输入校验钩子**：这类钩子在用户文本抵达 Anthropic API 之前就拦截它。它们扫描提示词注入攻击、恶意格式或违禁关键词，一旦发现违规立刻阻断这一轮。
- **执行后 payload 校验钩子**：这类钩子在 Claude 的输出触发你基础设施中的动作之前把它抓住。如果 Claude 生成的参数违反了你的系统安全规则，钩子就取消这次操作，让底层网络毫发无损。

## Domain 2：Claude Code 配置与工作流

**1\. 三层 CLAUDE.md 层级**

在使用 Claude Code 这类智能体开发助手时，系统通过一棵结构化的三层配置树来映射你的工程偏好。越具体越优先，也就是说发生冲突时低层级会安全地覆盖高层级。

1\. 全局用户级（~/.config/claude/rules.md）-\> 通用的个人偏好 2. 仓库项目级（仓库根目录的 /CLAUDE.md）-\> 核心技术栈与构建脚本 3. 本地目录级（子文件夹中的 /\*\*/CLAUDE.md）-\> 针对性的微服务规则

一个常见错误是把团队规则存进个人全局配置空间。因为这些设置只留在你本地的笔记本上，队友的智能体拿不到必需的上下文，构建循环就会崩掉。生产环境里的正确做法是，把 /CLAUDE.md 文件直接提交到远程 Git 仓库的根目录，让整个团队共享同一套规则。

**2\. 用斜杠命令和 Skills 扩展能力**

为了绕开自然语言的模糊性，你可以在项目配置 payload 里构造**自定义斜杠命令**。斜杠命令相当于硬编码的意图路由器。

```json
{
  "commands": {
    "review": {
      "description": "Performs a security audit on code changes.",
      "system_prompt_overlay": "You are a Principal Security Architect. Scan files for OWASP vulnerabilities.",
      "allowed_tools": ["file_read", "file_grep"]
    }
  }
}
```

当开发者输入 /review 时，系统会跳过标准分类流程，直接注入你指定的 system\_prompt\_overlay。

为了让这套东西可维护，把这些各自独立的行为打包成模块化的 **Agent Skills** 文件，放进专门的 /.claude/skills/ 目录。这样核心工作流逻辑就和主配置资产解耦了。

**3\. CI/CD 流水线中的无头自动化**

要在 GitHub Actions 这类持续集成与持续部署环境里跑 Claude Code，必须使用非交互式的执行方式。如果你在虚拟化的构建节点里直接触发裸的 claude 命令，runner 会卡死或抛出致命的终端错误。

不容商量的规则是使用 **\-p**（或 --print）参数。它强制 Claude Code 进入无头模式：接收一个输入字符串，锁掉终端交互提示，完全自主地跑完任务，把遥测数据直接输出到 stdout，然后以明确的系统退出码结束。

## Domain 3：提示词工程与结构化输出

**1\. 为什么 tool\_use 胜过基于提示词的 JSON**

如果你想靠在提示词里写「只返回一个合法的 JSON 对象」来强制拿到结构化数据，你的应用迟早会遇到解析器崩溃。Claude 可能会加上 markdown 包裹、留一个尾逗号，或者附上一段对话文字。

专业做法是改用原生的 **tool\_use（Function Calling）** API 层。当你向 API 注册一个明确的工具 schema 时，Anthropic 会修改 Claude 底层的 token 概率分布。模型在采样层被物理性地约束成只能输出符合你 schema 的数据，语法错误、markdown 包裹和多余的客套话被彻底消除。

**2\. 限制性 JSON Schema 设计**

要阻止 Claude 在你的 JSON 对象里幻觉出随机或前后不一的数据值，你必须写高度限制性的 schema。能用严格的逻辑护栏锁死字段时，就别用泛泛的基础类型。

```json
{
  "status": {
    "type": "string",
    "enum": ["PENDING", "APPROVED", "REJECTED"]
  },
  "quality_score": {
    "type": "number",
    "minimum": 0.0,
    "maximum": 1.0
  }
}
```
- **enum（枚举）**：把合法取值限定在一个严格的许可字符串数组里。
- **minimum 和 maximum**：给数值字段设死下限和上限，防止指标越界。

**3\. Few-Shot 学习的 2 到 4 个示例法则**

Few-shot 提示词在教 Claude 复杂模式时效果极好，但塞太多示例会带来严重的架构负担。你应该严格遵守 **2 到 4 个示例法则**。

给 8 个或 10 个示例，会让每一次 API 调用都消耗过多输入 token，还会造成**过拟合**——Claude 会盯住示例表面的机械形式，而不是背后的抽象规则。

另外，绝不要把宝贵的 few-shot 位置浪费在显而易见的简单场景上。把示例全部聚焦在**灰色地带**，也就是表面现象与你真实业务逻辑相冲突的那些高度模糊的边界线上。

**4\. 强制的结构化推理标签**

如果你的 few-shot 示例从输入文本 payload 直接跳到最终输出标签，就限制了模型的分析能力。永远要在 few-shot 样本里直接嵌入一个明确的 <reasoning\> 块。

```xml
<example>
  <input>Error: Timeout encountered during TLS handshake.</input>
  <reasoning>
    1. The error occurs during the TLS handshake phase.
    2. TLS operations exist at the transport network layer, prior to application logic.
    3. Therefore, this must map to the NETWORK infrastructure category.
  </reasoning>
  <output>{"category": "NETWORK"}</output>
</example>
```

在示例里强制这种思维链结构，会训练 Claude 内部的注意力层在运行时走同样的分步逻辑，从而大幅减少幻觉。

## Domain 4：工具设计与 MCP 集成

**1\. Model Context Protocol（MCP）蓝图**

**Model Context Protocol（MCP）**是一个开放标准，它把 AI 模型和数据层解耦。你不必为每个新数据库或内部平台写定制的、脆弱的 API 集成，而是构建或接入独立的 **MCP Server**。

\[ Claude Agent 客户端 \] -\> （标准化的 MCP 协议）-\> \[ MCP Server \] -\> \[ 你的私有数据库 \]

MCP Server 扮演一道安全、隔离的边界闸门。它向客户端暴露三个干净的原语：

- **Prompts**：预配置的提示词模板，把任务的表述方式标准化。
- **Resources**：只读的数据点（比如本地文件路径或 API 端点），Claude 可以安全地读取。
- **Tools**：可执行函数，让 Claude 在明确的安全约束下，在你的环境里执行安全的动作。

**2\. 进阶的工具错误处理与重试循环**

当一个工具在多步骤智能体流水线中失败时，你必须能区分两种不同类型的错误：

**错误分类核心判定方法解决策略语法错误**结构是否违反了基本解析规则（比如 JSON.parse()）？在本地快速修代码，或强化结构 schema。**语义错误**结构合法，但里面的数据违反了业务规则？通过**自纠正重试循环**把它送回给 Claude。

要实现自纠正循环，就在代码里拦下这次校验失败。不要清空对话历史。而是用明确的 **tool\_result** 角色类型往对话数组里追加一条新消息，把 is\_error 参数设为 true，并把完整的错误堆栈字符串原样传回给 Claude。

```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "tool_u_123XYZ",
      "is_error": true,
      "content": "Validation Failed: 'URGENT' is not a valid enum member. Allowed values are ['CRITICAL', 'MAJOR']."
    }
  ]
}
```

Claude 会分析自己上一次的错误，读取你的错误遥测信息，并在紧接着的下一轮生成结构正确的响应。

## Domain 5：上下文管理与可靠性

**1\. 击败上下文膨胀和「中间迷失」效应**

大语言模型每一轮都要从头处理整段文本历史，而它们内部的召回效率呈现出一条明显的曲线：对提示词最开头的 token（系统指令）和最末尾的 token（当前用户回合）召回率极高。埋在大上下文窗口中间那三分之一的信息，注意力会严重衰减。这就是所谓的**「中间迷失」（Lost in the Middle）效应**。

\[示意图：展示中间迷失效应，上下文窗口的顶部和底部注意力很高，中间部分注意力衰减\]

上下文膨胀的头号推手是没管住的工具响应。如果你的智能体查了一次数据库，就把成千上万 token 的原始 JSON 元数据塞进对话数组，你的核心规则就会被挤到低注意力的中间区域。

要彻底解决这个问题，实施两种上下文工程模式：

- **钩子层的工具输出裁剪**：用一层本地代码中间件，在把内容追加进对话历史之前，剥掉不必要的数据库字段、截断长数组，把庞杂的 JSON 块压平成高度紧凑、省 token 的 Markdown 表格。
- **Case Facts Block 模式**：每一轮都用程序抽取出一个小而结构化的摘要框，里面装着最关键的项目变量，并把它重新钉在**最新那条用户消息**的最顶部。这就把要害事实从正在褪色的中间区域拽出来，塞进模型当前计算中高注意力的处理层。

**2\. 多轮批处理架构**

在处理大规模工作负载时（比如审计 5 万条后台交易记录），跑同步的实时 API 调用会浪费大量资金。**Anthropic Message Batches API** 正是为这类高吞吐、异步的工作流设计的。

Batch API 直接给你输入和输出 token 各 **50% 的成本折扣**，代价是一个弹性的 **24 小时完成窗口**。

因为批处理是异步执行的，你必须在摄取用的 JSONL 文件里使用 **custom\_id 设计模式**，把 Claude 顺序打乱的响应映射回你自己数据库的记录主键：

```json
{"custom_id": "product_sku_1092A", "params": {"model": "claude-3-5-sonnet-20241022", "messages": [...]}}
```
- **什么时候该用 Batch**：夜间的法务合规审查、数据富化流水线、大规模 ETL 转换，以及**多轮评审模式**——第一阶段生成分析，第二阶段批判这些结论。
- **什么时候不该用 Batch**：合并前的 CI/CD 阻断关卡、实时聊天用户界面，以及**多轮工具调用循环**。因为 Batch API 完全无头地跑在 Anthropic 的集群里，客户端没有任何中间件在运行，无法拦截一次工具调用并在流程中途把结果注回去。批处理文件里的每一行都被严格限定为单次、孤立的请求-响应回合。

**3\. 程序化熔断器与人工升级触发条件**

为了防止自主智能体陷入校验反复失败的无限循环、烧光你的 API 预算，你必须实施严格的**熔断器（Circuit Breaker）**模式。把系统限制在任一任务步骤最多 **2 次连续重试**。如果智能体在两轮反馈循环内还无法自我纠正，就触发熔断，把这笔事务转入人工复核队列。

**不容商量的升级法则**：如果在对话流中的任何时刻，人类用户明确要求转人工客服、输入了「support」，或者要求退出 AI 交互，编排框架**必须**立即停止所有自动化处理，把整个事务上下文移交给人工坐席。你不该试图用提示词工程搞一层劝说逻辑，而应该在紧接着的下一轮就把会话转走。

**4\. 论断-来源映射与时间数据完整性**

在设计综述类或报告类智能体时，你面临很高的**归因丢失（Attribution Loss）**风险——这种失效模式是指，智能体陈述了一个正确的事实，却把它挂到了错误的源文档上。

要消除归因丢失，你的 JSON 输出 schema 必须强制要求明确的**论断-来源映射（Claim-Source Mapping）**。除非 Claude 能用程序把一条叙述性论断和一段精确的原文引用锚点、以及匹配的文档标识字符串配对起来，否则就禁止它生成这条论断。

最后，你还必须防范**时间数据陷阱**。除非你的基础设施明确给出时间锚点，否则模型对时间完全没有状态感知。为了不让 Claude 把 2019 年的一个财务统计数据当成当下的事实，你的摄取流水线必须执行两步的**时间锚定模式**：

用程序给每一份原始文档文件打上明确的 metadata\_timestamp 头，然后再把它注入上下文窗口。

把当前的绝对日历日期直接注入根系统提示词的头部字符串（例如 Current Time Anchor: Monday, June 22, 2026）。

这就给了 Claude 计算时间相关性所需的明确时间基线，让它自动优先采用新鲜数据，而不是过期的上下文资产。

## 考试技巧

- **Domain 1**：用程序检查 stop\_reason 变量（tool\_use 还是 end\_turn）。用沙箱式上下文隔离来防止多智能体之间的污染。
- **Domain 2**：把 /CLAUDE.md 提交到仓库根目录以同步团队规则。用 claude -p 做无头的 CI/CD 自动化。
- **Domain 3**：用原生 tool\_use schema，而不是在提示词里用文字提要求，才能保证 JSON 合法。把 2 到 4 个 few-shot 示例严格聚焦在灰色地带的边界情况上，并且必须包含 <reasoning\> 标签。
- **Domain 4**：用 MCP server 干净地暴露 prompts、resources 和 tools。用带 is\_error: true 的 tool\_result 块来实现自纠正的语义循环。
- **Domain 5**：用钩子层的工具输出裁剪和 Case Facts Block 击败「中间迷失」效应。对单回合的异步任务用 Message Batches API 拿 50% 成本折扣。用户明确提出要求时立即升级到人工。

就这些了，各位……加油！！
