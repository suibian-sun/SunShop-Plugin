# SunShop 插件接口说明文档

> 本文档为 SunShop 插件开发者提供标准化的安全接口规范。
> 插件通过调用这些接口实现功能扩展，**所有接口均经过权限校验和安全审计**，不会危害网站安全。

---

## 目录

1. [插件系统架构](#1-插件系统架构)
2. [插件生命周期](#2-插件生命周期)
3. [权限体系](#3-权限体系)
4. [事件钩子](#4-事件钩子)
5. [系统安全 API](#5-系统安全-api)
6. [安全限制](#6-安全限制)
7. [数据访问规范](#7-数据访问规范)
8. [配置管理](#8-配置管理)
9. [最佳实践](#9-最佳实践)

---

## 1. 插件系统架构

```
┌─────────────────────────────────────────────────┐
│                   SunShop 系统                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │  路由层   │  │  中间件   │  │  插件管理器   │  │
│  └────┬─────┘  └────┬─────┘  └──────┬───────┘  │
│       │             │               │           │
│  ┌────▼─────────────▼───────────────▼───────┐   │
│  │             安全沙箱 (Sandbox)            │   │
│  │  ┌─────────────┐  ┌──────────────────┐   │   │
│  │  │  权限校验器   │  │  API 白名单网关   │   │   │
│  │  └─────────────┘  └──────────────────┘   │   │
│  └────────────────┬─────────────────────────┘   │
│                   │                             │
│  ┌────────────────▼─────────────────────────┐   │
│  │              插件实例                      │   │
│  │  ┌──────────┐  ┌──────────┐  ┌────────┐  │   │
│  │  │ 权限域 A  │  │ 权限域 B  │  │ 权限域 C│  │   │
│  │  └──────────┘  └──────────┘  └────────┘  │   │
│  └───────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

### 关键原则

| 原则 | 说明 |
|------|------|
| **最小权限** | 插件只拥有 manifest.json 中声明的权限 |
| **API 白名单** | 插件只能通过系统提供的安全 API 操作数据，不得直接访问数据库 |
| **输入过滤** | 所有插件输入经过严格校验和转义 |
| **沙箱隔离** | 插件运行在隔离环境中，无法访问系统文件或执行系统命令 |
| **速率限制** | 插件 API 调用受频率限制，防止滥用 |

---

## 2. 插件生命周期

插件在 manifest.json 中声明生命周期钩子函数，系统在特定时机自动调用。

### 生命周期钩子

| 钩子 | 触发时机 | 签名 | 说明 |
|------|----------|------|------|
| `on_install` | 商户安装插件时 | `func(config json.RawMessage) error` | 验证配置，初始化数据 |
| `on_uninstall` | 商户卸载插件时 | `func() error` | 清理数据，释放资源 |
| `on_enable` | 商户启用插件时 | `func(config json.RawMessage) error` | 验证配置可用性，建立连接 |
| `on_disable` | 商户禁用插件时 | `func() error` | 安全关闭连接，暂停服务 |

### 生命周期流程

```
安装 → on_install → 初始化完成 → 已安装
                                      ↓
                                 启用 → on_enable → 启用成功
                                                       ↓
                                                  运行中
                                                       ↓
                                 禁用 → on_disable → 已禁用
                                      ↓
                                 卸载 → on_uninstall → 已卸载
```

### 示例

```go
// 安装时校验配置
func OnInstall(config json.RawMessage) error {
    var cfg Config
    if err := json.Unmarshal(config, &cfg); err != nil {
        return fmt.Errorf("配置解析失败: %w", err)
    }
    if cfg.APIURL == "" || cfg.AppID == "" {
        return fmt.Errorf("缺少必要配置项")
    }
    return nil
}

// 启用时验证连通性
func OnEnable(config json.RawMessage) error {
    var cfg Config
    json.Unmarshal(config, &cfg)
    return verifyConnection(cfg)
}
```

---

## 3. 权限体系

插件在 manifest.json 的 `permissions` 字段中声明所需权限。
**系统运行时严格按声明权限进行访问控制，未声明的权限拒绝授予。**

### 权限列表

| 权限标识 | 说明 | 风险等级 | 用途示例 |
|----------|------|----------|----------|
| `order:read` | 读取订单信息 | 低 | 查询订单状态、支付结果 |
| `order:write` | 创建/修改订单 | 中 | 创建子订单、更新订单备注 |
| `payment:process` | 处理支付请求 | 中 | 发起支付、生成支付链接 |
| `payment:callback` | 处理支付回调 | 中 | 验证支付通知、更新订单 |
| `payment:refund` | 执行退款操作 | 高 | 发起退款申请 |
| `product:read` | 读取商品信息 | 低 | 获取商品详情、价格 |
| `product:write` | 创建/修改商品 | 中 | 批量创建商品 |
| `card:read` | 读取卡密信息 | 低 | 查询卡密库存 |
| `card:write` | 导入/导出卡密 | 中 | 批量导入卡密 |
| `user:info:read` | 读取用户基本信息 | 中 | 获取用户联系方式 |
| `merchant:setting:read` | 读取商户设置 | 低 | 获取店铺配置 |
| `merchant:setting:write` | 修改商户设置 | 中 | 更新店铺参数 |
| `log:write` | 写入操作日志 | 低 | 记录插件操作日志 |
| `notification:send` | 发送通知消息 | 低 | 发送订单通知 |
| `http:request` | 发起外部 HTTP 请求 | 中 | 调用第三方 API |

### 权限声明示例

```json
{
  "permissions": [
    "order:read",
    "payment:process",
    "payment:callback",
    "http:request"
  ]
}
```

### 权限校验规则

1. 插件安装时，系统记录所需权限清单
2. 运行时，每次 API 调用检查权限是否已声明
3. 未声明的权限调用将被拒绝并记录安全日志
4. 高权限操作（如 `payment:refund`）需额外二次确认
5. 权限变更需重新安装插件

---

## 4. 事件钩子

插件可以监听系统事件，在事件发生时自动执行回调函数。
事件在 manifest.json 的 `events` 字段中声明。

### 系统事件列表

| 事件标识 | 触发时机 | 负载数据 | 风险等级 |
|----------|----------|----------|----------|
| `order:created` | 新订单创建 | 订单号、金额、商品信息 | 低 |
| `order:paid` | 订单支付成功 | 订单号、支付方式、支付时间 | 低 |
| `order:delivered` | 订单发货完成 | 订单号、发货方式 | 低 |
| `order:refunded` | 订单退款完成 | 订单号、退款金额 | 低 |
| `payment:created` | 支付请求发起 | 支付单号、金额、支付方式 | 低 |
| `payment:callback` | 收到支付回调 | 回调参数、验证结果 | 中 |
| `payment:refund` | 退款发起 | 交易号、退款金额 | 中 |
| `product:sold_out` | 商品售罄 | 商品ID、商品名称 | 低 |
| `card:stock_low` | 卡密库存不足 | 商品ID、剩余数量 | 低 |
| `merchant:registered` | 新商户注册 | 商户ID、商户名称 | 低 |

### 事件声明示例

```json
{
  "events": {
    "order:paid": "epay.onOrderPaid",
    "payment:callback": "epay.onPaymentCallback",
    "payment:refund": "epay.onPaymentRefund"
  }
}
```

### 事件处理函数签名

```go
// 事件处理函数接收上下文和事件数据
func OnOrderPaid(ctx EventContext, data OrderPaidEvent) error {
    // 事件处理逻辑
    return nil
}

// 事件上下文
type EventContext struct {
    MerchantID uint   // 触发事件的商户ID
    PluginID   uint   // 当前插件ID
    Timestamp  int64  // 事件触发时间戳
}

// 事件负载
type OrderPaidEvent struct {
    OrderNo     string  `json:"order_no"`
    Amount      float64 `json:"amount"`
    PayMethod   string  `json:"pay_method"`
    PaidAt      string  `json:"paid_at"`
}
```

### 事件安全规则

1. 事件处理函数**不能阻塞主流程**，超时（默认 5 秒）后自动跳过
2. 事件处理失败不影响系统正常业务流程
3. 事件处理中产生的错误会记录到插件日志，不影响系统稳定性
4. 同一事件可被多个插件监听，按插件安装顺序依次调用

---

## 5. 系统安全 API

插件可以通过系统提供的安全 API 操作数据，所有 API 都经过权限校验。

### 5.1 订单 API

#### 读取订单

```go
// 根据订单号查询订单
func GetOrder(orderNo string) (*Order, error)
// 权限要求: order:read
// 返回: 订单信息（脱敏处理，隐藏完整手机号/邮箱）

// 查询订单列表
func ListOrders(filter OrderFilter) ([]Order, int64, error)
// 权限要求: order:read
// 参数: 支持按状态、时间范围、商品ID过滤
// 返回: 订单列表和总数
```

#### 更新订单

```go
// 更新订单备注
func UpdateOrderNote(orderNo string, note string) error
// 权限要求: order:write
// 限制: 只能修改备注字段，不能修改金额、状态等关键字段

// 标记订单异常
func MarkOrderAbnormal(orderNo string, reason string) error
// 权限要求: order:write
```

### 5.2 支付 API

#### 处理支付

```go
// 创建支付请求
func CreatePayment(req PaymentRequest) (*PaymentResponse, error)
// 权限要求: payment:process
// 限制: 金额不能超过订单金额，支付方式必须在系统配置中启用

// 验证支付回调签名
func VerifyCallback(params CallbackParams) (bool, error)
// 权限要求: payment:callback
```

#### 退款

```go
// 发起退款
func Refund(tradeNo string, amount float64) error
// 权限要求: payment:refund
// 限制: 退款金额不能超过原支付金额，同一订单累计退款不超过原金额
// 安全: 需要二次确认，记录详细退款日志
```

### 5.3 商品 API

```go
// 读取商品信息
func GetProduct(productID uint) (*Product, error)
// 权限要求: product:read

// 查询商品列表
func ListProducts(filter ProductFilter) ([]Product, int64, error)
// 权限要求: product:read
```

### 5.4 卡密 API

```go
// 查询卡密库存
func GetCardStock(productID uint) (int64, error)
// 权限要求: card:read

// 批量导入卡密
func ImportCards(productID uint, cards []string) (int, error)
// 权限要求: card:write
// 限制: 单次最多导入 1000 条，每条卡密不能超过 500 字符
```

### 5.5 HTTP 请求 API

```go
// 发起外部 HTTP 请求（通过系统代理）
func HTTPRequest(req HTTPRequestConfig) (*HTTPResponse, error)
// 权限要求: http:request
// 限制:
//   - 只支持 HTTPS
//   - 超时时间 15 秒
//   - 禁止访问内网地址（127.0.0.1, 10.x.x.x, 172.16.x.x, 192.168.x.x）
//   - 禁止访问敏感端口（22, 3306, 6379 等）
//   - 请求频率限制：每分钟 60 次

type HTTPRequestConfig struct {
    Method  string            // GET / POST / PUT
    URL     string            // 仅支持 HTTPS
    Headers map[string]string // 自定义请求头
    Body    []byte            // 请求体
    Timeout int               // 超时时间（秒），默认 15，最大 30
}

type HTTPResponse struct {
    StatusCode int
    Headers    map[string]string
    Body       []byte
}
```

### 5.6 日志 API

```go
// 记录插件操作日志
func LogPluginAction(action string, detail string) error
// 权限要求: log:write
// 说明: 日志会记录到系统操作日志表中，可用于审计追踪

// 获取插件日志
func GetPluginLogs(filter LogFilter) ([]OperationLog, int64, error)
// 权限要求: log:write
// 说明: 只返回当前插件的日志，不能查看其他插件的日志
```

### 5.7 通知 API

```go
// 发送通知消息
func SendNotification(merchantID uint, title string, content string) error
// 权限要求: notification:send
// 限制: 不能伪造系统通知，通知类型会被标记为"插件通知"
```

---

## 6. 安全限制

### 6.1 绝对禁止的操作

| 操作 | 说明 | 后果 |
|------|------|------|
| 直接操作数据库 | 插件不能执行 SQL 语句或直接访问数据库连接 | 拒绝执行，记录安全警告 |
| 执行系统命令 | 插件不能调用 `exec`、`shell` 等系统命令 | 拒绝执行，记录安全警告 |
| 读写文件系统 | 插件不能创建、读取、修改或删除服务器文件 | 拒绝执行，记录安全警告 |
| 访问内网地址 | 插件不能请求内网 IP 或 localhost | 请求被拦截，记录安全警告 |
| 加载动态代码 | 插件不能动态加载外部代码或脚本 | 拒绝执行，记录安全警告 |
| 修改系统配置 | 插件不能修改 SunShop 系统级别的配置 | 拒绝执行，记录安全警告 |
| 访问其他插件数据 | 插件不能读取或修改其他插件的数据 | 拒绝执行，记录安全警告 |
| 发送未授权请求 | 插件不能发起未在 manifest 中声明的 HTTP 请求 | 请求被拦截 |

### 6.2 自动安全措施

| 措施 | 说明 |
|------|------|
| 输入消毒 | 所有用户输入和外部数据自动进行 XSS 过滤和 SQL 注入防护 |
| 输出编码 | 输出的 HTML 内容自动编码，防止 XSS 攻击 |
| 速率限制 | 每个插件每分钟 API 调用不超过 120 次 |
| 请求超时 | 所有外部 HTTP 请求强制 15 秒超时 |
| 敏感数据脱敏 | 手机号、邮箱、银行卡等敏感信息自动脱敏 |
| 数据量限制 | 列表查询单次最多返回 100 条记录 |
| 并发控制 | 单个插件同一时间最多 5 个并发操作 |

### 6.3 安全审计

所有插件操作均记录审计日志，包括：

- API 调用时间、调用方、调用参数（脱敏）
- 权限校验结果（通过/拒绝）
- 数据变更前后的关键字段值
- 异常操作和安全违规事件

---

## 7. 数据访问规范

### 7.1 数据获取范围

| 数据类型 | 可访问范围 | 说明 |
|----------|------------|------|
| 订单 | 仅当前商户的订单 | 不能访问其他商户的订单 |
| 商品 | 仅当前商户的商品 | 不能访问其他商户的商品 |
| 卡密 | 仅当前商户的卡密 | 不能访问其他商户的卡密 |
| 用户 | 与当前商户订单相关的用户 | 不能批量查询所有用户 |
| 配置 | 仅当前插件的配置 | 不能访问系统配置或其他插件配置 |

### 7.2 数据脱敏规则

| 字段类型 | 脱敏方式 | 示例 |
|----------|----------|------|
| 手机号 | 中间 4 位隐藏 | `138****1234` |
| 邮箱 | @ 前保留首字母 | `u***@example.com` |
| 银行卡号 | 仅显示后 4 位 | `**** **** **** 1234` |
| 密码/密钥 | 始终不返回 | `[已加密]` |
| IP 地址 | 仅保留前 3 段 | `192.168.1.*` |

### 7.3 数据写入限制

| 操作 | 限制 |
|------|------|
| 单次写入数据量 | 不超过 1MB |
| 批量导入条数 | 单次不超过 1000 条 |
| 字段长度 | 字符串字段不超过 5000 字符 |
| 写入频率 | 每分钟不超过 60 次写入操作 |

---

## 8. 配置管理

### 8.1 配置声明

插件在 manifest.json 的 `config_schema` 中声明配置项，系统自动生成配置页面。

```json
{
  "config_schema": {
    "type": "object",
    "required": ["api_url", "api_key"],
    "properties": {
      "api_url": {
        "type": "string",
        "title": "API 接口地址",
        "description": "第三方服务的 API 地址",
        "format": "uri",
        "minLength": 5,
        "maxLength": 500
      },
      "api_key": {
        "type": "string",
        "title": "API 密钥",
        "description": "第三方服务分配的密钥",
        "secret": true,
        "minLength": 8
      },
      "timeout": {
        "type": "integer",
        "title": "超时时间（秒）",
        "default": 15,
        "minimum": 5,
        "maximum": 30
      }
    }
  }
}
```

### 8.2 配置安全处理

| 特性 | 说明 |
|------|------|
| 加密存储 | 标记为 `secret: true` 的字段自动加密存储（AES-256-GCM） |
| 敏感信息掩码 | 返回配置时，密钥类字段自动掩码显示 |
| 配置校验 | 系统根据 schema 自动校验配置格式 |
| 变更审计 | 配置变更记录到操作日志，包含变更人、时间、变更字段 |

---

## 9. 最佳实践

### 9.1 权限声明建议

- **只申请必要权限**：仅声明插件确实需要的权限，最小化安全风险
- **按需升级权限**：插件新版本需要更多权限时，在更新说明中说明原因
- **定期审查**：定期检查插件权限，移除不再需要的权限

### 9.2 安全开发建议

- **使用系统 HTTP 客户端**：不要自行实现 HTTP 请求，使用系统提供的 `HTTPRequest` API
- **过滤外部数据**：对第三方 API 返回的数据进行校验和限制
- **避免敏感日志**：不要在日志中记录密钥、密码等敏感信息
- **错误处理**：合理处理错误，避免将系统内部错误信息暴露给用户
- **配置校验**：在 `on_install` 和 `on_enable` 中充分校验配置有效性

### 9.3 事件处理建议

- **保持幂等**：事件处理函数应该是幂等的，重复执行不会产生副作用
- **快速返回**：事件处理应尽快完成，避免耗时操作
- **错误容忍**：事件处理失败不应影响主流程
- **异步处理**：耗时操作（如调用第三方 API）使用异步方式

### 9.4 性能建议

- **控制数据量**：避免一次查询过多数据，合理使用分页
- **缓存外部数据**：对第三方 API 返回的数据进行本地缓存，减少重复请求
- **避免频繁写入**：批量操作时合并写入，减少数据库压力
- **合理设置超时**：HTTP 请求超时时间不宜过长，推荐 5-15 秒

---

## 附录

### A. 快速参考

```go
// 插件入口函数
func Info() map[string]interface{}           // 返回插件信息
func OnInstall(config json.RawMessage) error  // 安装钩子
func OnUninstall() error                       // 卸载钩子
func OnEnable(config json.RawMessage) error    // 启用钩子
func OnDisable() error                         // 禁用钩子

// 安全 API
func GetOrder(orderNo string) (*Order, error)
func ListOrders(filter OrderFilter) ([]Order, int64, error)
func CreatePayment(req PaymentRequest) (*PaymentResponse, error)
func VerifyCallback(params CallbackParams) (bool, error)
func Refund(tradeNo string, amount float64) error
func GetProduct(productID uint) (*Product, error)
func ImportCards(productID uint, cards []string) (int, error)
func HTTPRequest(req HTTPRequestConfig) (*HTTPResponse, error)
func LogPluginAction(action string, detail string) error
func SendNotification(merchantID uint, title string, content string) error
```

### B. 完整示例

参见 [epay/](epay/) 目录下的易支付插件实现，展示了完整的插件开发流程。