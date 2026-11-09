// Display currency helpers (Expbook). Pure functions, no Vue, so they are easy to test.

export interface CachedDisplayCurrency {
    readonly username: string;
    readonly currency: string;
}

export interface RateEntry {
    readonly currency: string;
}

/** Returns an upper-case 3-letter code, or "" when the value is not a currency code. */
export function normalizeCurrencyCode(value: unknown): string {
    if (typeof value !== 'string') {
        return '';
    }

    const code = value.trim().toUpperCase();
    return /^[A-Z]{3}$/.test(code) ? code : '';
}

/** The currency screens should show: the chosen display currency, otherwise the profile currency. */
export function resolveDisplayCurrency(chosen: string, profileCurrency: string): string {
    return normalizeCurrencyCode(chosen) || profileCurrency;
}

/** Parses the browser cache. Returns "" when the cache belongs to somebody else or is damaged. */
export function readCachedCurrency(raw: string | null, username: string): string {
    if (!raw || !username) {
        return '';
    }

    try {
        const parsed = JSON.parse(raw) as Partial<CachedDisplayCurrency>;
        return parsed.username === username ? normalizeCurrencyCode(parsed.currency) : '';
    } catch {
        return '';
    }
}

export function serializeCachedCurrency(username: string, currency: string): string {
    return JSON.stringify({ username, currency } satisfies CachedDisplayCurrency);
}

/** True when a conversion into this currency is possible with the loaded rates. */
export function hasRateFor(currency: string, rates: readonly RateEntry[] | null | undefined): boolean {
    if (!rates || !rates.length) {
        return false;
    }

    return rates.some(rate => rate.currency === currency);
}

/** Currencies the picker offers: those that have a rate, the profile currency first. */
export function pickerCurrencies(rates: readonly RateEntry[] | null | undefined, profileCurrency: string): string[] {
    const codes = (rates || []).map(rate => rate.currency).filter(code => !!normalizeCurrencyCode(code));
    const unique = Array.from(new Set(codes)).sort();
    return [profileCurrency, ...unique.filter(code => code !== profileCurrency)];
}
