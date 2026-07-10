// process.go — 主入口（Process / ProcessWithOptions / ProcessContext / ProcessStream）
// + 内部统一执行流 runOnce + 三段拆分（prepare / dispatch / finalize）。
//
// 设计：流式与非流式合并为同一执行流，靠 onEvent 是否为 nil 区分：
//
//	Process / ProcessContext / ProcessWithOptions    → runOnce(..., nil)
//	ProcessStream                                    → runOnce(..., onEvent)
//
// runOnce 编排：
//
//	prepare（STM 写入 + 偏好提取 + 路由决策 + 上下文装配 + 历史构建）
//	  ↓
//	dispatch（按 mode 分发到 chat / tool / rag / react，单一 mode handler 同时支持流/非流）
//	  ↓
//	finalize（assistant STM 写入 + 异步记忆抽取 + 异步合并 + 事件发布 + 计数填充）
package chat

import (
	"agi-assistant/internal/domain/memory/longterm"
	"agi-assistant/internal/domain/tool"
	"agi-assistant/internal/infrastructure/llm"
	"agi-assistant/internal/usercontext"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
)

// ─────────────────────────── 公开入口 ───────────────────────────

func (a *UnifiedAgent) Process(query string) *Response {
	ctx, cancel := context.WithCancel(context.Background())
	unregister := a.registerCancel(cancel)
	defer unregister()
	return a.runOnce(ctx, query, ChatOptions{Explicit: false}, nil)
}

// ProcessWithOptions 带显式选项的入口，供前端精确控制路由
func (a *UnifiedAgent) ProcessWithOptions(query string, opts ChatOptions) *Response {
	ctx, cancel := context.WithCancel(context.Background())
	unregister := a.registerCancel(cancel)
	defer unregister()
	return a.runOnce(ctx, query, opts, nil)
}

// ProcessContext 带 context 的入口，支持 SSE 流式和取消
func (a *UnifiedAgent) ProcessContext(ctx context.Context, query string, opts ChatOptions) *Response {
	ctx, cancel := context.WithCancel(ctx)
	unregister := a.registerCancel(cancel)
	defer unregister()
	return a.runOnce(ctx, query, opts, nil)
}

// ProcessStream 流式处理入口，在关键节点通过 onEvent 回调推送 SSE 事件。
// 返回完整的 Response（与 Process 一致），同时通过回调实时推送中间事件。
func (a *UnifiedAgent) ProcessStream(ctx context.Context, query string, opts ChatOptions, onEvent func(StreamEvent)) *Response {
	// context.WithCancel(parent) 不是“注册一个全局 cancel”，而是：
	//   1. 返回一个继承 parent 的子 ctx；
	//   2. 再返回一个 cancel 函数，用来主动终止这个子 ctx。
	// 这个 cancel 代表“当前这次流式请求”的终止开关，谁持有它，谁就能让这条请求链路尽快停止。
	//
	// 两个典型用法：
	//   示例 1：函数正常退出时，直接 defer cancel() / defer unregister()，
	//   让本次请求结束后立即释放子 ctx 和运行时登记，避免过期请求一直挂在表里。
	//
	//   示例 2：把 cancel 存进 taskRuntime，后续在 UnifiedAgent.Cancel() 或 CancelAll() 中统一调用，
	//   这样 SSE 流式请求、ReAct 任务、下游 LLM 调用都会收到 ctx.Done() 并提前退出。
	ctx, cancel := context.WithCancel(ctx)
	unregister := a.registerCancel(cancel)
	defer unregister()
	//如果没用流式的话还是走普通
	if onEvent == nil {
		// 上层若错误地传 nil，仍走非流式路径，保证 Response 完整
		return a.runOnce(ctx, query, opts, nil)
	}
	return a.runOnce(ctx, query, opts, onEvent)
}

// ─────────────────────────── 内部执行流 ───────────────────────────

// preparedRequest 是路由决策完成后、模式分发开始前的中间产物
type preparedRequest struct {
	query      string
	mode       string               // chat / tool / rag / react
	routeTools map[string]tool.Tool // 仅在 mode == tool / react 时非空
	memPrefix  string
	histMsgs   []llm.Message
	extracted  string // 同步规则提取的偏好回显（可能为空）
}

// runOnce 是 process / processStream 的统一编排：
// onEvent == nil → 非流式（不推送任何事件，但内部仍走完同一段逻辑）
// onEvent != nil → 流式（在 route / step / token / tool_call / rag_result / done 等节点推送）
func (a *UnifiedAgent) runOnce(ctx context.Context, query string, opts ChatOptions, onEvent func(StreamEvent)) *Response {
	pre := a.prepareAndSave(ctx, query, opts)

	resp := &Response{
		Query:         query,
		Mode:          pre.mode,
		ExtractedInfo: pre.extracted,
	}

	if pre.extracted != "" {
		emit(onEvent, "memory", map[string]string{"extracted_info": pre.extracted})
	}
	emit(onEvent, "route", map[string]string{"mode": pre.mode})

	// 检查 context 是否已取消（在分发前）
	if ctx.Err() != nil {
		resp.Interrupted = true
		resp.Answer = "[已中断] 请求在开始前被取消"
		emit(onEvent, "done", resp)
		return resp
	}

	a.dispatch(ctx, pre, resp, onEvent)

	if ctx.Err() != nil {
		resp.Interrupted = true
	}

	a.finalize(ctx, query, resp)

	emit(onEvent, "done", resp)
	return resp
}

// prepareAndSave 完成 STM 写入 / 偏好提取 / 路由 / 上下文装配 / 历史构建。
// 不推任何事件——事件由 runOnce 统一推送，便于 finalize 控制顺序。
func (a *UnifiedAgent) prepareAndSave(ctx context.Context, query string, opts ChatOptions) preparedRequest {
	userID := usercontext.UserIDFromContext(ctx)
	// 多租户：未登录请求所有 mem 写入都跳过——HTTP 层已经在 RequireAuth 中拦了，
	// 这里再加一道防御应付未来 CLI / 测试入口忘传 ctx 的情况
	hasUser := userID != ""

	// 1. 更新短期记忆 + 持久化
	if hasUser {
		//.STM(userID)是创建用户短期记忆桶,并从预热函数取出历史短期记忆
		// Add("user", query)是把用户输入的query写入短期记忆桶,如果记忆桶满了,会自动丢弃最旧的记忆
		a.mem.STM(userID).Add("user", query)
		//存入数据库
		a.repos.chat.Save(userID, "user", query)
	}

	// 2. 偏好提取：同步提取一份 kvs，同时用于前端回显与异步持久化。
	// 安全策略保持不变：整段 query 先过一次检查；正式落库前每条 k-v 再复检。
	var extracted string
	if hasUser {
		// 对用户消息先做整段检查，避免为高风险内容触发 LLM 抽取。
		if pre := inspectMemoryContent(query); !pre.Safe() {
			log.Printf("🛡️  [pref-extract] 整段拒绝：risk=%s reason=%s",
				pre.Risk, pre.Reason)
		} else {
			kvs := a.llm.ExtractPreferences(query)
			extracted = formatExtractedPreferences(kvs)
			if len(kvs) > 0 {
				// 异步持久化复用同一批提取结果，避免再次调用 ExtractPreferences。
				a.goSafe("process.preference-extract", func() {
					a.persistExtractedPreferences(userID, kvs)
				})
			}
		}
	}

	// 路由决策
	mode, routeTools := a.routeDecide(query, opts)

	// 装配 Schema-driven 上下文前缀 + 对话历史
	memPrefix := a.buildContextPrefix(ctx, query, mode)
	histMsgs := a.buildHistoryMessages(userID, query)

	return preparedRequest{
		query:      query,
		mode:       mode,
		routeTools: routeTools,
		memPrefix:  memPrefix,
		histMsgs:   histMsgs,
		extracted:  extracted,
	}
}

// formatExtractedPreferences 把偏好 kvs 格式化为稳定顺序的“已记住”回显文案。
func formatExtractedPreferences(kvs map[string]string) string {
	if len(kvs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(kvs))
	for k := range kvs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s = %s", k, kvs[k]))
	}
	return "已记住：" + joinWithChineseSemicolon(parts)
}

// joinWithChineseSemicolon 用中文分号拼接多条回显片段。
func joinWithChineseSemicolon(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "；" + parts[i]
	}
	return out
}

// persistExtractedPreferences 把一批已提取的偏好异步写入 Pref / PG / LTM。
func (a *UnifiedAgent) persistExtractedPreferences(userID string, kvs map[string]string) {
	// 先把提取结果写入用户偏好内存桶，供本进程后续上下文装配直接读取。
	a.mem.Pref(userID).SaveBatch(kvs)
	for k, v := range kvs {
		// 单条 k-v 复检
		if insp := inspectKVPair(k, v); !insp.Safe() {
			log.Printf("🛡️  [pref-extract] 拒绝写入 k=%q: risk=%s reason=%s",
				k, insp.Risk, insp.Reason)
			continue
		}
		// 这一条没问题就保存到数据库中
		a.repos.pref.Save(userID, k, v)
		// 把结构化 kv 拼成一条可入记忆的自然语言文本。
		// 之所以先拼接再审查，是因为后续要写入的图记忆和长期记忆，
		// 存的都是最终文本形态，而不是拆开的 k/v 片段。
		content := fmt.Sprintf("用户%s: %s", k, v)
		if insp := inspectMemoryContent(content); !insp.Safe() {
			log.Printf("🛡️  [pref-extract] 拼接后命中：risk=%s", insp.Risk)
			continue
		}
		// 先对最终文本做 embedding，得到用于相似度、去重和图召回的向量。
		emb, _ := a.llm.Embed(content)
		// GraphMemory.Store 先在内存长期记忆里做去重判断：
		//   - added=true 表示这条内容是新记忆，需要继续落库
		//   - added=false 表示已存在相似记忆，不再重复写图节点和 PG 记录
		// 这一步不是查数据库，而是查当前进程内的 LongTerm.Items。
		if a.mem.graphMem != nil {
			if added, _ := a.mem.graphMem.Store(userID, content, 0.8, emb); added {
				// 只有确认是新记忆时，才把 embedding 序列化后写入 PostgreSQL。
				// 这样可以避免重复插入，并保证 PG 与内存图看到的是同一条新增记忆。
				embJSON, _ := json.Marshal(emb)
				// LTM 仓储返回数据库自增 ID；这个 ID 后面要回填到刚写入的
				// LongTerm / GraphMemory 条目里，保证内存 ID 与 PGID 对齐。
				pgID := a.repos.ltm.Save(userID, content, 0.8, embJSON)
				// 回填 PGID 后，图记忆和长期记忆都能引用同一个持久化 ID，
				// 便于后续召回、合并、审计和图节点同步。
				a.mem.graphMem.SyncLastItemPGID(pgID)
			}
			continue
		}
		if a.mem.ltm.StoreClassified(userID, content, 0.8, emb, "general", nil, "") {
			embJSON, _ := json.Marshal(emb)
			newID := a.repos.ltm.SaveClassified(userID, content, 0.8, embJSON, "general", nil, "")
			a.mem.ltm.SyncLastItemPGID(newID)
		}
	}
}

// routeDecide 把"显式 vs 自动路由"的判断从 process 主流程中抽出。
// 显式优先（Explicit=true 时按 SelectedTools / UseRAG 决定），否则按关键词启发式判定。
func (a *UnifiedAgent) routeDecide(query string, opts ChatOptions) (mode string, routeTools map[string]tool.Tool) {
	if opts.Explicit {
		switch {
		case len(opts.SelectedTools) > 0:
			routeTools = a.filterTools(opts.SelectedTools)
			if a.needReActFromTools(query, routeTools) {
				mode = "react"
			} else {
				mode = "tool"
			}
		case opts.UseRAG && a.rag.Loaded:
			mode = "rag"
		default:
			mode = "chat"
		}
		return
	}

	switch {
	case a.needReAct(query):
		mode = "react"
		routeTools = a.toolsSnapshot()
	case a.needTool(query):
		mode = "tool"
		routeTools = a.toolsSnapshot()
	case a.needRAG(query):
		mode = "rag"
	default:
		mode = "chat"
	}
	return
}

// dispatch 按 mode 调对应 handler，把结果填回 resp。
// 流式与非流式共用同一组 handler，由 onEvent 区分。
func (a *UnifiedAgent) dispatch(ctx context.Context, pr preparedRequest, resp *Response, onEvent func(StreamEvent)) {
	switch pr.mode {
	case "react":
		answer, steps, task := a.runReAct(ctx, pr.query, pr.routeTools, pr.memPrefix, pr.histMsgs, onEvent)
		resp.Answer, resp.Steps, resp.Task = answer, steps, task
	case "tool":
		answer, tc := a.runTool(ctx, pr.query, pr.routeTools, pr.memPrefix, pr.histMsgs, onEvent)
		resp.Answer, resp.ToolCall = answer, tc
	case "rag":
		// RAG history rewriter 需要 userID 才能取到本用户 STM；
		// 未登录请求 recentHistoryForRAG 返回 nil，rewriter 自动退化到原始 query。
		userID := usercontext.UserIDFromContext(ctx)
		answer, results := a.rag.QueryWithHistory(pr.query, a.recentHistoryForRAG(userID))
		resp.Answer, resp.SearchResults = answer, results
		// RAG 暂不流式合成，但也通过事件回放结果以保持 SSE 输出形状一致
		emit(onEvent, "rag_result", map[string]interface{}{"search_results": results})
		emit(onEvent, "token", map[string]string{"content": answer})
	default: // chat
		systemPrompt := a.buildSystemPrompt(pr.memPrefix, "你是一个简洁的AI助手。结合你掌握的用户信息，使回答更个性化。")
		resp.Answer = a.chatLLM(ctx, systemPrompt, pr.histMsgs, onEvent)
	}
}

// finalize 完成 assistant 写回、异步记忆抽取、异步合并、事件发布、计数填充。
// 所有"无论流式或非流式都要做"的副作用集中在这里。
func (a *UnifiedAgent) finalize(ctx context.Context, query string, resp *Response) {
	userID := usercontext.UserIDFromContext(ctx)
	hasUser := userID != ""

	if hasUser {
		a.mem.STM(userID).Add("assistant", resp.Answer)
		a.repos.chat.Save(userID, "assistant", resp.Answer)
	}

	// 双源记忆抽取：
	//   1) 从用户消息抽 "用户偏好/身份/事实陈述"（高可信，importance=0.7）
	//   2) 从对话对抽 "用户问题主题相关的客观事实"（次级可信，importance=0.5）
	// 两条都过 poison gate；reply 路径额外要求 "key 必须与用户问题主题锚定"，
	// 切断 "AI 被越狱后吐出无关 PII → 入库" 的攻击放大链。
	if hasUser {
		a.goSafe("process.memory-extract", func() {
			a.extractMemoryFromUserMsg(userID, query)
			a.extractMemoryFromExchange(userID, query, resp.Answer)
		})
	}

	// 异步触发记忆合并（去重+合并+衰减+过期；有图层时使用图感知合并以保护高中心度节点）
	a.goSafe("process.consolidate", func() {
		if a.mem.ltm.NeedConsolidation() {
			var result longterm.ConsolidationResult
			if a.mem.graphMem != nil {
				result = a.mem.graphMem.GraphAwareConsolidate()
			} else {
				result = a.mem.ltm.Consolidate()
			}
			a.syncConsolidationToDB(result)
		}
	})

	eventData, _ := json.Marshal(map[string]interface{}{"query": query, "mode": resp.Mode})
	a.repos.events.Publish("agent.chat", string(eventData))

	// 计数 / 偏好回显（响应内只看本用户的桶）
	resp.ShortTermCount = a.mem.stmCount(userID)
	resp.LongTermCount = a.mem.ltm.Count()
	resp.Preferences = a.mem.prefSnapshot(userID)
}
