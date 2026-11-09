// Shared logic for the desktop and mobile display currency pickers (Expbook).

import { computed, onMounted, ref } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';

import { useDisplayCurrencyStore } from './store.ts';

export interface CurrencyChoice {
    readonly code: string;
    readonly name: string;
    readonly selected: boolean;
}

export function useDisplayCurrencyPicker(syncOnMount: boolean = true) {
    const { tt, getCurrencyName, formatDateTimeToLongDate } = useI18n();
    const store = useDisplayCurrencyStore();
    const saving = ref<boolean>(false);
    const failed = ref<boolean>(false);

    const choices = computed<CurrencyChoice[]>(() => store.availableCurrencies.map(code => ({
        code,
        name: getCurrencyName(code),
        selected: code === store.currency
    })));

    const ratesAsOf = computed<string>(() => {
        if (!store.ratesUpdateTime) {
            return '';
        }

        return tt('Rates as of') + ' ' + formatDateTimeToLongDate(parseDateTimeFromUnixTime(store.ratesUpdateTime));
    });

    async function pick(code: string): Promise<void> {
        if (saving.value || code === store.currency) {
            return;
        }

        saving.value = true;
        failed.value = false;

        try {
            await store.choose(code);
            // Totals are cached by many screens, so reload once to make all of them agree.
            window.location.reload();
        } catch {
            failed.value = true;
            saving.value = false;
        }
    }

    onMounted(async () => {
        if (!syncOnMount) {
            return;
        }

        store.ensureRates();

        if (await store.syncFromServer()) {
            window.location.reload();
        }
    });

    return { store, choices, ratesAsOf, saving, failed, pick, tt };
}
