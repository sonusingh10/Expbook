// Portfolio trend helpers (Expbook). Pure functions, no Vue, so they are easy to test.
// All amounts here are minor units in the display currency.

export type TrendRange = '6M' | '1Y' | '3Y' | 'ALL';

export const TREND_RANGES: TrendRange[] = ['6M', '1Y', '3Y', 'ALL'];

export interface AccountDay {
    /** Unix time of the end of that day. Sorted ascending. */
    readonly time: number;
    /** Sum of all accounts at the close of that day. */
    readonly total: number;
}

export interface HistoryPointDto {
    readonly time: number;
    readonly invested: number;
    readonly value: number;
    readonly estimated: boolean;
}

export interface TrendRow {
    readonly time: number;
    readonly accounts: number;
    /** Market value of investments (cost basis where no price was recorded yet). */
    readonly investments: number;
    /** Cost basis of investments. */
    readonly invested: number;
    /** accounts + investments */
    readonly total: number;
    /** True when `investments` is partly the cost basis because no market price was recorded yet. */
    readonly estimated: boolean;
}

const DAY = 86400;

/** The start of a range and how far apart the points are. start is null for "all data". */
export function rangeSpec(range: TrendRange, now: number): { start: number | null, stepDays: number } {
    switch (range) {
        case '6M': return { start: now - 183 * DAY, stepDays: 2 };
        case '1Y': return { start: now - 365 * DAY, stepDays: 3 };
        case '3Y': return { start: now - 3 * 365 * DAY, stepDays: 7 };
        default: return { start: null, stepDays: 7 };
    }
}

/** For "all data": widen the step so a very long history still stays around 200 points. */
export function stepDaysForSpan(startTime: number, endTime: number, minimum: number): number {
    const spanDays = Math.max(0, Math.round((endTime - startTime) / DAY));
    return Math.max(minimum, Math.ceil(spanDays / 200));
}

/** Total of all accounts at time t: the last known day at or before t (0 before the first record). */
export function totalAt(days: readonly AccountDay[], t: number): number {
    let low = 0;
    let high = days.length;

    while (low < high) {
        const mid = (low + high) >> 1;

        if (days[mid]!.time <= t) {
            low = mid + 1;
        } else {
            high = mid;
        }
    }

    return low === 0 ? 0 : days[low - 1]!.total;
}

/**
 * Merges account totals and investment history into rows.
 * `convert` turns minor units from the profile currency into the display currency, or null when no rate exists.
 * When it returns null the investments cannot be shown and `missingRate` is true.
 */
export function buildRows(points: readonly HistoryPointDto[], days: readonly AccountDay[], convert: (minor: number) => number | null): { rows: TrendRow[], missingRate: boolean } {
    const rows: TrendRow[] = [];
    let missingRate = false;

    for (const point of points) {
        const value = convert(point.value);
        const invested = convert(point.invested);

        if (value === null || invested === null) {
            missingRate = true;
        }

        const accounts = totalAt(days, point.time);
        const investments = value ?? 0;

        rows.push({
            time: point.time,
            accounts,
            investments,
            invested: invested ?? 0,
            total: accounts + investments,
            estimated: point.estimated
        });
    }

    return { rows, missingRate };
}

export interface Scale {
    readonly min: number;
    readonly max: number;
    readonly ticks: number[];
}

/** A tidy axis: round tick values that cover [min, max]. */
export function niceScale(min: number, max: number, tickCount: number = 4): Scale {
    if (!isFinite(min) || !isFinite(max)) {
        return { min: 0, max: 1, ticks: [0, 1] };
    }

    if (max === min) {
        max = min + Math.max(1, Math.abs(min) * 0.1);
    }

    const rawStep = (max - min) / Math.max(1, tickCount);
    const magnitude = Math.pow(10, Math.floor(Math.log10(rawStep)));
    const normalised = rawStep / magnitude;
    const factor = normalised <= 1 ? 1 : normalised <= 2 ? 2 : normalised <= 2.5 ? 2.5 : normalised <= 5 ? 5 : 10;
    const step = factor * magnitude;
    const niceMin = Math.floor(min / step) * step;
    const niceMax = Math.ceil(max / step) * step;
    const ticks: number[] = [];

    for (let v = niceMin; v <= niceMax + step / 2; v += step) {
        ticks.push(Math.round(v * 1e6) / 1e6);
    }

    return { min: niceMin, max: niceMax, ticks };
}

export interface Segments {
    /** Path of the parts that use recorded market prices. */
    readonly solid: string;
    /** Path of the parts that use cost as a stand-in. */
    readonly dashed: string;
}

function fmt(n: number): string {
    return (Math.round(n * 100) / 100).toString();
}

/** One continuous path through the points. */
export function linePath(xs: readonly number[], ys: readonly number[]): string {
    return xs.map((x, i) => `${i === 0 ? 'M' : 'L'}${fmt(x)} ${fmt(ys[i]!)}`).join(' ');
}

/** Splits a line into solid and dashed parts: a segment is dashed when either end is an estimate. */
export function segmentPaths(xs: readonly number[], ys: readonly number[], estimated: readonly boolean[]): Segments {
    let solid = '';
    let dashed = '';

    for (let i = 0; i < xs.length - 1; i++) {
        const piece = `M${fmt(xs[i]!)} ${fmt(ys[i]!)} L${fmt(xs[i + 1]!)} ${fmt(ys[i + 1]!)} `;

        if (estimated[i] || estimated[i + 1]) {
            dashed += piece;
        } else {
            solid += piece;
        }
    }

    return { solid: solid.trim(), dashed: dashed.trim() };
}

/** Index of the x position closest to x (xs ascending). -1 for no points. */
export function nearestIndex(xs: readonly number[], x: number): number {
    if (!xs.length) {
        return -1;
    }

    let best = 0;
    let bestDistance = Math.abs(xs[0]! - x);

    for (let i = 1; i < xs.length; i++) {
        const distance = Math.abs(xs[i]! - x);

        if (distance < bestDistance) {
            best = i;
            bestDistance = distance;
        } else if (xs[i]! > x) {
            break;
        }
    }

    return best;
}

/** Change between the first and the last row. percent is null when the start is 0. */
export function changeOver(rows: readonly TrendRow[]): { amount: number, percent: number | null } {
    if (rows.length < 2) {
        return { amount: 0, percent: null };
    }

    const first = rows[0]!.total;
    const last = rows[rows.length - 1]!.total;
    const amount = last - first;

    return { amount, percent: first > 0 ? (amount / first) * 100 : null };
}

/** Drops leading rows where nothing exists yet (no accounts and no investments). */
export function trimEmptyStart(rows: readonly TrendRow[]): TrendRow[] {
    const first = rows.findIndex(row => row.accounts !== 0 || row.investments !== 0 || row.invested !== 0);
    return first < 0 ? [] : rows.slice(first);
}

/** Evenly spaced points with no investments, used when the user has no trades (accounts can still be charted). */
export function gridPoints(start: number, end: number, stepSeconds: number): HistoryPointDto[] {
    if (stepSeconds <= 0 || end < start) {
        return [];
    }

    const points: HistoryPointDto[] = [];

    for (let t = start; t < end; t += stepSeconds) {
        points.push({ time: t, invested: 0, value: 0, estimated: false });
    }

    points.push({ time: end, invested: 0, value: 0, estimated: false });

    return points;
}

/** Unix time of the end of a calendar day (browser time zone). */
export function endOfDay(year: number, month1: number, day: number): number {
    return Math.floor(new Date(year, month1 - 1, day, 23, 59, 59).getTime() / 1000);
}
