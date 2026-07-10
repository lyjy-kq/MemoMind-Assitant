// Package preference 用户偏好记忆。
//
// 以键值对形式存储用户偏好（姓名 / 喜好 / 城市 等结构化属性）。
// 写入由上层统一完成：先用 ExtractPreferences 抽取，再异步写入此存储。
//
// 并发安全：所有公共方法持锁。直接读 .Data 字段（旧代码 / JSON 序列化）请用 Snapshot()。
package preference

import (
	"fmt"
	"strings"
	"sync"
)

// Preference 以键值对形式存储用户偏好信息
type Preference struct {
	mu   sync.RWMutex
	Data map[string]string `json:"data"`
}

// New 创建用户偏好存储
func New() *Preference {
	return &Preference{Data: make(map[string]string)}
}

// Save 保存单条偏好
func (p *Preference) Save(key, value string) {
	if key == "" || value == "" {
		return
	}
	p.mu.Lock()
	p.Data[key] = value
	p.mu.Unlock()
}

// SaveBatch 批量保存偏好（从 LLM 提取结果）
func (p *Preference) SaveBatch(kvs map[string]string) {
	p.mu.Lock()
	for k, v := range kvs {
		if k != "" && v != "" {
			p.Data[k] = v
		}
	}
	p.mu.Unlock()
}

// Get 安全读取单条偏好
func (p *Preference) Get(key string) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.Data[key]
	return v, ok
}

// Snapshot 返回偏好数据的浅拷贝（持读锁）
func (p *Preference) Snapshot() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cp := make(map[string]string, len(p.Data))
	for k, v := range p.Data {
		cp[k] = v
	}
	return cp
}

// BuildContext 将偏好数据格式化为给 LLM 的上下文字符串
func (p *Preference) BuildContext() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.Data) == 0 {
		return ""
	}
	items := make([]string, 0, len(p.Data))
	for k, v := range p.Data {
		items = append(items, fmt.Sprintf("%s: %s", k, v))
	}
	return "【用户偏好】\n" + strings.Join(items, "\n")
}
