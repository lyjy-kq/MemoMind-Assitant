
% 文件说明：本文件用于生成“后端实习-网盘+AGENT-1页”简历，
% 当前版本聚焦后端/Agent 方向，保留教育背景与核心项目，
% 并按投递需求隐藏 eBPF 项目、补充 GitHub 主页信息。
\documentclass{styles/resume}
\usepackage{styles/zh_CN-Adobefonts_external}
\usepackage{styles/linespacing_fix}
\usepackage{indentfirst}
\usepackage{cite}
\usepackage{setspace}
\usepackage{enumitem} % 添加 enumitem 包以压缩列表间距
\usepackage{tikz}
\usepackage{graphicx} % 确保 tikz 包已加载

% 适度压缩全局行距，给一页简历留出更稳定的纵向空间。
\setstretch{1.18}% 设置行间距
% 统一收紧列表间距与缩进，只影响排版密度，不改正文内容。
\setlist[itemize]{leftmargin=1.4em, itemsep=0.2ex, topsep=0.3ex, parsep=0pt, partopsep=0pt}

\begin{document}

\pagenumbering{gobble}

\newgeometry{% 强制设置边距
	a4paper,
	left=0.66in,
	right=0.66in,
	top=0.4in,
	bottom=0.35in,
	nohead
}
\begin{center}
	\begin{minipage}{\textwidth}
		\centering
		\name{孟高翔}
		\basicInfo{
			\email{1514190951@qq.com} \textperiodcentered\
			\faGithub\ \href{https://github.com/lyjy-kq}{lyjy-kq}
			\textperiodcentered\
			\phone{(+86) 136-7557-3771} 

		}
	\end{minipage}
\end{center}

\begin{tikzpicture}[remember picture, overlay]
	% 头像位置略向右上收拢，避免与正文挤压。
	\node[anchor=north east, inner sep=0pt] at ([xshift=-1.55cm, yshift=-0.55cm]current page.north east) {
		\includegraphics[width=1.0in]{images/photo2.png} };
\end{tikzpicture}

# 教育背景
\section{\faGraduationCap\ 教育背景}

\datedsubsection{\textbf{武汉大学（双一流A类，985高校） | 计算机学院 | 软件工程专业}}
{\qquad 本科}{\textit{2022.09 -- 2026.06}}

\datedsubsection{\textbf{武汉大学（双一流A类，985高校） | 计算机学院 | 软件工程专业}}
{\qquad 硕士}{2026.09 -- \qquad \quad}

\begin{itemize}
	\item \textbf{本科绩点}：3.883\quad \textbf{专业排名}：16/218（7\%）
	\item \textbf{荣誉奖项}：校级乙等奖学金（前30\%）、三好学生、优秀学生干部、甲等新生奖学金
	\item \textbf{竞赛奖项}：2024全国大学生数学建模省三，第九届ICT大赛编程赛省一，第十五届蓝桥杯省二
	\item \textbf{英语水平}：雅思6.5、CET-4、CET-6
\end{itemize}

\section{项目经历}
# 云盘
\datedsubsection{\textbf{MyCloudDrive 个人云盘系统}}{}{https://github.com/lyjy-kq/MyCloudDriver}
\role{核心开发}{Go, MySQL, Redis, Local/S3, Docker}
\begin{itemize}
	\item 设计多工作空间与多存储源架构，解耦业务逻辑与存储实现，支持本地盘与 S3 兼容存储的统一接入。
	\item 实现并发分片上传链路，支持断点续传、秒传识别与过期分片清理，解决大文件上传耗时长、失败后重传成本高的问题。
	\item 上传文件加入分片校验、幂等键与传输状态控制，保障可恢复性与一致性。
	\item 扩展短链分享功能，增加提取码校验和过期校验机制，增强文件分享的安全性，并异步统计访问记录。
\end{itemize}
# 网盘agent
\datedsubsection{\textbf{智能网盘 Agent}}{}{https://github.com/lyjy-kq/MyCloudDriver}
\role{核心开发}{LLM, RAG, Function Calling, Workflow}
\begin{itemize}
	\item 围绕不同意图的稳定响应，将对话、知识检索和任务执行拆分为独立流程，并以 Plan-Confirm-Execute、能力白名单和参数校验约束多步骤操作。
	\item 实现知识库文档异步导入，串联解析、切片、向量化、索引构建流程，支持混合召回与 RAG 问答。
	\item 基于文档 Hash 实现增量更新，模块化解耦解析、切分、向量化与重排流程，提高工程可维护性。
    \item 混合检索 RAG 架构，融合向量检索与BM25召回，以 RRF 排序提升稳定性与准确率。
	\item 基于千篇技术文档构建 Chunk 语料和基准问题，对比稠密、关键词与混合检索；Hybrid + RRF 的 NDCG@10 达到 0.83，据此优化检索与生成方案。
\end{itemize}

# 记忆助手
\datedsubsection{\textbf{MemoMind Assistant 智能记忆助手}}{}{https://github.com/lyjy-kq/MemoMind-Assitant}
项目简介：一个面向个人的多模态智能体系统，融合了检索增强生成(RAG)、三层记忆、知识图谱、沙箱执行与可恢复执行流，支持多轮对话、知识检索、工具调用与复杂推理。

\role{核心开发}{Go, Agent, RAG, Memory, Workflow}
\begin{itemize}
    \item 设计意图路由与 Agent 调度层，按请求分发至对话、检索和 ReAct-Planner 流程，支持按需工具调用，平衡响应质量、延迟与 Token 成本。
	\item 将记忆拆分为短期会话(滑动窗口)、用户偏好(LLM+规则)、长期记忆(Embedding/TF)，并引入图增强召回，配套抽取、去重、合并与过期策略，使用户画像能够持续演化并按任务按需装配。
    \item 对记忆检索增强，根据当前意图召回相关历史、用户画像与关联图实体，将高置信记忆配合RAG检索统一注入提示上下文，提升多轮问答的个性化与上下文连贯性。
    \item 基于 DAG 编排复杂任务，采用拓扑排序调度可并行节点，配合 Race Strategy 实现多搜索源、多模型、多检索策略竞速，并提供超时重试与状态持久化，实现Harness Runtime。
     \item 设计可隔离 Sandbox Runtime，将 Agent 命令工具纳入风险校验、用户确认、Docker 隔离执行、Kafka 审计过程，通过网络隔离、只读文件及CPU/内存/PID 限制约束执行风险。
\end{itemize}
# 专业技能
% 定义其他信息部分
\section{\faInfo\ 专业技能}
	\begin{itemize}
		\item 熟悉 RAG 全链路设计与优化，能够结合业务场景对召回效果、响应质量与系统性能进行针对性优化
		\item 熟悉 Agent 基本原理，了解任务规划、工具调用、流程编排等核心机制，掌握 Agent 记忆系统的基础设计思路
		\item 熟悉Golang基础知识与集合、并发等基础框架。
		\item 掌握MySQL及基本原理 ，熟悉Redis原理及应用。 
		
		\item 熟练使用 Docker、Kubernetes、Git、Claude Code、Codex 等工具，具备项目开发、部署与调试能力
        		% \item 熟悉 Claude Code、Codex、OpenClaw、Hermes 等 Agent 的使用，了解架构设计原理，能够结合实际场景进行应用与实践
                % \item 熟练使用Docker、Kubernetes、Git，实现容器化部署与项目管理。
	\end{itemize}

\end{document}



% eBPF 项目按当前投递方向暂时隐藏，保留源码便于后续切换版本时恢复。
% \datedsubsection{\textbf{基于 eBPF 的微服务故障注入系统}}{}{实验室项目}
% \role{核心开发}{eBPF, Go, Kubernetes, 微服务}
% 围绕实验室对微服务稳定性评估的需求，将本科毕设落成一套可控的故障注入原型，希望在不改业务代码的前提下，更低扰动地复现线上常见异常。
% \begin{itemize}
% 	\item 针对插件方案侵入性强、代理方案开销高的问题，重新拆分控制面与执行面，把注入逻辑下沉到内核流量路径，并支持按服务、接口和故障类型精确选中目标流量。
% 	\item 围绕“既要可控、又不能误伤正常请求”这个核心约束，重点处理了有限报文解析、规则查表匹配、报文改写一致性和热更新下发等难点，让实验规则能稳定生效且便于反复验证。
% 	\item 在 sock-shop、bookinfo 两套微服务上完成丢包、延迟、错误响应三类实验，压测范围覆盖 100 到 50000 req/s；结果验证了方案的可用性，也说明它在高并发下更容易兼顾吞吐、时延与资源开销。
% \end{itemize}
```
