import { describe, expect, test } from 'vitest';

import {
    rangeSpec,
    stepDaysForSpan,
    totalAt,
    buildRows,
    niceScale,
    linePath,
    segmentPaths,
    nearestIndex,
    changeOver,
    trimEmptyStart,
    gridPoints,
    endOfDay,
    type TrendRow
} from '../lib.ts';

const DAY = 86400;

describe('rangeSpec', () => {
    test('named ranges look back from now', () => {
        expect(rangeSpec('1Y', 1_000_000_000)).toEqual({ start: 1_000_000_000 - 365 * DAY, stepDays: 3 });
        expect(rangeSpec('ALL', 5).start).toBeNull();
    });

    test('a long history is thinned to about 200 points', () => {
        expect(stepDaysForSpan(0, 365 * DAY, 7)).toBe(7);
        expect(stepDaysForSpan(0, 10 * 365 * DAY, 7)).toBe(19);
    });
});

describe('totalAt', () => {
    const days = [{ time: 100, total: 10 }, { time: 200, total: 30 }, { time: 300, total: 20 }];

    test('uses the last day at or before the time', () => {
        expect(totalAt(days, 100)).toBe(10);
        expect(totalAt(days, 250)).toBe(30);
        expect(totalAt(days, 999)).toBe(20);
    });

    test('is 0 before the first record and for no records', () => {
        expect(totalAt(days, 99)).toBe(0);
        expect(totalAt([], 5)).toBe(0);
    });
});

describe('buildRows', () => {
    const points = [
        { time: 100, invested: 1000, value: 1000, estimated: true },
        { time: 200, invested: 1000, value: 1500, estimated: false }
    ];
    const days = [{ time: 50, total: 500 }, { time: 150, total: 700 }];

    test('adds accounts and investments and keeps the estimate flag', () => {
        const { rows, missingRate } = buildRows(points, days, minor => minor * 2);
        expect(missingRate).toBe(false);
        expect(rows[0]).toEqual({ time: 100, accounts: 500, investments: 2000, invested: 2000, total: 2500, estimated: true });
        expect(rows[1]).toEqual({ time: 200, accounts: 700, investments: 3000, invested: 2000, total: 3700, estimated: false });
    });

    test('without a rate investments are left out and flagged, never guessed', () => {
        const { rows, missingRate } = buildRows(points, days, () => null);
        expect(missingRate).toBe(true);
        expect(rows[1]!.investments).toBe(0);
        expect(rows[1]!.total).toBe(700);
    });
});

describe('niceScale', () => {
    test('covers the data with round ticks', () => {
        const scale = niceScale(0, 1730, 4);
        expect(scale.min).toBe(0);
        expect(scale.max).toBeGreaterThanOrEqual(1730);
        expect(scale.ticks[0]).toBe(0);
        expect(scale.ticks.at(-1)).toBe(scale.max);
        expect(scale.ticks.every(t => t === Math.round(t))).toBe(true);
    });

    test('a flat series still gets a range', () => {
        const scale = niceScale(5, 5);
        expect(scale.max).toBeGreaterThan(scale.min);
    });

    test('bad input does not throw', () => {
        expect(niceScale(NaN, 3).ticks.length).toBeGreaterThan(1);
    });
});

describe('paths', () => {
    test('linePath', () => {
        expect(linePath([0, 10], [5, 7.456])).toBe('M0 5 L10 7.46');
        expect(linePath([], [])).toBe('');
    });

    test('segmentPaths puts a segment in dashed when either end is an estimate', () => {
        const { solid, dashed } = segmentPaths([0, 10, 20, 30], [1, 2, 3, 4], [false, false, true, false]);
        expect(solid).toBe('M0 1 L10 2');
        expect(dashed).toBe('M10 2 L20 3 M20 3 L30 4');
    });
});

describe('nearestIndex', () => {
    test('picks the closest point', () => {
        expect(nearestIndex([0, 10, 20], 4)).toBe(0);
        expect(nearestIndex([0, 10, 20], 6)).toBe(1);
        expect(nearestIndex([0, 10, 20], 99)).toBe(2);
        expect(nearestIndex([], 1)).toBe(-1);
    });
});

describe('changeOver and trimEmptyStart', () => {
    const row = (time: number, accounts: number, investments = 0, invested = 0): TrendRow => ({ time, accounts, investments, invested, total: accounts + investments, estimated: false });

    test('change in total', () => {
        expect(changeOver([row(1, 100), row(2, 150)])).toEqual({ amount: 50, percent: 50 });
        expect(changeOver([row(1, 0), row(2, 150)])).toEqual({ amount: 150, percent: null });
        expect(changeOver([row(1, 100)])).toEqual({ amount: 0, percent: null });
    });

    test('empty leading rows are dropped', () => {
        expect(trimEmptyStart([row(1, 0), row(2, 0), row(3, 5)]).map(r => r.time)).toEqual([3]);
        expect(trimEmptyStart([row(1, 0)])).toEqual([]);
    });
});

describe('gridPoints and endOfDay', () => {
    test('a grid always ends at the end time', () => {
        expect(gridPoints(0, 25, 10).map(p => p.time)).toEqual([0, 10, 20, 25]);
        expect(gridPoints(0, 20, 10).map(p => p.time)).toEqual([0, 10, 20]);
        expect(gridPoints(5, 1, 10)).toEqual([]);
        expect(gridPoints(0, 10, 0)).toEqual([]);
    });

    test('end of day is within the day', () => {
        const noon = Math.floor(new Date(2026, 4, 3, 12).getTime() / 1000);
        const end = endOfDay(2026, 5, 3);
        expect(end).toBeGreaterThan(noon);
        expect(end - noon).toBeLessThan(86400);
    });
});
