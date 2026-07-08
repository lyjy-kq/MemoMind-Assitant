# mem0 🆚 viking

# 两种不同的 AI Memory 思想流派
虽然最终目标都一样：

```plain
让 Agent “长期记住东西”
```

但是两者是截然不同的思想和实现方式：

1:中间件思想：Mem0 更偏“Memory Middleware”

2:操作系统运行时：Viking 更偏“Memory-Native AI Runtime”

它们不是简单替代关系。

| **对比** | **Mem0** | **Viking** |
| --- | --- | --- |
| 核心定位 | 独立记忆层 | Agent Runtime 内置记忆 |
| 思想 | Memory as Service | Memory as Context Infrastructure |
| 重点 | Memory Retrieval | Context Orchestration |
| 优势 | 开源、通用、易接入 | 工程化强、系统统一 |
| 缺点 | 偏外挂 | 耦合更深 |
| 适合 | 快速集成记忆 | 做完整 Agent OS |
| 本质 | Memory SDK | AI Runtime |

# 一、Mem0 的核心思想
Mem0 最核心的一点：

**“不要存聊天记录，而是存事实”**

这是它真正厉害的地方。

传统 memory：

```plain
把全部聊天存进去
```

然后：

```plain
下一次全塞 prompt
```

问题：

+ token 爆炸
+ latency 爆炸
+ 噪声巨大

Mem0 的思路：

**从对话中“蒸馏”记忆**

例如：

用户说：

```plain
我喜欢 Go
我在东京
我在做 Agent
```

Mem0 不会存整个对话。

而是提取：

```plain
用户喜欢 Go
用户在东京
用户正在开发 AI Agent
```

这叫：

**Fact-based Memory（基于事实的记忆）**

这是 Mem0 的核心创新。 (Mem0)

---

# 二、Mem0 的架构其实很高级
它本质上：

已经不是 Vector DB 了

而是：

“Memory Pipeline”

包括：

```plain
Conversation
    ↓
Memory Extraction
    ↓
Deduplication
    ↓
Entity Linking
    ↓
Memory Store
    ↓
Hybrid Retrieval
```

里面有几个关键点：

---

## ADD-only Memory
Mem0 很重要的设计：

**不覆盖旧记忆**

例如：

```plain
2025：用户喜欢 Java
2026：用户喜欢 Go
```

它不会删。

而是：

```plain
同时存在
```

因为：

**记忆是时间态的**

笔者这个设计非常聪明且有道理。 (Mem0)

---

## Multi-Signal Retrieval
Mem0 不只是向量搜索。

而是：

```plain
Semantic
+
Keyword(BM25)
+
Entity Graph
```

联合召回。 (Mem0)

这是很多简单 memory 系统做不到的。

---

## Async Memory Write
Mem0 后面发现：

**Memory 写入不能阻塞响应**

所以：

```plain
用户响应先返回
Memory 后台异步写
```

这是很典型的生产级优化。 (Agent Market Cap)

---

# 三、Mem0 最大优点
## 工程解耦做得非常好
它不像 LangGraph 那种：

```plain
必须绑定 runtime
```

Mem0：

```plain
memory.add()
memory.search()
```

就能接。

所以：

## 很适合作为中间件
这也是为什么：

现在很多 Agent：

+ CrewAI
+ AutoGen
+ LangChain
+ OpenHands

都能接 Mem0。

---

# 四、Mem0 最大缺点
也是它最大的局限：

## 本质是“外挂 Memory”
即：

```plain
Agent Runtime
    ↓
调用 Mem0
```

所以：

**Memory 不是 Runtime Native**

这会导致：

---

**Context 不统一**

很多时候：

```plain
Prompt Layer
Memory Layer
RAG Layer
Tool Layer
```

是分裂的。

---

## 记忆只是“检索”
而不是：

**Runtime State**

这点非常关键。

Mem0 更像：

```plain
长期事实数据库
```

但：

**Agent 真正需要的是“运行时状态”**

例如：

```plain
任务做到哪一步
工具执行状态
workflow checkpoint
planner state
```

这些：

Mem0 不擅长。

---

# 五、Viking 的思想其实更先进
字节 Viking（包括 Coze/Viking Memory 那套）：

核心思想是：

**Memory 是 Context Infrastructure**

而不是：

**一个外挂 Retrieval System**

---

Viking 更偏：

```plain
统一上下文引擎
```

即：

```plain
User Profile
Memory
RAG
Runtime State
Tool State
Task State
```

统一进入：

Context Assembly Pipeline

而不是：

```plain
需要时 search memory
```


# 六、两者最本质的区别


# Mem0 是“记忆检索”
```plain
Memory -> Retrieval -> Prompt
```

---

# Viking 是“上下文编排”
```plain
Memory
RAG
State
Tool
Workflow
User
    ↓
Unified Context Engine
    ↓
LLM
```



---

# 七、Viking 的优点
---

## 更适合 Agent
因为 Agent 不是聊天机器人。

Agent 需要：

```plain
状态
任务
流程
规划
checkpoint
```

而不是只有：

```plain
用户事实
```

---

Context Native

Memory 不再是外挂。

而是：

Prompt 构建的一部分

这个很高级。

---

## 更容易做复杂工作流
例如：

```plain
Planner
Executor
Reviewer
```

共享状态。

---

# 八、Viking 的缺点
---

## 耦合重
它更像：

一个完整 Runtime

不是：

```plain
一个 SDK
```

所以：

+ 改造成本高
+ 迁移难
+ 灵活性差

---

## 不够通用
Mem0：

```plain
接啥都行
```

Viking：

```plain
更偏字节自己的 Runtime 哲学
```



---

# 九、现在行业已经开始分层
---

## 第一代 Memory
```plain
Conversation Buffer
```

就是存聊天记录。

---

## 第二代 Memory
```plain
Vector Memory
```

embedding 检索。

---

## 第三代（Mem0）
```plain
Fact Extraction Memory
```

提取事实。

---

## 第四代（Viking / Runtime Memory）
```plain
Context Operating System
```

Memory 已经不再是“数据库”。

而是：

**Agent Runtime 的一部分**

---

# 十、业界现在的方向
## 更接近 Viking
而不是 Mem0。

因为大部分在做：

```plain
Memory
+
RAG
+
Context Engine
+
Runtime State
+
Harness
```

这已经不是：

```plain
memory.search()
```



而是：

**AI Runtime Architecture**

---

# 十一、笔者的建议
**借鉴 Mem0 的“记忆提纯”**

**采用 Viking 的“统一上下文”**

这是目前最合理的路线。

即：

---

## 用 Mem0 思想解决：怎么存记忆
包括：

+ fact extraction
+ dedup
+ entity graph
+ semantic retrieval

---

## 用 Viking 思想解决：怎么组织上下文
包括：

+ runtime state
+ planner state
+ task memory
+ tool state
+ context assembly

---

# 十二、真正未来的方向
**Memory 不再是数据库问题**

而是：

**Context Engineering**

问题。

未来 Agent 拼的核心：

不是模型。

而是：

```plain
上下文调度能力
```

谁能：

+ 用更少 token
+ 注入更准上下文
+ 保持长期一致性
+ 管理 runtime state

谁就更强。



> 更新: 2026-05-11 03:18:24  
> 原文: 