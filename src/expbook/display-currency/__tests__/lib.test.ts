import { describe, expect, test } from 'vitest';

import {
    normalizeCurrencyCode,
    resolveDisplayCurrency,
    readCachedCurrency,
    serializeCachedCurrency,
    hasRateFor,
    pickerCurrencies
} from '../lib.ts';

describe('normalizeCurrencyCode', () => {
    test('accepts three letters in any case', () => {
        expect(normalizeCurrencyCode(' pln ')).toBe('PLN');
        expect(normalizeCurrencyCode('INR')).toBe('INR');
    });

    test('rejects everything else', () => {
        expect(normalizeCurrencyCode('')).toBe('');
        expect(normalizeCurrencyCode('EURO')).toBe('');
        expect(normalizeCurrencyCode('E1R')).toBe('');
        expect(normalizeCurrencyCode(undefined)).toBe('');
        expect(normalizeCurrencyCode(42)).toBe('');
    });
});

describe('resolveDisplayCurrency', () => {
    test('uses the chosen currency when there is one', () => {
        expect(resolveDisplayCurrency('PLN', 'INR')).toBe('PLN');
    });

    test('falls back to the profile currency', () => {
        expect(resolveDisplayCurrency('', 'INR')).toBe('INR');
        expect(resolveDisplayCurrency('nonsense', 'INR')).toBe('INR');
    });
});

describe('cached currency', () => {
    test('round-trips for the same user', () => {
        const raw = serializeCachedCurrency('test', 'PLN');
        expect(readCachedCurrency(raw, 'test')).toBe('PLN');
    });

    test('is ignored for another user', () => {
        const raw = serializeCachedCurrency('test', 'PLN');
        expect(readCachedCurrency(raw, 'someone-else')).toBe('');
    });

    test('is ignored when missing or damaged', () => {
        expect(readCachedCurrency(null, 'test')).toBe('');
        expect(readCachedCurrency('{not json', 'test')).toBe('');
        expect(readCachedCurrency(serializeCachedCurrency('test', 'PLN'), '')).toBe('');
    });
});

describe('rates', () => {
    const rates = [{ currency: 'EUR' }, { currency: 'PLN' }, { currency: 'INR' }];

    test('hasRateFor', () => {
        expect(hasRateFor('PLN', rates)).toBe(true);
        expect(hasRateFor('JPY', rates)).toBe(false);
        expect(hasRateFor('PLN', [])).toBe(false);
        expect(hasRateFor('PLN', null)).toBe(false);
    });

    test('pickerCurrencies puts the profile currency first and removes duplicates', () => {
        expect(pickerCurrencies([...rates, { currency: 'PLN' }], 'INR')).toEqual(['INR', 'EUR', 'PLN']);
        expect(pickerCurrencies(null, 'INR')).toEqual(['INR']);
    });
});
