import { ref, computed, watch } from "vue";

import { useI18n } from "@/locales/helpers.ts";

import { useUserStore } from "@/stores/user.ts";
import { useAccountsStore } from "@/stores/account.ts";
import { useInvestmentsStore } from "@/stores/investment.ts";

import {
    InvestmentTarget,
    parseInvestmentAmount,
} from "@/models/investment.ts";

import { parseBigDecimal } from "@/lib/numeral.ts";
import { parseDateTimeFromUnixTime } from "@/lib/datetime.ts";

export const INVESTMENT_MAX_RECORD_COUNT = 50;

export interface InvestmentHoldingRow {
    readonly id: string;
    readonly targetName: string;
    readonly accountName: string;
    readonly units: string;
    readonly avgCost: string;
    readonly cost: string;
    readonly marketValue: string;
    readonly floatProfit: string;
    readonly floatProfitClass: string;
    readonly balance: string;
    readonly diff: string;
    readonly diffPercent: string;
    readonly diffClass: string;
}

export interface InvestmentRecordRow {
    readonly key: string;
    readonly time: number;
    readonly timeText: string;
    readonly directionText: string;
    readonly amount: string;
    readonly units: string;
    readonly unitPrice: string;
    readonly fee: string;
    readonly comment: string;
}

export interface InvestmentPendingRow {
    readonly transactionId: string;
    readonly timeText: string;
    readonly directionText: string;
    readonly amount: string;
    readonly comment: string;
}

export interface InvestmentConfigurationRow {
    readonly id: string;
    readonly accountName: string;
    readonly initialUnits: string;
    readonly initialCost: string;
    readonly initialGrams: string;
    readonly initialCostValue: string;
    readonly hasInitialUnits: boolean;
}

export interface InvestmentAccountOption {
    readonly id: string;
    readonly name: string;
}

export function useInvestmentPageBase() {
    const {
        formatAmountToLocalizedNumeralsWithCurrency,
        formatBigDecimalToLocalizedNumerals,
        formatDateTimeToLongDateTime,
    } = useI18n();

    const userStore = useUserStore();
    const accountsStore = useAccountsStore();
    const investmentsStore = useInvestmentsStore();

    const submitting = ref<boolean>(false);
    const formError = ref<string | null>(null);

    const addAccountId = ref<string>("");
    const addInitialGrams = ref<string>("");
    const addInitialCost = ref<string>("");

    const editingId = ref<string>("");
    const editingGrams = ref<string>("");
    const editingCost = ref<string>("");

    const defaultCurrency = computed<string>(
        () => userStore.currentUserDefaultCurrency,
    );
    const overview = computed(() => investmentsStore.overview);
    const loading = computed<boolean>(() => investmentsStore.loading);
    const lastError = computed<string | null>(() => investmentsStore.lastError);
    const hasAccounts = computed<boolean>(
        () => !!overview.value && overview.value.accounts.length > 0,
    );
    const priceAvailable = computed<boolean>(
        () => !!overview.value && overview.value.priceAvailable,
    );

    const priceText = computed<string>(() => {
        if (!overview.value || !hasAccounts.value) {
            return "—"; // 没有投资账户时不取价
        } else if (!priceAvailable.value) {
            return "金价获取失败";
        }

        return (
            formatAmount(overview.value.price, defaultCurrency.value) + "/克"
        );
    });

    const priceFetchedAtText = computed<string>(() => {
        if (!overview.value || !overview.value.priceFetchedAt) {
            return "";
        }

        return formatDateTimeToLongDateTime(
            parseDateTimeFromUnixTime(overview.value.priceFetchedAt),
        );
    });

    const priceSourceText = computed<string>(() =>
        overview.value ? overview.value.priceSource : "",
    );

    const totalMarketValueText = computed<string>(() => {
        if (!priceAvailable.value || !overview.value) {
            return "—";
        }

        return formatAmount(
            overview.value.totalMarketValue,
            defaultCurrency.value,
        );
    });

    const totalCostText = computed<string>(() =>
        overview.value
            ? formatAmount(overview.value.totalCost, defaultCurrency.value)
            : "—",
    );

    const totalFloatProfitText = computed<string>(() => {
        if (!priceAvailable.value || !overview.value) {
            return "—";
        }

        return formatProfit(
            overview.value.totalFloatProfit,
            defaultCurrency.value,
        );
    });

    const totalFloatProfitClass = computed<string>(() =>
        overview.value ? profitClass(overview.value.totalFloatProfit) : "",
    );

    const holdings = computed<InvestmentHoldingRow[]>(() => {
        if (!overview.value) {
            return [];
        }

        return overview.value.accounts.map((account) => {
            const currency = account.currency || defaultCurrency.value;

            return {
                id: account.id,
                targetName:
                    account.target === InvestmentTarget.Gold
                        ? "黄金"
                        : account.targetName,
                accountName: account.accountName,
                units: formatUnits(account.units),
                avgCost: formatAmount(account.avgCost, currency),
                cost: formatAmount(account.cost, currency),
                marketValue: priceAvailable.value
                    ? formatAmount(account.marketValue, currency)
                    : "—",
                floatProfit: priceAvailable.value
                    ? formatProfit(account.floatProfit, currency)
                    : "—",
                floatProfitClass: priceAvailable.value
                    ? profitClass(account.floatProfit)
                    : "",
                balance: formatAmount(account.balance, currency),
                diff: priceAvailable.value
                    ? formatProfit(account.diff, currency)
                    : "—",
                diffPercent: priceAvailable.value
                    ? account.diffPercent + "%"
                    : "—",
                diffClass: priceAvailable.value
                    ? profitClass(account.diff)
                    : "",
            };
        });
    });

    const records = computed<InvestmentRecordRow[]>(() => {
        const allRecords: InvestmentRecordRow[] = [];

        if (!overview.value) {
            return allRecords;
        }

        for (const account of overview.value.accounts) {
            const currency = account.currency || defaultCurrency.value;

            for (const record of account.records) {
                allRecords.push({
                    key: account.accountId + "_" + record.transactionId,
                    time: record.time,
                    timeText: formatDateTimeToLongDateTime(
                        parseDateTimeFromUnixTime(record.time),
                    ),
                    directionText:
                        record.direction === "sell" ? "卖出" : "买入",
                    amount: formatAmount(record.amount, currency),
                    units: formatUnits(record.units),
                    unitPrice: formatAmount(record.unitPrice, currency),
                    fee: formatAmount(record.fee, currency),
                    comment: record.comment,
                });
            }
        }

        allRecords.sort((a, b) => b.time - a.time);

        return allRecords.slice(0, INVESTMENT_MAX_RECORD_COUNT);
    });

    const pendingTransfers = computed<InvestmentPendingRow[]>(() => {
        if (!overview.value) {
            return [];
        }

        return overview.value.pendingTransfers.map((pendingTransfer) => {
            return {
                transactionId: pendingTransfer.transactionId,
                timeText: formatDateTimeToLongDateTime(
                    parseDateTimeFromUnixTime(pendingTransfer.time),
                ),
                directionText:
                    pendingTransfer.direction === "sell" ? "卖出" : "买入",
                amount: formatAmount(pendingTransfer.amount, "CNY"), // 投资账户仅支持 CNY
                comment: pendingTransfer.comment,
            };
        });
    });

    const configurations = computed<InvestmentConfigurationRow[]>(() => {
        return investmentsStore.accounts.map((account) => {
            const accountModel =
                accountsStore.allAccountsMap[account.accountId];

            return {
                id: account.id,
                accountName: accountModel
                    ? accountModel.name
                    : account.accountId,
                initialUnits: formatUnits(account.initialUnits),
                initialCost: formatAmount(
                    account.initialCost,
                    accountModel
                        ? accountModel.currency
                        : defaultCurrency.value,
                ),
                initialGrams: account.initialUnits,
                initialCostValue: account.initialCost,
                hasInitialUnits: parseBigDecimal(
                    account.initialUnits,
                ).greaterThan(0),
            };
        });
    });

    const availableAccountOptions = computed<InvestmentAccountOption[]>(() => {
        const configuredAccountIds: Record<string, boolean> = {};

        for (const account of investmentsStore.accounts) {
            configuredAccountIds[account.accountId] = true;
        }

        return accountsStore.allPlainAccounts
            .filter((account) => !configuredAccountIds[account.id])
            .map((account) => {
                return {
                    id: account.id,
                    name: account.name,
                };
            });
    });

    watch(
        availableAccountOptions,
        (options) => {
            if (!options.some((option) => option.id === addAccountId.value)) {
                addAccountId.value = options.length > 0 ? options[0]!.id : "";
            }
        },
        { immediate: true },
    );

    function formatAmount(value: string, currency: string): string {
        return formatAmountToLocalizedNumeralsWithCurrency(
            parseInvestmentAmount(value),
            currency,
        );
    }

    function formatUnits(value: string): string {
        return (
            formatBigDecimalToLocalizedNumerals(parseBigDecimal(value), 4) +
            " 克"
        );
    }

    function formatProfit(value: string, currency: string): string {
        const amount = parseInvestmentAmount(value);
        const text = formatAmountToLocalizedNumeralsWithCurrency(
            amount,
            currency,
        );

        return amount.greaterThan(0) ? "+" + text : text;
    }

    function profitClass(value: string): string {
        const amount = parseBigDecimal(value);

        if (amount.greaterThan(0)) {
            return "investment-profit-positive";
        } else if (amount.lessThan(0)) {
            return "investment-profit-negative";
        } else {
            return "";
        }
    }

    function errorMessage(error: {
        error?: { errorMessage?: string };
        message?: string;
    }): string {
        if (error && error.error && error.error.errorMessage) {
            return error.error.errorMessage;
        } else if (error && error.message) {
            return error.message;
        } else {
            return "操作失败";
        }
    }

    function load(): void {
        accountsStore.loadAllAccounts({ force: false }).catch(() => {});
        investmentsStore.loadAccounts().catch(() => {});
        investmentsStore.loadOverview(false).catch(() => {});
    }

    function refresh(): void {
        investmentsStore.loadOverview(true).catch(() => {});
    }

    // 配置变更后同时刷新配置列表与概览
    function refreshAfterChange(): void {
        investmentsStore.loadAccounts().catch(() => {});
        refresh();
    }

    function submitAdd(): void {
        if (!addAccountId.value) {
            formError.value = "请选择投资账户";
            return;
        }

        submitting.value = true;
        formError.value = null;

        investmentsStore
            .addAccount({
                target: InvestmentTarget.Gold,
                accountId: addAccountId.value,
                initialGrams: addInitialGrams.value.trim() || "0",
                initialCost: addInitialCost.value.trim() || "0",
            })
            .then(() => {
                submitting.value = false;
                addInitialGrams.value = "";
                addInitialCost.value = "";
                refreshAfterChange();
            })
            .catch((error) => {
                submitting.value = false;
                formError.value = errorMessage(error);
            });
    }

    function startEdit(configuration: InvestmentConfigurationRow): void {
        editingId.value = configuration.id;
        editingGrams.value = configuration.initialGrams;
        editingCost.value = configuration.initialCostValue;
        formError.value = null;
    }

    function cancelEdit(): void {
        editingId.value = "";
        editingGrams.value = "";
        editingCost.value = "";
        formError.value = null;
    }

    function submitEdit(): void {
        if (!editingId.value) {
            return;
        }

        submitting.value = true;
        formError.value = null;

        investmentsStore
            .modifyAccount({
                id: editingId.value,
                initialGrams: editingGrams.value.trim() || "0",
                initialCost: editingCost.value.trim() || "0",
            })
            .then(() => {
                submitting.value = false;
                cancelEdit();
                refreshAfterChange();
            })
            .catch((error) => {
                submitting.value = false;
                formError.value = errorMessage(error);
            });
    }

    function remove(configuration: InvestmentConfigurationRow): void {
        if (
            !window.confirm(
                "确定删除该投资账户配置吗？只会删除配置，不会删除账户和交易。",
            )
        ) {
            return;
        }

        submitting.value = true;
        formError.value = null;

        investmentsStore
            .deleteAccount(configuration.id)
            .then(() => {
                submitting.value = false;

                if (editingId.value === configuration.id) {
                    cancelEdit();
                }

                refreshAfterChange();
            })
            .catch((error) => {
                submitting.value = false;
                formError.value = errorMessage(error);
            });
    }

    return {
        // states
        submitting,
        formError,
        addAccountId,
        addInitialGrams,
        addInitialCost,
        editingId,
        editingGrams,
        editingCost,
        // computed states
        overview,
        loading,
        lastError,
        hasAccounts,
        priceAvailable,
        priceText,
        priceFetchedAtText,
        priceSourceText,
        totalMarketValueText,
        totalCostText,
        totalFloatProfitText,
        totalFloatProfitClass,
        holdings,
        records,
        pendingTransfers,
        configurations,
        availableAccountOptions,
        // functions
        load,
        refresh,
        submitAdd,
        startEdit,
        cancelEdit,
        submitEdit,
        remove,
    };
}
