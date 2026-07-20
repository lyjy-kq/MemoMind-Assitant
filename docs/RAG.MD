# RAG测选型
## chunk的切分策略
##向量化模型的选择

# 怎么抑制幻觉
①检索层用Reranker提精度；②Prompt约束"只基于资料回答，
不知道就说不知道"；③生成后用Faithfulness评估验证。

# RAG流程

## step 1: 改写（Rewriter）

1) 先把当前问题改写成一句**自包含的独立查询**（消除指代、补全省略）。
2) 再生成若干条**等价但措辞不同**的查询变体（同义词替换、抽象/具体切换、不同语序）。

输出**严格 JSON**，不要任何说明文字、不要 markdown 代码块：
{"queries": ["独立查询", "变体1", "变体2","原查询"]}


## step 2-4: 三路检索 + 一级 RRF + 二级 RRF（核心）

#### 一级 RRF（单条 query 内三路融合）

每条 query 都单独跑一次"三路检索 + 三路 RRF"

RRF公式（必背，面试肯定问）：

**score(d) = Σᵢ wᵢ / (k + rankᵢ(d))**

* `d` = 一条候选文档
* `i` = 第 i 路检索器
* `rankᵢ(d)` = 文档 d 在第 i 路检索结果里排第几（从 0 开始）
* `k` = 平滑常数，默认 60，越大越削弱前几名优势
* `wᵢ` = 第 i 路的权重（语义和关键词都是 1.0，KG 用 `cfg.KGWeight`）

#### 二级 RRF（rerank）

一级 RRF 算完一条 query 的结果。但我们有 N 条 query，结果怎么合？


\*\*为什么需要二级 RRF？\
\*\*Rewriter 生成的 N 条 query 召回的 chunk 大部分相同，但顺序略有差异。简单合并 + 排序会有"重复 + 顺序不稳定"问题。RRF 让"被多条 query 同时命中的 chunk"自然排前——符合用户的潜在意图，因为 N 条 query 都想找它。

 候选池 = 4×topK：rerank 需要"宽召回"

\_**如果 topK=5，没 reranker 时召回 10 条，有 reranker 时召回 20 条。为什么这么设计？\ **\_rerank 能"精排"但不能"召新"。如果一开始召回的 10 条里就没有正确答案，rerank 再厉害也变不出来。所以**有 rerank 时召回宽一点**，让真实答案有更高概率落进候选池。

### step 5: LLM 精排（Reranker）

***这一步在干嘛？ ***\
让 LLM 给候选池里每一条 chunk 打一个 0-10 的分，按分数重排，截到 topK。

```go
你是检索系统的精排器。给定用户问题和若干候选段落（每条带编号 idx），
判断每条段落对回答该问题的**相关性 + 信息密度**，给 0~10 的整数分。

打分准则：
- 10：直接回答了问题
- 7~9：包含明确相关事实
- 4~6：弱相关 / 部分相关
- 1~3：仅出现共现关键词，不能用来回答
- 0：无关 / 噪声

输出**严格 JSON**...
{"scores": [{"idx": 0, "score": 9}, {"idx": 1, "score": 3}, ...]}
```

**关键设计：****LLM**** 分数主排序，原 RRF 做二级分类

**另一个关键兜底**reranker.go#L108-114：LLM 漏给某条打分时，给它一个"低分但不是负无穷"的兜底值（-1）。**不丢弃**——避免 LLM 漏判把正确答案删掉。

### step 6: small-to-big 父块回填

***这一步在干嘛？***\*\*\
**之前检索命中的是**子块\*\*（200 字精准命中），但喂给 LLM 看的应该是**父块**（800 字完整上下文）。这一步就是做这个替换。

***为什么不直接用大块检索？***\*\*\
\*\* 关键在 embedding 的特性：

* 把一段 800 字塞进 embedding 模型，输出一个固定维度的向量。
* 这段 800 字里同时讲了 ABC 三件事，向量是**三件事的"语义平均"**。
* 你问 A，问题向量和这个"平均向量"的距离是中等——既不近也不远。
* 而那段 200 字的子块里只讲 A 这一件事，向量集中指向 A，问 A 时它命中得很准。


### step 7: 拼提示词喂 LLM

**最后一步**：把若干个父块拼成 context，套上 system prompt 和用户问题，调 LLM 生成最终答案

***代码 rag.go#L284-300：***

```go
context := strings.Join(parts, "\n\n")
if context == "" {
    return "知识库中未找到相关内容。", results
}
if e.generateFn != nil {
    systemPrompt := "你是一个基于知识库回答问题的助手。请仅根据提供的上下文内容回答问题，不要编造信息。如果上下文不足以回答，请说明。"
    askQuery := question
    if len(queries) > 0 && queries[0] != "" {
        askQuery = queries[0]   // ← 用独立化后的版本
    }
    userMsg := fmt.Sprintf("上下文：\n%s\n\n问题：%s", context, askQuery)
    answer := e.generateFn(systemPrompt, userMsg)
    return answer, results
}
return fmt.Sprintf("【知识库检索结果】\n%s", context), results
```

**关键细节 1**：\
system prompt 明确要求"仅根据上下文回答，不编造"——这是防 LLM 幻觉的最后一道闸。\
**关键细节 2**：\
`askQuery` 用 Rewriter 的第一条（独立化后的版本），不用原始的"上面讲的微服务……"。因为独立化版本对 LLM 来说更好理解。\
**关键细节 3**：\
`generateFn == nil`（没注入 LLM）时**不会崩**，而是直接把检索原文返回。这是又一处优雅降级。

***总结：Query 是「改写 → 三路检索 → 一级 RRF（路间）→ 二级 RRF（query 间）→ LLM 精排 → small-to-big 父块回填 → 喂 LLM 合成」，每一步失败都有兜底。***

