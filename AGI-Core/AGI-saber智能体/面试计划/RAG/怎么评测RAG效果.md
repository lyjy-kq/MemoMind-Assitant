# 怎么评测RAG效果

# 省流：
测这些

```plain
Recall@K	看能不能召回来
MRR	看排序质量
NDCG	看整体排序质量
HitRate	是否命中
```

RAG 评测其实分四层：

```plain
1. Retrieval（检索）
2. Rerank（重排）
3. Generation（生成）
4. End-to-End（整体问答）
```

工业界最重要的是：

```plain
先测 Retrieval
再测 Generation
```

因为很多问题根本不是 LLM 的锅。

---

# 一、先理解：RAG 最容易坏在哪
通常：

| 模块 | 常见问题 |
| --- | --- |
| Chunk | 切太短/太长 |
| Embedding | 语义表达差 |
| Recall | 没召回来 |
| BM25 | 关键词不准 |
| Rerank | 排序错 |
| Context Assembly | 上下文拼接乱 |
| Generation | 幻觉 |


但实际上：

最大问题通常是 Recall 不够。

即：

```plain
正确chunk根本没进入上下文
```

---

# 二、Retrieval 怎么测（最重要）
---

# 1. 准备评测集（黄金数据）
你需要：

```plain
(question, ground_truth_chunk)
```

例如：

| Question | 正确Chunk |
| --- | --- |
| Redis 为什么会缓存雪崩 | “缓存雪崩章节” |
| JWT 为什么无状态 | “JWT 原理” |
| Kafka 如何保证顺序消费 | “Partition 顺序机制” |


这个叫：

```plain
Golden Dataset（黄金数据集）
```

---

# 2. 测 Recall@K
最核心指标。

意思：

```plain
正确chunk有没有出现在TopK里
```

---

# 举例
100个问题：

+ 83个问题正确chunk出现在Top5

那么：

```plain
Recall@5 = 83%
```

---

# 为什么 Recall 最重要
因为：

```plain
没召回来 = LLM 不可能答对
```

所以：

```plain
Recall 是 RAG 的天花板
```

---

# 3. 测 MRR（Mean Reciprocal Rank）
看：

```plain
正确chunk排得靠不靠前
```

---

# 举例
正确答案：

+ 第1名 → 1
+ 第2名 → 0.5
+ 第5名 → 0.2

平均后：

```plain
MRR 越高越好
```

---

# 4. 测 NDCG（高级一点）
看：

```plain
排序质量
```

适合：

```plain
一个query对应多个相关chunk
```

工业界常用。

---

# 三、Generation 怎么测
即：

```plain
最终回答好不好
```

---

# 1. Faithfulness（是否胡编）
最关键。

看：

```plain
回答是否基于context
```

不是：

```plain
LLM 自己瞎编
```

---

# 2. Answer Relevance
回答是否真正回答了问题。

---

# 3. Context Precision
给的context是否有用。

避免：

```plain
塞了一堆垃圾chunk
```



# 四、工业界最常用评测框架
现在最主流：



# 1. RAGAS
最火。

可测：

| 指标 | 含义 |
| --- | --- |
| faithfulness | 是否忠于context |
| answer_relevancy | 回答相关性 |
| context_precision | context质量 |
| context_recall | 是否召回正确context |


非常适合：

```plain
个人RAG项目
```



# 2. DeepEval
更工程化。

适合：

+ CI/CD
+ 自动化测试
+ Agent评测



# 3. LangSmith
适合：

+ trace
+ 调试
+ 可视化



# 五、最标准的 RAG Benchmark 流程
工业界通常：

```plain
构建测试问题集
↓
人工标注正确chunk
↓
测试 Recall@K
↓
测试 MRR/NDCG
↓
测试最终回答
↓
人工评估
```

---

# 六、一个非常重要的现实
很多人：

```plain
只看最终回答
```

这是错误的。

因为：

你根本不知道：

+ 是 retrieval 错
+ rerank 错
+ prompt 错
+ 还是 LLM 错

所以：

RAG 一定要“分层评测”。

否则根本无法优化。



> 更新: 2026-06-26 16:51:37  
> 原文: <https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/gqkcyldld0c7afly>