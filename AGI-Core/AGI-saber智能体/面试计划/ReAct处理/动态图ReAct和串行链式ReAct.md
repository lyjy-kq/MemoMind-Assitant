# 动态图ReAct和串行链式ReAct

前言：👀这篇文档之前建议去力扣刷一下课程表（[207. 课程表 - 力扣（LeetCode）](https://leetcode.cn/problems/course-schedule/description/)）这道题，会对你有意想不到的帮助！

## 串行链式 ReAct（Linear / Sequential ReAct）
传统 ReAct 本质：

```plain
Thought -> Action -> Observation
         ↓
Thought -> Action -> Observation
         ↓
Thought -> Action -> Observation
```

即：

+ 一步一步串行推进
+ 当前步骤完成后才能进入下一步
+ 上下文是线性的
+ 每次只维护一个执行路径

典型特点：

| 特征 | 串行链式 ReAct |
| --- | --- |
| 执行方式 | 单链路串行 |
| 状态结构 | Linear Context |
| 推理模式 | 单线程思考 |
| Tool调用 | 顺序调用 |
| 容错 | 弱 |
| 可恢复 | 一般 |
| 并行能力 | 无 |
| 复杂任务 | 容易爆context |


---

### 适合场景
适用于：

+ 简单问答
+ 单工具任务
+ 一次性推理
+ 短链路 Agent

例如：

```plain
用户：北京天气？

Thought: 需要查询天气
Action: weather_api(Beijing)
Observation: 26℃
Answer: 北京26℃
```

这是经典 ReAct。

---

## 动态图 ReAct（Graph-based ReAct）
动态图 ReAct 的核心：

不是“链”。

而是：

```plain
一个动态扩展的任务图（Task Graph）
```

结构类似：

```plain
Root Thought
    │
    ├─► Tool A ──► Result
    │
    ├─► Tool B ──┬─► Subtask B1 ──► B1 Result
    │            └─► Subtask B2
    │
    └─► Tool C ──┬─► Memory Recall
                 └─► Planner (Re-plan)
```

它本质是：

```plain
ReAct + DAG Runtime + Stateful Scheduler
```

---

核心区别

| 维度 | 串行链式 ReAct | 动态图 ReAct |
| --- | --- | --- |
| 执行结构 | 链表 | DAG/Graph |
| 推理方式 | 单路径 | 多路径 |
| Context | 单上下文 | 节点级上下文 |
| Tool调用 | 串行 | 可并行 |
| 状态管理 | Prompt里硬塞 | Runtime State |
| 容错 | Retry困难 | 节点恢复 |
| 长任务 | 容易崩 | 可持续执行 |
| Memory | 临时上下文 | 图状态持久化 |
| 调度 | 无 | Scheduler |
| 适合 | Chat Agent | Production Agent |


---

### 为什么动态图更先进
因为真实 Agent 任务不是线性的。

例如：

```plain
“帮我分析公司财报并生成投资建议”
```

实际上会拆成：

```plain
1. 拉取财报
2. 拉取行业数据
3. 拉取新闻
4. 分析风险
5. 分析增长
6. 汇总生成报告
```

这里天然就是 DAG：

```plain
      拉取数据
       /   \
 风险分析  增长分析
       \   /
      最终汇总
```

不是线性链。

---

### 动态图 ReAct 的关键组件
1. Planner（任务规划）

先把任务拆图：

```plain
Goal -> Subtasks
```

例如：

```json
{
  "nodes": [
    "search_news",
    "analyze_finance",
    "generate_report"
  ]
}
```

---

2. Runtime State

每个节点有独立状态：

```python
NodeState:
    status
    retries
    outputs
    dependencies
```

而不是全部塞 prompt。

---

3. Scheduler（调度器）

决定：

+ 哪些节点可以执行
+ 哪些依赖完成
+ 哪些失败重试
+ 哪些并行

类似：

```plain
Airflow / Ray / Temporal
```

思想。

---

4. Memory Graph

记忆不再只是：

```plain
conversation history
```

而是：

```plain
Task Graph State
+
Semantic Memory
+
Working Memory
```

例如：

```plain
节点A输出 -> 节点D输入
```

直接图连接。

---

一个直观例子

串行 ReAct

```plain
Thought
↓
Search
↓
Observation
↓
Thought
↓
Code
↓
Observation
```

像：

```plain
流水线工人
```

---

动态图 ReAct

```plain
           Planner
               ↓
    ┌──────────┼──────────┐
 Search      Code      Memory
    │           │          │
    └──────Merge/Summary───┘
```

像：分布式调度系统

```plain
最直观的感受，是执行速度加快了，因为毕竟并行度增加了
```

---

## 区别
### 串行链式 ReAct
本质：

```plain
Prompt 驱动推理
```

---

### 动态图 ReAct
本质：

```plain
Runtime 驱动智能体
```

即：

LLM 不再直接控制全部流程。

而是：

```plain
LLM 负责认知
Runtime 负责执行
Scheduler 负责调度
Memory 负责状态
```

这是现代 Agent 的核心方向。

---

## 业界演进趋势
大致路线：

```plain
Chain-of-Thought
    ↓
ReAct
    ↓
Plan-and-Execute
    ↓
Graph-based Agent
    ↓
Stateful Runtime Agent
```

代表：

+ OpenAI Deep Research
+ Anthropic Computer Use
+ LangChain LangGraph
+ Microsoft AutoGen
+ Temporal Technologies Temporal Agent Runtime

基本都在：

```plain
ReAct Runtime 化
```

而不是继续做 Prompt chaining。

## 动态图的实现方法
```plain
Kahn Topological Sort
```

是业界 DAG Runtime 最常见的一种实现思路。

本质流程：

```plain
1. 找入度为0的节点
2. 执行
3. 删除边
4. 更新其他节点入度
5. 继续调度
```

但真正的 Agent Runtime / Workflow Engine 里，会在这个基础上再加：

+ 状态机
+ 调度器
+ 优先级
+ 并行池
+ 重试机制
+ Future/Promise
+ 依赖追踪

最后就演化成：

```plain
DAG Scheduler Runtime
```

---

最基础 DAG 执行模型

例如：

```plain
A → C
B → C
C → D
```

---

数据结构

一般：

```python
class Node:
    id
    deps        # 依赖谁
    nexts       # 后继节点
    indegree    # 入度
```

图：

```python
graph = {
    "A": ["C"],
    "B": ["C"],
    "C": ["D"]
}
```

入度：

```python
A:0
B:0
C:2
D:1
```

---

### 经典 Kahn 算法
Step1 初始化队列

把：

```plain
indegree == 0
```

的节点放进去。

即：

```plain
[A, B]
```

---

Step2 执行节点

执行：

```plain
A
```

然后：

```plain
A -> C
```

这条边被“删除”。

于是：

```plain
C indegree:
2 -> 1
```

---

再执行：

```plain
B
```

则：

```plain
C indegree:
1 -> 0
```

于是：

```plain
C 可执行
```

进入队列。

---

为什么很多系统用小顶堆

因为：

```plain
多个节点同时可执行
```

时，需要：

```plain
调度策略
```

例如：

| 策略 | 小顶堆排序依据 |
| --- | --- |
| BFS | depth |
| 优先级 | priority |
| 最早创建 | timestamp |
| 最短任务优先 | cost |
| AI Agent | importance score |


所以：

```python
heapq.heappush(heap, (priority, node))
```

而不是普通 queue。

---

### 现代 Runtime 会怎么做
真正复杂的是：

```plain
节点执行不是同步函数
```

而是：

```plain
Future / Async Task
```

例如：

```python
async def execute(node):
    result = await tool_call()
```

于是 Scheduler 会：

```plain
提交任务
↓
等待完成
↓
更新依赖
↓
激活下游节点
```

这已经接近：

+ Ray DAG
+ Airflow
+ Prefect
+ Temporal
+ LangGraph

了。

---

### Agent Runtime 比普通 DAG 多的东西
普通 DAG：

```plain
节点 = 函数
```

但 Agent：

```plain
节点 = LLM Cognitive Step
```

于是节点状态会变成：

```python
class AgentNode:
    thought
    action
    observation
    retries
    memory
    tool_result
```

不是单纯函数调用。

---

### 真正工业级 Scheduler
一般会维护：

1. Ready Queue

```plain
所有 indegree=0 且未执行节点
```

---

2. Running Set

```plain
当前执行中的节点
```

---

3. Finished Set

```plain
完成节点
```

---

4. Failed Set

```plain
失败节点
```

---

5. Dependency Map

```plain
dependency_map = {
    "C": ["A", "B"]
}
```

---

# 


> 更新: 2026-05-28 18:57:19  
> 原文: <https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/rdbcbr1vy2oaenr6>