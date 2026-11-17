package services

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// InvestmentUnitsScale is the scale of investment units (1 unit = 0.0001g)
const InvestmentUnitsScale int64 = 10000

// InvestmentValuationCategoryName is the name of the category used by investment valuation transactions
const InvestmentValuationCategoryName = "投资估值变动"

// InvestmentValuationCommentPrefix is the comment prefix of investment valuation transactions
const InvestmentValuationCommentPrefix = "【投资估值】"

// InvestmentTransferCategoryName is the name of the transfer category for investment
const InvestmentTransferCategoryName = "投资"

var investmentUnitsPattern = regexp.MustCompile(`(?i)(?:(?:克重|重量)[:：\s]*(\d+(?:\.\d+)?))|(?:(\d+(?:\.\d+)?)\s*(?:克|g))`)

// InvestmentService represents investment service
type InvestmentService struct {
	ServiceUsingDB

	priceMutex  sync.Mutex
	cachedPrice *goldPrice
}

// Initialize an investment service singleton instance
var (
	Investments = &InvestmentService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
	}
)

// investmentHolding represents the current holding of an investment account
type investmentHolding struct {
	Units          int64 // 0.0001g
	Cost           int64 // cents
	RealizedProfit int64 // cents
	AvgCost        int64 // cents per gram
}

// parseInvestmentUnitsFromComment parses investment units from the transaction comment.
// It returns units in 0.0001g, -1 when the comment marks the transaction as ignored, or 0 when no units can be parsed.
func parseInvestmentUnitsFromComment(comment string) int64 {
	if strings.Contains(comment, "不计入") || strings.Contains(comment, "忽略") {
		return -1
	}

	// "千克" and "kg" must not be recognized as grams
	if strings.Contains(comment, "千克") || strings.Contains(strings.ToLower(comment), "kg") {
		return 0
	}

	matches := investmentUnitsPattern.FindStringSubmatch(comment)

	if len(matches) < 3 {
		return 0
	}

	value := matches[1]

	if value == "" {
		value = matches[2]
	}

	if value == "" {
		return 0
	}

	parsed, err := strconv.ParseFloat(value, 64)

	if err != nil || parsed <= 0 {
		return 0
	}

	return int64(math.Round(parsed * float64(InvestmentUnitsScale)))
}

// calculateInvestmentHolding calculates the current holding of an investment account.
// Records are the transfer out transactions which involve the given investment account.
func calculateInvestmentHolding(initialUnits int64, initialCost int64, records []*models.Transaction, investmentAccountId int64) investmentHolding {
	holding := investmentHolding{
		Units: initialUnits,
		Cost:  initialCost,
	}

	sortedRecords := make([]*models.Transaction, len(records))
	copy(sortedRecords, records)
	sort.SliceStable(sortedRecords, func(i, j int) bool {
		return sortedRecords[i].TransactionTime < sortedRecords[j].TransactionTime
	})

	for i := 0; i < len(sortedRecords); i++ {
		record := sortedRecords[i]

		if record.InvestmentTarget != models.INVESTMENT_TARGET_GOLD || record.InvestmentUnits <= 0 {
			continue
		}

		if record.RelatedAccountId == investmentAccountId {
			// transfer to the investment account means buying
			holding.Units += record.InvestmentUnits
			holding.Cost += record.Amount
		} else if record.AccountId == investmentAccountId {
			// transfer from the investment account means selling
			soldUnits := record.InvestmentUnits

			if soldUnits > holding.Units {
				soldUnits = holding.Units
			}

			if soldUnits <= 0 {
				continue
			}

			costRemoved := calculateProportionalAmount(holding.Cost, soldUnits, holding.Units)

			holding.Units -= soldUnits
			holding.Cost -= costRemoved
			holding.RealizedProfit += record.Amount - costRemoved
		}
	}

	if holding.Units > 0 {
		holding.AvgCost = calculateUnitPrice(holding.Cost, holding.Units)
	}

	return holding
}

// calculateProportionalAmount returns the proportional part of the amount.
func calculateProportionalAmount(total int64, part int64, base int64) int64 {
	if base <= 0 || part <= 0 {
		return 0
	}

	return int64(math.Round(float64(total) * float64(part) / float64(base)))
}

// calculateUnitPrice returns the price of each gram in cents.
func calculateUnitPrice(amount int64, units int64) int64 {
	if units <= 0 {
		return 0
	}

	return int64(math.Round(float64(amount) * float64(InvestmentUnitsScale) / float64(units)))
}

// calculateInvestmentMarketValue returns the market value in cents according to units and price of each gram.
func calculateInvestmentMarketValue(units int64, pricePerGram int64) int64 {
	return int64(math.Round(float64(units) * float64(pricePerGram) / float64(InvestmentUnitsScale)))
}

// FormatInvestmentUnits returns the textual representation of investment units.
func FormatInvestmentUnits(units int64) string {
	sign := ""

	if units < 0 {
		sign = "-"
		units = -units
	}

	return fmt.Sprintf("%s%d.%04d", sign, units/InvestmentUnitsScale, units%InvestmentUnitsScale)
}

// ParseInvestmentUnits parses the textual representation of investment units (in grams) to 0.0001g.
func ParseInvestmentUnits(value string) (int64, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, errs.ErrInvestmentUnitsInvalid
	}

	if parsed < 0 {
		return 0, errs.ErrInvestmentUnitsInvalid
	}

	return int64(math.Round(parsed * float64(InvestmentUnitsScale))), nil
}

// checkInvestmentTransactionFields verifies the investment fields of a transaction.
func checkInvestmentTransactionFields(transaction *models.Transaction, investmentTargets map[int64]models.InvestmentTarget) error {
	if transaction.InvestmentTarget == models.INVESTMENT_TARGET_NONE {
		if transaction.InvestmentUnits != 0 || transaction.InvestmentFee != 0 {
			return errs.ErrInvestmentUnitsInvalid
		}

		return nil
	}

	if transaction.InvestmentTarget != models.INVESTMENT_TARGET_GOLD {
		return errs.ErrInvestmentTargetInvalid
	}

	if transaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		return errs.ErrInvestmentTransactionMustBeTransfer
	}

	sourceIsInvestment := investmentTargets[transaction.AccountId] == models.INVESTMENT_TARGET_GOLD
	destinationIsInvestment := investmentTargets[transaction.RelatedAccountId] == models.INVESTMENT_TARGET_GOLD

	if sourceIsInvestment && destinationIsInvestment {
		return errs.ErrCannotTransferBetweenInvestmentAccounts
	}

	if !sourceIsInvestment && !destinationIsInvestment {
		return errs.ErrInvestmentAccountNotFound
	}

	if transaction.InvestmentUnits <= 0 {
		return errs.ErrInvestmentUnitsInvalid
	}

	if transaction.InvestmentFee < 0 {
		return errs.ErrInvestmentFeeInvalid
	}

	return nil
}

// validateInvestmentTransaction checks the investment fields of a transaction, only querying
// the investment account configurations when the transaction carries an investment target.
func validateInvestmentTransaction(sess datastore.DatabaseSession, transaction *models.Transaction) error {
	if transaction.InvestmentTarget == models.INVESTMENT_TARGET_NONE {
		return checkInvestmentTransactionFields(transaction, nil)
	}

	investmentTargets, err := getInvestmentTargetsByAccountIds(sess, transaction.Uid, transaction.AccountId, transaction.RelatedAccountId)
	if err != nil {
		return err
	}

	return checkInvestmentTransactionFields(transaction, investmentTargets)
}

// getInvestmentTargetsByAccountIds returns the investment targets by the given account ids
func getInvestmentTargetsByAccountIds(sess datastore.DatabaseSession, uid int64, accountIds ...int64) (map[int64]models.InvestmentTarget, error) {
	targets := make(map[int64]models.InvestmentTarget)
	ids := make([]int64, 0, len(accountIds))

	for i := 0; i < len(accountIds); i++ {
		if accountIds[i] > 0 {
			ids = append(ids, accountIds[i])
		}
	}

	if len(ids) < 1 {
		return targets, nil
	}

	var accounts []*models.InvestmentAccount
	err := sess.Where("uid=?", uid).In("account_id", ids).Find(&accounts)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(accounts); i++ {
		targets[accounts[i].AccountId] = accounts[i].Target
	}

	return targets, nil
}

// GetInvestmentAccounts returns all investment account configurations of user
func (s *InvestmentService) GetInvestmentAccounts(c core.Context, uid int64) ([]*models.InvestmentAccount, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var accounts []*models.InvestmentAccount
	err := s.UserDataDB(uid).NewSession(c).Where("uid=?", uid).OrderBy("id asc").Find(&accounts)

	if accounts == nil {
		accounts = make([]*models.InvestmentAccount, 0)
	}

	return accounts, err
}

// GetInvestmentAccountById returns an investment account configuration according to id
func (s *InvestmentService) GetInvestmentAccountById(c core.Context, uid int64, id int64) (*models.InvestmentAccount, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if id <= 0 {
		return nil, errs.ErrInvestmentAccountIdInvalid
	}

	account := &models.InvestmentAccount{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(id).Where("uid=?", uid).Get(account)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrInvestmentAccountNotFound
	}

	return account, nil
}

// AddInvestmentAccount saves a new investment account configuration to database
func (s *InvestmentService) AddInvestmentAccount(c core.Context, uid int64, target models.InvestmentTarget, accountId int64, initialUnits int64, initialCost int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if target != models.INVESTMENT_TARGET_GOLD {
		return errs.ErrInvestmentTargetInvalid
	}

	if accountId <= 0 {
		return errs.ErrAccountIdInvalid
	}

	account, err := s.getAccount(c, uid, accountId)
	if err != nil {
		return err
	}

	if account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
		return errs.ErrAccountTypeInvalid
	}

	if account.Currency != "CNY" {
		return errs.ErrInvestmentAccountCurrencyInvalid
	}

	if initialUnits < 0 || initialCost < 0 {
		return errs.ErrInvestmentUnitsInvalid
	}

	exists, err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND account_id=?", uid, accountId).Exist(&models.InvestmentAccount{})

	if err != nil {
		return err
	} else if exists {
		return errs.ErrInvestmentAccountAlreadyExists
	}

	now := time.Now().Unix()

	investmentAccount := &models.InvestmentAccount{
		Uid:             uid,
		Target:          target,
		AccountId:       accountId,
		InitialUnits:    initialUnits,
		InitialCost:     initialCost,
		CreatedUnixTime: now,
		UpdatedUnixTime: now,
	}

	_, err = s.UserDataDB(uid).NewSession(c).Insert(investmentAccount)
	if err != nil {
		return err
	}

	_, err = s.ensureCategory(c, uid, models.CATEGORY_TYPE_TRANSFER, InvestmentTransferCategoryName)
	if err != nil {
		return err
	}

	return s.backfillInvestmentTransactions(c, uid, investmentAccount)
}

// ModifyInvestmentAccount saves the initial units and cost of an existed investment account configuration
func (s *InvestmentService) ModifyInvestmentAccount(c core.Context, uid int64, id int64, initialUnits int64, initialCost int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	account, err := s.GetInvestmentAccountById(c, uid, id)
	if err != nil {
		return err
	}

	if initialUnits < 0 || initialCost < 0 {
		return errs.ErrInvestmentUnitsInvalid
	}

	account.InitialUnits = initialUnits
	account.InitialCost = initialCost
	account.UpdatedUnixTime = time.Now().Unix()

	_, err = s.UserDataDB(uid).NewSession(c).ID(id).Cols("initial_units", "initial_cost", "updated_unix_time").Where("uid=?", uid).Update(account)
	if err != nil {
		return err
	}

	return s.backfillInvestmentTransactions(c, uid, account)
}

// DeleteInvestmentAccount deletes an existed investment account configuration from database
func (s *InvestmentService) DeleteInvestmentAccount(c core.Context, uid int64, id int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if id <= 0 {
		return errs.ErrInvestmentAccountIdInvalid
	}

	deletedRows, err := s.UserDataDB(uid).NewSession(c).ID(id).Where("uid=?", uid).Delete(&models.InvestmentAccount{})

	if err != nil {
		return err
	} else if deletedRows < 1 {
		return errs.ErrInvestmentAccountNotFound
	}

	return nil
}

// backfillInvestmentTransactions parses and fills the investment fields of the existed transfers of the given investment account
func (s *InvestmentService) backfillInvestmentTransactions(c core.Context, uid int64, account *models.InvestmentAccount) error {
	var transactions []*models.Transaction
	err := s.UserDataDB(uid).NewSession(c).Where(
		"uid=? AND deleted=? AND type=? AND investment_target=? AND (account_id=? OR related_account_id=?)",
		uid, false, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, models.INVESTMENT_TARGET_NONE, account.AccountId, account.AccountId).Find(&transactions)
	if err != nil {
		return err
	}

	now := time.Now().Unix()

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		units := parseInvestmentUnitsFromComment(transaction.Comment)

		if units > 0 {
			transaction.InvestmentTarget = models.INVESTMENT_TARGET_GOLD
			transaction.InvestmentUnits = units
			transaction.InvestmentFee = 0
		} else if units < 0 {
			// marked as ignored by the comment, keep it out of holdings and out of the pending list
			transaction.InvestmentTarget = models.INVESTMENT_TARGET_GOLD
			transaction.InvestmentUnits = 0
			transaction.InvestmentFee = 0
		} else {
			// unparseable comment stays pending
			continue
		}

		transaction.UpdatedUnixTime = now

		_, err := s.UserDataDB(uid).NewSession(c).ID(transaction.TransactionId).Cols(
			"investment_target", "investment_units", "investment_fee", "updated_unix_time").Where("uid=?", uid).Update(transaction)
		if err != nil {
			return err
		}
	}

	return nil
}

// getInvestmentTransactions returns all investment transactions which involve the given account
func (s *InvestmentService) getInvestmentTransactions(c core.Context, uid int64, accountId int64) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	err := s.UserDataDB(uid).NewSession(c).Where(
		"uid=? AND deleted=? AND type=? AND investment_target=? AND (account_id=? OR related_account_id=?)",
		uid, false, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, models.INVESTMENT_TARGET_GOLD, accountId, accountId).OrderBy("transaction_time asc").Find(&transactions)
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

// hasPendingInvestmentTransfers returns whether there are transfers which involve the given account but have no investment units
func (s *InvestmentService) hasPendingInvestmentTransfers(c core.Context, uid int64, accountId int64) (bool, error) {
	return s.UserDataDB(uid).NewSession(c).Where(
		"uid=? AND deleted=? AND type=? AND investment_target=? AND (account_id=? OR related_account_id=?)",
		uid, false, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, models.INVESTMENT_TARGET_NONE, accountId, accountId).Exist(&models.Transaction{})
}

// getAccount returns an account model according to account id
func (s *InvestmentService) getAccount(c core.Context, uid int64, accountId int64) (*models.Account, error) {
	account := &models.Account{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(accountId).Where("uid=? AND deleted=?", uid, false).Get(account)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrAccountNotFound
	}

	return account, nil
}

// ensureCategory returns the sub transaction category with the given type and name, creates it if it does not exist
func (s *InvestmentService) ensureCategory(c core.Context, uid int64, categoryType models.TransactionCategoryType, name string) (*models.TransactionCategory, error) {
	existedCategory := &models.TransactionCategory{}
	has, err := s.UserDataDB(uid).NewSession(c).Where(
		"uid=? AND deleted=? AND type=? AND name=? AND parent_category_id<>?",
		uid, false, categoryType, name, models.LevelOneTransactionCategoryParentId).OrderBy("display_order asc").Limit(1).Get(existedCategory)

	if err != nil {
		return nil, err
	} else if has {
		return existedCategory, nil
	}

	parentCategory := &models.TransactionCategory{}
	has, err = s.UserDataDB(uid).NewSession(c).Where(
		"uid=? AND deleted=? AND type=? AND parent_category_id=?",
		uid, false, categoryType, models.LevelOneTransactionCategoryParentId).OrderBy("display_order asc").Limit(1).Get(parentCategory)
	if err != nil {
		return nil, err
	}

	if !has {
		maxDisplayOrder, err := TransactionCategories.GetMaxDisplayOrder(c, uid, categoryType)
		if err != nil {
			return nil, err
		}

		parentCategory = &models.TransactionCategory{
			Uid:              uid,
			Type:             categoryType,
			ParentCategoryId: models.LevelOneTransactionCategoryParentId,
			Name:             name,
			DisplayOrder:     maxDisplayOrder + 1,
			Icon:             102,
			Color:            "ffcc00",
		}

		err = TransactionCategories.CreateCategory(c, parentCategory)
		if err != nil {
			return nil, err
		}
	}

	maxSubDisplayOrder, err := TransactionCategories.GetMaxSubCategoryDisplayOrder(c, uid, categoryType, parentCategory.CategoryId)
	if err != nil {
		return nil, err
	}

	category := &models.TransactionCategory{
		Uid:              uid,
		Type:             categoryType,
		ParentCategoryId: parentCategory.CategoryId,
		Name:             name,
		DisplayOrder:     maxSubDisplayOrder + 1,
		Icon:             102,
		Color:            "ffcc00",
	}

	err = TransactionCategories.CreateCategory(c, category)
	if err != nil {
		return nil, err
	}

	return category, nil
}

// UpdateAllInvestmentValuations updates the valuations of all investment accounts of all users
func (s *InvestmentService) UpdateAllInvestmentValuations(c core.Context) error {
	for i := 0; i < s.UserDataDBCount(); i++ {
		var accounts []*models.InvestmentAccount
		err := s.UserDataDBByIndex(i).NewSession(c).Where("target=?", models.INVESTMENT_TARGET_GOLD).Find(&accounts)
		if err != nil {
			log.Errorf(c, "[investments.UpdateAllInvestmentValuations] failed to get investment accounts from database \"index:%d\", because %s", i, err.Error())
			continue
		}

		for j := 0; j < len(accounts); j++ {
			err = s.updateInvestmentValuation(c, accounts[j].Uid, accounts[j])
			if err != nil {
				log.Errorf(c, "[investments.UpdateAllInvestmentValuations] failed to update investment valuation of account \"id:%d\" for user \"uid:%d\", because %s", accounts[j].AccountId, accounts[j].Uid, err.Error())
			}
		}
	}

	return nil
}

// updateInvestmentValuation creates an income or expense transaction when the market value of the investment account differs from its balance
func (s *InvestmentService) updateInvestmentValuation(c core.Context, uid int64, account *models.InvestmentAccount) error {
	if account.Target != models.INVESTMENT_TARGET_GOLD {
		return nil
	}

	pending, err := s.hasPendingInvestmentTransfers(c, uid, account.AccountId)

	if err != nil {
		return err
	} else if pending {
		log.Infof(c, "[investments.updateInvestmentValuation] skip investment valuation of account \"id:%d\" for user \"uid:%d\", because there are transfers without investment units", account.AccountId, uid)
		return nil
	}

	transactions, err := s.getInvestmentTransactions(c, uid, account.AccountId)
	if err != nil {
		return err
	}

	holding := calculateInvestmentHolding(account.InitialUnits, account.InitialCost, transactions, account.AccountId)

	if holding.Units <= 0 {
		log.Infof(c, "[investments.updateInvestmentValuation] skip investment valuation of account \"id:%d\" for user \"uid:%d\", because there is no holding", account.AccountId, uid)
		return nil
	}

	accountModel, err := s.getAccount(c, uid, account.AccountId)
	if err != nil {
		return err
	}

	if accountModel.Currency != "CNY" {
		return nil
	}

	price, err := s.getGoldPrice(c, true)
	if err != nil {
		log.Warnf(c, "[investments.updateInvestmentValuation] skip investment valuation of account \"id:%d\" for user \"uid:%d\", because %s", account.AccountId, uid, err.Error())
		return nil
	}

	marketValue := calculateInvestmentMarketValue(holding.Units, price.Price)
	diff := marketValue - accountModel.Balance

	if diff == 0 {
		return nil
	}

	categoryType := models.CATEGORY_TYPE_INCOME
	transactionType := models.TRANSACTION_DB_TYPE_INCOME
	diffAmount := diff

	if diff < 0 {
		categoryType = models.CATEGORY_TYPE_EXPENSE
		transactionType = models.TRANSACTION_DB_TYPE_EXPENSE
		diffAmount = -diff
	}

	category, err := s.ensureCategory(c, uid, categoryType, InvestmentValuationCategoryName)
	if err != nil {
		return err
	}

	now := time.Now()
	_, zoneOffsetSeconds := now.Zone()

	comment := fmt.Sprintf("%sAu99.99 %s 元/克 × %s 克 = %s 元，较账面 %s 元",
		InvestmentValuationCommentPrefix, utils.FormatAmount(price.Price), FormatInvestmentUnits(holding.Units),
		utils.FormatAmount(marketValue), formatSignedAmount(diff))

	transaction := &models.Transaction{
		Uid:               uid,
		Type:              transactionType,
		CategoryId:        category.CategoryId,
		TransactionTime:   utils.GetMinTransactionTimeFromUnixTime(now.Unix()),
		TimezoneUtcOffset: int16(zoneOffsetSeconds / 60),
		AccountId:         account.AccountId,
		Amount:            diffAmount,
		Comment:           comment,
	}

	err = Transactions.CreateTransaction(c, transaction, nil, nil)
	if err != nil {
		return err
	}

	log.Infof(c, "[investments.updateInvestmentValuation] user \"uid:%d\" has created investment valuation transaction \"id:%d\" for account \"id:%d\"", uid, transaction.TransactionId, account.AccountId)
	return nil
}

func formatSignedAmount(amount int64) string {
	if amount >= 0 {
		return "+" + utils.FormatAmount(amount)
	}

	return "-" + utils.FormatAmount(-amount)
}

// InvestmentRecord represents a record of buying or selling an investment
type InvestmentRecord struct {
	TransactionId int64
	Time          int64
	Direction     string
	Amount        int64
	Units         int64
	Fee           int64
	UnitPrice     int64
	Comment       string
}

// PendingInvestmentTransfer represents a transfer which involves an investment account but has no investment units
type PendingInvestmentTransfer struct {
	TransactionId int64
	Time          int64
	Direction     string
	Amount        int64
	Comment       string
}

// InvestmentAccountOverview represents the overview of an investment account
type InvestmentAccountOverview struct {
	Id             int64
	Target         models.InvestmentTarget
	AccountId      int64
	AccountName    string
	Currency       string
	Units          int64
	InitialUnits   int64
	InitialCost    int64
	AvgCost        int64
	Cost           int64
	RealizedProfit int64
	MarketValue    int64
	FloatProfit    int64
	Balance        int64
	Diff           int64
	Records        []*InvestmentRecord
}

// InvestmentOverview represents the overview of all investment accounts of a user
type InvestmentOverview struct {
	Accounts         []*InvestmentAccountOverview
	TotalMarketValue int64
	TotalCost        int64
	TotalFloatProfit int64
	Price            int64
	PriceFetchedAt   int64
	PriceAvailable   bool
	PendingTransfers []*PendingInvestmentTransfer
}

// GetInvestmentOverview returns the overview of all investment accounts of user
func (s *InvestmentService) GetInvestmentOverview(c core.Context, uid int64, force bool) (*InvestmentOverview, error) {
	accounts, err := s.GetInvestmentAccounts(c, uid)
	if err != nil {
		return nil, err
	}

	overview := &InvestmentOverview{
		Accounts:         make([]*InvestmentAccountOverview, 0, len(accounts)),
		PendingTransfers: make([]*PendingInvestmentTransfer, 0),
	}

	if len(accounts) < 1 {
		return overview, nil
	}

	accountIds := make([]int64, len(accounts))

	for i := 0; i < len(accounts); i++ {
		accountIds[i] = accounts[i].AccountId
	}

	accountModels, err := Accounts.GetAccountsByAccountIds(c, uid, accountIds)
	if err != nil {
		return nil, err
	}

	var price *goldPrice

	price, err = s.getGoldPrice(c, force)

	if err != nil {
		log.Warnf(c, "[investments.GetInvestmentOverview] failed to get gold price for user \"uid:%d\", because %s", uid, err.Error())
		price = nil
	} else {
		overview.PriceAvailable = true
		overview.Price = price.Price
		overview.PriceFetchedAt = price.FetchedAt
	}

	for i := 0; i < len(accounts); i++ {
		account := accounts[i]
		item := &InvestmentAccountOverview{
			Id:           account.Id,
			Target:       account.Target,
			AccountId:    account.AccountId,
			InitialUnits: account.InitialUnits,
			InitialCost:  account.InitialCost,
			Records:      make([]*InvestmentRecord, 0),
		}

		if accountModel := accountModels[account.AccountId]; accountModel != nil {
			item.AccountName = accountModel.Name
			item.Currency = accountModel.Currency
			item.Balance = accountModel.Balance
		}

		transactions, err := s.getInvestmentTransactions(c, uid, account.AccountId)
		if err != nil {
			return nil, err
		}

		holding := calculateInvestmentHolding(account.InitialUnits, account.InitialCost, transactions, account.AccountId)
		item.Units = holding.Units
		item.Cost = holding.Cost
		item.AvgCost = holding.AvgCost
		item.RealizedProfit = holding.RealizedProfit
		item.Records = buildInvestmentRecords(transactions, account.AccountId)

		if price != nil {
			item.MarketValue = calculateInvestmentMarketValue(holding.Units, price.Price)
			item.FloatProfit = item.MarketValue - holding.Cost
			item.Diff = item.MarketValue - item.Balance
		}

		overview.Accounts = append(overview.Accounts, item)
		overview.TotalMarketValue += item.MarketValue
		overview.TotalCost += item.Cost
		overview.TotalFloatProfit += item.FloatProfit
	}

	overview.PendingTransfers, err = s.getPendingInvestmentTransfers(c, uid, accountIds)
	if err != nil {
		return nil, err
	}

	return overview, nil
}

func buildInvestmentRecords(transactions []*models.Transaction, accountId int64) []*InvestmentRecord {
	records := make([]*InvestmentRecord, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.InvestmentTarget != models.INVESTMENT_TARGET_GOLD || transaction.InvestmentUnits <= 0 {
			continue
		}

		direction := "buy"

		if transaction.AccountId == accountId {
			direction = "sell"
		}

		records = append(records, &InvestmentRecord{
			TransactionId: transaction.TransactionId,
			Time:          utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime),
			Direction:     direction,
			Amount:        transaction.Amount,
			Units:         transaction.InvestmentUnits,
			Fee:           transaction.InvestmentFee,
			UnitPrice:     calculateUnitPrice(transaction.Amount, transaction.InvestmentUnits),
			Comment:       transaction.Comment,
		})
	}

	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Time > records[j].Time
	})

	return records
}

func (s *InvestmentService) getPendingInvestmentTransfers(c core.Context, uid int64, accountIds []int64) ([]*PendingInvestmentTransfer, error) {
	result := make([]*PendingInvestmentTransfer, 0)

	if len(accountIds) < 1 {
		return result, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(accountIds)), ",")
	condition := "uid=? AND deleted=? AND type=? AND investment_target=? AND (account_id IN (" + placeholders + ") OR related_account_id IN (" + placeholders + "))"
	params := make([]any, 0, 4+len(accountIds)*2)
	params = append(params, uid, false, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, models.INVESTMENT_TARGET_NONE)

	for i := 0; i < 2; i++ {
		for j := 0; j < len(accountIds); j++ {
			params = append(params, accountIds[j])
		}
	}

	var transactions []*models.Transaction
	err := s.UserDataDB(uid).NewSession(c).Where(condition, params...).OrderBy("transaction_time desc").Find(&transactions)
	if err != nil {
		return nil, err
	}

	accountIdSet := make(map[int64]bool, len(accountIds))

	for i := 0; i < len(accountIds); i++ {
		accountIdSet[accountIds[i]] = true
	}

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		direction := "buy"

		if accountIdSet[transaction.AccountId] {
			direction = "sell"
		}

		result = append(result, &PendingInvestmentTransfer{
			TransactionId: transaction.TransactionId,
			Time:          utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime),
			Direction:     direction,
			Amount:        transaction.Amount,
			Comment:       transaction.Comment,
		})
	}

	return result, nil
}
