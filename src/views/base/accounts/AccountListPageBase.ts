import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useInvestmentsStore } from '@/stores/investment.ts';

import { INVESTMENT_VALUATION_UP_COLOR, INVESTMENT_VALUATION_DOWN_COLOR, type InvestmentValuationDisplay } from '@/views/base/overview/AssetSummaryWidgetBase.ts';

import type { BigDecimal, HiddenAmount, BigDecimalWithSuffix } from '@/core/numeral.ts';
import type { WeekDayValue } from '@/core/datetime.ts';
import { AccountCategory, AccountType } from '@/core/account.ts';
import { ACCOUNT_CURRENCY_NOT_SET_VALUE } from '@/consts/currency.ts';

import type { Account, CategorizedAccount } from '@/models/account.ts';
import { parseInvestmentAmount } from '@/models/investment.ts';

import { isDefined, isObject, isString } from '@/lib/common.ts';
import { isBigDecimal, parseBigDecimal } from '@/lib/numeral.ts';

export function useAccountListPageBase() {
    const { formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

    const settingsStore = useSettingsStore();
    const userStore = useUserStore();
    const accountsStore = useAccountsStore();
    const investmentsStore = useInvestmentsStore();

    // a failed request just hides the investment valuation
    if (!investmentsStore.loading) {
        investmentsStore.loadOverview(false).catch(() => {});
    }

    const loading = ref<boolean>(true);
    const showHidden = ref<boolean>(false);
    const displayOrderModified = ref<boolean>(false);

    const showAccountBalance = computed<boolean>({
        get: () => settingsStore.appSettings.showAccountBalance,
        set: (value) => settingsStore.setShowAccountBalance(value)
    });

    const customAccountCategoryOrder = computed<string>(() => settingsStore.appSettings.accountCategoryOrders);
    const defaultAccountCategory = computed<AccountCategory>(() => AccountCategory.values(customAccountCategoryOrder.value)[0] ?? AccountCategory.Default);

    const firstDayOfWeek = computed<WeekDayValue>(() => userStore.currentUserFirstDayOfWeek);
    const fiscalYearStart = computed<number>(() => userStore.currentUserFiscalYearStart);
    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
    const useLastReconciledTime = computed(() => userStore.currentUserUseLastReconciledTime);

    const allAccounts = computed<Account[]>(() => accountsStore.allAccounts);
    const allCategorizedAccountsMap = computed<Record<number, CategorizedAccount>>(() => accountsStore.allCategorizedAccountsMap);
    const allAccountCount = computed<number>(() => accountsStore.allAvailableAccountsCount);
    const maxCategoryAccountCount = computed<number>(() => accountsStore.maxCategoryAccountCount);

    const netAssets = computed<string>(() => {
        const netAssets: BigDecimal | HiddenAmount | BigDecimalWithSuffix = accountsStore.getNetAssets(showAccountBalance.value, settingsStore.appSettings.totalAmountExcludeAccountIds);
        return formatAmountToLocalizedNumeralsWithCurrency(netAssets, defaultCurrency.value);
    });

    const totalAssets = computed<string>(() => {
        const totalAssets: BigDecimal | HiddenAmount | BigDecimalWithSuffix = accountsStore.getTotalAssets(showAccountBalance.value, settingsStore.appSettings.totalAmountExcludeAccountIds);
        return formatAmountToLocalizedNumeralsWithCurrency(totalAssets, defaultCurrency.value);
    });

    const totalLiabilities = computed<string>(() => {
        const totalLiabilities: BigDecimal | HiddenAmount | BigDecimalWithSuffix = accountsStore.getTotalLiabilities(showAccountBalance.value, settingsStore.appSettings.totalAmountExcludeAccountIds);
        return formatAmountToLocalizedNumeralsWithCurrency(totalLiabilities, defaultCurrency.value);
    });

    // the investment valuation is an additional line and is hidden when amounts are hidden, there is no investment account or the gold price is unavailable
    const showInvestmentValuation = computed<boolean>(() => {
        const overview = investmentsStore.overview;
        return showAccountBalance.value && !!overview && overview.priceAvailable && overview.accounts.length > 0;
    });

    const investmentMarketValue = computed<string>(() => {
        if (!showInvestmentValuation.value) {
            return '';
        }

        return formatAmountToLocalizedNumeralsWithCurrency(investmentsStore.totalMarketValue, defaultCurrency.value);
    });

    const investmentDiffText = computed<string>(() => {
        if (!showInvestmentValuation.value) {
            return '';
        }

        const diff: BigDecimal = investmentsStore.totalDiff;
        const prefix = diff.isPositive() ? '+' : (diff.isNegative() ? '-' : '');

        return prefix + formatAmountToLocalizedNumeralsWithCurrency(diff.abs(), defaultCurrency.value);
    });

    const investmentDiffColor = computed<string>(() => {
        const diff: BigDecimal = investmentsStore.totalDiff;

        if (diff.isPositive()) {
            return INVESTMENT_VALUATION_UP_COLOR;
        } else if (diff.isNegative()) {
            return INVESTMENT_VALUATION_DOWN_COLOR;
        } else {
            return '';
        }
    });

    function canShowAvailableCredit(account: Account): boolean {
        return account.category === AccountCategory.CreditCard.type && account.numericCreditCardLimit > 0 && !!account.currency && account.currency !== ACCOUNT_CURRENCY_NOT_SET_VALUE;
    }

    function accountCategoryTotalBalance(accountCategory: AccountCategory | undefined, showAvailableCreditForCreditCard: boolean): string {
        if (!accountCategory) {
            return '';
        }

        const totalBalance: BigDecimal | HiddenAmount | BigDecimalWithSuffix | undefined = accountsStore.getAccountCategoryTotalBalance(showAccountBalance.value, accountCategory, showAvailableCreditForCreditCard);

        if (isDefined(totalBalance)) {
            return formatAmountToLocalizedNumeralsWithCurrency(totalBalance, defaultCurrency.value);
        } else {
            return '';
        }
    }

    function accountBalance(account: Account, currentSubAccountId: string | undefined, showBalance: boolean, onlyShowSelectedAccountIds?: string[]): string | null {
        if (account.type === AccountType.SingleAccount.type) {
            const balance: BigDecimal | HiddenAmount | null = accountsStore.getAccountBalance(showBalance, account);

            if (isBigDecimal(balance) || isString(balance)) {
                return formatAmountToLocalizedNumeralsWithCurrency(balance, account.currency);
            } else {
                return '';
            }
        } else if (account.type === AccountType.MultiSubAccounts.type) {
            const balanceResult = accountsStore.getAccountSubAccountBalance(showBalance, showHidden.value, account, currentSubAccountId, onlyShowSelectedAccountIds);

            if (!isObject(balanceResult)) {
                return '';
            }

            return formatAmountToLocalizedNumeralsWithCurrency(balanceResult.balance, balanceResult.currency);
        } else {
            return null;
        }
    }

    function accountAvailableCredit(account: Account, showBalance: boolean): string | null {
        if (!canShowAvailableCredit(account)) {
            return null;
        }

        const creditCardLimit = parseBigDecimal(account.creditCardLimit);

        if (account.type === AccountType.SingleAccount.type) {
            const balance: BigDecimal | HiddenAmount | null = accountsStore.getAccountBalance(showBalance, account);

            if (isBigDecimal(balance)) {
                return formatAmountToLocalizedNumeralsWithCurrency(creditCardLimit.subtract(balance), account.currency);
            } else if (isString(balance)) {
                return formatAmountToLocalizedNumeralsWithCurrency(balance, account.currency);
            } else {
                return null;
            }
        } else if (account.type === AccountType.MultiSubAccounts.type) {
            const balanceResult = accountsStore.getAccountSubAccountBalance(showBalance, showHidden.value, account, undefined, undefined);

            if (!isObject(balanceResult)) {
                return null;
            }

            const balance: BigDecimal | HiddenAmount | BigDecimalWithSuffix = balanceResult.balance;

            if (isBigDecimal(balance)) {
                return formatAmountToLocalizedNumeralsWithCurrency(creditCardLimit.subtract(balance), balanceResult.currency);
            } else if (isObject(balance) && !balance.suffix) {
                return formatAmountToLocalizedNumeralsWithCurrency(creditCardLimit.subtract(balance.value), balanceResult.currency);
            } else if (isString(balance)) {
                return formatAmountToLocalizedNumeralsWithCurrency(balance, balanceResult.currency);
            } else {
                return null;
            }
        } else {
            return null;
        }
    }

    function accountBalanceOrAvailableCredit(account: Account, currentSubAccountId: string | undefined, showAvailableCreditForCreditCard: boolean, showBalance: boolean, onlyShowSelectedAccountIds?: string[]): string | null {
        if (showAvailableCreditForCreditCard && account.category === AccountCategory.CreditCard.type) {
            return accountAvailableCredit(account, showBalance);
        } else {
            return accountBalance(account, currentSubAccountId, showBalance, onlyShowSelectedAccountIds);
        }
    }

    // valuations of all investment accounts keyed by account id, empty when amounts are hidden or the gold price is unavailable
    const investmentValuations = computed<Record<string, InvestmentValuationDisplay>>(() => {
        const valuations: Record<string, InvestmentValuationDisplay> = {};

        if (!showInvestmentValuation.value) {
            return valuations;
        }

        for (const accountId of Object.keys(investmentsStore.overviewByAccountId)) {
            const accountOverview = investmentsStore.overviewByAccountId[accountId];

            if (!accountOverview) {
                continue;
            }

            const diff: BigDecimal = parseInvestmentAmount(accountOverview.diff);
            const prefix = diff.isPositive() ? '+' : (diff.isNegative() ? '-' : '');
            const currency: string = accountOverview.currency || defaultCurrency.value;

            valuations[accountId] = {
                marketValue: formatAmountToLocalizedNumeralsWithCurrency(parseInvestmentAmount(accountOverview.marketValue), currency),
                diff: prefix + formatAmountToLocalizedNumeralsWithCurrency(diff.abs(), currency),
                color: diff.isPositive() ? INVESTMENT_VALUATION_UP_COLOR : (diff.isNegative() ? INVESTMENT_VALUATION_DOWN_COLOR : '')
            };
        }

        return valuations;
    });

    return {
        // states
        loading,
        showHidden,
        displayOrderModified,
        // computed states
        showAccountBalance,
        customAccountCategoryOrder,
        defaultAccountCategory,
        firstDayOfWeek,
        fiscalYearStart,
        defaultCurrency,
        useLastReconciledTime,
        allAccounts,
        allCategorizedAccountsMap,
        allAccountCount,
        maxCategoryAccountCount,
        netAssets,
        totalAssets,
        totalLiabilities,
        showInvestmentValuation,
        investmentMarketValue,
        investmentDiffText,
        investmentDiffColor,
        investmentValuations,
        // functions
        canShowAvailableCredit,
        accountCategoryTotalBalance,
        accountBalance,
        accountAvailableCredit,
        accountBalanceOrAvailableCredit
    };
}
