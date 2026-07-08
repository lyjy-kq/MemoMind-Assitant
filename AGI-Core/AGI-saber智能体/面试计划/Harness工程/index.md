# Harness工程

**Harness（容错执行 Runtime）**

这个部分本质上：

在做一个 AI Agent 的可靠执行引擎。

不是简单 retry。

而是：

+ 有状态
+ 可恢复
+ 可追踪
+ 可编排

的执行系统。

---

# Harness 解决什么问题
我们介绍这样开场：

LLM Agent 本质上是一个长链路 IO 系统。

每一步：

+ LLM 调用
+ Tool 调用
+ MCP
+ Web Search
+ RAG Retrieval

都是不稳定 IO。

所以我引入了 Harness Runtime 去解决：

+ 超时
+ 抖动
+ 工具失败
+ Agent 中断
+ 上下文丢失
+ 长任务恢复
+ 多步骤一致性

这7个点面试的时候多说一个点就多加十分。

接下来是每个点的解决方案

---

# Harness Runtime：问题与解决方案
---

# 1. 超时（Timeout）
## 问题
LLM / Tool 本质是远程 IO：

+ 网络不稳定
+ 模型推理慢
+ Tool 卡死
+ 外部 API 无响应

如果不控制：

+ 整个 Agent Workflow 阻塞
+ Planner 无法推进
+ Runtime State 长时间占用

---

## 解决方案
---

### （1）分层超时
不同任务：

不同 timeout。

例如：

| 类型 | 超时 |
| --- | --- |
| LLM | 30s |
| Web Search | 10s |
| DB Query | 3s |
| Sandbox Exec | 60s |


避免：

```plain
一个慢 Tool 拖死整个 Agent
```

---

### （2）超时状态流转
超时后：

```plain
running -> timeout -> retrying/failed
```

而不是进程直接挂。

---

# 2. 抖动（Jitter / Retry Storm）
| 概念 | 本质 | 例子 |
| --- | --- | --- |
| 超时（timeout） | 超过可接受时间上限 | Tool 30 秒没返回 |
| 抖动（jitter/latency variance） | 响应时间不稳定 | 同一个 Tool 一会 2s，一会 15s |


## 问题
同一个tool，执行时间忽快忽慢

---

## 解决方案
### （1）自适应恢复
不能无脑重试，根据具体错误具体分析，实现方法就是靠返回日志信息的形势

如果是网络波动，则可以重试

但如果是工具参数错误、权限错误、模型输出格式错误，这种逻辑问题就不会继续重试，而是直接终止或者重新规划步骤，避免任务一直卡死。

### （2）软失败
直接终止的tool任务不能导致整个流程失败，尽量保证整个流程跑完，让generator可以给出最终结果，并告诉我们哪个工具执行失败



# 3. 工具失败（Tool Failure）
## 问题
Tool 不可靠：

+ HTTP 500
+ JSON 格式错误
+ MCP 崩溃
+ Tool 返回 hallucination
+ Tool schema 不匹配

---

## 解决方案
---

## （1）Tool Isolation
每个 Tool：

独立执行上下文。

失败不会影响整个 Runtime。

---

## （2）统一 Tool Result Schema
你可以讲：

我没有让 Tool 返回自由文本，而是统一成结构化 Result。

例如：

```json
{
  "success": false,
  "error": "timeout",
  "retryable": true
}
```

这样 Planner 能决策：

+ retry 
+ fallback 
+ skip

---

## （3）Fallback Strategy
例如：

```plain
SearchTool 失败→ fallback 到 cached memory
```

或者：

```plain
主模型失败→ fallback 小模型
```

---

# 4. Agent 中断+长任务恢复（Crash / Interrupt）
## 问题
Agent 是长链路：

可能：

+ 服务重启
+ 容器销毁
+ Worker crash
+ Runtime panic

如果没有恢复：

整个任务丢失。

---

## 解决方案
### （1）快照保存
Agent 每执行一步，都会把当前状态保存到 Redis 里，包括：

+  当前执行到哪一步
+  每个任务规划状态 
+  已经成功的工具调用结果 
+ 运行时用户的query变量

这样即使服务突然重启、进程崩溃或者任务被打断，也不会整个任务丢失。

恢复的时候，系统会先去 Redis 读取之前保存的快照，然后重新加载任务状态，从上一次中断的位置继续执行，而不是让大模型从头重新推理。因为很多任务本身是长链路的，如果每次失败都全部重来，成本会非常高，而且上下文也容易漂移。

## （2）事件记录机制
 agent生命周期里的关键动作都会按顺序记录下来，比如：

+  planner （start到running到done） 
+  工具调用 （start到running到done）
+  快照保存 （start到running到done）

这些事件只追加、不修改。这样任务崩溃之后，可以根据事件记录重新恢复整个执行过程，也方便后面做调试和问题排查。

---

# 5. 上下文丢失（Context Loss）
## 问题
LLM 有 Context Window 限制。

长任务：

会出现：

+ Tool Result 被截断
+ Planner 忘记历史
+ Memory 污染
+ Prompt 漂移

---

## 解决方案
---

因为大模型的上下文窗口是有限的，一个 Agent 如果执行很多轮，很容易出现：

+ 工具结果太多被截断
+ 任务规划忘记之前做过什么
+ 旧信息污染当前推理
+ 提示词方向逐渐偏移

所以我没有直接把所有聊天记录无脑塞给模型，而是做了动态上下文组装。

系统会根据当前任务，动态挑选真正需要的信息，例如：

+ 当前运行状态
+ 当前任务规划
+ 和任务相关的历史记录
+ 最近的工具结果
+ 当前阶段的短期记忆

这样可以减少无关信息干扰，也能降低上下文长度。

另外我把记忆分成了几层：

第一层是原始对话，保存完整历史；

第二层是长期记忆，提取稳定事实，比如用户长期偏好、长期任务目标；

第三层是运行时记忆，只保存当前任务短期状态。

这样不同类型的信息不会混在一起，模型也更容易保持推理稳定。

最后，对于过长的旧上下文，我会做压缩处理，比如摘要、语义合并，只保留核心信息，避免上下文越来越大导致推理质量下降。

---

# 
# 6. 多步骤一致性（Multi-step Consistency）
---

## 问题
Agent：

不是单步调用。

而是：

```plain
plan
→ tool
→ memory
→ reasoning
→ action
```

如果中间一步失败：

可能出现：

+ Memory 已写入
+ Tool 已执行
+ Planner 未更新

导致状态不一致。

---

# 解决方案
---

### （1）显式状态流转
因为 Agent 不是一次模型调用就结束，而是：  
任务规划  
→ 工具调用  
→ 记忆更新  
→ 推理  
→ 执行动作

这样一步一步推进的。

如果中间某一步失败，就很容易出现状态不一致的问题。比如：

+ 工具已经执行成功了
+ 记忆已经写入了
+ 但任务状态还没更新

这时候如果直接重新执行，就可能出现重复写入、重复调用工具，甚至状态混乱。

所以我把整个 Agent Runtime 做成了显式状态机。每个任务都会有明确状态，比如：



start- running-done/fall



这样任务状态变化是可追踪、可恢复的，而不是隐藏在代码流程里。



### （2）幂等性设计
比如每次工具调用都会生成唯一调用编号，即使任务恢复后再次执行，也不会重复触发已经成功的操作，避免副作用重复发生。

---

# 最后一个总结
---

我后来发现 Agent 最大的问题并不是 模型和prompt。

而是：

如何让一个不稳定的认知系统，在真实生产环境里可靠运行。

所以我开始把 Agent 当成：

一个分布式、有状态、可恢复的 Runtime 系统去设计。



> 更新: 2026-06-17 21:52:26  
> 原文: <https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/eem938c7rw31md15>