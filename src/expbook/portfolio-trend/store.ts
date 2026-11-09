// Portfolio trend store (Expbook).
//
// Two things are charted together over time, all in the display currency:
//   - the balance of every account (from the asset trends the Home page already uses), and
//   - the investment portfolio (value and cost) from /investments/history.json.
// The server sends raw numbers; conversion to the display currency happens here so that the chart follows
// the display currency, the rates and the "hide account from overview" setting without another request.

import { ref, computed } from 'vue';
import { defineStore } from 'pinia';
import axios from 'axios';

import { useSettingsStore } from '@/stores/setting.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import type { TransactionStatisticAssetTrendsResponseItem } from '@/models/transaction.ts';

import { BIG_DECIMAL_ZERO, parseBigDecimal } from '@/lib/numeral.ts';
import logger from '@/lib/logger.ts';

import { useDisplayCurrencyStore } from '../display-currency/store.ts';

import {
    type TrendRange,
    type TrendRow,
    type AccountDay,
    type HistoryPointDto,
    rangeSpec,
    stepDaysForSpan,
    buildRows,
    trimEmptyStart,
    gridPoints,
    endOfDay
} from './lib.ts';

export interface InvestmentHistoryResponse {
    readonly baseCurrency: string;
    readonly ratesAvailable: boolean;
    readonly unconvertedCurrencies: string[];
    readonly firstTradeTime: number;
    readonly points: HistoryPointDto[];
}

interface ApiResult<T> {
    readonly success: boolean;
    readonly result?: T;
}

interface RawDay {
    readonly time: number;
    readonly closing: Record<string, string>;
}

const DAY = 86400;

export const usePortfolioTrendStore = defineStore('expbookPortfolioTrend', () => {
    const settingsStore = useSettingsStore();
    const accountsStore = useAccountsStore();
    const exchangeRatesStore = useExchangeRatesStore();
    const displayCurrencyStore = useDisplayCurrencyStore();

    const range = ref<TrendRange>('1Y');
    const loading = ref<boolean>(false);
    const failed = ref<boolean>(false);
    const loaded = ref<boolean>(false);

    const rawDays = ref<RawDay[]>([]);
    const history = ref<InvestmentHistoryResponse | null>(null);
    const windowStart = ref<number>(0);

    const currency = computed<string>(() => displayCurrencyStore.currency);

    /** Sum of all shown accounts at the close of each day that had activity, in the display currency. */
    const accountDays = computed<AccountDay[]>(() => {
        const target = currency.value;
        const hidden = settingsStore.appSettings.overviewAccountFilterInHomePage;
        const last: Record<string, string> = {};
        const days: AccountDay[] = [];

        for (const day of rawDays.value) {
            Object.assign(last, day.closing);

            let total = BIG_DECIMAL_ZERO;

            for (const accountId of Object.keys(last)) {
                const account = accountsStore.allAccountsMap[accountId];

                if (!account || hidden[account.id]) {
                    continue;
                }

                let balance = parseBigDecimal(last[accountId]!);

                if (account.currency !== target) {
                    const exchanged = exchangeRatesStore.getExchangedAmount(balance, account.currency, target);

                    if (!exchanged) {
                        continue;
                    }

                    balance = exchanged.truncate();
                }

                if (account.isAsset || account.isLiability) {
                    total = total.add(balance);
                }
            }

            days.push({ time: day.time, total: Number(total.toString()) });
        }

        return days;
    });

    function convertInvestments(minor: number): number | null {
        const base = history.value ? history.value.baseCurrency : '';
        const target = currency.value;

        if (!base || base === target) {
            return minor;
        }

        const exchanged = exchangeRatesStore.getExchangedAmount(parseBigDecimal(minor), base, target);

        return exchanged ? Math.round(Number(exchanged.toString())) : null;
    }

    const built = computed(() => {
        const points = history.value ? history.value.points : [];
        const result = buildRows(points, accountDays.value, convertInvestments);

        return { rows: trimEmptyStart(result.rows), missingRate: result.missingRate };
    });

    const rows = computed<TrendRow[]>(() => built.value.rows);

    /** True when investments could not be converted to the display currency, so they are left out. */
    const investmentsMissingRate = computed<boolean>(() => built.value.missingRate);

    const hasInvestments = computed<boolean>(() => rows.value.some(row => row.invested !== 0 || row.investments !== 0));

    const hasEstimates = computed<boolean>(() => rows.value.some(row => row.estimated && row.invested !== 0));

    function unwrap<T>(data: ApiResult<T> | undefined): T | undefined {
        return data && data.success ? data.result : undefined;
    }

    async function load(): Promise<void> {
        loading.value = true;
        failed.value = false;

        try {
            await accountsStore.loadAllAccounts({ force: false });
            displayCurrencyStore.ensureRates();

            const assets = await axios.get<ApiResult<TransactionStatisticAssetTrendsResponseItem[]>>('v1/transactions/statistics/asset_trends.json');
            const items = unwrap(assets.data);

            if (!items) {
                throw new Error('asset trends are not available');
            }

            const days: RawDay[] = items.map(item => {
                const closing: Record<string, string> = {};

                for (const entry of item.items) {
                    closing[entry.accountId] = entry.accountClosingBalance;
                }

                return { time: endOfDay(item.year, item.month, item.day), closing };
            });

            days.sort((a, b) => a.time - b.time);

            const now = Math.floor(Date.now() / 1000);
            const spec = rangeSpec(range.value, now);
            const earliest = days.length ? days[0]!.time - DAY : 0;
            const start = spec.start !== null ? spec.start : earliest;
            const stepDays = spec.start !== null || start <= 0 ? spec.stepDays : stepDaysForSpan(start, now, spec.stepDays);

            const response = await axios.get<ApiResult<InvestmentHistoryResponse>>('v1/investments/history.json', {
                params: { startTime: Math.max(0, Math.floor(start)), stepDays }
            });
            const result = unwrap(response.data);

            if (!result) {
                throw new Error('investment history is not available');
            }

            // No trades yet: still chart the accounts on an even grid
            const points = result.points.length ? result.points : gridPoints(Math.max(start, 0) || now - 30 * DAY, now, stepDays * DAY);

            rawDays.value = days;
            windowStart.value = start;
            history.value = { ...result, points };
            loaded.value = true;
        } catch (error) {
            logger.error('failed to load portfolio trend', error);
            failed.value = true;
        } finally {
            loading.value = false;
        }
    }

    async function setRange(next: TrendRange): Promise<void> {
        if (next === range.value && loaded.value) {
            return;
        }

        range.value = next;
        await load();
    }

    return {
        range,
        loading,
        failed,
        loaded,
        currency,
        rows,
        hasInvestments,
        hasEstimates,
        investmentsMissingRate,
        load,
        setRange
    };
});
