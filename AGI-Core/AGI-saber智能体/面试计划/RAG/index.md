# RAG

# RAG（检索增强生成）能讲的点：

## RAG 解决什么问题：

RAG（Retrieval-Augmented Generation）本质上是：**“让大模型先查资料，再回答问题。”**

主要解决几个核心问题：

1. 大模型知识过时

LLM 参数是训练时固化的。

例如：

* 公司内部文档
* 最新代码
* 用户知识库
* 实时数据

模型本身不知道。

RAG 可以：

* 动态检索最新知识 （因为你的知识库是可以动态删除和增加的）
* 不需要重新训练模型（微调成本大，效果像开盲盒，越调越傻逼的可能性很大）

## 完整 RAG 流程：

```graphql
1:数据提取
2:用户提问
3:Query 理解 / 改写
4:检索（Retrieval）
5:重排序（Rerank）
6:组装prompt传入大模型，大模型给出最终答案
```

### 数据提取流程

```graphql
原始文档
  ↓
清洗
  ↓
切片 Chunk
  ↓
Embedding 向量化
  ↓
存入向量数据库
```

首先清洗对于多种文件的方式是不同的，因为我们的知识库只支持上传md文件，所以清洗也只在md文件的基础上考虑就行。

**Markdown 清洗核心目标**

做Markdown 清洗时，不是单纯“删字符”。

而是：“保留对检索有价值的信息，去掉会污染 embedding 的噪声。”

因为 RAG 的问题很多时候不是模型不够强，而是：

```plain
Embedding 输入质量差
```

***

**1. 删除无意义 Markdown 噪声**

会清理：

```plain
HTML 标签
重复空行
无意义分隔符
导航链接
```

例如：

```plain
上一篇
下一篇
返回顶部
```

这些内容：

* 不具备语义价值
* 会污染向量表示
* 增加 retrieval 噪声

***

**2. 保留标题层级结构**

不会把 markdown flatten 成纯文本。

而是保留：

```plain
# 一级标题
## 二级标题
### 小节
```

因为：

标题层级本身就是语义结构。

***

**3. 保留代码块（重点）**

技术知识库里：

```plain
代码本身就是知识
```

所以不会删除：

````plain
```go
func SendMessage() {}
````

***

**4. 保留列表结构**

例如：

```plain
- embedding
- rerank
- retrieval
```

不会简单拼成一行。

因为列表本身表示：

```plain
并列语义关系
```

对检索有帮助。

***

**5. 表格结构规范化**

Markdown table 会转成：

```plain
key-value
```

或者结构化文本。

避免：

* 表格直接塌陷成乱码
* embedding 丢失字段关系

***

**6. 去重（Deduplication）**

很多知识库：

```plain
导航重复
模板重复
引用重复
```

会做去重。

否则：

* retrieval 容易重复召回
* context 被冗余内容污染

**以上六条都可以用python脚本实现固定规则**

**然后md文件会变得更加规范，切片也就更方便更有效**

## 3:切片策略：

### Markdown 技术知识库切片策略

**1. Header-aware Chunking（按标题层级切分）**

优先按照：

```markdown
# 一级标题
## 二级标题
### 小节
```

进行切分。

因为技术文档天然章节化：

* topic 更集中
* retrieval precision 更高
* 不容易语义断裂

***

**2. Recursive Split（超长 Section 递归切分）**

如果某个 section：

* token 过大
* 内容跨度过广

才继续：

* 按段落
* 按列表
* 按代码块

递归细切。

避免：

* chunk 过大
* embedding 被多 topic 污染

***

**3. Code-aware Chunking（代码块完整保留）**

不会从中间截断：

```go
func SendMessage() {}
```

会完整保留 code block。

因为：

* 函数名
* API 名
* 错误码

都是高价值 retrieval signal。

***

**4. Small-overlap Strategy（小重叠策略）**

仅保留少量 overlap：

```plain
50~100 token
```

避免：

* 上下文断裂
* retrieval 重复召回
* embedding 冗余

***

**5. Neighbor Expansion（邻居扩展）**

检索命中 chunk 后：

自动补充：

* 前一个 chunk
* 后一个 chunk

恢复章节连续语义。

因为技术文档：

* topic 通常沿 section 连续分布

***

**6. Parent-Child Chunking（父子块检索）**

区分：

Child Chunk

小粒度：

* 用于 retrieval
* embedding 更精准

Parent Context

大粒度：

* 用于 generation
* 保证上下文完整

流程：

```plain
先召回 child
↓
再回溯 parent section
↓
最终给 LLM 更完整上下文
```

切完存入向量数据库即可

## 4:混合检索做法：

向量召回（稠密向量索引）：召回语义相似的chunk

缺点：容易忽略精确名词，容易走歪

关键词召回（倒排索引 + BM25 打分）：召回关键词次数最多的文档chunk

缺点：无法理解语义。

我们做法是混合检索

```graphql
Dense Retrieval（向量召回 TopK）
+
BM25 Retrieval（关键词召回 TopK）
↓
Merge + Dedup（合并去重）
↓
RRF Fusion（融合排序）
↓
TopN Context
```

## 5:为什么混合检索：

Dense（向量检索）和 BM25（关键词检索）各有盲区。

混合检索本质上是在解决：

```plain
“单一路径召回不稳定”
```

***

**1. 向量检索 的问题**

Dense 擅长：

* 语义相似
* 同义表达
* 意图理解

比如：

Query：

```plain
如何防止数据库被拖垮
```

即使文档里写的是：

```plain
缓存击穿解决方案
```

向量也能召回。

因为 embedding 学到了：

```plain
数据库被拖垮 ≈ 缓存击穿
```

***

但 Dense 有几个致命问题：

***

**（1）关键词不敏感**

比如：

```plain
GPT-4o mini token limit
```

Dense 可能只理解：

```plain
GPT模型 + token
```

但：

* `4o mini`
* `128k`
* 特定版本号

这些精确词可能丢。

尤其：

* API 名
* 类名
* 错误码
* 人名
* SKU
* 函数名

Dense 很容易翻车。

***

**（2）容易语义漂移**

Query：

```plain
redis 主从同步延迟
```

Dense 可能召回：

```plain
redis 高可用
redis cluster
redis 持久化
```

因为 embedding 觉得：

```plain
“都属于 redis”
```

但其实不精准。

***

**2. BM25 的问题**

BM25 擅长：

* 精确关键词
* ID
* 专有名词
* 错误码
* API 名称

比如：

```plain
ERR_CONNECTION_RESET
```

BM25 极强。

***

但 BM25 不懂语义。

例如：

Query：

```plain
怎么防止数据库被拖垮
```

文档：

```plain
缓存击穿解决方案
```

BM25 根本召不回来。

因为：

```plain
没有共同关键词
```

***

所以两者互补

| 能力 | Dense | BM25 |
| --- | --- | --- |
| 语义理解 | 强 | 弱 |
| 精确关键词 | 弱 | 强 |
| 同义词 | 强 | 弱 |
| 专有名词 | 弱 | 强 |
| 长尾错误码 | 弱 | 强 |
| 自然语言问题 | 强 | 弱 |

所以工业界会：

```plain
Dense 负责“广”
BM25 负责“准”
```

***

### 为什么不能只靠 Dense

很多人一开始觉得：

```plain
embedding 已经很强了
```

但实际线上：

Dense 经常漏掉：

* 函数名
* 版本号
* 配置项
* SQL字段
* API路径
* 报错文本

这些在技术 RAG 里极重要。

比如：

```plain
gorm.ErrRecordNotFound
```

BM25 秒杀 Dense。

***

### 为什么不能只靠 BM25

因为用户根本不会用文档原词提问。

用户会说：

```plain
为什么服务雪崩
```

文档写：

```plain
级联故障传播
```

只有 Dense 能理解。

***

### 混合检索真正解决的问题

核心是：

```plain
提高 Recall（召回率）
```

RAG 最大问题通常不是：

```plain
生成不好
```

而是：

```plain
根本没召回来
```

LLM 再强：

```plain
没喂进去也没用
```

所以工业界非常重视：

```plain
Recall First
```

***

### 为什么先召回很多，再 rerank

因为：

```plain
召回阶段宁可“错杀少”
```

即：

```plain
宁可多召
不要漏召
```

后面再：

* RRF
* CrossEncoder
* LLM rerank

慢慢过滤。

***

# 工业界经典思路

一般：

```plain
Dense Top50
+
BM25 Top50
↓
Merge
↓
RRF
↓
CrossEncoder
↓
Top5
```

本质：

**第一阶段**

尽可能：

```plain
“把可能相关的都捞上来”
```

**第二阶段**

再：

```plain
“精细排序”
```

这是现代 RAG 的核心思想。

# RAG评测

标准流程介绍：[怎么评测RAG效果](https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/gqkcyldld0c7afly)

我们项目真实评测过程及其数据：[AGI项目的真实RAG评测](https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/kg90qoh7vm8o79bg)


> 更新: 2026-07-02 09:06:16  
> 原文: <https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/gbkmp56a6ki3z383>