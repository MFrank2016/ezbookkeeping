package api

import (
	"fmt"
	"math"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// InvestmentsApi represents investment api
type InvestmentsApi struct {
	investments *services.InvestmentService
}

// Initialize an investment api singleton instance
var (
	Investments = &InvestmentsApi{
		investments: services.Investments,
	}
)

// InvestmentAccountOverviewResponse represents a view-object of an investment account overview
type InvestmentAccountOverviewResponse struct {
	Id             int64                       `json:"id,string"`
	Target         models.InvestmentTarget     `json:"target"`
	TargetName     string                      `json:"targetName"`
	AccountId      int64                       `json:"accountId,string"`
	AccountName    string                      `json:"accountName"`
	Currency       string                      `json:"currency"`
	Units          string                      `json:"units"`
	InitialUnits   string                      `json:"initialUnits"`
	InitialCost    string                      `json:"initialCost"`
	AvgCost        string                      `json:"avgCost"`
	Cost           string                      `json:"cost"`
	MarketValue    string                      `json:"marketValue"`
	FloatProfit    string                      `json:"floatProfit"`
	Balance        string                      `json:"balance"`
	Diff           string                      `json:"diff"`
	DiffPercent    string                      `json:"diffPercent"`
	Records        []*InvestmentRecordResponse `json:"records"`
	RealizedProfit string                      `json:"realizedProfit"`
}

// InvestmentRecordResponse represents a view-object of an investment record
type InvestmentRecordResponse struct {
	TransactionId int64  `json:"transactionId,string"`
	Time          int64  `json:"time"`
	Direction     string `json:"direction"`
	Amount        string `json:"amount"`
	Units         string `json:"units"`
	Fee           string `json:"fee"`
	UnitPrice     string `json:"unitPrice"`
	Comment       string `json:"comment"`
}

// PendingInvestmentTransferResponse represents a view-object of a pending investment transfer
type PendingInvestmentTransferResponse struct {
	TransactionId int64  `json:"transactionId,string"`
	Time          int64  `json:"time"`
	Direction     string `json:"direction"`
	Amount        string `json:"amount"`
	Comment       string `json:"comment"`
}

// InvestmentOverviewResponse represents a view-object of the investment overview
type InvestmentOverviewResponse struct {
	Accounts         []*InvestmentAccountOverviewResponse `json:"accounts"`
	TotalMarketValue string                               `json:"totalMarketValue"`
	TotalCost        string                               `json:"totalCost"`
	TotalFloatProfit string                               `json:"totalFloatProfit"`
	Price            string                               `json:"price"`
	PriceFetchedAt   int64                                `json:"priceFetchedAt"`
	PriceSource      string                               `json:"priceSource"`
	PriceAvailable   bool                                 `json:"priceAvailable"`
	PendingTransfers []*PendingInvestmentTransferResponse `json:"pendingTransfers"`
}

// InvestmentAccountResponse represents a view-object of an investment account configuration
type InvestmentAccountResponse struct {
	Id           int64                   `json:"id,string"`
	Target       models.InvestmentTarget `json:"target"`
	TargetName   string                  `json:"targetName"`
	AccountId    int64                   `json:"accountId,string"`
	InitialUnits string                  `json:"initialUnits"`
	InitialCost  string                  `json:"initialCost"`
}

// InvestmentAccountsResponse represents a view-object of the investment account configuration list
type InvestmentAccountsResponse struct {
	Accounts []*InvestmentAccountResponse `json:"accounts"`
}

// InvestmentOverviewRequest represents all parameters of investment overview request
type InvestmentOverviewRequest struct {
	Refresh bool `form:"refresh"`
}

// InvestmentAccountAddRequest represents all parameters of investment account adding request
type InvestmentAccountAddRequest struct {
	Target       models.InvestmentTarget `json:"target" binding:"min=0,max=1"`
	AccountId    int64                   `json:"accountId,string" binding:"required,min=1"`
	InitialGrams string                  `json:"initialGrams"`
	InitialCost  string                  `json:"initialCost"`
}

// InvestmentAccountModifyRequest represents all parameters of investment account modifying request
type InvestmentAccountModifyRequest struct {
	Id           int64  `json:"id,string" binding:"required,min=1"`
	InitialGrams string `json:"initialGrams"`
	InitialCost  string `json:"initialCost"`
}

// InvestmentAccountDeleteRequest represents all parameters of investment account deleting request
type InvestmentAccountDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// InvestmentOverviewHandler returns the investment overview of current user
func (a *InvestmentsApi) InvestmentOverviewHandler(c *core.WebContext) (any, *errs.Error) {
	var request InvestmentOverviewRequest

	if err := c.ShouldBindQuery(&request); err != nil {
		log.Warnf(c, "[investments.InvestmentOverviewHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	overview, err := a.investments.GetInvestmentOverview(c, uid, request.Refresh)
	if err != nil {
		log.Errorf(c, "[investments.InvestmentOverviewHandler] failed to get investment overview for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	response := &InvestmentOverviewResponse{
		Accounts:         make([]*InvestmentAccountOverviewResponse, 0, len(overview.Accounts)),
		TotalMarketValue: utils.FormatAmount(overview.TotalMarketValue),
		TotalCost:        utils.FormatAmount(overview.TotalCost),
		TotalFloatProfit: utils.FormatAmount(overview.TotalFloatProfit),
		PriceFetchedAt:   overview.PriceFetchedAt,
		PriceSource:      "上海黄金交易所 Au99.99",
		PriceAvailable:   overview.PriceAvailable,
		PendingTransfers: make([]*PendingInvestmentTransferResponse, 0, len(overview.PendingTransfers)),
	}

	if overview.PriceAvailable {
		response.Price = utils.FormatAmount(overview.Price)
	}

	for i := 0; i < len(overview.Accounts); i++ {
		account := overview.Accounts[i]
		item := &InvestmentAccountOverviewResponse{
			Id:             account.Id,
			Target:         account.Target,
			TargetName:     account.Target.String(),
			AccountId:      account.AccountId,
			AccountName:    account.AccountName,
			Currency:       account.Currency,
			Units:          services.FormatInvestmentUnits(account.Units),
			InitialUnits:   services.FormatInvestmentUnits(account.InitialUnits),
			InitialCost:    utils.FormatAmount(account.InitialCost),
			AvgCost:        utils.FormatAmount(account.AvgCost),
			Cost:           utils.FormatAmount(account.Cost),
			MarketValue:    utils.FormatAmount(account.MarketValue),
			FloatProfit:    utils.FormatAmount(account.FloatProfit),
			Balance:        utils.FormatAmount(account.Balance),
			Diff:           utils.FormatAmount(account.Diff),
			DiffPercent:    formatDiffPercent(account.Diff, account.Balance),
			RealizedProfit: utils.FormatAmount(account.RealizedProfit),
			Records:        make([]*InvestmentRecordResponse, 0, len(account.Records)),
		}

		for j := 0; j < len(account.Records); j++ {
			record := account.Records[j]
			item.Records = append(item.Records, &InvestmentRecordResponse{
				TransactionId: record.TransactionId,
				Time:          record.Time,
				Direction:     record.Direction,
				Amount:        utils.FormatAmount(record.Amount),
				Units:         services.FormatInvestmentUnits(record.Units),
				Fee:           utils.FormatAmount(record.Fee),
				UnitPrice:     utils.FormatAmount(record.UnitPrice),
				Comment:       record.Comment,
			})
		}

		response.Accounts = append(response.Accounts, item)
	}

	for i := 0; i < len(overview.PendingTransfers); i++ {
		pendingTransfer := overview.PendingTransfers[i]
		response.PendingTransfers = append(response.PendingTransfers, &PendingInvestmentTransferResponse{
			TransactionId: pendingTransfer.TransactionId,
			Time:          pendingTransfer.Time,
			Direction:     pendingTransfer.Direction,
			Amount:        utils.FormatAmount(pendingTransfer.Amount),
			Comment:       pendingTransfer.Comment,
		})
	}

	return response, nil
}

// InvestmentAccountsHandler returns the investment account configurations of current user
func (a *InvestmentsApi) InvestmentAccountsHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	accounts, err := a.investments.GetInvestmentAccounts(c, uid)
	if err != nil {
		log.Errorf(c, "[investments.InvestmentAccountsHandler] failed to get investment accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	response := &InvestmentAccountsResponse{
		Accounts: make([]*InvestmentAccountResponse, 0, len(accounts)),
	}

	for i := 0; i < len(accounts); i++ {
		account := accounts[i]
		response.Accounts = append(response.Accounts, &InvestmentAccountResponse{
			Id:           account.Id,
			Target:       account.Target,
			TargetName:   account.Target.String(),
			AccountId:    account.AccountId,
			InitialUnits: services.FormatInvestmentUnits(account.InitialUnits),
			InitialCost:  utils.FormatAmount(account.InitialCost),
		})
	}

	return response, nil
}

// InvestmentAccountAddHandler saves a new investment account configuration for current user
func (a *InvestmentsApi) InvestmentAccountAddHandler(c *core.WebContext) (any, *errs.Error) {
	var request InvestmentAccountAddRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Warnf(c, "[investments.InvestmentAccountAddHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	initialUnits, err := services.ParseInvestmentUnits(request.InitialGrams)
	if err != nil {
		return nil, errs.ErrInvestmentUnitsInvalid
	}

	initialCost, err := utils.ParseAmount(request.InitialCost)
	if err != nil {
		return nil, errs.ErrAmountInvalid
	}

	uid := c.GetCurrentUid()

	if err = a.investments.AddInvestmentAccount(c, uid, request.Target, request.AccountId, initialUnits, initialCost); err != nil {
		log.Errorf(c, "[investments.InvestmentAccountAddHandler] failed to add investment account for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.InvestmentAccountAddHandler] user \"uid:%d\" has added investment account \"id:%d\" successfully", uid, request.AccountId)
	return true, nil
}

// InvestmentAccountModifyHandler saves the initial units and cost of an existed investment account configuration
func (a *InvestmentsApi) InvestmentAccountModifyHandler(c *core.WebContext) (any, *errs.Error) {
	var request InvestmentAccountModifyRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Warnf(c, "[investments.InvestmentAccountModifyHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	initialUnits, err := services.ParseInvestmentUnits(request.InitialGrams)
	if err != nil {
		return nil, errs.ErrInvestmentUnitsInvalid
	}

	initialCost, err := utils.ParseAmount(request.InitialCost)
	if err != nil {
		return nil, errs.ErrAmountInvalid
	}

	uid := c.GetCurrentUid()

	if err = a.investments.ModifyInvestmentAccount(c, uid, request.Id, initialUnits, initialCost); err != nil {
		log.Errorf(c, "[investments.InvestmentAccountModifyHandler] failed to modify investment account \"id:%d\" for user \"uid:%d\", because %s", request.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.InvestmentAccountModifyHandler] user \"uid:%d\" has modified investment account \"id:%d\" successfully", uid, request.Id)
	return true, nil
}

// InvestmentAccountDeleteHandler deletes an existed investment account configuration
func (a *InvestmentsApi) InvestmentAccountDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	var request InvestmentAccountDeleteRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Warnf(c, "[investments.InvestmentAccountDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	if err := a.investments.DeleteInvestmentAccount(c, uid, request.Id); err != nil {
		log.Errorf(c, "[investments.InvestmentAccountDeleteHandler] failed to delete investment account \"id:%d\" for user \"uid:%d\", because %s", request.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.InvestmentAccountDeleteHandler] user \"uid:%d\" has deleted investment account \"id:%d\" successfully", uid, request.Id)
	return true, nil
}

func formatDiffPercent(diff int64, balance int64) string {
	if balance == 0 {
		return "0.00"
	}

	percent := math.Round(float64(diff) / float64(balance) * 10000)

	return fmt.Sprintf("%.2f", percent/100)
}
