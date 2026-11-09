// Investment totals are calculated by the server in the profile currency (the summary "baseCurrency").
// These helpers show them in the display currency instead (Expbook).

import { computed } from 'vue';

import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';

import { useDisplayCurrencyStore } from './store.ts';

export interface DisplayedAmount {
    /** Amount in minor units, in `currency`. */
    readonly minor: number;
    readonly currency: string;
}

export function useInvestmentDisplayCurrency() {
    const displayCurrencyStore = useDisplayCurrencyStore();
    const exchangeRatesStore = useExchangeRatesStore();

    /**
     * Converts an amount in minor units from `from` to the display currency.
     * Without a usable rate the amount stays in `from`, so a wrong number is never shown with the wrong code.
     */
    function toDisplay(minor: number, from: string | undefined): DisplayedAmount | null {
        if (!from) {
            return null;
        }

        const target = displayCurrencyStore.currency;

        if (from === target) {
            return { minor, currency: from };
        }

        const exchanged = exchangeRatesStore.getExchangedAmount(parseBigDecimal(minor), from, target);

        if (!exchanged) {
            return { minor, currency: from };
        }

        return { minor: Math.round(Number(exchanged.toString())), currency: target };
    }

    const displayCurrencyCode = computed<string>(() => displayCurrencyStore.currency);

    return { toDisplay, displayCurrencyCode };
}
