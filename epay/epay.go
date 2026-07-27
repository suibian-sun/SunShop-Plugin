// Package epay 易支付插件 - SunShop 发卡网支付通道
// 本文件为插件源码，运行时由 SunShop 插件系统动态加载
package epay

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// ==================== 配置结构 ====================

// Config 易支付插件配置
type Config struct {
	APIURL     string `json:"api_url"`     // 易支付接口地址
	AppID      string `json:"app_id"`      // 商户ID
	AppSecret  string `json:"app_secret"`  // 商户密钥
	PayType    string `json:"pay_type"`    // 默认支付方式
	NotifyURL  string `json:"notify_url"`  // 异步通知地址
	ReturnURL  string `json:"return_url"`  // 同步跳转地址
}

// ==================== 请求/响应结构 ====================

// PaymentRequest 创建支付请求
type PaymentRequest struct {
	OrderNo    string  `json:"order_no"`    // 商户订单号
	Amount     float64 `json:"amount"`      // 支付金额（元）
	ProductName string `json:"product_name"` // 商品名称
	PayType    string  `json:"pay_type"`    // 支付方式
	NotifyURL  string  `json:"notify_url"`  // 异步通知地址
	ReturnURL  string  `json:"return_url"`  // 同步跳转地址
	ClientIP   string  `json:"client_ip"`   // 客户端IP
}

// PaymentResponse 支付响应
type PaymentResponse struct {
	Code       int    `json:"code"`        // 状态码: 1=成功, 0=失败
	Msg        string `json:"msg"`         // 提示信息
	TradeNo    string `json:"trade_no"`    // 平台交易号
	PayURL     string `json:"pay_url"`     // 支付链接
	PayQRCode  string `json:"pay_qrcode"`  // 支付二维码内容
	RealAmount string `json:"real_amount"` // 实际支付金额
}

// CallbackParams 异步通知参数
type CallbackParams struct {
	PID          int     `json:"pid"`           // 商户ID
	TradeNo      string  `json:"trade_no"`      // 平台交易号
	OutTradeNo   string  `json:"out_trade_no"`  // 商户订单号
	Type         string  `json:"type"`          // 支付方式
	Name         string  `json:"name"`          // 商品名称
	Money        float64 `json:"money"`         // 金额
	TradeStatus  string  `json:"trade_status"`  // 支付状态: TRADE_SUCCESS
	Sign         string  `json:"sign"`          // 签名
	SignType     string  `json:"sign_type"`     // 签名方式: MD5 / HMAC-SHA256
	Param        string  `json:"param"`         // 附加参数
	PayTime      string  `json:"pay_time"`      // 支付时间
}

// ==================== 插件接口实现 ====================

// Info 返回插件信息
func Info() map[string]interface{} {
	return map[string]interface{}{
		"slug":        "epay",
		"name":        "易支付",
		"version":     "1.0.0",
		"type":        "payment",
		"description": "集成易支付聚合支付平台，支持支付宝、微信支付、QQ钱包",
	}
}

// OnInstall 插件安装时调用
func OnInstall(config json.RawMessage) error {
	var cfg Config
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("配置解析失败: %w", err)
	}
	if cfg.APIURL == "" || cfg.AppID == "" || cfg.AppSecret == "" {
		return fmt.Errorf("请填写完整的易支付配置信息")
	}
	return nil
}

// OnUninstall 插件卸载时调用
func OnUninstall() error {
	return nil
}

// OnEnable 插件启用时调用
func OnEnable(config json.RawMessage) error {
	var cfg Config
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("配置解析失败: %w", err)
	}
	// 验证配置可用性
	return verifyConfig(cfg)
}

// OnDisable 插件禁用时调用
func OnDisable() error {
	return nil
}

// ==================== 支付核心接口 ====================

// CreatePayment 创建支付订单
// 返回: 支付URL, 交易号, 错误
func CreatePayment(cfg Config, req PaymentRequest) (*PaymentResponse, error) {
	payType := req.PayType
	if payType == "" {
		payType = cfg.PayType
	}
	if payType == "" || payType == "all" {
		payType = "alipay" // 默认支付宝
	}

	notifyURL := cfg.NotifyURL
	if req.NotifyURL != "" {
		notifyURL = req.NotifyURL
	}

	returnURL := cfg.ReturnURL
	if req.ReturnURL != "" {
		returnURL = req.ReturnURL
	}

	// 构建请求参数
	params := map[string]string{
		"pid":          cfg.AppID,
		"type":         payType,
		"out_trade_no": req.OrderNo,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         req.ProductName,
		"money":        fmt.Sprintf("%.2f", req.Amount),
		"client_ip":    req.ClientIP,
		"sign_type":    "MD5",
	}

	// 计算签名
	params["sign"] = generateSign(params, cfg.AppSecret, "MD5")

	// 发起支付请求
	resp, err := httpPostForm(cfg.APIURL+"/mapi.php", params)
	if err != nil {
		return nil, fmt.Errorf("支付请求失败: %w", err)
	}

	var result PaymentResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}

	return &result, nil
}

// VerifyCallback 验证支付回调签名
// 返回: 是否验证通过
func VerifyCallback(cfg Config, params CallbackParams) bool {
	// 构建签名参数（排除sign和sign_type）
	signParams := map[string]string{
		"pid":           fmt.Sprintf("%d", params.PID),
		"trade_no":      params.TradeNo,
		"out_trade_no":  params.OutTradeNo,
		"type":          params.Type,
		"name":          params.Name,
		"money":         fmt.Sprintf("%.2f", params.Money),
		"trade_status":  params.TradeStatus,
		"param":         params.Param,
		"pay_time":      params.PayTime,
	}

	signType := params.SignType
	if signType == "" {
		signType = "MD5"
	}

	expectedSign := generateSign(signParams, cfg.AppSecret, signType)
	return strings.EqualFold(params.Sign, expectedSign)
}

// QueryOrder 查询订单状态
func QueryOrder(cfg Config, orderNo string) (map[string]interface{}, error) {
	params := map[string]string{
		"pid":          cfg.AppID,
		"out_trade_no": orderNo,
		"sign_type":    "MD5",
	}
	params["sign"] = generateSign(params, cfg.AppSecret, "MD5")

	resp, err := httpPostForm(cfg.APIURL+"/api.php?act=order", params)
	if err != nil {
		return nil, fmt.Errorf("订单查询失败: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}

	return result, nil
}

// Refund 退款
func Refund(cfg Config, tradeNo string, amount float64) (map[string]interface{}, error) {
	params := map[string]string{
		"pid":          cfg.AppID,
		"trade_no":     tradeNo,
		"money":        fmt.Sprintf("%.2f", amount),
		"sign_type":    "MD5",
	}
	params["sign"] = generateSign(params, cfg.AppSecret, "MD5")

	resp, err := httpPostForm(cfg.APIURL+"/api.php?act=refund", params)
	if err != nil {
		return nil, fmt.Errorf("退款请求失败: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}

	return result, nil
}

// ==================== 内部工具函数 ====================

// generateSign 生成易支付签名
func generateSign(params map[string]string, secret string, signType string) string {
	// 剔除空值和sign参数
	keys := make([]string, 0)
	for k, v := range params {
		if k != "sign" && k != "sign_type" && v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
		sb.WriteString("&")
	}
	sb.WriteString("key=")
	sb.WriteString(secret)

	raw := sb.String()
	switch strings.ToUpper(signType) {
	case "MD5":
		hash := md5.Sum([]byte(raw))
		return hex.EncodeToString(hash[:])
	case "SHA256", "HMAC-SHA256":
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(raw))
		return hex.EncodeToString(mac.Sum(nil))
	case "SHA512":
		hash := sha512.Sum512([]byte(raw))
		return hex.EncodeToString(hash[:])
	default:
		hash := md5.Sum([]byte(raw))
		return hex.EncodeToString(hash[:])
	}
}

// verifyConfig 验证配置是否可用
func verifyConfig(cfg Config) error {
	params := map[string]string{
		"pid":       cfg.AppID,
		"sign_type": "MD5",
	}
	params["sign"] = generateSign(params, cfg.AppSecret, "MD5")

	resp, err := httpPostForm(cfg.APIURL+"/api.php?act=query", params)
	if err != nil {
		return fmt.Errorf("无法连接到易支付平台: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("响应解析失败: %w", err)
	}

	return nil
}

// httpPostForm 发送HTTP POST表单请求
func httpPostForm(urlStr string, data map[string]string) ([]byte, error) {
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.PostForm(urlStr, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}