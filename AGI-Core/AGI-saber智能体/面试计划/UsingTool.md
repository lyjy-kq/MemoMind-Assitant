# Using Tool

### Tool Agent（工具调用）
**能讲的点：**

+ **Function Calling 原理**：给 LLM 提供工具 Schema（name/description/parameters），LLM 决定是否调用及参数填写
+ **工具三要素**：名称、描述（越清晰 LLM 越准确）、参数 Schema（JSON Schema 格式）

**面试常问：**

"如何保证工具调用的安全性？" → 参数校验、白名单工具、权限控制、执行沙箱

"工具调用失败怎么处理？" → 引出 Stage 6 Harness 的重试机制



> 更新: 2026-05-12 21:04:19  
> 原文: <https://www.yuque.com/yuqueyonghu-ng3vtk/agi-saber/kztaldw0d8n4ef3k>