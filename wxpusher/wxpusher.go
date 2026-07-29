// Package wxpusher 基于WxPusher系统的微信通知SDK
// 完全独立于SunShop系统，可作为独立的Go库使用
package wxpusher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ==================== 配置结构 ====================

// WxPusherConfig WxPusher 通知配置
type WxPusherConfig struct {
	AppToken string   `json:"app_token"` // 应用令牌
	UIDs     []string `json:"uids"`      // 用户UID列表
	TopicIDs []int    `json:"topic_ids"` // 主题ID列表（可选）
}

// ==================== 请求/响应结构 ====================

// SendRequest 发送消息请求
type SendRequest struct {
	AppToken      string   `json:"appToken"`                // 应用令牌
	Content       string   `json:"content"`                 // 消息内容
	Summary       string   `json:"summary,omitempty"`       // 消息摘要
	ContentType   int      `json:"contentType"`             // 内容类型: 1=文字, 2=HTML, 3=Markdown
	TopicIDs      []int    `json:"topicIds,omitempty"`      // 主题ID列表
	UIDs          []string `json:"uids,omitempty"`          // 用户UID列表
	URL           string   `json:"url,omitempty"`           // 原文链接
	VerifyPayType int      `json:"verifyPayType,omitempty"` // 验证类型: 0=不验证, 1=仅付费用户, 2=仅未订阅/过期用户
}

// SendResponse 发送消息响应
type SendResponse struct {
	Code    int          `json:"code"`    // 状态码，1000表示成功
	Msg     string       `json:"msg"`     // 提示消息
	Data    []SendStatus `json:"data"`    // 发送状态
	Success bool         `json:"success"` // 是否成功
}

// SendStatus 发送状态
type SendStatus struct {
	UID              string `json:"uid"`               // 用户UID
	TopicID          *int   `json:"topicId"`           // 主题ID
	MessageContentID int    `json:"messageContentId"`  // 消息内容ID
	SendRecordID     int    `json:"sendRecordId"`      // 消息发送记录ID
	Code             int    `json:"code"`              // 1000表示发送成功
	Status           string `json:"status"`            // 发送状态描述
}

// ==================== 常量 ====================

const (
	// SendAPI WxPusher 发送消息接口地址
	SendAPI = "https://wxpusher.zjiecode.com/api/send/message"
	// DefaultTimeout HTTP 请求默认超时时间（秒）
	DefaultTimeout = 30
)

// ==================== 核心发送接口 ====================

// SendMessage 发送微信通知消息
//   - cfg: WxPusher 配置（AppToken、UIDs、TopicIDs）
//   - content: 消息内容（支持HTML格式，当 contentType=2 时）
//   - summary: 消息摘要（可选，不传则自动截取content前100字）
//   - contentType: 内容类型（1=文字, 2=HTML, 3=Markdown）
//   - url: 原文链接（可选）
//
// 返回: 发送响应（含每个目标的发送状态）
func SendMessage(cfg WxPusherConfig, content, summary string, contentType int, url string) (*SendResponse, error) {
	if cfg.AppToken == "" {
		return nil, fmt.Errorf("AppToken 未配置")
	}
	if len(cfg.UIDs) == 0 && len(cfg.TopicIDs) == 0 {
		return nil, fmt.Errorf("未配置接收目标（UIDs 或 TopicIDs）")
	}

	req := SendRequest{
		AppToken:      cfg.AppToken,
		Content:       content,
		Summary:       summary,
		ContentType:   contentType,
		TopicIDs:      cfg.TopicIDs,
		UIDs:          cfg.UIDs,
		URL:           url,
		VerifyPayType: 0,
	}

	// 自动填充摘要
	if summary == "" {
		runes := []rune(stripHTMLTags(content))
		if len(runes) > 100 {
			req.Summary = string(runes[:100])
		} else {
			req.Summary = string(runes)
		}
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("请求序列化失败: %w", err)
	}

	client := &http.Client{Timeout: DefaultTimeout * time.Second}
	resp, err := client.Post(SendAPI, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result SendResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}

	if result.Code != 1000 || !result.Success {
		return nil, fmt.Errorf("发送失败: code=%d, msg=%s", result.Code, result.Msg)
	}

	return &result, nil
}

// SendHTML 发送HTML格式消息（便捷方法）
// contentType 固定为 2 (HTML)
func SendHTML(cfg WxPusherConfig, htmlContent, summary string) (*SendResponse, error) {
	return SendMessage(cfg, htmlContent, summary, 2, "")
}

// SendText 发送纯文本消息（便捷方法）
// contentType 固定为 1 (文字)
func SendText(cfg WxPusherConfig, text string) (*SendResponse, error) {
	return SendMessage(cfg, text, text, 1, "")
}

// SendMarkdown 发送Markdown消息（便捷方法）
// contentType 固定为 3 (Markdown)
func SendMarkdown(cfg WxPusherConfig, markdown, summary string) (*SendResponse, error) {
	return SendMessage(cfg, markdown, summary, 3, "")
}

// ==================== 内置实用函数 ====================

// stripHTMLTags 去除HTML标签，提取纯文本
func stripHTMLTags(html string) string {
	var result []byte
	inTag := false
	for i := 0; i < len(html); i++ {
		if html[i] == '<' {
			inTag = true
			continue
		}
		if html[i] == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result = append(result, html[i])
		}
	}
	return string(result)
}