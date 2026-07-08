# 任务 DAG 设计

## 一. 整体链路

### 1.1 DAG 在 Agent 中处于什么位置

```plain
              ┌─────────────────────────────┐
              │   ChatOptions / 路由决策    	│
              └──────────────┬──────────────┘
                             ▼
            ┌───────────────────────────────┐
            │ 4 个 Mode：chat/tool/rag/react	│
            └────────────────┬──────────────┘
                             ▼
                      ┌──────────────┐
                      │  ReAct Mode  │ ◄── DAG 在这里登场
                      └──────┬───────┘
                             │
                ┌────────────┼────────────┐
                ▼            ▼            ▼
            Planner       GraphRuntime  Generator
            LLM 规划      并行+竞速调度  LLM 合成
                │            │            │
                ▼            ▼            ▼
          一组 planNode   Topological     最终答案
          (带依赖+竞速)   级并行执行     综合所有观察
```

DAG **只服务于 ReAct 模式**——多步推理 + 多工具协作的场景。其他三个模式（chat/tool/rag）是单步或线性流程，不走 DAG。

### 1.2 三个核心抽象

| 抽象 | 职责 |
| --- | --- |
| **Node** | 单个执行单元（工具调用 / 子 Agent / 思考 / 聚合） |
| **TaskGraph** | 有向无环图，带邻接表 + 入度表 + 拓扑层缓存 |
| **GraphRuntime** | 调度器：拓扑分层 + 信号量并发 + 竞速执行 |

### 1.3 Node 的关键字段

```go
type Node struct {
    ID         NodeID            // 节点唯一标识
    Type       NodeType          // tool / sub_agent / think / aggregate
    Name       string            // Planner 给出的 reason（人类可读）
    ToolName   string            // tool 节点必填
    AgentName  string            // sub_agent 节点必填
    Goal       string            // 子 Agent 的任务目标
    Params     map[string]string // 工具参数
    DependsOn  []NodeID          // 入边：依赖哪些节点
    RaceGroup  string            // 空=独立；同 group=竞速
    Status     NodeStatus        // pending/running/done/failed/skipped/cancelled
    Result     string
    Error      string
    RetryCount int
}
```

**两个超出常规 DAG 的字段**：

* RaceGroup ：同组节点**竞速**执行，首个成功的获胜，其余被取消
* AgentName + Goal ：节点可以是**子 Agent**，不只是工具

### 1.4 三阶段执行链路

```plain
┌────────────────────────────────────────────────────────────┐
│  Stage 1：Planning（Planner LLM 规划）                     	 │
│  query + 工具集 + 子 Agent → JSON 节点数组（含依赖和竞速）    	 │
│  失败降级：rulePlanNodes 关键词规则                         	 │
└──────────────────────────┬─────────────────────────────────┘
                           ▼
┌────────────────────────────────────────────────────────────┐
│  Stage 2：Build & Validate（构图）                         	 │
│  NewTaskGraph 计算邻接表 / 入度                            	 │
│  Validate 检测环 + 悬空依赖                                	 │
│  校验失败降级：DependsOn 清空 → 全并行                         │
└──────────────────────────┬─────────────────────────────────┘
                           ▼
┌────────────────────────────────────────────────────────────┐
│  Stage 3：Execute（GraphRuntime 调度）                    	 │
│  TopologicalLevels Kahn 算法分层                          	 │
│  每层：按 RaceGroup 分组 → 信号量并发 → 竞速 / 普通          	 │
│  每节点：状态机 + 重试 + ctx 中断 + TaskMem 写入             	 │
└──────────────────────────┬─────────────────────────────────┘
                           ▼
                  ┌─────────────────┐
                  │ Generator LLM   │
                  │ 合成最终答案     	│
                  └─────────────────┘
```

### 1.5 端到端时序

```plain
用户提问 → 路由到 ReAct 模式 
   │
   ▼
llmPlanGraph                     
   ├─ 构造工具描述 + 子 Agent 描述
   ├─ Planner LLM 输出 planNode JSON
   ├─ 失败兜底：解析 legacy 格式 / 关键词规则 rulePlanNodes
   └─ 过滤：只保留实际存在的工具/Agent
   │
   ▼
NewTaskGraph                      
   ├─ 注册节点 → AdjList / InDegree
   └─ Validate (悬空依赖 + 环检测)
   │
   ▼  失败降级：DependsOn=nil 全并行重建
   │
   ▼
GraphRuntime.Execute              
   │
   ├─ TopologicalLevels (Kahn 算法)
   │  levels[0] = [n1, n2]
   │  levels[1] = [n3]   (依赖 n1, n2)
   │  levels[2] = [n4]   (依赖 n3)
   │
   ├─ 对每一层：
   │   ├─ groupByRace 分组
   │   ├─ 同 RaceGroup → raceGroup goroutine（竞速）
   │   ├─ 独立节点 → executeNode goroutine（普通）
   │   ├─ 信号量 sem 限并发
   │   ├─ wg.Wait 等本层完成
   │   └─ saveSnapshot 持久化
   │
   ├─ executeSingleNode：
   │   ├─ 推送 node_start / thought / action SSE
   │   ├─ run() → 工具 t.Execute 或 子 Agent sa.Run
   │   ├─ 失败重试 maxRetries 次
   │   ├─ TaskMem.Push + ToolTracker.Record
   │   └─ 推送 node_done / observation SSE
   │
   └─ buildResult / buildInterruptedResult
   │
   ▼
llmGenerate                      
   └─ 把所有 observations 喂给 Generator LLM
   │
   ▼
最终答案 + ReActStep 列表
```

## 二. 本项目的独到之处

### 2.1 运行时 DAG 而非编译时 DAG

**LangGraph**：图结构在代码里写死，运行时不变。

```python
graph.add_node("a", ...)
graph.add_edge("a", "b")  # 编译期定义
```

**本项目**：图结构**由 LLM 动态产出**——同一个用户问题，不同的 LLM 输出可能产生完全不同的图。

```go
planNodes := a.llmPlanGraph(ctx, query, ts, memPrefix)
tg := graph.NewTaskGraph(planNodes)
```

**好处**：

* Agent 真正"会规划"——不是按固定流程走
* 工具集变化时 Planner 自动适配，不用改代码
* 用户问"研究 X 写报告"，Planner 自动生成 research → writer → review 三节点链

**代价**：

* 需要降级路径（LLM 解析失败 → 规则 → 全并行）
* 调试比 LangGraph 难——图结构每次都可能不同

### 2.2 同层竞速（RaceGroup）

这是 LangGraph 完全没有的能力。

**场景**：用户问"X 是什么"，可以同时调 `search_web` 和 `rag_search`——谁先返回用谁。

**Planner 输出**：

```json
[
  {"id":"n1","tool":"search_web","race_group":"search","depends_on":[]},
  {"id":"n2","tool":"rag_search","race_group":"search","depends_on":[]}
]
```

**调度细节**：

```go
ch := make(chan raceResult, len(g.NodeIDs))
raceCtx, cancel := context.WithCancel(ctx)

for _, nodeID := range g.NodeIDs {
    go func(id graph.NodeID) {
        res, err := rt.executeSingleNode(raceCtx, id)
        ch <- raceResult{nodeID: id, result: res, err: err}
    }(nodeID)
}

// 首个成功 → 取消其余
winnerFound := false
for i := 0; i < len(g.NodeIDs); i++ {
    r := <-ch
    if r.err == nil && !winnerFound {
        winnerFound = true
        cancel()                                      // 关键：取消其余
        rt.results[r.nodeID] = r.result
        rt.graph.SetNodeStatus(r.nodeID, graph.StatusDone)
    }
}
```

**好处**：

* **延迟降低**：N 路并发，wall-clock = min(各路延迟)
* **可靠性提升**：单源故障不致命，其他源继续
* **Cypher 级语义**：First-success-wins 比 LangGraph 的"全等"语义更适合 agent 场景

**例子**：用户问"研究 React 18 写一份报告并保存到知识库"——Planner 生成：

```plain
n1 [sub_agent: research_agent]  ← 研究
n2 [sub_agent: writer_agent]    ← 写报告，depends_on: [n1]
n3 [sub_agent: review_agent]    ← 审查，depends_on: [n2]
n4 [sub_agent: doc_agent]       ← 保存到 RAG，depends_on: [n2, n3]
```

### 2.3 拓扑分层 vs 单步推进

**LangGraph**：每次推进一个"step"，状态机驱动，需要 conditional edge 决定下一步。

* 优点：可控性强、便于回放
* 缺点：并发需要手动 Send，复杂度上来

**本项目**：Kahn 算法分层，**同层自动并行**。

```go
levels[0] = [n1, n2]    // 入度=0，可立即执行
levels[1] = [n3]        // 入度=2（依赖 n1, n2），等 L0 完成
levels[2] = [n4]        // 入度=1（依赖 n3）
```

**调度伪代码**：

```go
for levelIdx, level := range levels {
    groups := rt.groupByRace(level)
    var wg sync.WaitGroup
    for _, g := range groups {
        if g.RaceGroup != "" {
            wg.Add(1)
            go rt.raceGroup(ctx, g, &wg)
        } else {
            for _, nodeID := range g.NodeIDs {
                wg.Add(1)
                go rt.executeNode(ctx, nodeID, &wg)
            }
        }
    }
    wg.Wait()
}
```

**好处**：

* **自动并行**：开发者不用思考"哪些可以并发"
* **延迟下界**：wall-clock = sum(各层最长节点) 而非 sum(全部节点)
* **简单确定**：一层完成才进下一层，状态清晰

####


> 更新: 2026-06-30 17:28:15  
> 原文: <https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/lrfvxx45p6qm1g9p>