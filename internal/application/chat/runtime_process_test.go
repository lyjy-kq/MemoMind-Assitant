// runtime_process_test.go — 验证 prepare 阶段的偏好提取与 extracted 回显行为。
//
// 本文件聚焦一个回归点：extracted 必须直接来自 ExtractPreferences 的结果，
// 不能再依赖 Preference.ExtractAndSave 那套有限的同步规则兜底。
package chat

import (
	"agi-assistant/config"
	"agi-assistant/internal/domain/rag"
	"agi-assistant/internal/infrastructure/llm"
	"agi-assistant/internal/infrastructure/persistence/chathistory"
	ltmrepo "agi-assistant/internal/infrastructure/persistence/longterm"
	"agi-assistant/internal/usercontext"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeChatRepo 是 prepare 单测使用的聊天记录仓储桩。
type fakeChatRepo struct{}

// Save 是聊天记录持久化桩实现。
func (r *fakeChatRepo) Save(userID, role, content string) {}

// Load 是聊天记录读取桩实现。
func (r *fakeChatRepo) Load(userID string, limit int) []chathistory.Entry { return nil }

// fakePrefRepo 是 prepare 单测使用的偏好仓储桩。
type fakePrefRepo struct{}

// Save 是偏好持久化桩实现。
func (r *fakePrefRepo) Save(userID, key, value string) {}

// Load 是偏好读取桩实现。
func (r *fakePrefRepo) Load(userID string) map[string]string { return map[string]string{} }

// fakeLongTermRepo 是 prepare 单测使用的长期记忆仓储桩。
type fakeLongTermRepo struct{}

// Save 是长期记忆普通写入桩实现。
func (r *fakeLongTermRepo) Save(userID, content string, importance float64, embeddingJSON []byte) int {
	return 1
}

// SaveClassified 是长期记忆分类写入桩实现。
func (r *fakeLongTermRepo) SaveClassified(userID, content string, importance float64, embeddingJSON []byte, category string, tags []string, slotHint string) int {
	return 1
}

// Load 是长期记忆全量加载桩实现。
func (r *fakeLongTermRepo) Load() []ltmrepo.Row { return nil }

// LoadByUser 是长期记忆按用户加载桩实现。
func (r *fakeLongTermRepo) LoadByUser(userID string) []ltmrepo.Row { return nil }

// Update 是长期记忆更新桩实现。
func (r *fakeLongTermRepo) Update(id int, content string, importance float64, embeddingJSON []byte) {}

// UpdateImportanceBatch 是长期记忆批量重要度更新桩实现。
func (r *fakeLongTermRepo) UpdateImportanceBatch(items []ltmrepo.ImportanceUpdate) {}

// Delete 是长期记忆删除桩实现。
func (r *fakeLongTermRepo) Delete(ids []int) {}

// SetQuarantine 是长期记忆隔离标记桩实现。
func (r *fakeLongTermRepo) SetQuarantine(id int, quarantined bool, reason string) {}

// MarkSuperseded 是长期记忆替代关系桩实现。
func (r *fakeLongTermRepo) MarkSuperseded(oldIDs []int, newID int) {}

// newPrepareTestAgent 构造一个只覆盖 prepare 所需依赖的最小 agent。
func newPrepareTestAgent(cfg *config.APIConfig) *UnifiedAgent {
	return &UnifiedAgent{
		cfg: cfg,
		llm: llm.New(cfg),
		rag: &rag.Engine{},
		mem: newMemoryStack(cfg, nil, nil),
		repos: &repoBundle{
			chat: &fakeChatRepo{},
			pref: &fakePrefRepo{},
			ltm:  &fakeLongTermRepo{},
		},
	}
}

// TestPrepare_ExtractedInfoUsesExtractPreferences 验证 extracted 直接来自 ExtractPreferences。
//
// 这里故意让 LLM 返回 “职业=工程师”：
//   - 旧逻辑中的 ExtractAndSave 不支持“我是工程师”这类模式，extracted 会为空
//   - 新逻辑应直接消费 ExtractPreferences 的结果，返回 “已记住：职业 = 工程师”
func TestPrepare_ExtractedInfoUsesExtractPreferences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"职业\":\"工程师\"}"}}]}`))
	}))
	defer server.Close()

	cfg := &config.APIConfig{
		LLMConfig: config.LLMConfig{
			LLMAPIUrl:   server.URL,
			LLMAPIKey:   "test-key",
			LLMModel:    "test-model",
			Temperature: 0,
		},
		MemoryConfig: config.MemoryConfig{
			ShortTermMaxTurns: 5,
		},
	}
	agent := newPrepareTestAgent(cfg)
	ctx := usercontext.With(context.Background(), "user-1", "alice")

	got := agent.prepareAndSave(ctx, "我是工程师", ChatOptions{})

	if got.extracted != "已记住：职业 = 工程师" {
		t.Fatalf("extracted 回显不正确，期望 %q，得到 %q", "已记住：职业 = 工程师", got.extracted)
	}
}
