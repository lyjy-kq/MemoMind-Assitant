# PDF 处理流程说明

## 1. 功能目标

PDF 处理能力的目标是让用户可以直接上传 PDF 文件，并把其中可提取的文本转成 RAG 可处理的知识内容。

在这个提交之前，上传内容更偏向纯文本；这个提交之后，系统可以识别 PDF、解析 PDF 字节、提取正文、切分成 chunk，并尝试进入混合检索索引。

## 2. 当前已实现能力

当前已经实现：

* 前端使用 `multipart/form-data` 上传原始文件
* `/api/upload` 支持真实文件上传，同时保留旧版 JSON 文本上传兼容
* 服务端可以识别 `.pdf` 文件和 `application/pdf` 类型
* PDF 不再按普通 UTF-8 文本读取，而是走专门的 PDF 解析流程
* PDF 文本提取优先级为：
  * `pdfplumber`
  * `pdftotext`
* 解析出的文本会进行归一化，减少 PDF 排版噪声
* 文本会进入 RAG 的父子 chunk 切分流程
* 上传响应会返回解析器、页数、文本字符数、chunk 数、索引数、`doc_hash` 等信息
* 如果 PDF 可提取文本太少，会返回 `needs_ocr: true`，避免把扫描版 PDF 当作有效知识入库
* PostgreSQL 不可用时，会跳过实际索引和 embedding 调用，避免无效写入

## 3.PDF 解析策略

系统通过两个信号判断文件是否是 PDF：

* `Content-Type` 是 `application/pdf`
* 文件扩展名是 `.pdf`

解析优先级如下：

1. `pdfplumber`
   * 通过 Python 执行。
   * 会优先查找环境变量指定的 Python：
     * `PDF_EXTRACT_PYTHON`
     * `PDF_PYTHON`
   * 然后查找系统 `python3`。
   * 会输出类似 `--- page 1 ---` 的页标记。
2. `pdftotext`
   * 如果系统 PATH 中存在 `pdftotext`，会作为第二解析方案。
   * 使用 layout preservation 和 UTF-8 输出。
3. Go fallback
   * 使用 `github.com/ledongthuc/pdf`
   * 先尝试按行提取，再退回普通文本提取。

如果所有方案都无法提取有效文本，接口会返回需要 OCR 的错误或标记。

## 5. 文本归一化

PDF 提取出的文本通常带有换行、断词、空格和页眉页脚等噪声。当前归一化会处理：

* `\r\n` 和 `\r` 统一为 `\n`
* 移除空字节和软连字符。
* 修复英文断行连字符，例如 `RE-\nWARDS`
* 合并重复空格和 tab。
* 压缩过多空行。
* 去掉首尾空白。

这样可以减少无意义 chunk，提高后续检索质量。

## 6. RAG 切分方式

PDF 文本进入 RAG 后使用 small-to-big chunking：

```latex
完整文档文本
  |
  v
父 chunk 切分
  - size: max(rag.chunk_size * 4, 600)
  - overlap: rag.chunk_overlap * 2
  |
  v
子 chunk 切分
  - size: rag.chunk_size
  - overlap: rag.chunk_overlap
  |
  v
索引子 chunk，并保留父 chunk 上下文
```

当前默认配置：

* `rag.chunk_size = 200`
* `rag.chunk_overlap = 50`
* 父 chunk 大小约 `800`
* 父 chunk overlap 为 `100`
* 子 chunk 大小为 `200`
* 子 chunk overlap 为 `50`

检索时先命中更精准的子 chunk，再扩展到更大的父 chunk 给 LLM 使用。

## 7. 索引语义

当前混合索引顺序是：

1. 检查 PostgreSQL 是否可用。
2. 如果 PostgreSQL 不可用：
   * 返回已计算的 `doc_hash`
   * 返回 chunk 数。
   * `indexed_count = 0`
   * 跳过 embedding。
   * 跳过 Milvus / Elasticsearch / Neo4j 写入。
3. 如果 PostgreSQL 可用：
   * 生成 embedding。
   * 保存 chunk、parent content 和 embedding JSON 到 PG。
   * 可用时写入 Milvus。
   * 可用时写入 Elasticsearch。
   * 可用时异步触发 Neo4j 知识图谱索引。

`Engine.Loaded` 现在由实际成功索引数量决定。也就是说，PDF 解析成功但索引失败时，不会误报 RAG 已加载。

## 8. 上传响应示例

成功上传并解析后，响应类似：

```json
{
  "filename": "paper.pdf",
  "content_type": "application/pdf",
  "parser": "pdfplumber",
  "pages": 22,
  "text_chars": 76642,
  "needs_ocr": false,
  "chunk_count": 539,
  "parent_count": 102,
  "indexed_count": 0,
  "chunk_preview": [],
  "doc_hash": "sha256..."
}
```

如果 `needs_ocr = true`，说明该 PDF 可能是扫描版或图片型 PDF，当前不会进入 RAG。


> 更新: 2026-06-30 10:25:32  
> 原文: <https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/hhr0liyp3tqmkokx>