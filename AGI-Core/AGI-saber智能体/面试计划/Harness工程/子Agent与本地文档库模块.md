# 子 Agent 与本地文档库模块

## 1. 功能目标

目标是让系统不只是“回答问题”，而是可以像一个小型工作流智能体一样：

1. 规划复杂任务。
2. 调用不同子 Agent 协作。
3. 生成 Markdown 报告。
4. 保存到本地文档库。
5. 后续可以重新写入 RAG，成为可检索知识。

## 2. 新增子 Agent

当前内置了 4 个子 Agent：

### research\_agent

负责研究和证据收集。

主要职责：

* 根据用户任务规划检索问题。
* 使用 RAG 或搜索工具获取信息。
* 整理观察结果和证据片段。
* 输出结构化研究摘要。

### writer\_agent

负责写作。

主要职责：

* 接收 `research_agent` 的上游结果。
* 整理为 Markdown 报告。
* 输出摘要、分析、建议和下一步。

### review\_agent

负责审查。

主要职责：

* 检查报告结构。
* 检查事实一致性。
* 找出证据缺口。
* 给出风险和可信度判断。

### doc\_agent

负责文档落库。

主要职责：

* 获取 `writer_agent` 生成的报告正文。
* 提取文档标题。
* 写入本地文档库。
* 尝试同步写入 RAG。
* 将 `review_agent` 的审查内容写入 metadata。

## 3. 典型调用链

对于研究、调研、总结、报告、文档、方案、分析类任务，planner 会优先生成子 Agent DAG。

只研究不保存时：

```latex
research_agent -> writer_agent -> review_agent
```

需要保存报告时：

```latex
research_agent -> writer_agent -> review_agent -> doc_agent
```

真实测试中已经验证过这条调用链：

```latex
research_agent -> writer_agent -> review_agent -> doc_agent
```

### 4. Runtime 改动

任务图节点现在支持两种类型：

* `tool`
* `sub_agent`

图节点新增字段：

* `type`
* `agent_name`
* `goal`

GraphRuntime 执行节点时会判断节点类型：

* 如果是工具节点，继续走原来的 `tool.Execute()`。
* 如果是子 Agent 节点，则从 sub-agent registry 中取出对应 Agent，并调用它的 `Run()`。

上游节点结果会带上 executor 名称，例如：

```latex
n2:writer_agent
n3:review_agent
```

这样 `doc_agent` 可以明确识别哪个上游结果是正文，哪个是审查意见。

## 5. Planner 改动

Planner 增加了对子 Agent 的支持。

LLM planner 输出节点时可以选择：

```json
{
  "type": "sub_agent",
  "agent": "research_agent",
  "goal": "围绕用户任务进行研究",
  "depends_on": []
}
```

为了避免研究类任务被误规划成单个搜索工具，当前对以下关键词采用确定性子 Agent 路由：

* 研究
* 调研
* 总结
* 报告
* 文档
* 方案
* 分析

这样可以保证复杂任务稳定进入子 Agent 工作流。

## 6. 本地文档库

新增了本地文档库领域模型：

### documents

表示稳定的文档实体。

主要字段：

* `id`
* `title`
* `doc_type`
* `source`
* `status`
* `created_by`
* `created_at`
* `updated_at`
* `latest_version`
* `latest_version_id`

### document\_versions

表示文档版本。

主要字段：

* `id`
* `document_id`
* `version`
* `content_md`
* `summary`
* `metadata`
* `created_at`

## 7. 存储策略

文档库支持两种存储方式：

1. PostgreSQL 可用时写入数据库。
2. PostgreSQL 不可用时降级写入本地 `.data/documents`。

本地 fallback 的意义是：即使本机没有启动 PostgreSQL，也可以继续测试文档库和前端查看能力。

## 8. 新增文档工具

新增工具：

* `write_document`
* `list_documents`
* `read_document`
* `ingest_document`

这些工具让 Agent 可以主动写入、读取和重新入库文档。

`write_document` 示例参数：

```json
{
  "title": "子Agent真实调用测试",
  "doc_type": "report",
  "content_md": "# 子Agent真实调用测试\n\n...",
  "source": "agent_generated",
  "ingest_to_rag": true
}
```

## 9. 新增 HTTP API

新增接口：

```latex
GET  /api/documents
POST /api/documents
GET  /api/documents/{id}
POST /api/documents/{id}/ingest
```

用途：

* 列出本地文档。
* 创建或更新文档。
* 查看文档最新版本。
* 将文档重新写入 RAG。

## 10. RAG 元数据

RAG chunk 现在可以记录文档来源：

* `document_id`
* `version_id`
* `section`

这使得后续检索结果可以反向追踪到本地文档库中的完整文档和版本。

## 11. 前端改动

前端新增：

* 左侧“本地文档库”区域。
* 文档列表刷新按钮。
* 文档查看弹窗。
* 文档重新入库按钮。

一个关键修复是：点击本地文档时，不再把文档内容追加到当前聊天，而是打开独立文档查看器。


> 更新: 2026-06-29 11:32:29  
> 原文: <https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/ld9xgiguusbeam2g>