package models

import "fmt"

// InvestmentTarget represents the investment target type
type InvestmentTarget byte

// Investment targets
const (
	INVESTMENT_TARGET_NONE InvestmentTarget = 0
	INVESTMENT_TARGET_GOLD InvestmentTarget = 1
)

// String returns a textual representation of the investment target
func (t InvestmentTarget) String() string {
	switch t {
	case INVESTMENT_TARGET_NONE:
		return "None"
	case INVESTMENT_TARGET_GOLD:
		return "Gold"
	default:
		return fmt.Sprintf("Invalid(%d)", int(t))
	}
}

// InvestmentAccount represents the investment account configuration stored in database
type InvestmentAccount struct {
	Id              int64            `xorm:"PK AUTOINCR"`
	Uid             int64            `xorm:"UNIQUE(UQE_investment_account_uid_account) INDEX NOT NULL"`
	Target          InvestmentTarget `xorm:"NOT NULL"`
	AccountId       int64            `xorm:"UNIQUE(UQE_investment_account_uid_account) NOT NULL"`
	InitialUnits    int64            `xorm:"NOT NULL"`
	InitialCost     int64            `xorm:"NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
}
