---
title: "Python's Inner Working - Behind The Scenes"
url: "https://x.com/Harry_The_Nerd/status/2078116519029129646"
category: "Engineering Articles"
date: "2026-07-17"
description: "How Python executes code under the hood."
lang: "zh-CN"
---

# Python 的内部运作——幕后揭秘

> Python 在底层是如何执行代码的。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2078116519029129646](https://x.com/Harry_The_Nerd/status/2078116519029129646) · 2026-07-17

![封面图](https://pbs.twimg.com/media/HMtU-24bkAA20V5.jpg)

大多数人写好 Python 代码，运行，看到输出。感觉很简单。你敲下 print("Hello")，点运行，文字就出现在屏幕上。但从敲下那一行到看见结果之间，发生了很多事情。我们用平实简单的语言，一步步走完这整段旅程。

## 第 1 步：你写下源代码

一切都从一个 .py 文件开始。这个文件只是纯文本。计算机并不会把纯文本当成指令来理解，它只懂机器码，也就是一堆针对你 CPU 的 1 和 0。所以 Python 必须把你可读的代码转换成机器真正能运行的东西。

## 第 2 步：分词（词法分析）

真正的第一步叫做分词。Python 逐个字符地读取你的源代码，把它切成一个个小片段，这些片段叫做 token。

举个例子，下面这一行：

x = 5 + 3

会被切成这样一串 token：

- x（一个名字）
- \=（一个运算符）
- 5（一个数字）
- \+（一个运算符）
- 3（一个数字）

可以把 token 看成代码里的「单词」。就像句子由单词组成一样，你的代码由 token 组成。这一步还不关心语义，它只负责识别出这些片段。

## 第 3 步：语法分析与构建 AST

有了 token 之后，Python 需要理解代码的结构和语法。这就轮到解析器（parser）登场了。

解析器接收这些 token，把它们组织成一棵树形结构，叫做 AST，即抽象语法树（Abstract Syntax Tree）。这棵树表示代码的语法结构，就像一个句子有主语、谓语、宾语一样。

对于 x = 5 + 3，这棵树在概念上大致长这样：

Assign ├── Target: x └── Value: BinOp ├── Left: 5 ├── Op: + └── Right: 3

这棵树让 Python 很容易看清哪个操作依赖哪个操作。如果你的代码有语法错误，比如少了个冒号或者括号没闭合，通常就是在这个阶段被 Python 抓住并抛出 SyntaxError。

你其实可以用 Python 内置的 ast 模块亲眼看看这棵树：

```python
import ast
tree = ast.parse("x = 5 + 3")
print(ast.dump(tree))
```

## 第 4 步：编译成字节码

现在 Python 有了一棵干净的树形结构，接下来它把这棵树编译成所谓的字节码（bytecode）。字节码是一套层次较低、经过简化的指令集。它和机器码不是一回事，但已经离机器能快速处理的东西近多了。

当你导入模块时，这些字节码会存放在名为 \_\_pycache\_\_ 的目录下的 .pyc 文件里。这就是为什么第二次运行程序时，启动有时会稍快一些。只要源文件没变，Python 就不需要重新分词和解析，直接复用已保存的字节码即可。

你可以用 dis 模块查看任意函数的字节码：

```python
import dis

def add(a, b):
    return a + b

dis.dis(add)
```

输出大致是这样：

LOAD\_FAST a LOAD\_FAST b BINARY\_ADD RETURN\_VALUE

这些指令都是简单的基于栈的操作。Python 在对自己说：加载这个值，加载那个值，把它们相加，然后返回结果。

## 第 5 步：Python 虚拟机（PVM）

重点来了。你的 CPU 并不直接执行字节码。执行它的是一个叫 Python 虚拟机的系统，常缩写为 PVM。它本质上是一个用 C 写的大循环（标准 Python 也就是 CPython，是用 C 语言编写的），逐条读取字节码指令，并执行对应的动作。

这个循环在内部有时被称为「eval loop」。可以把它想成站在你的字节码和实际硬件之间的一个翻译。它读到一条像 BINARY\_ADD 这样的指令，理解它的含义，然后调用正确的 C 函数，用你的 CPU 真正完成这次加法。

这也是 Python 被认为是解释型语言的原因。不像 C 或 Rust 那样直接编译成 CPU 自己就能跑的机器码，Python 始终需要中间这台虚拟机，动态地翻译指令。

## 第 6 步：基于栈的执行

Python 虚拟机用一种叫做栈（stack）的结构，在执行指令时跟踪各个值。栈是一种很简单的结构：最后放进去的东西最先取出来，就像一摞盘子。

回到前面那个例子：

LOAD\_FAST a # push value of a onto the stack LOAD\_FAST b # push value of b onto the stack BINARY\_ADD # pop both values, add them, push result RETURN\_VALUE # pop and return the result

Python 里的每一个操作，无论你的代码看起来多复杂，最终都会被拆解成这些微小的栈操作。

## 第 7 步：帧与调用栈

Python 里每次调用函数，都会创建一个新的帧（frame）。帧就像一个小工作区，里面装着：

- 该函数的局部变量
- 正在执行的字节码
- 一个指向当前执行位置的指针
- 一个指回调用方帧的引用

这些帧一个摞一个，构成了所谓的调用栈（call stack）。如果函数 A 调用函数 B，B 又调用函数 C，你就得到三个叠起来的帧。C 执行完毕后，它的帧被移除，控制权回到 B 的帧，以此类推。

这也是为什么 Python 里过深的递归会引发 RecursionError。每次递归调用都会新增一个帧，而能堆叠的帧数是有上限的。

## 第 8 步：内存管理与引用计数

在这一切执行的同时，Python 还在幕后管理内存。Python 里的每个对象，无论是数字、字符串、列表还是自定义类的实例，都有一个引用计数。这个数字记录了程序中当前有多少处指向这个对象。

```python
a = [1, 2, 3]   # reference count of the list becomes 1
b = a           # reference count becomes 2
del a           # reference count becomes 1
del b           # reference count becomes 0, memory is freed
```

当引用计数降到零，Python 就知道再没人需要这个对象了，于是立刻释放内存。

## 第 9 步：垃圾回收器

单靠引用计数有个弱点。它处理不了这种情况：两个对象互相引用，但没有别的地方引用它们中的任何一个。这叫做引用循环。

```
class Node:
    def __init__(self):
        self.other = None

a = Node()
b = Node()
a.other = b
b.other = a
```

即使你在代码里删掉了 a 和 b，它们仍然互相引用，引用计数自己永远降不到零。为了应对这种情况，Python 另有一套系统叫垃圾回收器，专门寻找这类循环并定期清理。

## 第 10 步：全局解释器锁（GIL）

还有一个重要部件是 GIL，即全局解释器锁（Global Interpreter Lock）。在标准 Python（CPython）里，任意时刻只有一个线程能执行 Python 字节码，哪怕你的机器有很多个 CPU 核心。

它的存在主要是为了让内存管理，尤其是引用计数，保持安全和简单。没有 GIL 的话，两个线程可能同时去修改同一个引用计数，把数据搞坏。

这就是为什么 Python 线程很适合那些大量时间在等待的任务，比如下载文件或读数据库，却不适合需要同时进行大量 CPU 计算的任务。要做真正并行的 CPU 计算，人们通常改用多进程而不是多线程，比如用 multiprocessing 模块。

把整条链路串起来（从头到尾的总结）：

你在一个 .py 文件里写下源代码

Python 把代码分词成一个个叫 token 的小片段

解析器用这些 token 构建出抽象语法树

编译器把这棵树变成字节码

字节码被缓存到 \_\_pycache\_\_ 中，让以后的运行更快

Python 虚拟机逐条读取字节码指令

每条指令都通过一套基于栈的系统执行

函数调用创建出一个个互相堆叠的帧

引用计数自动跟踪并释放内存

垃圾回收器清理残留的引用循环

GIL 确保同一时刻只有一个线程碰字节码

这个过程能解释很多你作为开发者每天都会遇到的真实现象：

- 为什么 Python 比 C 这类编译型语言慢，因为多了虚拟机这一层翻译
- 为什么第一次导入大模块感觉慢，之后就快了，因为有字节码缓存
- 为什么深递归会报错失败，因为帧栈有上限
- 为什么线程没能像你预期的那样加速 CPU 密集型任务，因为有 GIL
- 为什么内存通常能自己清理得很干净，多亏引用计数和垃圾回收器的配合

Python 把这些复杂性全都藏在一门看起来干净简单的语言背后。但简单之下，是一条设计精良的流水线，一步一步、小心翼翼地把你可读的代码变成机器真正能执行的东西。

以上就是全部内容，干杯！！

欢迎点赞、评论、分享和转发！！
