// Expbook: investment tracking models. Quantities and prices travel as decimal strings,
// money as integer minor units (same factor as the rest of the app).

export type SecurityTypeValue = 1 | 2 | 3 | 4 | 5;
export type PriceSourceValue = 'manual' | 'yahoo' | 'amfi';
export type TradeTypeValue = 1 | 2 | 3;

export const SECURITY_TYPES: { value: SecurityTypeValue, name: string }[] = [
    { value: 1, name: 'Stock' },
    { value: 2, name: 'ETF' },
    { value: 3, name: 'Mutual Fund' },
    { value: 4, name: 'Crypto' },
    { value: 5, name: 'Other' }
];

export const PRICE_SOURCES: { value: PriceSourceValue, name: string }[] = [
    { value: 'manual', name: 'Manual' },
    { value: 'yahoo', name: 'Yahoo Finance' },
    { value: 'amfi', name: 'AMFI (India mutual funds)' }
];

export const TRADE_TYPES: { value: TradeTypeValue, name: string }[] = [
    { value: 1, name: 'Buy' },
    { value: 2, name: 'Sell' },
    { value: 3, name: 'Dividend' }
];

export const ALLOCATION_TYPE_NAMES: Record<string, string> = {
    stock: 'Stock',
    etf: 'ETF',
    mutual_fund: 'Mutual Fund',
    crypto: 'Crypto',
    other: 'Other'
};

export interface SecurityInfoResponse {
    readonly id: string;
    readonly name: string;
    readonly symbol: string;
    readonly securityType: number;
    readonly currency: string;
    readonly priceSource: string;
    readonly priceSymbol: string;
    readonly latestPrice: string;
    readonly latestPriceTime: number;
    readonly comment: string;
}

export interface SecurityEditRequest {
    readonly id?: string;
    readonly name: string;
    readonly symbol: string;
    readonly securityType: number;
    readonly currency: string;
    readonly priceSource: string;
    readonly priceSymbol: string;
    readonly comment: string;
}

export interface SecuritySetPriceRequest {
    readonly id: string;
    readonly price: string;
    readonly time?: number;
}

export interface InvestmentTradeInfoResponse {
    readonly id: string;
    readonly securityId: string;
    readonly tradeType: number;
    readonly time: number;
    readonly quantity: string;
    readonly price: string;
    readonly amount: number;
    readonly fees: number;
    readonly accountId: string;
    readonly source: string;
    readonly externalId: string;
    readonly comment: string;
}

export interface InvestmentTradeEditRequest {
    readonly id?: string;
    readonly securityId: string;
    readonly tradeType: number;
    readonly time: number;
    readonly quantity: string;
    readonly price: string;
    readonly amount: number;
    readonly fees: number;
    readonly accountId: string;
    readonly comment: string;
}

export interface InvestmentImportRowResponse {
    readonly line: number;
    readonly symbol: string;
    readonly status: 'valid' | 'duplicate' | 'error';
    readonly message: string;
}

export interface InvestmentImportResponse {
    readonly dryRun: boolean;
    readonly rows: InvestmentImportRowResponse[];
    readonly created: number;
    readonly skipped: number;
    readonly failed: number;
    readonly createdSecurities: number;
}

export interface InvestmentHolding {
    readonly securityId: string;
    readonly name: string;
    readonly symbol: string;
    readonly securityType: number;
    readonly currency: string;
    readonly quantity: string;
    readonly averageCost: string;
    readonly totalCost: number;
    readonly price: string;
    readonly priceTime: number;
    readonly priceAvailable: boolean;
    readonly value: number;
    readonly unrealised: number;
    readonly unrealisedPercent: number | null;
    readonly realised: number;
    readonly dividends: number;
    readonly xirr: number | null;
    readonly tradeCount: number;
    readonly warning: '' | 'no_price' | 'no_exchange_rate';
}

export interface InvestmentTotals {
    readonly invested: number;
    readonly pricedCost: number;
    readonly value: number;
    readonly unrealised: number;
    readonly realised: number;
    readonly dividends: number;
    readonly xirr: number | null;
}

export interface InvestmentAllocation {
    readonly key: string;
    readonly value: number;
    readonly percent: number;
}

export interface InvestmentSummaryResponse {
    readonly baseCurrency: string;
    readonly ratesAvailable: boolean;
    readonly holdings: InvestmentHolding[];
    readonly totals: InvestmentTotals;
    readonly allocationByType: InvestmentAllocation[];
    readonly allocationByCurrency: InvestmentAllocation[];
    readonly unconvertedCurrencies: string[];
}

export interface SecurityPriceRefreshResult {
    readonly securityId: string;
    readonly symbol: string;
    readonly ok: boolean;
    readonly price: string;
    readonly priceTime: number;
    readonly message: string;
}

export interface SecurityPriceRefreshResponse {
    readonly results: SecurityPriceRefreshResult[];
}
