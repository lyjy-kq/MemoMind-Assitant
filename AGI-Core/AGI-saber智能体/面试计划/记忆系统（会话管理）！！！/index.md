# 记忆系统（会话管理）！！！

# 引言
笔者认为在未来最牛逼的agent技术部分绝对是围绕记忆系统去建设的，所以此章内容吃透绝对是无与伦比的提升！

科普文章：

# 业界承认度最高的两个方案
1:mem0框架

2:字节的viking框架

个人观点：[mem0 🆚 viking](https://www.yuque.com/yuqueyonghu-ng3vtk/buapta/uy509v64cse72mgk)

# 我们教学项目的具体实现
## 1:用 Mem0 思想解决：
```plain
怎么存记忆
```

+ fact extraction
+ dedup
+ entity graph
+ semantic retrieval

---

## 2:用 Viking 思想解决：
```plain
怎么组织上下文
```

+ runtime state
+ planner state
+ task memory
+ tool state
+ context assembly

# 实现方案
+ 短期记忆：滑动窗口维持最近的多轮对话摘要
+ 长期记忆：存入milvs和es，混合检索，保证精确性和语义性，对长期记忆进行纬度维护。
+ 长期记忆管理：去重、合并、过期、重要性衰减机制

（模仿mem0 有 memory consolidation（相似记忆自动合并）和去重机制）

+ 使用记忆的时候怎么组织上下文呢？（借鉴viking思想）：[如何组织记忆上下文](https://www.yuque.com/yuqueyonghu-ng3vtk/buapta/nrv4gew7slmipetx)



## 1:去重
双重去重：（硬去重）哈希去重+（软去重）向量化去重

向量化去重：本条记忆如果与长期记忆表中检索到一条相似度大于0.92的，即放弃存入长期记忆表

## 2:合并
### Step1：召回相似记忆
```plain
topK semantic search
```

---

### Step2：相似度判断
例如：

```plain
sim > 0.85
```

进入 consolidation。

---

### Step3：LLM 合并
Prompt：

```plain
请将以下记忆合并成一条更稳定、更长期、更抽象的用户事实：
```

输出：

```plain
用户长期使用 Go 开发 AI Agent 系统
```

---

### Step4：更新 Memory
不是新增。

而是：

```plain
replace old memory
```

## 3:过期和重要性
ttl和importance本质是分不开的

ttl随着importance的动态变化而变化



> 更新: 2026-06-17 15:10:42  
> 原文: <https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/wrf2f1sgen39slzh>