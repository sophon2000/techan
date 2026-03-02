package techan

import (
	"github.com/sdcoffey/big"
)

// TradingRecord 记录一系列已平仓交易与当前持仓
type TradingRecord struct {
	Trades          []*Position
	currentPosition *Position
}

// NewTradingRecord 创建新的交易记录
func NewTradingRecord() (t *TradingRecord) {
	t = new(TradingRecord)
	t.Trades = make([]*Position, 0)
	t.currentPosition = new(Position)
	return t
}

// CurrentPosition 返回当前持仓
func (tr *TradingRecord) CurrentPosition() *Position {
	return tr.currentPosition
}

// LastTrade 返回最近一笔已平仓交易
func (tr *TradingRecord) LastTrade() *Position {
	if len(tr.Trades) == 0 {
		return nil
	}
	return tr.Trades[len(tr.Trades)-1]
}

// Operate 处理一笔订单，支持：
// - 开仓：当前无仓位时按订单方向开仓
// - 加仓：当前有仓位且同向时追加入场
// - 分批止盈：当前有仓位且反向、数量小于剩余持仓时部分平仓
// - 平仓：反向且数量等于剩余持仓时全部平仓
// - 反手：反向且数量大于剩余持仓时先平仓，再用超出部分开反向仓
func (tr *TradingRecord) Operate(order Order) {
	if tr.currentPosition.IsOpen() {
		if order.ExecutionTime.Before(tr.currentPosition.LastActivityTime()) {
			return
		}
		tr.operateWithOpenPosition(order)
		return
	}
	if tr.currentPosition.IsNew() {
		if tr.LastTrade() != nil && tr.LastTrade().ExitOrder() != nil &&
			order.ExecutionTime.Before(tr.LastTrade().ExitOrder().ExecutionTime) {
			return
		}
		tr.currentPosition.Enter(order)
	}
}

// operateWithOpenPosition 在已有持仓时处理订单：加仓、分批止盈、平仓或反手
func (tr *TradingRecord) operateWithOpenPosition(order Order) {
	pos := tr.currentPosition
	remaining := pos.RemainingAmount()

	// 同向：加仓
	if (pos.IsLong() && order.Side == BUY) || (pos.IsShort() && order.Side == SELL) {
		pos.Enter(order)
		return
	}

	// 反向
	if order.Amount.LT(remaining) {
		// 分批止盈：部分平仓
		pos.Exit(order)
		if pos.IsClosed() {
			tr.Trades = append(tr.Trades, pos)
			tr.currentPosition = new(Position)
		}
		return
	}

	// 平仓：先按剩余数量平掉当前仓
	closeOrder := order
	closeOrder.Amount = remaining
	pos.Exit(closeOrder)
	tr.Trades = append(tr.Trades, pos)
	tr.currentPosition = new(Position)

	// 反手：若订单数量大于剩余，用超出部分开反向仓
	excess := order.Amount.Sub(remaining)
	if excess.GT(big.ZERO) {
		reverseOrder := Order{
			Side:          order.Side,
			Security:      order.Security,
			Price:         order.Price,
			Amount:        excess,
			ExecutionTime: order.ExecutionTime,
		}
		tr.currentPosition.Enter(reverseOrder)
	}
}
