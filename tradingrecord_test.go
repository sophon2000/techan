package techan

import (
	"testing"
	"time"

	"github.com/sdcoffey/big"
	"github.com/stretchr/testify/assert"
)

func TestNewTradingRecord(t *testing.T) {
	record := NewTradingRecord()

	assert.Len(t, record.Trades, 0)
	assert.True(t, record.CurrentPosition().IsNew())
}

func TestTradingRecord_CurrentTrade(t *testing.T) {
	record := NewTradingRecord()

	yesterday := time.Now().Add(-time.Hour * 24)
	record.Operate(Order{
		Side:          BUY,
		Amount:        big.ONE,
		Price:         big.NewFromString("2"),
		ExecutionTime: yesterday,
	})

	assert.EqualValues(t, "1", record.CurrentPosition().EntranceOrder().Amount.String())
	assert.EqualValues(t, "2", record.CurrentPosition().EntranceOrder().Price.String())
	assert.EqualValues(t, yesterday.UnixNano(),
		record.CurrentPosition().EntranceOrder().ExecutionTime.UnixNano())

	now := time.Now()
	// 平仓：卖出数量等于持仓，无反手
	record.Operate(Order{
		Side:          SELL,
		Amount:        big.ONE,
		Price:         big.NewFromString("4"),
		ExecutionTime: now,
	})
	assert.True(t, record.CurrentPosition().IsNew())

	lastTrade := record.LastTrade()

	assert.EqualValues(t, "1", lastTrade.ExitOrder().Amount.String())
	assert.EqualValues(t, "4", lastTrade.ExitOrder().Price.String())
	assert.EqualValues(t, now.UnixNano(),
		lastTrade.ExitOrder().ExecutionTime.UnixNano())
}

func TestTradingRecord_Enter(t *testing.T) {
	t.Run("Does not add trades older than last trade", func(t *testing.T) {
		record := NewTradingRecord()

		now := time.Now()

		record.Operate(Order{
			Side:          BUY,
			Amount:        big.ONE,
			Price:         big.NewFromString("2"),
			ExecutionTime: now,
		})

		// 平仓：卖出数量等于持仓 1，平仓后无持仓
		record.Operate(Order{
			Side:          SELL,
			Amount:        big.ONE,
			Price:         big.NewFromString("2"),
			ExecutionTime: now.Add(time.Minute),
		})

		// 时间早于上一笔平仓，应被忽略
		record.Operate(Order{
			Side:          BUY,
			Amount:        big.NewFromString("2"),
			Price:         big.NewFromString("2"),
			ExecutionTime: now.Add(-time.Minute),
		})

		assert.True(t, record.CurrentPosition().IsNew())
		assert.Len(t, record.Trades, 1)
	})
}

func TestTradingRecord_Exit(t *testing.T) {
	t.Run("Does not add trades older than last trade", func(t *testing.T) {
		record := NewTradingRecord()

		now := time.Now()
		record.Operate(Order{
			Side:          BUY,
			Amount:        big.ONE,
			Price:         big.NewFromString("2"),
			ExecutionTime: now,
		})

		record.Operate(Order{
			Side:          SELL,
			Amount:        big.NewFromString("2"),
			Price:         big.NewFromString("2"),
			ExecutionTime: now.Add(-time.Minute),
		})

		assert.True(t, record.CurrentPosition().IsOpen())
	})
}

// TestTradingRecord_AddPosition 加仓：同向订单追加到当前持仓
func TestTradingRecord_AddPosition(t *testing.T) {
	record := NewTradingRecord()
	now := time.Now()
	record.Operate(Order{Side: BUY, Amount: big.ONE, Price: big.NewFromString("10"), Security: "X", ExecutionTime: now})
	record.Operate(Order{Side: BUY, Amount: big.NewFromString("2"), Price: big.NewFromString("11"), Security: "X", ExecutionTime: now.Add(time.Minute)})

	assert.True(t, record.CurrentPosition().IsLong())
	assert.True(t, record.CurrentPosition().IsOpen())
	assert.EqualValues(t, "3", record.CurrentPosition().TotalEntryAmount().String())
	assert.Len(t, record.CurrentPosition().EntryOrders(), 2)
}

// TestTradingRecord_PartialTakeProfit 分批止盈：反向且数量小于持仓时部分平仓
func TestTradingRecord_PartialTakeProfit(t *testing.T) {
	record := NewTradingRecord()
	now := time.Now()
	record.Operate(Order{Side: BUY, Amount: big.NewFromString("10"), Price: big.NewFromString("10"), Security: "X", ExecutionTime: now})
	record.Operate(Order{Side: SELL, Amount: big.NewFromString("4"), Price: big.NewFromString("12"), Security: "X", ExecutionTime: now.Add(time.Minute)})

	assert.True(t, record.CurrentPosition().IsOpen())
	assert.EqualValues(t, "6", record.CurrentPosition().RemainingAmount().String())
	assert.Len(t, record.CurrentPosition().ExitOrders(), 1)
	assert.Len(t, record.Trades, 0)
}

// TestTradingRecord_Reverse 反手：反向且数量大于持仓时先平仓再开反向仓
func TestTradingRecord_Reverse(t *testing.T) {
	record := NewTradingRecord()
	now := time.Now()
	record.Operate(Order{Side: BUY, Amount: big.ONE, Price: big.NewFromString("10"), Security: "X", ExecutionTime: now})
	// 卖 3：平掉多 1，多出 2 开空
	record.Operate(Order{Side: SELL, Amount: big.NewFromString("3"), Price: big.NewFromString("11"), Security: "X", ExecutionTime: now.Add(time.Minute)})

	assert.Len(t, record.Trades, 1)
	assert.EqualValues(t, "1", record.Trades[0].ExitOrder().Amount.String())
	assert.True(t, record.CurrentPosition().IsShort())
	assert.EqualValues(t, "2", record.CurrentPosition().TotalEntryAmount().String())
}
