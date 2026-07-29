# 微信通知插件 - WxPusher

> 基于 WxPusher 系统的微信通知插件，向微信用户推送订单通知、工单通知等消息。
> 本插件完全独立于 SunShop 系统，可作为独立的 Go SDK 或 HTTP 服务使用。

---

## 目录

1. [插件概述](#1-插件概述)
2. [安装与配置](#2-安装与配置)
3. [API 接口](#3-api-接口)
4. [消息格式](#4-消息格式)
5. [事件通知](#5-事件通知)
6. [安全限制](#6-安全限制)
7. [最佳实践](#7-最佳实践)

---

## 1. 插件概述

### 1.1 功能特性

| 功能 | 说明 |
|------|------|
| 文字消息 | 发送纯文本消息到微信 |
| HTML 消息 | 发送富文本 HTML 消息，支持样式和 `<copy>` 复制标签 |
| Markdown 消息 | 发送 Markdown 格式消息 |
| 多目标推送 | 支持按 UID 单发或按 TopicID 群发 |
| 消息摘要 | 自定义消息摘要，控制通知栏显示内容 |
| 原文链接 | 可选的消息原文链接 |

### 1.2 架构说明

```
┌─────────────────────────────────────────────────────────┐
│                    第三方系统 / SunShop                    │
│                         │                                │
│                         ▼                                │
│  ┌─────────────────────────────────────────────────┐    │
│  │              WxPusher 插件 (本 SDK)               │    │
│  │    ┌─────────────┐  ┌──────────────────────┐    │    │
│  │    │  配置管理     │  │  消息构建与格式化     │    │    │
│  │    └──────┬──────┘  └──────────┬───────────┘    │    │
│  │           │                    │                 │    │
│  │    ┌──────▼────────────────────▼───────────┐    │    │
│  │    │           HTTP 请求发送层               │    │    │
│  │    └──────────────────┬────────────────────┘    │    │
│  └───────────────────────┼─────────────────────────┘    │
│                          │                              │
└──────────────────────────┼──────────────────────────────┘
                           │ HTTPS
                           ▼
              ┌──────────────────────┐
              │  WxPusher API 服务    │
              │  wxpusher.zjiecode.com│
              └──────────┬───────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │     微信用户          │
              └──────────────────────┘
```

---

## 2. 安装与配置

### 2.1 获取 WxPusher 凭证

1. 访问 [WxPusher 官网](https://wxpusher.zjiecode.com) 注册账号
2. 创建应用，获取 **AppToken**（应用令牌）
3. 在 [关注管理页](https://wxpusher.zjiecode.com/wxuser/?type=1&id=133355#/follow) 获取用户的 **UID**

### 2.2 配置参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `app_token` | string | 是 | WxPusher 应用令牌，用于 API 鉴权 |
| `uids` | string[] | 否* | 接收消息的微信用户 UID 列表 |
| `topic_ids` | int[] | 否* | 群发主题 ID 列表 |

> *`uids` 和 `topic_ids` 至少需要配置一个

### 2.3 配置示例

```json
{
  "app_token": "AT_xxxxxxxxxxxxxxxxxxxxxxxxxx",
  "uids": ["UID_xxxxxxxxxxxxx"],
  "topic_ids": [123]
}
```

---

## 3. API 接口

### 3.1 发送消息

发送消息到微信用户，支持文字、HTML、Markdown 三种格式。

#### 请求

```
POST https://wxpusher.zjiecode.com/api/send/message
Content-Type: application/json
```

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `appToken` | string | 是 | WxPusher 应用令牌 |
| `content` | string | 是 | 消息内容 |
| `summary` | string | 否 | 消息摘要，最长100字，不传则截取 content 前100字 |
| `contentType` | int | 是 | 内容类型: 1=文字, 2=HTML, 3=Markdown |
| `uids` | string[] | 否 | 用户 UID 列表 |
| `topicIds` | int[] | 否 | 主题 ID 列表 |
| `url` | string | 否 | 原文链接 |
| `verifyPayType` | int | 否 | 验证类型: 0=不验证, 1=仅付费用户, 2=仅未订阅/过期用户 |

#### 请求示例

```json
{
  "appToken": "AT_xxx",
  "content": "<h1>订单通知</h1><p>您的订单已支付成功</p>",
  "summary": "订单支付成功通知",
  "contentType": 2,
  "uids": ["UID_xxxx"],
  "verifyPayType": 0
}
```

#### 响应参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `code` | int | 状态码，1000 表示成功 |
| `msg` | string | 提示消息 |
| `success` | bool | 是否成功 |
| `data` | array | 每个目标的发送状态 |

#### 响应示例

```json
{
  "code": 1000,
  "msg": "处理成功",
  "success": true,
  "data": [
    {
      "uid": "UID_xxx",
      "messageContentId": 2123,
      "sendRecordId": 12313,
      "code": 1000,
      "status": "创建发送任务成功"
    }
  ]
}
```

### 3.2 Go SDK 调用

本插件可作为 Go SDK 直接引用：

```go
import "github.com/suibian-sun/sunshop-plugin-wxpusher"

// 配置
cfg := wxpusher.WxPusherConfig{
    AppToken: "AT_xxxxxxxxxxxxxxxxxxxx",
    UIDs:     []string{"UID_xxxxxxxxxx"},
}

// 发送 HTML 消息
resp, err := wxpusher.SendHTML(cfg, 
    "<h2>订单通知</h2><p>订单已支付成功</p>", 
    "订单支付成功通知")

// 发送纯文本消息
resp, err := wxpusher.SendText(cfg, "您的订单已支付成功")

// 发送 Markdown 消息
resp, err := wxpusher.SendMarkdown(cfg, 
    "# 订单通知\n\n您的订单已支付成功", 
    "订单支付成功通知")

// 自定义发送
resp, err := wxpusher.SendMessage(cfg, content, summary, contentType, url)
```

### 3.3 HTTP 直接调用

不依赖 Go SDK，直接通过 HTTP 调用 WxPusher API：

```bash
curl -X POST https://wxpusher.zjiecode.com/api/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "appToken": "AT_xxx",
    "content": "<h2>订单通知</h2><p>订单已支付成功</p>",
    "summary": "订单支付成功通知",
    "contentType": 2,
    "uids": ["UID_xxx"]
  }'
```

---

## 4. 消息格式

### 4.1 HTML 消息 (contentType=2)

支持完整的 HTML 标签，推荐使用以下结构：

```html
<h2 style="color:#409eff;">消息标题</h2>
<div style="padding:10px 0;line-height:1.8;color:#333;">
  <p>消息内容</p>
</div>
<table style="width:100%;border-collapse:collapse;">
  <tr>
    <td style="padding:8px;border:1px solid #ddd;color:#666;">字段名</td>
    <td style="padding:8px;border:1px solid #ddd;">字段值</td>
  </tr>
</table>
```

### 4.2 复制按钮

HTML 消息支持通过 `<copy>` 标签实现一键复制：

```html
<copy data-clipboard-text="需要复制的内容">
  复制按钮文字
</copy>
```

示例 - 复制订单号：

```html
<copy data-clipboard-text="ORD20260729123456" style="color:#409eff;cursor:pointer;">
  点击复制订单号
</copy>
```

### 4.3 消息模板

#### 订单通知模板

```html
<h2 style="color:#409eff;">订单通知</h2>
<table style="width:100%;border-collapse:collapse;margin:10px 0;">
  <tr>
    <td style="padding:8px;border:1px solid #ddd;color:#666;">订单编号</td>
    <td style="padding:8px;border:1px solid #ddd;">
      <copy data-clipboard-text="ORD20260729123456" style="color:#409eff;cursor:pointer;">
        ORD20260729123456
      </copy>
    </td>
  </tr>
  <tr>
    <td style="padding:8px;border:1px solid #ddd;color:#666;">商品名称</td>
    <td style="padding:8px;border:1px solid #ddd;">商品名称</td>
  </tr>
  <tr>
    <td style="padding:8px;border:1px solid #ddd;color:#666;">订单金额</td>
    <td style="padding:8px;border:1px solid #ddd;font-weight:bold;color:#e4393c;">¥99.00</td>
  </tr>
  <tr>
    <td style="padding:8px;border:1px solid #ddd;color:#666;">订单状态</td>
    <td style="padding:8px;border:1px solid #ddd;color:#67c23a;">已付款</td>
  </tr>
</table>
```

#### 工单通知模板

```html
<h2 style="color:#e6a23c;">工单通知</h2>
<table style="width:100%;border-collapse:collapse;margin:10px 0;">
  <tr>
    <td style="padding:8px;border:1px solid #ddd;color:#666;">工单编号</td>
    <td style="padding:8px;border:1px solid #ddd;">
      <copy data-clipboard-text="TK1712345678" style="color:#409eff;cursor:pointer;">
        TK1712345678
      </copy>
    </td>
  </tr>
  <tr>
    <td style="padding:8px;border:1px solid #ddd;color:#666;">订单编号</td>
    <td style="padding:8px;border:1px solid #ddd;">ORD20260729123456</td>
  </tr>
  <tr>
    <td style="padding:8px;border:1px solid #ddd;color:#666;">问题类型</td>
    <td style="padding:8px;border:1px solid #ddd;">未收到卡密</td>
  </tr>
  <tr>
    <td style="padding:8px;border:1px solid #ddd;color:#666;">问题描述</td>
    <td style="padding:8px;border:1px solid #ddd;">已付款但未收到卡密信息</td>
  </tr>
</table>
```

---

## 5. 事件通知

### 5.1 支持的事件

| 事件 | 触发时机 | 推送内容 |
|------|----------|----------|
| 订单支付成功 | 买家完成支付 | 订单号、商品名、金额、联系方式 |
| 新工单创建 | 买家提交售后工单 | 工单号、订单号、问题类型、描述 |
| 系统通知 | 商户后台发送通知 | 通知标题和内容 |

### 5.2 接入 SunShop 系统

在 SunShop 系统中，商户后台配置通知渠道后，以下事件会自动推送微信通知：

1. **买家付款通知** - 订单支付成功后自动推送
2. **新工单通知** - 买家提交售后工单后自动推送
3. **系统通知** - 商户后台发送的所有通知消息

---

## 6. 安全限制

### 6.1 API 调用限制

| 限制项 | 说明 |
|--------|------|
| 请求频率 | 建议每分钟不超过 60 次 |
| 内容长度 | content 建议不超过 5000 字符 |
| 摘要长度 | summary 最长 100 字符 |
| 超时时间 | HTTP 请求默认 30 秒超时 |

### 6.2 安全建议

| 建议 | 说明 |
|------|------|
| 令牌保护 | AppToken 等同于密码，请勿在客户端代码中暴露 |
| HTTPS 传输 | 所有 API 通信均使用 HTTPS 加密 |
| 敏感信息脱敏 | 不要在消息内容中包含密码、密钥等敏感信息 |
| 摘要截断 | 摘要不宜过长，控制在 20 字以内效果最佳 |

---

## 7. 最佳实践

### 7.1 消息设计

- **标题清晰**：使用 `<h2>` 标签区分不同消息类型（订单用蓝色、工单用橙色）
- **关键信息突出**：金额用红色加粗、状态用绿色、可复制内容用蓝色
- **摘要简洁**：摘要控制在 20 字以内，展示最核心信息
- **复制功能**：对订单号、工单号等关键编号添加 `<copy>` 复制按钮

### 7.2 错误处理

- 发送失败时记录日志，但不要阻塞主业务流程
- 使用异步方式发送消息，避免影响接口响应时间
- 对 AppToken 和 UIDs 进行有效性校验

### 7.3 性能优化

- 合并多条消息为一条发送，减少 API 调用次数
- 对相同内容的消息进行去重
- 使用连接池复用 HTTP 连接

---

## 附录

### A. 状态码说明

| 状态码 | 说明 |
|--------|------|
| 1000 | 处理成功 |
| 1001 | 参数错误 |
| 1002 | 鉴权失败，AppToken 无效 |
| 1003 | 发送目标不存在 |
| 1004 | 消息内容违规 |
| 1005 | 频率限制 |
| 9999 | 系统异常 |

### B. 快速参考

```go
// 发送消息
func SendMessage(cfg WxPusherConfig, content, summary string, contentType int, url string) (*SendResponse, error)

// 便捷方法
func SendHTML(cfg WxPusherConfig, htmlContent, summary string) (*SendResponse, error)
func SendText(cfg WxPusherConfig, text string) (*SendResponse, error)
func SendMarkdown(cfg WxPusherConfig, markdown, summary string) (*SendResponse, error)
```

### C. 相关链接

- [WxPusher 官方文档](https://wxpusher.zjiecode.com/docs)
- [WxPusher 关注管理](https://wxpusher.zjiecode.com/wxuser/?type=1&id=133355#/follow)
- [SunShop 插件系统文档](../server/plugins/API.md)