package cron

import (
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// RemoveExpiredTokensJob represents the cron job which periodically remove expired user tokens from the database
var RemoveExpiredTokensJob = &CronJob{
	Name:        "RemoveExpiredTokens",
	Description: "Periodically remove expired user tokens from the database.",
	Period: CronJobFixedHourPeriod{
		Hour: 0,
	},
	Run: func(c *core.CronContext) error {
		return services.Tokens.DeleteAllExpiredTokens(c)
	},
}

// CreateScheduledTransactionJob represents the cron job which periodically create transaction by scheduled transaction template
var CreateScheduledTransactionJob = &CronJob{
	Name:        "CreateScheduledTransaction",
	Description: "Periodically create transaction by scheduled transaction template.",
	Period: CronJobEvery15MinutesPeriod{
		Second: 0,
	},
	Run: func(c *core.CronContext) error {
		return services.Transactions.CreateScheduledTransactions(c, time.Now().Unix(), c.GetInterval())
	},
}

// UpdateInvestmentValuationsJob represents the cron job which weekly updates the valuations of investment accounts
var UpdateInvestmentValuationsJob = &CronJob{
	Name:        "UpdateInvestmentValuations",
	Description: "Weekly compare the market value of investment accounts with their balances and create valuation transactions.",
	Period: CronJobWeeklyPeriod{
		Weekday: time.Sunday,
		Hour:    23,
		Minute:  30,
	},
	Run: func(c *core.CronContext) error {
		return services.Investments.UpdateAllInvestmentValuations(c)
	},
}
