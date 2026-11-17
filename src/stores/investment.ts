import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

import type { BigDecimal } from '@/core/numeral.ts';

import { parseInvestmentAmount } from '@/models/investment.ts';

import type {
    InvestmentOverview,
    InvestmentAccount,
    InvestmentAccountOverview,
    InvestmentAccountCreateRequest,
    InvestmentAccountModifyRequest
} from '@/models/investment.ts';

import { BIG_DECIMAL_ZERO } from '@/lib/numeral.ts';
import services from '@/lib/services.ts';
import logger from '@/lib/logger.ts';

interface InvestmentApiError {
    readonly response?: {
        readonly data?: {
            readonly errorMessage?: string;
        };
    };
    readonly processed?: boolean;
}

export const useInvestmentsStore = defineStore('investments', () => {
    const overview = ref<InvestmentOverview | null>(null);
    const accounts = ref<InvestmentAccount[]>([]);
    const loading = ref<boolean>(false);
    const lastError = ref<string | null>(null);

    const investmentAccounts = computed<InvestmentAccount[]>(() => accounts.value);

    const overviewByAccountId = computed<Record<string, InvestmentAccountOverview>>(() => {
        const overviewMap: Record<string, InvestmentAccountOverview> = {};

        if (overview.value) {
            for (const account of overview.value.accounts) {
                overviewMap[account.accountId] = account;
            }
        }

        return overviewMap;
    });

    const totalDiff = computed<BigDecimal>(() => {
        if (!overview.value) {
            return BIG_DECIMAL_ZERO;
        }

        let totalDiffAmount: BigDecimal = BIG_DECIMAL_ZERO;

        for (const account of overview.value.accounts) {
            totalDiffAmount = totalDiffAmount.add(parseInvestmentAmount(account.diff));
        }

        return totalDiffAmount;
    });

    const totalMarketValue = computed<BigDecimal>(() => {
        if (!overview.value) {
            return BIG_DECIMAL_ZERO;
        }

        return parseInvestmentAmount(overview.value.totalMarketValue);
    });

    function handleError(error: InvestmentApiError, message: string, reject: (reason?: unknown) => void): void {
        if (error.response && error.response.data && error.response.data.errorMessage) {
            lastError.value = error.response.data.errorMessage;
            reject({ error: error.response.data });
        } else if (!error.processed) {
            lastError.value = message;
            reject({ message: message });
        } else {
            reject(error);
        }
    }

    function loadOverview(force = false): Promise<InvestmentOverview> {
        loading.value = true;

        return new Promise((resolve, reject) => {
            services.getInvestmentOverview(force).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    lastError.value = 'Unable to retrieve investment overview';
                    reject({ message: lastError.value });
                    return;
                }

                lastError.value = null;
                overview.value = data.result;
                resolve(data.result);
            }).catch((error: InvestmentApiError) => {
                logger.error('failed to load investment overview', error);
                handleError(error, 'Unable to retrieve investment overview', reject);
            }).finally(() => {
                loading.value = false;
            });
        });
    }

    function loadAccounts(): Promise<InvestmentAccount[]> {
        return new Promise((resolve, reject) => {
            services.getInvestmentAccounts().then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    lastError.value = 'Unable to retrieve investment accounts';
                    reject({ message: lastError.value });
                    return;
                }

                lastError.value = null;
                accounts.value = data.result.accounts;
                resolve(data.result.accounts);
            }).catch((error: InvestmentApiError) => {
                logger.error('failed to load investment accounts', error);
                handleError(error, 'Unable to retrieve investment accounts', reject);
            });
        });
    }

    function addAccount(req: InvestmentAccountCreateRequest): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.addInvestmentAccount(req).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to add investment account' });
                    return;
                }

                loadAccounts().catch(() => {});
                resolve(data.result);
            }).catch((error: InvestmentApiError) => {
                logger.error('failed to add investment account', error);
                handleError(error, 'Unable to add investment account', reject);
            });
        });
    }

    function modifyAccount(req: InvestmentAccountModifyRequest): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.modifyInvestmentAccount(req).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to modify investment account' });
                    return;
                }

                loadAccounts().catch(() => {});
                resolve(data.result);
            }).catch((error: InvestmentApiError) => {
                logger.error('failed to modify investment account', error);
                handleError(error, 'Unable to modify investment account', reject);
            });
        });
    }

    function deleteAccount(id: string): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.deleteInvestmentAccount({
                id: id
            }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to delete investment account' });
                    return;
                }

                loadAccounts().catch(() => {});
                resolve(data.result);
            }).catch((error: InvestmentApiError) => {
                logger.error('failed to delete investment account', error);
                handleError(error, 'Unable to delete investment account', reject);
            });
        });
    }

    return {
        // states
        overview,
        accounts,
        loading,
        lastError,
        // computed states
        investmentAccounts,
        overviewByAccountId,
        totalDiff,
        totalMarketValue,
        // functions
        loadOverview,
        loadAccounts,
        addAccount,
        modifyAccount,
        deleteAccount
    };
});
