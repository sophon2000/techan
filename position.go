package techan

import (
	"time"

	"github.com/sdcoffey/big"
)

// Position 表示一笔持仓，支持多笔入场（加仓）和多笔出场（分批止盈）
type Position struct {
	entryOrders []*Order
	exitOrders  []*Order
}

// NewPosition 用给定的开仓单创建新持仓
func NewPosition(openOrder Order) (t *Position) {
	t = new(Position)
	t.entryOrders = make([]*Order, 0)
	t.exitOrders = make([]*Order, 0)
	o := openOrder
	t.entryOrders = append(t.entryOrders, &o)
	return t
}

// Enter 开仓或加仓：追加一笔入场单
func (p *Position) Enter(order Order) {
	if p.entryOrders == nil {
		p.entryOrders = make([]*Order, 0)
		p.exitOrders = make([]*Order, 0)
	}
	o := order
	p.entryOrders = append(p.entryOrders, &o)
}

// Exit 平仓或分批止盈：追加一笔出场单
func (p *Position) Exit(order Order) {
	if p.exitOrders == nil {
		p.exitOrders = make([]*Order, 0)
	}
	o := order
	p.exitOrders = append(p.exitOrders, &o)
}

// EntryOrders 返回所有入场单（用于加仓场景）
func (p *Position) EntryOrders() []*Order {
	if p.entryOrders == nil {
		return nil
	}
	return p.entryOrders
}

// ExitOrders 返回所有出场单（用于分批止盈场景）
func (p *Position) ExitOrders() []*Order {
	if p.exitOrders == nil {
		return nil
	}
	return p.exitOrders
}

// TotalEntryAmount 返回总入场数量
func (p *Position) TotalEntryAmount() big.Decimal {
	var sum big.Decimal = big.ZERO
	for _, o := range p.entryOrders {
		if o != nil {
			sum = sum.Add(o.Amount)
		}
	}
	return sum
}

// TotalExitAmount 返回已出场总数量
func (p *Position) TotalExitAmount() big.Decimal {
	var sum big.Decimal = big.ZERO
	for _, o := range p.exitOrders {
		if o != nil {
			sum = sum.Add(o.Amount)
		}
	}
	return sum
}

// RemainingAmount 返回当前剩余持仓数量（未平仓部分）
func (p *Position) RemainingAmount() big.Decimal {
	return p.TotalEntryAmount().Sub(p.TotalExitAmount())
}

// IsLong 是否为多头
func (p *Position) IsLong() bool {
	return p.EntranceOrder() != nil && p.EntranceOrder().Side == BUY
}

// IsShort 是否为空头
func (p *Position) IsShort() bool {
	return p.EntranceOrder() != nil && p.EntranceOrder().Side == SELL
}

// IsOpen 是否仍有持仓（有入场且未完全平仓）
func (p *Position) IsOpen() bool {
	if p.entryOrders == nil || len(p.entryOrders) == 0 {
		return false
	}
	return p.TotalExitAmount().LT(p.TotalEntryAmount())
}

// IsClosed 是否已完全平仓
func (p *Position) IsClosed() bool {
	if p.entryOrders == nil || len(p.entryOrders) == 0 {
		return false
	}
	return p.TotalExitAmount().GTE(p.TotalEntryAmount())
}

// IsNew 是否为空仓位（无任何入场）
func (p *Position) IsNew() bool {
	return p.entryOrders == nil || len(p.entryOrders) == 0
}

// EntranceOrder 返回第一笔入场单（兼容旧用法）
func (p *Position) EntranceOrder() *Order {
	if p.entryOrders == nil || len(p.entryOrders) == 0 {
		return nil
	}
	return p.entryOrders[0]
}

// ExitOrder 返回最后一笔出场单（兼容旧用法）
func (p *Position) ExitOrder() *Order {
	if p.exitOrders == nil || len(p.exitOrders) == 0 {
		return nil
	}
	return p.exitOrders[len(p.exitOrders)-1]
}

// CostBasis 返回持仓成本（所有入场单的金额之和）
func (p *Position) CostBasis() big.Decimal {
	var sum big.Decimal = big.ZERO
	for _, o := range p.entryOrders {
		if o != nil {
			sum = sum.Add(o.Amount.Mul(o.Price))
		}
	}
	return sum
}

// ExitValue 返回已出场部分对应的市值（所有出场单的金额之和）
func (p *Position) ExitValue() big.Decimal {
	var sum big.Decimal = big.ZERO
	for _, o := range p.exitOrders {
		if o != nil {
			sum = sum.Add(o.Amount.Mul(o.Price))
		}
	}
	return sum
}

// LastActivityTime 返回当前持仓最后一次操作时间（用于订单时序校验）
func (p *Position) LastActivityTime() time.Time {
	var t time.Time
	for _, o := range p.entryOrders {
		if o != nil && o.ExecutionTime.After(t) {
			t = o.ExecutionTime
		}
	}
	for _, o := range p.exitOrders {
		if o != nil && o.ExecutionTime.After(t) {
			t = o.ExecutionTime
		}
	}
	return t
}
