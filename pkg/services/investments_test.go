package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func TestParseInvestmentUnitsFromComment(t *testing.T) {
	assert.Equal(t, int64(46316), parseInvestmentUnitsFromComment("4.6316克黄金"))
	assert.Equal(t, int64(100000), parseInvestmentUnitsFromComment("10g"))
	assert.Equal(t, int64(100000), parseInvestmentUnitsFromComment("10G"))
	assert.Equal(t, int64(100000), parseInvestmentUnitsFromComment("克重：10"))
	assert.Equal(t, int64(100000), parseInvestmentUnitsFromComment("重量 10"))
	assert.Equal(t, int64(0), parseInvestmentUnitsFromComment("10千克"))
	assert.Equal(t, int64(0), parseInvestmentUnitsFromComment("10kg"))
	assert.Equal(t, int64(-1), parseInvestmentUnitsFromComment("不计入 5克"))
	assert.Equal(t, int64(-1), parseInvestmentUnitsFromComment("忽略：5克"))
	assert.Equal(t, int64(0), parseInvestmentUnitsFromComment("无"))
	assert.Equal(t, int64(0), parseInvestmentUnitsFromComment(""))
}

func TestParseInvestmentUnitsFromCommentMultipleNumbers(t *testing.T) {
	assert.Equal(t, int64(46316), parseInvestmentUnitsFromComment("11-04 转账 4.6316克，手续费 2.5 元"))
	assert.Equal(t, int64(100000), parseInvestmentUnitsFromComment("买入 10克 @ 480"))
}

func TestCalculateInvestmentHoldingBuyAndSell(t *testing.T) {
	records := []*models.Transaction{
		{
			TransactionId:    1,
			Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
			AccountId:        100,
			RelatedAccountId: 200,
			Amount:           83000,
			InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
			InvestmentUnits:  100000,
		},
		{
			TransactionId:    2,
			Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
			AccountId:        200,
			RelatedAccountId: 100,
			Amount:           42000,
			InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
			InvestmentUnits:  50000,
		},
	}

	holding := calculateInvestmentHolding(1000000, 80000, records, 200)

	assert.Equal(t, int64(1050000), holding.Units)
	assert.Equal(t, int64(155591), holding.Cost)
	assert.Equal(t, int64(34591), holding.RealizedProfit)
	assert.Equal(t, int64(1482), holding.AvgCost)
}

func TestCalculateInvestmentHoldingSellMoreThanHolding(t *testing.T) {
	records := []*models.Transaction{
		{
			TransactionId:    1,
			Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
			AccountId:        200,
			RelatedAccountId: 100,
			Amount:           100000,
			InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
			InvestmentUnits:  2000000,
		},
	}

	holding := calculateInvestmentHolding(1000000, 80000, records, 200)

	assert.Equal(t, int64(0), holding.Units)
	assert.Equal(t, int64(0), holding.Cost)
	assert.Equal(t, int64(20000), holding.RealizedProfit)
	assert.Equal(t, int64(0), holding.AvgCost)
}

func TestCalculateInvestmentHoldingIgnoresNonInvestmentRecords(t *testing.T) {
	records := []*models.Transaction{
		{
			TransactionId:    1,
			Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
			AccountId:        100,
			RelatedAccountId: 200,
			Amount:           83000,
			InvestmentTarget: models.INVESTMENT_TARGET_NONE,
		},
		{
			TransactionId:    2,
			Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
			AccountId:        100,
			RelatedAccountId: 300,
			Amount:           83000,
			InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
			InvestmentUnits:  100000,
		},
	}

	holding := calculateInvestmentHolding(1000000, 80000, records, 200)

	assert.Equal(t, int64(1000000), holding.Units)
	assert.Equal(t, int64(80000), holding.Cost)
}

func TestCheckInvestmentTransactionFields(t *testing.T) {
	investmentTargets := map[int64]models.InvestmentTarget{
		200: models.INVESTMENT_TARGET_GOLD,
	}

	// buy: transfer to the investment account
	buy := &models.Transaction{
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		AccountId:        100,
		RelatedAccountId: 200,
		InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
		InvestmentUnits:  100000,
	}
	assert.Nil(t, checkInvestmentTransactionFields(buy, investmentTargets))

	// sell: transfer from the investment account
	sell := &models.Transaction{
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		AccountId:        200,
		RelatedAccountId: 100,
		InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
		InvestmentUnits:  100000,
	}
	assert.Nil(t, checkInvestmentTransactionFields(sell, investmentTargets))

	// no investment target and no units
	assert.Nil(t, checkInvestmentTransactionFields(&models.Transaction{
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		AccountId:        100,
		RelatedAccountId: 200,
	}, investmentTargets))

	// no investment target but units are set
	assert.NotNil(t, checkInvestmentTransactionFields(&models.Transaction{
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		AccountId:        100,
		RelatedAccountId: 200,
		InvestmentUnits:  100000,
	}, investmentTargets))

	// not a transfer
	assert.NotNil(t, checkInvestmentTransactionFields(&models.Transaction{
		Type:             models.TRANSACTION_DB_TYPE_INCOME,
		AccountId:        200,
		InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
		InvestmentUnits:  100000,
	}, investmentTargets))

	// neither side is an investment account
	assert.NotNil(t, checkInvestmentTransactionFields(&models.Transaction{
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		AccountId:        100,
		RelatedAccountId: 300,
		InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
		InvestmentUnits:  100000,
	}, investmentTargets))

	// both sides are investment accounts
	assert.NotNil(t, checkInvestmentTransactionFields(&models.Transaction{
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		AccountId:        200,
		RelatedAccountId: 200,
		InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
		InvestmentUnits:  100000,
	}, investmentTargets))

	// units must be positive
	assert.NotNil(t, checkInvestmentTransactionFields(&models.Transaction{
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		AccountId:        100,
		RelatedAccountId: 200,
		InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
	}, investmentTargets))

	// negative fee
	assert.NotNil(t, checkInvestmentTransactionFields(&models.Transaction{
		Type:             models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		AccountId:        100,
		RelatedAccountId: 200,
		InvestmentTarget: models.INVESTMENT_TARGET_GOLD,
		InvestmentUnits:  100000,
		InvestmentFee:    -1,
	}, investmentTargets))
}

func TestFormatInvestmentUnits(t *testing.T) {
	assert.Equal(t, "4.6316", FormatInvestmentUnits(46316))
	assert.Equal(t, "10.0000", FormatInvestmentUnits(100000))
	assert.Equal(t, "0.0001", FormatInvestmentUnits(1))
	assert.Equal(t, "0.0000", FormatInvestmentUnits(0))
	assert.Equal(t, "-4.6316", FormatInvestmentUnits(-46316))
}

func TestParseInvestmentUnits(t *testing.T) {
	units, err := ParseInvestmentUnits("4.6316")
	assert.Nil(t, err)
	assert.Equal(t, int64(46316), units)

	units, err = ParseInvestmentUnits("10")
	assert.Nil(t, err)
	assert.Equal(t, int64(100000), units)

	units, err = ParseInvestmentUnits("")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), units)

	_, err = ParseInvestmentUnits("abc")
	assert.NotNil(t, err)
}
