import type { BigDecimal } from "@/core/numeral.ts";

import { parseBigDecimal } from "@/lib/numeral.ts";

export enum InvestmentTarget {
    None = 0,
    Gold = 1,
}

export const INVESTMENT_UNITS_SCALE: number = 10000; // 0.0001g (the unit of server-side investment units)

// the investment api returns decimal yuan strings, while amount values are cents in the app
export function parseInvestmentAmount(value: string): BigDecimal {
    return parseBigDecimal(value).multiply(100);
}

export interface InvestmentRecord {
    readonly transactionId: string;
    readonly time: number;
    readonly direction: string; // buy / sell
    readonly amount: string;
    readonly units: string;
    readonly fee: string;
    readonly unitPrice: string;
    readonly comment: string;
}

export interface PendingInvestmentTransfer {
    readonly transactionId: string;
    readonly time: number;
    readonly direction: string; // buy / sell
    readonly amount: string;
    readonly comment: string;
}

export interface InvestmentAccountOverview {
    readonly id: string;
    readonly target: number;
    readonly targetName: string;
    readonly accountId: string;
    readonly accountName: string;
    readonly currency: string;
    readonly units: string;
    readonly initialUnits: string;
    readonly initialCost: string;
    readonly avgCost: string;
    readonly cost: string;
    readonly marketValue: string;
    readonly floatProfit: string;
    readonly balance: string;
    readonly diff: string;
    readonly diffPercent: string;
    readonly realizedProfit: string;
    readonly records: InvestmentRecord[];
}

export interface InvestmentOverview {
    readonly accounts: InvestmentAccountOverview[];
    readonly totalMarketValue: string;
    readonly totalCost: string;
    readonly totalFloatProfit: string;
    readonly price: string;
    readonly priceFetchedAt: number;
    readonly priceSource: string;
    readonly priceAvailable: boolean;
    readonly pendingTransfers: PendingInvestmentTransfer[];
}

export interface InvestmentAccount {
    readonly id: string;
    readonly target: number;
    readonly targetName: string;
    readonly accountId: string;
    readonly initialUnits: string;
    readonly initialCost: string;
}

export interface InvestmentAccountsResponse {
    readonly accounts: InvestmentAccount[];
}

export interface InvestmentAccountCreateRequest {
    readonly target: number;
    readonly accountId: string;
    readonly initialGrams: string;
    readonly initialCost: string;
}

export interface InvestmentAccountModifyRequest {
    readonly id: string;
    readonly initialGrams: string;
    readonly initialCost: string;
}

export interface InvestmentAccountDeleteRequest {
    readonly id: string;
}
