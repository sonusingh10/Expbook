// Display currency store (Expbook).
//
// The user can look at every total in another currency without changing the profile currency.
// The server only stores the choice; the amounts are converted in the browser with the latest exchange rates.

import { ref, computed } from 'vue';
import { defineStore } from 'pinia';
import axios from 'axios';

import { useUserStore } from '@/stores/user.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import {
    normalizeCurrencyCode,
    resolveDisplayCurrency,
    readCachedCurrency,
    serializeCachedCurrency,
    hasRateFor,
    pickerCurrencies
} from './lib.ts';

const cacheKey = 'ebk_expbook_display_currency';

interface DisplayCurrencyResult {
    readonly success: boolean;
    readonly result?: { currency: string };
}

function currentUsername(): string {
    const info = useUserStore().currentUserBasicInfo;
    return info ? info.username : '';
}

function readCache(): string {
    return readCachedCurrency(localStorage.getItem(cacheKey), currentUsername());
}

function writeCache(currency: string): void {
    localStorage.setItem(cacheKey, serializeCachedCurrency(currentUsername(), currency));
}

export const useDisplayCurrencyStore = defineStore('expbookDisplayCurrency', () => {
    const userStore = useUserStore();
    const exchangeRatesStore = useExchangeRatesStore();

    // Read the browser cache first so the first paint already uses the right currency.
    const chosen = ref<string>(readCache());

    /** Currency to show amounts in. */
    const currency = computed<string>(() => resolveDisplayCurrency(chosen.value, userStore.currentUserDefaultCurrency));

    /** True while amounts are shown in a currency other than the profile currency. */
    const isOverridden = computed<boolean>(() => currency.value !== userStore.currentUserDefaultCurrency);

    const rateList = computed(() => Object.values(exchangeRatesStore.latestExchangeRateMap));

    /** False when a chosen currency has no rate (for example the rates are not loaded yet). */
    const hasRate = computed<boolean>(() => !isOverridden.value || hasRateFor(currency.value, rateList.value));

    const availableCurrencies = computed<string[]>(() => pickerCurrencies(rateList.value, userStore.currentUserDefaultCurrency));

    const ratesUpdateTime = computed<number | null>(() => exchangeRatesStore.exchangeRatesLastUpdateTime);

    function ensureRates(): void {
        if (!rateList.value.length) {
            exchangeRatesStore.getLatestExchangeRates({ silent: true, force: false }).catch(() => undefined);
        }
    }

    /**
     * Ask the server for the saved choice (it follows the user across devices).
     * Returns true when it differs from what this browser was using, so the caller can reload.
     */
    async function syncFromServer(): Promise<boolean> {
        try {
            const response = await axios.get<DisplayCurrencyResult>('v1/expbook/preferences/display_currency.json');
            const saved = response.data && response.data.success && response.data.result
                ? normalizeCurrencyCode(response.data.result.currency)
                : '';

            writeCache(saved);

            if (saved !== chosen.value) {
                chosen.value = saved;
                return true;
            }
        } catch {
            // Offline or not signed in yet: keep what this browser has.
        }

        return false;
    }

    /** Save a new choice. An empty code goes back to the profile currency. */
    async function choose(code: string): Promise<void> {
        const next = normalizeCurrencyCode(code);
        const previous = chosen.value;
        const profile = userStore.currentUserDefaultCurrency;
        const stored = next === profile ? '' : next;

        chosen.value = stored;
        writeCache(stored);

        try {
            await axios.post('v1/expbook/preferences/display_currency/update.json', { currency: stored });
        } catch (error) {
            chosen.value = previous;
            writeCache(previous);
            throw error;
        }
    }

    return {
        chosen,
        currency,
        isOverridden,
        hasRate,
        availableCurrencies,
        ratesUpdateTime,
        ensureRates,
        syncFromServer,
        choose
    };
});
