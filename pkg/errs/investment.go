package errs

import "net/http"

// Error codes related to investment
var (
	ErrInvestmentTargetInvalid                 = NewNormalError(NormalSubcategoryInvestment, 0, http.StatusBadRequest, "investment target is invalid")
	ErrInvestmentAccountNotFound               = NewNormalError(NormalSubcategoryInvestment, 1, http.StatusBadRequest, "investment account not found")
	ErrInvestmentAccountAlreadyExists          = NewNormalError(NormalSubcategoryInvestment, 2, http.StatusBadRequest, "investment account already exists")
	ErrCannotTransferBetweenInvestmentAccounts = NewNormalError(NormalSubcategoryInvestment, 3, http.StatusBadRequest, "cannot transfer between investment accounts")
	ErrInvestmentTransactionMustBeTransfer     = NewNormalError(NormalSubcategoryInvestment, 4, http.StatusBadRequest, "investment transaction must be a transfer")
	ErrInvestmentUnitsInvalid                  = NewNormalError(NormalSubcategoryInvestment, 5, http.StatusBadRequest, "investment units is invalid")
	ErrInvestmentFeeInvalid                    = NewNormalError(NormalSubcategoryInvestment, 6, http.StatusBadRequest, "investment fee is invalid")
	ErrInvestmentAccountCurrencyInvalid        = NewNormalError(NormalSubcategoryInvestment, 7, http.StatusBadRequest, "investment account currency must be CNY")
	ErrInvestmentPriceUnavailable              = NewNormalError(NormalSubcategoryInvestment, 8, http.StatusInternalServerError, "investment price is unavailable")
	ErrInvestmentAccountIdInvalid              = NewNormalError(NormalSubcategoryInvestment, 9, http.StatusBadRequest, "investment account id is invalid")
)
