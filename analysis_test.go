package techan

import (
	"testing"
	"time"

	"bytes"

	"bufio"

	"fmt"

	"github.com/sdcoffey/big"
	"github.com/stretchr/testify/assert"
)

const example = "EXM"

func TestTotalProfitAnalysis(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		record := NewTradingRecord()
		tpa := TotalProfitAnalysis{}

		// 两笔完整交易：多头 1@1 平 1@2 盈利 1；空头 1@2 平 1@1 盈利 1；总盈利 2
		orders := []Order{
			{Side: BUY, Amount: big.NewDecimal(1), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
			{Side: SELL, Amount: big.NewDecimal(1), Price: big.NewDecimal(2), Security: example, ExecutionTime: time.Now()},
			{Side: SELL, Amount: big.NewDecimal(1), Price: big.NewDecimal(2), Security: example, ExecutionTime: time.Now()},
			{Side: BUY, Amount: big.NewDecimal(1), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
		}
		for _, order := range orders {
			record.Operate(order)
		}
		assert.EqualValues(t, 2.0, tpa.Analyze(record))

		// 再开多 1@1，平 1@0.5，亏损 0.5；总盈利 1.5
		record.Operate(Order{Side: BUY, Amount: big.ONE, Price: big.ONE, Security: example, ExecutionTime: time.Now()})
		record.Operate(Order{Side: SELL, Amount: big.ONE, Price: big.NewFromString("0.5"), Security: example, ExecutionTime: time.Now()})

		assert.EqualValues(t, 1.5, tpa.Analyze(record))
	})
}

func TestPercentGainAnalysis(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		record := NewTradingRecord()

		pga := PercentGainAnalysis{}

		assert.EqualValues(t, 0, pga.Analyze(record))
	})

	t.Run("Simple gain", func(t *testing.T) {
		record := NewTradingRecord()
		pga := PercentGainAnalysis{}
		// 一笔：买 1@1 卖 1@2，收益率 100%
		orders := []Order{
			{Side: BUY, Amount: big.NewDecimal(1), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
			{Side: SELL, Amount: big.NewDecimal(1), Price: big.NewDecimal(2), Security: example, ExecutionTime: time.Now()},
		}
		for _, order := range orders {
			record.Operate(order)
		}
		gain := pga.Analyze(record)
		assert.EqualValues(t, 1, gain)
	})

	t.Run("Simple loss", func(t *testing.T) {
		record := NewTradingRecord()
		pga := PercentGainAnalysis{}
		// 一笔：买 2@1 卖 2@0.5，收益率 -50%
		orders := []Order{
			{Side: BUY, Amount: big.NewDecimal(2), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
			{Side: SELL, Amount: big.NewDecimal(2), Price: big.NewDecimal(0.5), Security: example, ExecutionTime: time.Now()},
		}
		for _, order := range orders {
			record.Operate(order)
		}
		gain := pga.Analyze(record)
		assert.EqualValues(t, -.5, gain)
	})

	t.Run("Small loss and gain", func(t *testing.T) {
		record := NewTradingRecord()
		pga := PercentGainAnalysis{}
		// 两笔：买 2@1 卖 2@0.5（亏）；卖 1@1 买 1@1.25（盈）。首笔成本 2，末笔出场 1.25，收益率 1.25/2-1 = -0.375
		orders := []Order{
			{Side: BUY, Amount: big.NewDecimal(2), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
			{Side: SELL, Amount: big.NewDecimal(2), Price: big.NewDecimal(0.5), Security: example, ExecutionTime: time.Now()},
			{Side: SELL, Amount: big.NewDecimal(1), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
			{Side: BUY, Amount: big.NewDecimal(1), Price: big.NewDecimal(1.25), Security: example, ExecutionTime: time.Now()},
		}
		for _, order := range orders {
			record.Operate(order)
		}
		gain := pga.Analyze(record)
		assert.EqualValues(t, -.375, gain)
	})
}

func TestNumTradesAnalysis(t *testing.T) {
	record := NewTradingRecord()

	var nta NumTradesAnalysis

	assert.EqualValues(t, 0, nta.Analyze(record))
}

func TestLogTradesAnalysis(t *testing.T) {
	buffer := bytes.NewBufferString("")

	logger := LogTradesAnalysis{
		Writer: buffer,
	}

	record := NewTradingRecord()

	now := time.Now().UTC()
	dates := []time.Time{
		now,
		now.AddDate(0, 0, 1),
		now.AddDate(0, 0, 2),
		now.AddDate(0, 0, 3),
	}

	orders := []Order{
		{
			Side:          BUY,
			Amount:        big.NewDecimal(1),
			Price:         big.NewDecimal(2),
			Security:      example,
			ExecutionTime: dates[0],
		},
		{
			Side:          SELL,
			Amount:        big.NewDecimal(1),
			Price:         big.NewDecimal(1),
			Security:      example,
			ExecutionTime: dates[1],
		},
		{
			Side:          BUY,
			Amount:        big.NewDecimal(1),
			Price:         big.NewDecimal(1),
			Security:      example,
			ExecutionTime: dates[2],
		},
		{
			Side:          SELL,
			Amount:        big.NewDecimal(1),
			Price:         big.NewDecimal(1.25),
			Security:      example,
			ExecutionTime: dates[3],
		},
	}

	for _, order := range orders {
		record.Operate(order)
	}

	val := logger.Analyze(record)
	assert.EqualValues(t, 0, val)

	scanner := bufio.NewScanner(buffer)

	var i int
	for scanner.Scan() {
		text := scanner.Text()

		var expected string
		switch i {
		case 0:
			expected = fmt.Sprintf("%s - enter with buy EXM (1 @ $2)", dates[0].Format(time.RFC822))
		case 1:
			expected = fmt.Sprintf("%s - exit with sell EXM (1 @ $1)", dates[1].Format(time.RFC822))
		case 2:
			expected = "Profit: $-1"
		case 3:
			expected = fmt.Sprintf("%s - enter with buy EXM (1 @ $1)", dates[2].Format(time.RFC822))
		case 4:
			expected = fmt.Sprintf("%s - exit with sell EXM (1 @ $1.25)", dates[3].Format(time.RFC822))
		case 5:
			expected = "Profit: $0.25"
		}

		assert.EqualValues(t, expected, text)
		i++
	}
}

func TestPeriodProfitAnalysis(t *testing.T) {
	record := NewTradingRecord()
	now := time.Now().Add(-time.Minute * 5)
	// 两笔交易总盈利 4，时间跨度 4 分钟，Period 2 分钟 → 2 个周期，平均每周期盈利 2
	orders := []Order{
		{Side: BUY, Amount: big.NewDecimal(1), Price: big.NewDecimal(1), Security: example, ExecutionTime: now},
		{Side: SELL, Amount: big.NewDecimal(1), Price: big.NewDecimal(3), Security: example, ExecutionTime: now.Add(time.Minute)},
		{Side: SELL, Amount: big.NewDecimal(1), Price: big.NewDecimal(3), Security: example, ExecutionTime: now.Add(time.Minute * 2)},
		{Side: BUY, Amount: big.NewDecimal(1), Price: big.NewDecimal(1), Security: example, ExecutionTime: now.Add(time.Minute * 4)},
	}
	for _, order := range orders {
		record.Operate(order)
	}
	ppa := PeriodProfitAnalysis{Period: time.Minute * 2}
	assert.EqualValues(t, 2, ppa.Analyze(record))
}

func TestProfitableTradesAnalysis(t *testing.T) {
	record := NewTradingRecord()
	// 第一笔盈利（买1@1卖1@2），第二笔持平（卖2@1买2@1）
	orders := []Order{
		{Side: BUY, Amount: big.NewDecimal(1), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
		{Side: SELL, Amount: big.NewDecimal(1), Price: big.NewDecimal(2), Security: example, ExecutionTime: time.Now()},
		{Side: SELL, Amount: big.NewDecimal(2), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
		{Side: BUY, Amount: big.NewDecimal(2), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
	}
	for _, order := range orders {
		record.Operate(order)
	}
	pta := ProfitableTradesAnalysis{}
	assert.EqualValues(t, 1, pta.Analyze(record))
}

func TestAverageProfitAnalysis(t *testing.T) {
	record := NewTradingRecord()
	// 两笔交易，每笔盈利 2，平均 2
	orders := []Order{
		{Side: BUY, Amount: big.NewDecimal(1), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
		{Side: SELL, Amount: big.NewDecimal(1), Price: big.NewDecimal(3), Security: example, ExecutionTime: time.Now()},
		{Side: SELL, Amount: big.NewDecimal(1), Price: big.NewDecimal(3), Security: example, ExecutionTime: time.Now()},
		{Side: BUY, Amount: big.NewDecimal(1), Price: big.NewDecimal(1), Security: example, ExecutionTime: time.Now()},
	}
	for _, order := range orders {
		record.Operate(order)
	}
	apa := AverageProfitAnalysis{}
	assert.EqualValues(t, 2, apa.Analyze(record))
}

func TestBuyAndHoldAnalysis(t *testing.T) {
	series := mockTimeSeries("1", "2", "3", "2", "6")
	record := NewTradingRecord()

	t.Run("== 0 trades returns zero", func(t *testing.T) {
		buyAndHoldAnalysis := BuyAndHoldAnalysis{
			TimeSeries:    series,
			StartingMoney: 1,
		}

		assert.EqualValues(t, 0, buyAndHoldAnalysis.Analyze(record))
	})

	t.Run("> 0 trades", func(t *testing.T) {
		orders := []Order{
			{
				Side:          BUY,
				Amount:        big.NewDecimal(1),
				Price:         big.NewDecimal(1),
				Security:      example,
				ExecutionTime: time.Now(),
			},
			{
				Side:          SELL,
				Amount:        big.NewDecimal(2),
				Price:         big.NewDecimal(1),
				Security:      example,
				ExecutionTime: time.Now(),
			},
			{
				Side:          BUY,
				Amount:        big.NewDecimal(3),
				Price:         big.NewDecimal(1),
				Security:      example,
				ExecutionTime: time.Now(),
			},
			{
				Side:          SELL,
				Amount:        big.NewDecimal(6),
				Price:         big.NewDecimal(1),
				Security:      example,
				ExecutionTime: time.Now(),
			},
		}

		for _, order := range orders {
			record.Operate(order)
		}

		buyAndHoldAnalysis := BuyAndHoldAnalysis{
			TimeSeries:    series,
			StartingMoney: 1,
		}

		assert.EqualValues(t, 5, buyAndHoldAnalysis.Analyze(record))
	})
}
