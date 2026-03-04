package calculator

import (
	"errors"
	"fmt"
)

// Scaling factors to avoid using float64 for monetary calculations
const (
	// MoneyScale is the factor by which monetary values are scaled to preserve precision.
	// e.g., 1 cent is stored as 1 * MoneyScale.
	MoneyScale = 1_000_000
	// SzScale is the factor by which position sizes (Sz) are scaled.
	// e.g., a size of 1.23 is stored as 123.
	SzScale = 100

	// 维持保证金率（默认0.4%）
	maintenanceRate = 0.004
)

// PositionType 持仓方向
type PositionType string

const (
	PositionLong       PositionType = "long"  // 做多
	PositionShort      PositionType = "short" // 做空
	PositionCloseLong               = "close-long"
	PositionCloseShort              = "close-short"
)

type PositionState int

const (
	PositionHold PositionState = 0 // 持仓中

	PositionManualClose PositionState = 1 // 已手动平仓

	PositionForceClose PositionState = 2 // 已强制平仓
)

// OpenPositionRequest 开仓请求参数
type OpenPositionRequest struct {
	Price    int64        `json:"price" binding:"required"`                     // 开仓价格（单位：分）
	Position PositionType `json:"position" binding:"required,oneof=long short"` // 持仓方向：long 或 short
	Leverage int32        `json:"leverage" binding:"required,min=1"`            // 杠杆倍数
	Sz       int64        `json:"sz" binding:"required,min=1"`                  // 开仓数量/张数 (scaled by SzScale, e.g., 100 for 1.00)
}

// ClosePositionResult 平仓结果
type ClosePositionResult struct {
	Profit         int64   `json:"profit"`          // 收益
	ProfitRate     float64 `json:"profit_rate"`     // 收益率
	ReturnedMargin int64   `json:"returned_margin"` // 返还保证金
	TotalReturn    int64   `json:"total_return"`    // 总返还金额
	IsLiquidation  bool    `json:"is_liquidation"`  // 是否爆仓
}

// OpenPositionResult 开仓计算结果
type OpenPositionResult struct {
	// 基础信息
	Price    int64        `json:"price"`    // 开仓价格
	Position PositionType `json:"position"` // 持仓方向
	Leverage int32        `json:"leverage"` // 杠杆倍数
	Sz       int64        `json:"sz"`       // 开仓数量 (scaled by SzScale)

	// 计算结果 (all scaled by MoneyScale)
	TotalValue     int64 `json:"totalValue"`     // 合约总价值
	RequiredMargin int64 `json:"requiredMargin"` // 所需保证金
	// 风险指标
	LiquidationPrice int64   `json:"liquidationPrice"` // 预估强平价格 (in cents)
	MarginRate       float64 `json:"marginRate"`       // 保证金率
	LeverageRatio    float64 `json:"leverageRatio"`    // 实际杠杆率
}

// ContractConfig 合约配置
type ContractConfig struct {
	ContractSize          int64 // 合约面值（每张合约代表的价值，单位：分）
	MaintenanceMarginRate int64 // 维持保证金率 (scaled by RateScale)
	MaxLeverage           int32 // 最大杠杆倍数
	MinLeverage           int32 // 最小杠杆倍数
}

// DefaultContractConfig 默认合约配置（以常见的永续合约为例）
var DefaultContractConfig = ContractConfig{
	MaxLeverage: 10000,
	MinLeverage: 1,
}

// CalculateLiquidationPrice 计算强平价格
// 参数:
//
//	position: 持仓数量(正数表示做多,负数表示做空)
//	avgPrice: 平均持仓价格
//	leverage: 杠杆倍数
//	marginBalance: 保证金余额
//	positionType: 持仓类型(做多/做空)
//
// 返回:
//
//	强平价格
func CalculateLiquidationPrice(sz int64, avgPrice int64, leverage int32, marginBalance int64, positionType PositionType) int64 {
	if sz == 0 || avgPrice == 0 || leverage == 0 {
		return 0
	}

	// sz: 实际数量 × 100
	// avgPrice: 实际价格 × 1,000,000
	// marginBalance: 假设也是 × 1,000,000

	// 计算初始保证金 (放大 1,000,000 倍)
	initialMargin := int64(0)

	// 最大亏损 (放大 1,000,000 倍)
	maxLoss := marginBalance - initialMargin

	var liquidationPrice int64

	if positionType == PositionLong {
		// 做多强平价格
		// maxLoss 是 ×1,000,000
		// sz 是 ×100
		// maxLoss * 100 / sz = (×1,000,000 × 100) / (×100) = ×1,000,000
		priceDiff := maxLoss * 100 / sz
		liquidationPrice = avgPrice - priceDiff
	} else {
		// 做空强平价格
		priceDiff := maxLoss * 100 / sz
		liquidationPrice = avgPrice + priceDiff
	}

	return liquidationPrice
}

// CalculateOpenPosition 计算开仓所需保证金
func CalculateOpenPosition(req *OpenPositionRequest, config *ContractConfig) (*OpenPositionResult, error) {
	if config == nil {
		config = &DefaultContractConfig
	}

	// 参数校验
	if req.Price <= 0 {
		return nil, errors.New("price must be positive")
	}
	if req.Sz <= 0 {
		return nil, errors.New("sz must be positive")
	}
	if req.Leverage < config.MinLeverage || req.Leverage > config.MaxLeverage {
		return nil, fmt.Errorf("leverage must be between %d and %d", config.MinLeverage, config.MaxLeverage)
	}
	if req.Position != PositionLong && req.Position != PositionShort {
		return nil, errors.New("invalid position type")
	}

	// 计算合约总价值 (scaled by MoneyScale)
	totalValue := (req.Sz * req.Price) / SzScale
	if totalValue < 0 {
		return nil, errors.New("total value overflow")
	}

	// 计算所需保证金（初始保证金） (scaled by MoneyScale)
	requiredMargin := totalValue / int64(req.Leverage)

	// 计算保证金率
	// MarginRate = RequiredMargin / TotalValue
	marginRate := 0.0
	if totalValue > 0 {
		marginRate = float64(requiredMargin) / float64(totalValue)
	}

	// 计算实际杠杆率
	// LeverageRatio = TotalValue / RequiredMargin
	leverageRatio := 0.0
	if requiredMargin > 0 {
		leverageRatio = float64(totalValue) / float64(requiredMargin)
	}

	return &OpenPositionResult{
		// 基础信息
		Price:    req.Price,
		Position: req.Position,
		Leverage: req.Leverage,
		Sz:       req.Sz,

		// 计算结果 (all scaled by MoneyScale)
		TotalValue:     totalValue,
		RequiredMargin: requiredMargin,

		// 风险指标
		LiquidationPrice: 0,
		MarginRate:       marginRate,
		LeverageRatio:    leverageRatio,
	}, nil
}

// validateOpenPositionRequest 校验开仓请求参数
func validateOpenPositionRequest(req *OpenPositionRequest, config *ContractConfig) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if req.Price <= 0 {
		return fmt.Errorf("price must be positive: %d", req.Price)
	}
	if req.Position != PositionLong && req.Position != PositionShort {
		return fmt.Errorf("invalid position: %s", req.Position)
	}
	if config != nil {
		if req.Leverage < config.MinLeverage || req.Leverage > config.MaxLeverage {
			return fmt.Errorf("leverage %d is out of range [%d, %d]", req.Leverage, config.MinLeverage, config.MaxLeverage)
		}
	} else if req.Leverage < 1 {
		return fmt.Errorf("leverage must be at least 1: %d", req.Leverage)
	}
	if req.Sz < 1 { // Corresponds to 0.01
		return fmt.Errorf("sz must be at least 0.01 (sent as 1): %d", req.Sz)
	}
	return nil
}

// CalculateMaxPosition 计算最大可开仓数量
// 返回值：最大持仓量（单位：张/币）
func CalculateMaxPosition(balance int64, price int64, leverage int32) int64 {
	if balance <= 0 || price <= 0 || leverage <= 0 {
		return 0
	}

	// 最大可开仓价值 = 余额 × 杠杆倍数
	maxPositionValue := balance * int64(leverage)

	// 最大可开仓数量 = 最大可开仓价值 / 价格
	maxPosition := float64(maxPositionValue) / float64(price)

	return int64(maxPosition * 100)
}

// CalculateClosePositionProfit 计算平仓收益
// openPrice: 开仓价格（已乘100）
// closePrice: 平仓价格（已乘100）
// position: 仓位方向（long/short）
// leverage: 杠杆倍数
// sz: 持仓数量（已乘100）
func CalculateClosePositionProfit(openPrice, closePrice int64, position PositionType, leverage int32, sz int64) (*ClosePositionResult, error) {
	// 参数校验
	if openPrice <= 0 {
		return nil, errors.New("开仓价格必须大于0")
	}
	if closePrice <= 0 {
		return nil, errors.New("平仓价格必须大于0")
	}
	if sz <= 0 {
		return nil, errors.New("持仓数量必须大于0")
	}
	if leverage <= 0 {
		return nil, errors.New("杠杆倍数必须大于0")
	}
	if position != PositionLong && position != PositionShort {
		return nil, errors.New("仓位方向必须是long或short")
	}

	result := &ClosePositionResult{}

	// 计算价格差
	var priceDiff int64
	if position == PositionLong {
		// 做多：平仓价 - 开仓价
		priceDiff = closePrice - openPrice
	} else {
		// 做空：开仓价 - 平仓价
		priceDiff = openPrice - closePrice
	}

	profit := (priceDiff * sz) / 100

	earnestMoney := (openPrice * sz / 100) / int64(leverage)

	// 计算收益率 = (收益 / 保证金) * 100
	profitRate := 0.0
	if earnestMoney > 0 {
		profitRate = float64(profit) / float64(earnestMoney)
	}

	// 总返还 = 保证金 + 收益
	totalReturn := earnestMoney + profit

	result.Profit = profit
	result.ProfitRate = profitRate
	result.ReturnedMargin = totalReturn

	return result, nil
}

// CalculateROI 计算投资回报率（盈亏比）
func CalculateROI(openPrice, closePrice int64, position PositionType, leverage int32, sz int64) *ROIResult {
	totalValue := (openPrice * sz * MoneyScale) / SzScale
	initialMargin := totalValue / int64(leverage)

	// 2. Calculate PnL (Profit and Loss)
	// pnl_cents = (closePrice - openPrice) * (sz / SzScale)
	// pnl_scaled = pnl_cents * MoneyScale = ((closePrice - openPrice) * sz * MoneyScale) / SzScale
	pnl := ((closePrice - openPrice) * sz * MoneyScale) / SzScale
	if position == PositionShort {
		pnl = -pnl
	}

	// 3. Net PnL (无手续费，净盈亏等于毛盈亏)
	netPnl := pnl

	// 4. ROI
	var roi float64
	if initialMargin != 0 {
		// roi = netPnl_scaled / initialMargin_scaled
		roi = float64(netPnl) / float64(initialMargin) * 100
	}

	return &ROIResult{
		OpenPrice:     openPrice,
		ClosePrice:    closePrice,
		Position:      position,
		Leverage:      leverage,
		Sz:            sz,
		PriceChange:   float64(closePrice-openPrice) / float64(openPrice) * 100,
		GrossPnL:      pnl,
		NetPnL:        netPnl,
		InitialMargin: initialMargin,
		ROI:           roi,
	}
}

// ROIResult 投资回报率计算结果
type ROIResult struct {
	OpenPrice     int64        `json:"openPrice"`     // 开仓价格
	ClosePrice    int64        `json:"closePrice"`    // 平仓价格
	Position      PositionType `json:"position"`      // 持仓方向
	Leverage      int32        `json:"leverage"`      // 杠杆倍数
	Sz            int64        `json:"sz"`            // 持仓数量 (scaled by SzScale)
	PriceChange   float64      `json:"priceChange"`   // 价格变动百分比
	GrossPnL      int64        `json:"grossPnl"`      // 毛盈亏 (scaled by MoneyScale)
	NetPnL        int64        `json:"netPnl"`        // 净盈亏 (scaled by MoneyScale)
	InitialMargin int64        `json:"initialMargin"` // 初始保证金 (scaled by MoneyScale)
	ROI           float64      `json:"roi"`           // 投资回报率（%）
}

// FormatMoney 格式化金额（分转元）
func FormatMoney(amount int64) string {
	yuan := float64(amount) / 100.0
	return fmt.Sprintf("%.2f", yuan)
}

// FormatMoneyWithUnit 格式化金额并带单位
func FormatMoneyWithUnit(amount int64) string {
	return fmt.Sprintf("¥%s", FormatMoney(amount))
}
