1. ltm做成懒加载
2. LongTerm 是用户隔离的，但是GraphMemory 只写mem_id / content / importance，那么图记忆怎么保证是用户的记忆，这里先按照只为单用户设计来预设
3. stm使用kv存储，放弃一些变化性,比如喜欢苹果会覆盖之前的喜欢西瓜，但是这两个在rag和kg都会有存储
4. 我先知道rag文档的增量更新是怎么做到的有uu知道具体方案吗
5. 一般我们一篇文档会有一个doc_id，chunk会有对应的chunk_id，这个前提是对更新这步比较重要的，chunk_id有一个比较好的分配方式是对（source_id, chunk_index,  section_path) 这部分做hash操作做一个hash操作得到chunk_id, chunk_hash 用来判断内容有没有变化，新文档来了就重新解析，得到chunk列表和hash，和之前构建好的索引做diff对比，然后update
就小厂没笔试
https://developers.llamaindex.ai/python/examples/ingestion/document_management_pipeline/
可以看看LlamaIndex他们的做法
核心是有一个元数据作为索引方便我们快速定位 更新就正常update即可
get
谢谢uu
https://cursor.com/cn/blog/secure-codebase-indexing
还有cursor的做法
很值得借鉴 因为像codebase这种频繁更新的内容 一般像cc这种agent是不会做语义检索的
cursor做了 而且是有提升的，所以网上可以能会看到一些文章说为什么cc选择grep而不是rag，会去贬低rag的价值，可能有失判断
是 这会有开销 cursor在尽力减少
  1. 用merkle tree来定位变化，merkle tree占用很小，网络传输压力小
    2. chunk级别更新，我们之前提到的
  3. 异步做embedding，不阻塞agent响应
  但其实你用cursor就是会拿你一部分额度去做embedding😂
7. saber的kg链路后续还会有优化吗？我看他那个逻辑好像是节点属性会被后写覆盖同一实体出现在多个 chunk 时，当前节点上的 pg_id/doc_hash 可能只保留最后一次 set 的值。
我感觉如果这样设计好像不太好用啊
我最近正在看这个，还要做实体分歧
对呀
这两块都有点问题
这个其实就加个id区分就行，主要做实体分歧
我还在看方式
因为我现在去实习的那个地方他就是让我负责做kg链路了

8. 抖音内容
10. 图记忆召回优化（优先提升实际回答连贯性）
    - 当前 `FOLLOWS` 仅表示记忆写入相邻，且 1-hop 只能取到直接相邻节点；不能可靠表达同一轮对话或同一主题的关联。
    - 第一阶段：为每轮对话建立 `Conversation/MemoryGroup`，本轮抽取出的全部 LTM 记忆通过 `IN_CONVERSATION` 关系归入该组；保留跨会话的 `SIMILAR_TO` 语义边。
    - 召回流程：先进行向量召回得到强相关 seed，再从同会话组补充 1～2 条高价值记忆，并最多补充一条 `SIMILAR_TO` 弱关联记忆；按关系类型、时间衰减和重要度重新评分后统一 Top-K。
    - 不以多跳 `FOLLOWS` 扩展作为主要方案；`CAUSES` / `BELONGS_TO` 等实体或主题关系留待第二阶段，通过规则/LLM 抽取并做归一化后再接入。
    - 验收：用户询问“上次的 X”时，能够稳定取回同轮相关记忆；扩展结果不应挤掉强相关向量命中，也不得跨用户召回。

