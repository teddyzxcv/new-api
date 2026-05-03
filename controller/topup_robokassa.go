package controller

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

const robokassaPayURL = "https://auth.robokassa.ru/Merchant/Index.aspx"

// getRobokassaPasswords 按当前模式（生产/沙箱）返回 (password1, password2)。
func getRobokassaPasswords() (string, string) {
	if setting.RobokassaSandbox {
		return setting.RobokassaTestPassword1, setting.RobokassaTestPassword2
	}
	return setting.RobokassaPassword1, setting.RobokassaPassword2
}

// robokassaSign 按配置的算法对 parts join(":") 后做哈希，返回十六进制小写字符串。
func robokassaSign(parts ...string) string {
	var h hash.Hash
	switch strings.ToLower(strings.TrimSpace(setting.RobokassaSignatureAlgo)) {
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		h = md5.New()
	}
	_, _ = io.WriteString(h, strings.Join(parts, ":"))
	return hex.EncodeToString(h.Sum(nil))
}

// formatRobokassaSum 把支付金额格式化为 Robokassa 接受的字符串（点号分隔，2 位小数）。
func formatRobokassaSum(amount float64) string {
	return strconv.FormatFloat(decimal.NewFromFloat(amount).Round(2).InexactFloat64(), 'f', 2, 64)
}

// getRobokassaPayMoney 计算实际收款金额（OutSum），逻辑与 Epay 的 getPayMoney 一致。
func getRobokassaPayMoney(amount int64, group string) float64 {
	dAmount := decimal.NewFromInt(amount)
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		dAmount = dAmount.Div(dQuotaPerUnit)
	}

	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}

	dTopupGroupRatio := decimal.NewFromFloat(topupGroupRatio)
	dUnitPrice := decimal.NewFromFloat(setting.RobokassaUnitPrice)
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]; ok && ds > 0 {
		discount = ds
	}
	dDiscount := decimal.NewFromFloat(discount)

	return dAmount.Mul(dUnitPrice).Mul(dTopupGroupRatio).Mul(dDiscount).InexactFloat64()
}

func getRobokassaMinTopup() int64 {
	min := int64(setting.RobokassaMinTopUp)
	if min < 1 {
		min = 1
	}
	return min
}

type RobokassaPayRequest struct {
	Amount int64 `json:"amount"`
}

// RequestRobokassaAmount 计算实际收款金额，前端预览用。
func RequestRobokassaAmount(c *gin.Context) {
	var req RobokassaPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getRobokassaMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getRobokassaMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getRobokassaPayMoney(req.Amount, group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": formatRobokassaSum(payMoney)})
}

// RequestRobokassaPay 创建 TopUp 订单并返回前端用于跳转 Robokassa 的表单参数。
func RequestRobokassaPay(c *gin.Context) {
	if !isRobokassaTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "Robokassa 支付未启用"})
		return
	}

	var req RobokassaPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getRobokassaMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getRobokassaMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getRobokassaPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	password1, _ := getRobokassaPasswords()
	if password1 == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}

	// Token 模式下归一化 Amount：存等价 USD/CNY 数量，避免 RechargeRobokassa 双重放大
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = int64(decimal.NewFromInt(req.Amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
		if amount < 1 {
			amount = 1
		}
	}

	tradeNo := fmt.Sprintf("RBK-%d-%d-%s", id, time.Now().UnixMilli(), common.GetRandomString(6))
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          amount,
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodRobokassa,
		PaymentProvider: model.PaymentProviderRobokassa,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Robokassa 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	invId := strconv.Itoa(topUp.Id)
	outSum := formatRobokassaSum(payMoney)
	description := fmt.Sprintf("Recharge %d credits", req.Amount)
	signature := robokassaSign(setting.RobokassaMerchantLogin, outSum, invId, password1)

	form := map[string]string{
		"MerchantLogin":  setting.RobokassaMerchantLogin,
		"OutSum":         outSum,
		"InvId":          invId,
		"Description":    description,
		"SignatureValue": signature,
		"Encoding":       "utf-8",
	}
	if currency := strings.ToUpper(strings.TrimSpace(setting.RobokassaCurrency)); currency != "" && currency != "RUB" {
		form["OutSumCurrency"] = currency
	}
	if setting.RobokassaSandbox {
		form["IsTest"] = "1"
	}
	if setting.RobokassaSuccessUrl != "" {
		form["SuccessURL"] = setting.RobokassaSuccessUrl
	} else {
		form["SuccessURL"] = system_setting.ServerAddress + "/console/topup?show_history=true"
	}
	if setting.RobokassaFailUrl != "" {
		form["FailURL"] = setting.RobokassaFailUrl
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("Robokassa 充值订单创建成功 user_id=%d trade_no=%s inv_id=%s amount=%d out_sum=%s", id, tradeNo, invId, req.Amount, outSum))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    form,
		"url":     robokassaPayURL,
	})
}

// RobokassaResult 处理 Robokassa 异步回调（ResultURL）。验签通过必须以纯文本
// "OK<InvId>" 响应；任何其他响应 Robokassa 都会重试。
func RobokassaResult(c *gin.Context) {
	if !isRobokassaWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Robokassa webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	var params map[string]string
	if c.Request.Method == http.MethodPost {
		if err := c.Request.ParseForm(); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Robokassa webhook 解析表单失败 client_ip=%s error=%q", c.ClientIP(), err.Error()))
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, k string, _ int) map[string]string {
			r[k] = c.Request.PostForm.Get(k)
			return r
		}, map[string]string{})
	} else {
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, k string, _ int) map[string]string {
			r[k] = c.Request.URL.Query().Get(k)
			return r
		}, map[string]string{})
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("Robokassa webhook 收到请求 client_ip=%s method=%s params=%q", c.ClientIP(), c.Request.Method, common.GetJsonString(params)))

	outSum := strings.TrimSpace(params["OutSum"])
	invIdStr := strings.TrimSpace(params["InvId"])
	signature := strings.TrimSpace(params["SignatureValue"])
	if outSum == "" || invIdStr == "" || signature == "" {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Robokassa webhook 缺少必要参数 client_ip=%s params=%q", c.ClientIP(), common.GetJsonString(params)))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	_, password2 := getRobokassaPasswords()
	if password2 == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Robokassa webhook 未配置 password2 client_ip=%s", c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	expected := robokassaSign(outSum, invIdStr, password2)
	if !strings.EqualFold(expected, signature) {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Robokassa webhook 验签失败 client_ip=%s inv_id=%s expected=%s actual=%s", c.ClientIP(), invIdStr, expected, signature))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	invId, err := strconv.Atoi(invIdStr)
	if err != nil || invId <= 0 {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Robokassa webhook 非法 InvId client_ip=%s inv_id=%s", c.ClientIP(), invIdStr))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	lockKey := "robokassa:" + invIdStr
	LockOrder(lockKey)
	defer UnlockOrder(lockKey)

	if err := model.RechargeRobokassa(invId, c.ClientIP()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Robokassa 充值处理失败 inv_id=%d client_ip=%s error=%q", invId, c.ClientIP(), err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("Robokassa 充值成功 inv_id=%d client_ip=%s", invId, c.ClientIP()))
	_, _ = c.Writer.Write([]byte("OK" + invIdStr))
}
