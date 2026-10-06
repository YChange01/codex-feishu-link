package orchestrator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

func (s *Service) handleQuotaCommand(surface *state.SurfaceConsoleRecord, action control.Action) []eventcontract.Event {
	if surface == nil {
		return nil
	}
	inst := s.root.Instances[surface.AttachedInstanceID]
	if inst == nil || !inst.Online {
		return notice(surface, "quota_unavailable", "当前没有连接到桌面 Codex。请先打开电脑端 Codex 和飞书桥接，再查询额度。")
	}
	if inst.Backend != agentproto.BackendCodex || !inst.Capabilities.SharedAppServer {
		return notice(surface, "quota_unavailable", "额度查询仅支持已连接的桌面 Codex 共享会话。")
	}
	return []eventcontract.Event{{
		Kind:             eventcontract.KindAgentCommand,
		SurfaceSessionID: surface.SurfaceSessionID,
		SourceMessageID:  action.MessageID,
		Command: &agentproto.Command{
			Kind: agentproto.CommandRateLimitsRead,
			Origin: agentproto.Origin{
				Surface:   surface.SurfaceSessionID,
				UserID:    surface.ActorUserID,
				ChatID:    surface.ChatID,
				MessageID: action.MessageID,
			},
		},
	}}
}

func formatQuotaNotice(update *agentproto.CapabilityStateUpdate) string {
	if update == nil {
		return "暂时无法读取 Codex 额度。"
	}
	if update.RateLimitsReadError != "" {
		return "读取 Codex 额度失败：" + update.RateLimitsReadError
	}
	lines := []string{"Codex 剩余额度"}
	appendQuotaWindows(&lines, "", update.RateLimits)
	if len(lines) == 1 {
		lines = append(lines, "Codex 暂未返回可用的限额窗口。请确认桌面端已登录 ChatGPT 账户。")
	}
	if update.RateLimitResetCreditsAvailable != nil {
		lines = append(lines, fmt.Sprintf("可用额度重置次数：%d", *update.RateLimitResetCreditsAvailable))
	}
	return strings.Join(lines, "\n")
}

func appendQuotaWindows(lines *[]string, prefix string, values map[string]map[string]any) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	// Sort to keep the Feishu result stable across responses.
	sort.Strings(keys)
	for _, key := range keys {
		window := values[key]
		label := strings.TrimSpace(strings.Join([]string{prefix, key}, " "))
		if used, ok := number(window["usedPercent"]); ok {
			remaining := 100 - used
			if remaining < 0 {
				remaining = 0
			}
			if remaining > 100 {
				remaining = 100
			}
			line := fmt.Sprintf("%s：剩余 %.0f%%（已用 %.0f%%）", label, remaining, used)
			if duration, ok := number(window["windowDurationMins"]); ok && duration > 0 {
				line += fmt.Sprintf("，窗口 %s", formatWindowDuration(int(duration)))
			}
			if reset := quotaResetTime(window["resetsAt"]); reset != "" {
				line += "，重置 " + reset
			}
			*lines = append(*lines, line)
			continue
		}
		nestedKeys := make([]string, 0, len(window))
		for nestedKey := range window {
			nestedKeys = append(nestedKeys, nestedKey)
		}
		sort.Strings(nestedKeys)
		for _, nestedKey := range nestedKeys {
			nestedValue := window[nestedKey]
			nested, ok := nestedValue.(map[string]any)
			if !ok || numberMap(nested) == false {
				continue
			}
			appendQuotaWindow(lines, label+" "+nestedKey, nested)
		}
	}
}

func numberMap(values map[string]any) bool { _, ok := number(values["usedPercent"]); return ok }

func appendQuotaWindow(lines *[]string, label string, window map[string]any) {
	used, ok := number(window["usedPercent"])
	if !ok {
		return
	}
	remaining := 100 - used
	if remaining < 0 {
		remaining = 0
	}
	if remaining > 100 {
		remaining = 100
	}
	line := fmt.Sprintf("%s：剩余 %.0f%%（已用 %.0f%%）", label, remaining, used)
	if duration, ok := number(window["windowDurationMins"]); ok && duration > 0 {
		line += fmt.Sprintf("，窗口 %s", formatWindowDuration(int(duration)))
	}
	if reset := quotaResetTime(window["resetsAt"]); reset != "" {
		line += "，重置 " + reset
	}
	*lines = append(*lines, line)
}

func number(value any) (float64, bool) {
	switch value := value.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func formatWindowDuration(minutes int) string {
	if minutes >= 60 && minutes%60 == 0 {
		return fmt.Sprintf("%d 小时", minutes/60)
	}
	return fmt.Sprintf("%d 分钟", minutes)
}

func quotaResetTime(value any) string {
	seconds, ok := number(value)
	if !ok || seconds <= 0 {
		return ""
	}
	return time.Unix(int64(seconds), 0).Local().Format("01-02 15:04")
}
