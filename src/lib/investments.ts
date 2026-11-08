import type { BigDecimal, HiddenAmount, BigDecimalWithSuffix } from '@/core/numeral.ts';

import { isBigDecimal } from '@/lib/numeral.ts';

// Expbook: small pure helpers for the investment screens.

const MAX_QUANTITY_DECIMALS = 8;
const STALE_PRICE_DAYS = 3;

export type ValueSign = 'positive' | 'negative' | 'zero';

export function getValueSign(minorUnits: number | null | undefined): ValueSign {
    if (!minorUnits) {
        return 'zero';
    }

    return minorUnits > 0 ? 'positive' : 'negative';
}

/**
 * Gain and loss are conveyed by sign first and colour second: positive values get an explicit "+".
 * Negative values already carry their own minus sign from the currency formatter.
 */
export function withExplicitSign(formatted: string, minorUnits: number | null | undefined): string {
    if (getValueSign(minorUnits) === 'positive' && !formatted.startsWith('+')) {
        return '+' + formatted;
    }

    return formatted;
}

/** Formats a ratio such as XIRR (0.1234) or a percentage already multiplied by 100, with an explicit sign. */
export function formatSignedPercent(value: number | null | undefined, isRatio: boolean, fractionDigits: number = 2): string {
    if (value === null || value === undefined || !Number.isFinite(value)) {
        return '-';
    }

    const percent = isRatio ? value * 100 : value;
    const text = Math.abs(percent).toFixed(fractionDigits);
    const isZero = Number(text) === 0;

    if (isZero) {
        return text + '%';
    }

    return (percent > 0 ? '+' : '-') + text + '%';
}

/** A price is stale when it is older than a few days, or when there is none at all. */
export function isPriceStale(priceTimeUnix: number, nowUnix: number): boolean {
    if (!priceTimeUnix || priceTimeUnix <= 0) {
        return true;
    }

    return nowUnix - priceTimeUnix > STALE_PRICE_DAYS * 24 * 60 * 60;
}

/** Mirrors the server: plain decimal, no thousands separators, no exponent, at most 8 decimals. */
export function isValidQuantityOrPrice(value: string, allowZero: boolean = false): boolean {
    const text = value.trim();

    if (!/^\d+(\.\d+)?$/.test(text)) {
        return false;
    }

    const fraction = text.split('.')[1];

    if (fraction && fraction.length > MAX_QUANTITY_DECIMALS) {
        return false;
    }

    return allowZero || Number(text) > 0;
}

/** Converts an amount typed in major units (for example "123.45") to minor units, or null if invalid. */
export function majorToMinorUnits(value: string): number | null {
    const text = value.trim();

    if (!/^\d+(\.\d{1,2})?$/.test(text)) {
        return null;
    }

    const [whole, fraction = ''] = text.split('.');

    return Number(whole) * 100 + Number((fraction + '00').slice(0, 2));
}

export function minorToMajorString(minor: number): string {
    const negative = minor < 0;
    const abs = Math.abs(minor);
    const whole = Math.floor(abs / 100);
    const fraction = String(abs % 100).padStart(2, '0');

    return (negative ? '-' : '') + whole + '.' + fraction;
}

export const IMPORT_CSV_HEADER = 'date,type,symbol,name,security_type,currency,quantity,price,amount,fees,external_id,comment';

export const IMPORT_CSV_EXAMPLE = [
    IMPORT_CSV_HEADER,
    '2025-04-02,buy,INFY,Infosys,stock,INR,10,1480.5,,20,,first purchase',
    '2025-06-15,dividend,INFY,Infosys,stock,INR,,,350,,,',
    '2025-09-01,sell,INFY,Infosys,stock,INR,4,1650,,18,,'
].join('\n');

export interface ImportCounts {
    readonly valid: number;
    readonly duplicate: number;
    readonly error: number;
}

export function countImportRows(rows: { status: string }[]): ImportCounts {
    let valid = 0;
    let duplicate = 0;
    let error = 0;

    for (const row of rows) {
        if (row.status === 'valid') {
            valid++;
        } else if (row.status === 'duplicate') {
            duplicate++;
        } else {
            error++;
        }
    }

    return { valid, duplicate, error };
}

/**
 * Adds the investment value (BigDecimal in the same minor-unit scale as accounts) to the accounts' net assets.
 * Hidden balances stay hidden and an incomplete accounts total keeps its "incomplete" suffix, so the sum never
 * claims to be more precise than its parts.
 */
export function combineNetWorth(accountsNet: BigDecimal | HiddenAmount | BigDecimalWithSuffix, investmentValue: BigDecimal): BigDecimal | HiddenAmount | BigDecimalWithSuffix {
    if (typeof accountsNet === 'string') {
        return accountsNet;
    }

    if (isBigDecimal(accountsNet)) {
        return accountsNet.add(investmentValue);
    }

    return {
        value: accountsNet.value.add(investmentValue),
        suffix: accountsNet.suffix
    };
}
