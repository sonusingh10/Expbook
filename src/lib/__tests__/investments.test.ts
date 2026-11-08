import { describe, expect, test } from 'vitest';

import { parseBigDecimal } from '@/lib/numeral.ts';
import {
    combineNetWorth,
    getValueSign,
    withExplicitSign,
    formatSignedPercent,
    isPriceStale,
    isValidQuantityOrPrice,
    majorToMinorUnits,
    minorToMajorString,
    countImportRows,
    IMPORT_CSV_EXAMPLE,
    IMPORT_CSV_HEADER
} from '@/lib/investments.ts';

describe('getValueSign / withExplicitSign', () => {
    test('classifies values', () => {
        expect(getValueSign(10)).toBe('positive');
        expect(getValueSign(-10)).toBe('negative');
        expect(getValueSign(0)).toBe('zero');
        expect(getValueSign(null)).toBe('zero');
    });

    test('adds a plus only to positive values', () => {
        expect(withExplicitSign('₹1,200.00', 120000)).toBe('+₹1,200.00');
        expect(withExplicitSign('-₹1,200.00', -120000)).toBe('-₹1,200.00');
        expect(withExplicitSign('₹0.00', 0)).toBe('₹0.00');
        expect(withExplicitSign('+₹5.00', 500)).toBe('+₹5.00');
    });
});

describe('formatSignedPercent', () => {
    test('formats ratios and percentages with sign', () => {
        expect(formatSignedPercent(0.1234, true)).toBe('+12.34%');
        expect(formatSignedPercent(-0.05, true)).toBe('-5.00%');
        expect(formatSignedPercent(12.3456, false)).toBe('+12.35%');
        expect(formatSignedPercent(0, true)).toBe('0.00%');
    });

    test('rounds a tiny value to zero without a sign', () => {
        expect(formatSignedPercent(0.000001, true)).toBe('0.00%');
        expect(formatSignedPercent(-0.000001, true)).toBe('0.00%');
    });

    test('shows a dash when the value is unknown', () => {
        expect(formatSignedPercent(null, true)).toBe('-');
        expect(formatSignedPercent(undefined, true)).toBe('-');
        expect(formatSignedPercent(Number.NaN, true)).toBe('-');
    });
});

describe('isPriceStale', () => {
    const now = 1_800_000_000;

    test('missing price is stale', () => {
        expect(isPriceStale(0, now)).toBe(true);
    });

    test('recent price is fresh and an old one is stale', () => {
        expect(isPriceStale(now - 60, now)).toBe(false);
        expect(isPriceStale(now - 3 * 86400, now)).toBe(false);
        expect(isPriceStale(now - 3 * 86400 - 1, now)).toBe(true);
    });
});

describe('isValidQuantityOrPrice', () => {
    test('accepts plain decimals up to 8 places', () => {
        expect(isValidQuantityOrPrice('10')).toBe(true);
        expect(isValidQuantityOrPrice('0.00000001')).toBe(true);
        expect(isValidQuantityOrPrice(' 1480.5 ')).toBe(true);
    });

    test('rejects separators, exponents, signs, too many decimals and zero', () => {
        expect(isValidQuantityOrPrice('1,000')).toBe(false);
        expect(isValidQuantityOrPrice('1e3')).toBe(false);
        expect(isValidQuantityOrPrice('-1')).toBe(false);
        expect(isValidQuantityOrPrice('0.000000001')).toBe(false);
        expect(isValidQuantityOrPrice('')).toBe(false);
        expect(isValidQuantityOrPrice('0')).toBe(false);
        expect(isValidQuantityOrPrice('0', true)).toBe(true);
    });
});

describe('majorToMinorUnits / minorToMajorString', () => {
    test('converts without floating point drift', () => {
        expect(majorToMinorUnits('123.45')).toBe(12345);
        expect(majorToMinorUnits('0.1')).toBe(10);
        expect(majorToMinorUnits('19.99')).toBe(1999);
        expect(majorToMinorUnits('5')).toBe(500);
    });

    test('rejects invalid input', () => {
        expect(majorToMinorUnits('1.234')).toBeNull();
        expect(majorToMinorUnits('abc')).toBeNull();
        expect(majorToMinorUnits('-1')).toBeNull();
        expect(majorToMinorUnits('')).toBeNull();
    });

    test('round trips', () => {
        expect(minorToMajorString(12345)).toBe('123.45');
        expect(minorToMajorString(5)).toBe('0.05');
        expect(minorToMajorString(-250)).toBe('-2.50');
    });
});

describe('import helpers', () => {
    test('counts row statuses', () => {
        expect(countImportRows([
            { status: 'valid' }, { status: 'valid' }, { status: 'duplicate' }, { status: 'error' }
        ])).toEqual({ valid: 2, duplicate: 1, error: 1 });
    });

    test('example csv starts with the documented header', () => {
        expect(IMPORT_CSV_EXAMPLE.split('\n')[0]).toBe(IMPORT_CSV_HEADER);
        expect(IMPORT_CSV_EXAMPLE.split('\n')).toHaveLength(4);
    });
});

describe('combineNetWorth', () => {
    test('adds investments to a complete accounts total', () => {
        const total = combineNetWorth(parseBigDecimal(100000), parseBigDecimal(25050));
        expect(String((total as { toString: () => string }).toString())).toBe(parseBigDecimal(125050).toString());
    });

    test('keeps the incomplete suffix when some accounts have no exchange rate', () => {
        const total = combineNetWorth({ value: parseBigDecimal(100), suffix: '+' }, parseBigDecimal(50));
        expect(total).toHaveProperty('suffix', '+');
        expect((total as { value: { toString: () => string } }).value.toString()).toBe(parseBigDecimal(150).toString());
    });

    test('hidden balances stay hidden', () => {
        expect(combineNetWorth('***', parseBigDecimal(50))).toBe('***');
    });
});
