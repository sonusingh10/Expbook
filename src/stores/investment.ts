// Expbook: investments store. Server is the source of truth (all maths is done by the Go ledger),
// so the store only holds the last responses and re-fetches after every write.
import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

import type {
    SecurityInfoResponse,
    SecurityEditRequest,
    InvestmentTradeInfoResponse,
    InvestmentTradeEditRequest,
    InvestmentImportResponse,
    InvestmentSummaryResponse,
    SecurityPriceRefreshResult
} from '@/models/investment.ts';

import logger from '@/lib/logger.ts';
import services, { type ApiResponsePromise } from '@/lib/services.ts';

interface StoreError {
    message?: string;
    error?: unknown;
    processed?: boolean;
}

function toStoreError(error: { response?: { data?: { errorMessage?: string } }, processed?: boolean }, fallbackMessage: string): StoreError {
    if (error.response && error.response.data && error.response.data.errorMessage) {
        return { error: error.response.data };
    } else if (!error.processed) {
        return { message: fallbackMessage };
    }

    return error as StoreError;
}

function call<T>(request: ApiResponsePromise<T>, failureMessage: string, logName: string, allowFalsyResult: boolean = false): Promise<T> {
    return new Promise((resolve, reject) => {
        request.then(response => {
            const data = response.data;

            if (!data || !data.success || (!allowFalsyResult && !data.result)) {
                reject({ message: failureMessage });
                return;
            }

            resolve(data.result);
        }).catch(error => {
            logger.error(logName, error);
            reject(toStoreError(error, failureMessage));
        });
    });
}

export const useInvestmentStore = defineStore('investments', () => {
    const securities = ref<SecurityInfoResponse[]>([]);
    const summary = ref<InvestmentSummaryResponse | null>(null);
    const trades = ref<InvestmentTradeInfoResponse[]>([]);

    const securityMap = computed<Record<string, SecurityInfoResponse>>(() => {
        const map: Record<string, SecurityInfoResponse> = {};

        for (const security of securities.value) {
            map[security.id] = security;
        }

        return map;
    });

    function loadSecurities(): Promise<SecurityInfoResponse[]> {
        return call(services.getAllSecurities(), 'Unable to retrieve security list', 'failed to load securities').then(result => {
            securities.value = result;
            return result;
        });
    }

    function loadSummary(): Promise<InvestmentSummaryResponse> {
        return call(services.getInvestmentSummary(), 'Unable to retrieve portfolio summary', 'failed to load investment summary').then(result => {
            summary.value = result;
            return result;
        });
    }

    function loadTrades(securityId?: string): Promise<InvestmentTradeInfoResponse[]> {
        return call(services.getInvestmentTrades(securityId), 'Unable to retrieve trade list', 'failed to load trades', true).then(result => {
            trades.value = result || [];
            return trades.value;
        });
    }

    function reloadAll(): Promise<void> {
        return Promise.all([loadSecurities(), loadSummary(), loadTrades()]).then(() => undefined);
    }

    function saveSecurity(req: SecurityEditRequest): Promise<SecurityInfoResponse> {
        const request = req.id ? services.modifySecurity(req) : services.addSecurity(req);

        return call(request, 'Unable to save this security', 'failed to save security').then(result => {
            return reloadAll().then(() => result);
        });
    }

    function deleteSecurity(id: string): Promise<boolean> {
        return call(services.deleteSecurity({ id }), 'Unable to delete this security', 'failed to delete security').then(result => {
            return reloadAll().then(() => result);
        });
    }

    function setSecurityPrice(id: string, price: string): Promise<SecurityInfoResponse> {
        return call(services.setSecurityPrice({ id, price }), 'Unable to save this price', 'failed to set security price').then(result => {
            return reloadAll().then(() => result);
        });
    }

    function saveTrade(req: InvestmentTradeEditRequest): Promise<InvestmentTradeInfoResponse> {
        const request = req.id ? services.modifyInvestmentTrade(req) : services.addInvestmentTrade(req);

        return call(request, 'Unable to save this trade', 'failed to save trade').then(result => {
            return reloadAll().then(() => result);
        });
    }

    function deleteTrade(id: string): Promise<boolean> {
        return call(services.deleteInvestmentTrade({ id }), 'Unable to delete this trade', 'failed to delete trade').then(result => {
            return reloadAll().then(() => result);
        });
    }

    function importTrades(csv: string, dryRun: boolean): Promise<InvestmentImportResponse> {
        return call(services.importInvestmentTrades({ csv, dryRun }), 'Unable to import trades', 'failed to import trades').then(result => {
            if (dryRun) {
                return result;
            }

            return reloadAll().then(() => result);
        });
    }

    function refreshPrices(): Promise<SecurityPriceRefreshResult[]> {
        return call(services.refreshSecurityPrices(), 'Unable to refresh prices', 'failed to refresh prices').then(result => {
            return reloadAll().then(() => result.results || []);
        });
    }

    return {
        securities,
        summary,
        trades,
        securityMap,
        loadSecurities,
        loadSummary,
        loadTrades,
        reloadAll,
        saveSecurity,
        deleteSecurity,
        setSecurityPrice,
        saveTrade,
        deleteTrade,
        importTrades,
        refreshPrices
    };
});
