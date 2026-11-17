import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useInvestmentsStore } from '@/stores/investment.ts';

import type { BigDecimal, HiddenAmount, BigDecimalWithSuffix } from '@/core/numeral.ts';

import { Account } from '@/models/account.ts';

// investment valuation colors: up is green and down is red (explicitly required by the user)
export const INVESTMENT_VALUATION_UP_COLOR: string = 'rgb(var(--ebk-amount-color-green))';
export const INVESTMENT_VALUATION_DOWN_COLOR: string = 'rgb(var(--ebk-amount-color-red))';

export interface InvestmentValuationDisplay {
    readonly marketValue: string;
    readonly diff: string;
    readonly color: string;
}

export function useAssetSummaryWidgetBase(scene: 'overview' | 'accountList') {
    const { formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

    const settingsStore = useSettingsStore();
    const userStore = useUserStore();
    const accountsStore = useAccountsStore();
    const investmentsStore = useInvestmentsStore();

    // a failed request just hides the investment valuation line
    if (!investmentsStore.loading) {
        investmentsStore.loadOverview(false).catch(() => {});
    }

    const showAmountInHomePage = computed<boolean>({
        get: () => settingsStore.appSettings.showAmountInHomePage,
        set: (value) => settingsStore.setShowAmountInHomePage(value)
    });

    const showAccountBalance = computed<boolean>({
        get: () => settingsStore.appSettings.showAccountBalance,
        set: (value) => settingsStore.setShowAccountBalance(value)
    });

    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
    const allAccounts = computed<Account[]>(() => accountsStore.allAccounts);

    const netAssets = computed<string>(() => {
        if (scene === 'overview') {
            const netAssets: BigDecimal | HiddenAmount | BigDecimalWithSuffix = accountsStore.getNetAssets(showAmountInHomePage.value, settingsStore.appSettings.overviewAccountFilterInHomePage);
            return formatAmountToLocalizedNumeralsWithCurrency(netAssets, defaultCurrency.value);
        } else if (scene === 'accountList') {
            const netAssets: BigDecimal | HiddenAmount | BigDecimalWithSuffix = accountsStore.getNetAssets(showAccountBalance.value, settingsStore.appSettings.totalAmountExcludeAccountIds);
            return formatAmountToLocalizedNumeralsWithCurrency(netAssets, defaultCurrency.value);
        } else {
            return '';
        }
    });

    const totalAssets = computed<string>(() => {
        if (scene === 'overview') {
            const totalAssets: BigDecimal | HiddenAmount | BigDecimalWithSuffix = accountsStore.getTotalAssets(showAmountInHomePage.value, settingsStore.appSettings.overviewAccountFilterInHomePage);
            return formatAmountToLocalizedNumeralsWithCurrency(totalAssets, defaultCurrency.value);
        } else if (scene === 'accountList') {
            const totalAssets: BigDecimal | HiddenAmount | BigDecimalWithSuffix = accountsStore.getTotalAssets(showAccountBalance.value, settingsStore.appSettings.totalAmountExcludeAccountIds);
            return formatAmountToLocalizedNumeralsWithCurrency(totalAssets, defaultCurrency.value);
        } else {
            return '';
        }
    });

    const totalLiabilities = computed<string>(() => {
        if (scene === 'overview') {
            const totalLiabilities: BigDecimal | HiddenAmount | BigDecimalWithSuffix = accountsStore.getTotalLiabilities(showAmountInHomePage.value, settingsStore.appSettings.overviewAccountFilterInHomePage);
            return formatAmountToLocalizedNumeralsWithCurrency(totalLiabilities, defaultCurrency.value);
        } else if (scene === 'accountList') {
            const totalLiabilities: BigDecimal | HiddenAmount | BigDecimalWithSuffix = accountsStore.getTotalLiabilities(showAccountBalance.value, settingsStore.appSettings.totalAmountExcludeAccountIds);
            return formatAmountToLocalizedNumeralsWithCurrency(totalLiabilities, defaultCurrency.value);
        } else {
            return '';
        }
    });

    // the investment valuation is an additional line and is hidden when amounts are hidden, there is no investment account or the gold price is unavailable
    const showInvestmentValuation = computed<boolean>(() => {
        const showAmount = scene === 'overview' ? showAmountInHomePage.value : showAccountBalance.value;
        const overview = investmentsStore.overview;
        return showAmount && !!overview && overview.priceAvailable && overview.accounts.length > 0;
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

    return {
        // computed states
        showAmountInHomePage,
        showAccountBalance,
        defaultCurrency,
        allAccounts,
        netAssets,
        totalAssets,
        totalLiabilities,
        showInvestmentValuation,
        investmentMarketValue,
        investmentDiffText,
        investmentDiffColor
    };
}
