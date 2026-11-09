<template>
    <section class="ebk-trend" :class="{ 'ebk-trend-compact': compact }" :aria-labelledby="titleId">
        <header class="ebk-trend-head">
            <div class="ebk-trend-summary">
                <h2 :id="titleId" class="ebk-trend-title">{{ tt('Portfolio trend') }}</h2>
                <div class="ebk-trend-total ebk-numeric">{{ money(active?.total) }}</div>
                <div class="ebk-trend-sub">
                    <template v-if="hovering && active">{{ formatDay(active.time) }}</template>
                    <template v-else>
                        <span class="ebk-trend-change ebk-numeric" :class="changeClass" v-if="rows.length > 1">{{ changeText }}</span>
                        <span class="ebk-trend-muted">{{ tt('Total assets over time') }}</span>
                    </template>
                </div>
            </div>
            <div class="ebk-trend-ranges" role="group" :aria-label="tt('Chart range')">
                <button type="button" class="ebk-trend-range" :class="{ 'ebk-trend-range-on': store.range === item }"
                        :aria-pressed="store.range === item" :disabled="store.loading"
                        :key="item" v-for="item in TREND_RANGES" @click="store.setRange(item)">{{ rangeLabel(item) }}</button>
            </div>
        </header>

        <div ref="box" class="ebk-trend-chart" :style="{ height: `${height}px` }">
            <div class="ebk-trend-state" v-if="store.failed">
                <span>{{ tt('Unable to load the trend') }}</span>
                <button type="button" class="ebk-trend-range" @click="store.load()">{{ tt('Retry') }}</button>
            </div>
            <div class="ebk-trend-state" v-else-if="store.loaded && rows.length < 2">
                <span>{{ tt('Add an account or an investment to see your trend.') }}</span>
            </div>
            <div class="ebk-trend-skeleton" aria-hidden="true" v-else-if="!store.loaded"></div>
            <svg ref="svg" class="ebk-trend-svg" :width="width" :height="height" :viewBox="`0 0 ${width} ${height}`"
                 tabindex="0" role="group" :aria-label="ariaLabel" :class="{ 'ebk-trend-busy': store.loading }"
                 @pointermove="onPointer" @pointerdown="onPointer" @pointerleave="hover = -1"
                 @keydown="onKey" @blur="hover = -1" v-else-if="width > 0">
                <g class="ebk-trend-grid">
                    <template v-for="tick in scale.ticks" :key="`y${tick}`">
                        <line :x1="plot.left" :x2="plot.right" :y1="yOf(tick)" :y2="yOf(tick)" />
                        <text class="ebk-trend-axis" :x="plot.left - 8" :y="yOf(tick)" text-anchor="end" dominant-baseline="middle"
                              v-if="showAmounts">{{ compactNumber(tick) }}</text>
                    </template>
                </g>
                <g>
                    <text class="ebk-trend-axis" :x="clampX(tick.x)" :y="height - 6" :text-anchor="tick.anchor"
                          :key="`x${tick.time}`" v-for="tick in xTicks">{{ tick.label }}</text>
                </g>

                <path class="ebk-trend-area" :d="areaPath" />
                <path class="ebk-trend-line ebk-trend-line-cost" :d="costPath" v-if="store.hasInvestments" />
                <template v-if="store.hasInvestments">
                    <path class="ebk-trend-line ebk-trend-line-invest" :d="investSegments.solid" />
                    <path class="ebk-trend-line ebk-trend-line-invest ebk-trend-line-estimate" :d="investSegments.dashed" />
                </template>
                <path class="ebk-trend-line ebk-trend-line-total" :d="totalPath" />

                <g v-if="active && activeX !== null">
                    <line class="ebk-trend-cursor" :x1="activeX" :x2="activeX" :y1="plot.top" :y2="plot.bottom" />
                    <circle class="ebk-trend-dot ebk-trend-dot-total" :cx="activeX" :cy="yOf(active.total)" r="4" />
                    <circle class="ebk-trend-dot ebk-trend-dot-invest" :cx="activeX" :cy="yOf(active.investments)" r="3.5" v-if="store.hasInvestments" />
                </g>
            </svg>
        </div>

        <ul class="ebk-trend-legend">
            <li>
                <span class="ebk-trend-key ebk-trend-key-total"></span>
                <span class="ebk-trend-legend-label">{{ tt('Total assets') }}</span>
                <span class="ebk-numeric">{{ money(active?.total) }}</span>
            </li>
            <li v-if="store.hasInvestments">
                <span class="ebk-trend-key ebk-trend-key-invest"></span>
                <span class="ebk-trend-legend-label">{{ tt('Investments') }}</span>
                <span class="ebk-numeric">{{ money(active?.investments) }}</span>
            </li>
            <li v-if="store.hasInvestments">
                <span class="ebk-trend-key ebk-trend-key-cost"></span>
                <span class="ebk-trend-legend-label">{{ tt('Cost of investments') }}</span>
                <span class="ebk-numeric">{{ money(active?.invested) }}</span>
            </li>
        </ul>

        <p class="ebk-trend-note" v-if="store.investmentsMissingRate">
            {{ tt('Investments are left out because there is no exchange rate for this currency yet.') }}
        </p>
        <p class="ebk-trend-note" v-else-if="store.hasEstimates">
            {{ tt('Dashed lines use cost where no market price was recorded yet. Prices you record from now on build the real history.') }}
        </p>
    </section>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, useId } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useSettingsStore } from '@/stores/setting.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';

import { useDisplayCurrencyStore } from '../display-currency/store.ts';

import { usePortfolioTrendStore } from './store.ts';
import {
    TREND_RANGES,
    type TrendRange,
    type TrendRow,
    niceScale,
    linePath,
    segmentPaths,
    nearestIndex,
    changeOver
} from './lib.ts';

const props = defineProps<{
    compact?: boolean;
}>();

const { tt, formatAmountToLocalizedNumeralsWithCurrency, getCurrentLanguageTag } = useI18n();

const store = usePortfolioTrendStore();
const settingsStore = useSettingsStore();
const displayCurrencyStore = useDisplayCurrencyStore();

const titleId = `ebk-trend-${useId()}`;
const box = ref<HTMLElement | null>(null);
const svg = ref<SVGSVGElement | null>(null);
const width = ref<number>(0);
const hover = ref<number>(-1);

const height = computed<number>(() => props.compact ? 180 : 260);
const showAmounts = computed<boolean>(() => settingsStore.appSettings.showAmountInHomePage);
const rows = computed<TrendRow[]>(() => store.rows);
const hovering = computed<boolean>(() => hover.value >= 0 && hover.value < rows.value.length);
const active = computed<TrendRow | undefined>(() => hovering.value ? rows.value[hover.value] : rows.value[rows.value.length - 1]);

const plot = computed(() => ({
    left: props.compact ? 40 : 52,
    right: Math.max(0, width.value - 10),
    top: 10,
    bottom: height.value - 26
}));

const scale = computed(() => {
    let max = 0;
    let min = Infinity;

    for (const row of rows.value) {
        max = Math.max(max, row.total, row.investments, row.invested);
        min = Math.min(min, row.total, row.investments, row.invested);
    }

    // The y axis starts at zero when the data is all positive, so that the height of the lines means something.
    return niceScale(min < 0 ? min : 0, max, props.compact ? 3 : 4);
});

const xRange = computed(() => {
    const first = rows.value[0]?.time ?? 0;
    const last = rows.value[rows.value.length - 1]?.time ?? first + 1;

    return { first, last: last === first ? first + 1 : last };
});

function xOf(time: number): number {
    const { first, last } = xRange.value;
    return plot.value.left + ((time - first) / (last - first)) * (plot.value.right - plot.value.left);
}

function yOf(value: number): number {
    const { min, max } = scale.value;
    return plot.value.bottom - ((value - min) / (max - min)) * (plot.value.bottom - plot.value.top);
}

const xs = computed<number[]>(() => rows.value.map(row => xOf(row.time)));
const totalPath = computed<string>(() => linePath(xs.value, rows.value.map(row => yOf(row.total))));
const costPath = computed<string>(() => linePath(xs.value, rows.value.map(row => yOf(row.invested))));
const investSegments = computed(() => segmentPaths(xs.value, rows.value.map(row => yOf(row.investments)), rows.value.map(row => row.estimated)));

const areaPath = computed<string>(() => {
    if (rows.value.length < 2) {
        return '';
    }

    const line = totalPath.value;
    const lastX = xs.value[xs.value.length - 1]!;
    const firstX = xs.value[0]!;

    return `${line} L${lastX} ${plot.value.bottom} L${firstX} ${plot.value.bottom} Z`;
});

const activeX = computed<number | null>(() => active.value ? xOf(active.value.time) : null);

function clampX(x: number): number {
    return Math.min(Math.max(x, plot.value.left), plot.value.right);
}

const dayFormat = computed(() => new Intl.DateTimeFormat(getCurrentLanguageTag(), { day: 'numeric', month: 'short', year: 'numeric' }));

function formatDay(time: number): string {
    return dayFormat.value.format(new Date(time * 1000));
}

const xTicks = computed(() => {
    if (rows.value.length < 2) {
        return [];
    }

    const { first, last } = xRange.value;
    const spanDays = (last - first) / 86400;
    const options: Intl.DateTimeFormatOptions = spanDays <= 200 ? { month: 'short', day: 'numeric' } : { month: 'short', year: 'numeric' };
    const format = new Intl.DateTimeFormat(getCurrentLanguageTag(), options);
    const count = props.compact ? 3 : (width.value < 520 ? 3 : 5);
    const ticks: { time: number, x: number, label: string, anchor: string }[] = [];

    for (let i = 0; i < count; i++) {
        const time = first + ((last - first) * i) / (count - 1);
        ticks.push({
            time,
            x: xOf(time),
            label: format.format(new Date(time * 1000)),
            anchor: i === 0 ? 'start' : (i === count - 1 ? 'end' : 'middle')
        });
    }

    return ticks;
});

const compactFormat = computed(() => new Intl.NumberFormat(getCurrentLanguageTag(), { notation: 'compact', maximumFractionDigits: 1 }));

function compactNumber(minor: number): string {
    return compactFormat.value.format(minor / 100);
}

function money(minor: number | undefined): string {
    if (minor === undefined) {
        return '-';
    }

    if (!showAmounts.value) {
        return '***';
    }

    return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(minor), displayCurrencyStore.currency);
}

const change = computed(() => changeOver(rows.value));

const changeClass = computed<string>(() => change.value.amount > 0 ? 'ebk-trend-up' : (change.value.amount < 0 ? 'ebk-trend-down' : ''));

const changeText = computed<string>(() => {
    if (!showAmounts.value) {
        return '***';
    }

    const amount = change.value.amount;
    const sign = amount > 0 ? '+' : (amount < 0 ? '-' : '');
    const text = `${sign}${money(Math.abs(amount))}`;
    const percent = change.value.percent;

    return percent === null ? text : `${text} (${sign}${Math.abs(percent).toFixed(1)}%)`;
});

const ariaLabel = computed<string>(() => {
    const last = rows.value[rows.value.length - 1];
    return `${tt('Portfolio trend')}: ${tt('Total assets')} ${last ? money(last.total) : '-'}`;
});

function rangeLabel(range: TrendRange): string {
    return range === 'ALL' ? tt('All') : tt(range);
}

function onPointer(event: PointerEvent): void {
    if (!svg.value || !rows.value.length) {
        return;
    }

    const rect = svg.value.getBoundingClientRect();
    hover.value = nearestIndex(xs.value, event.clientX - rect.left);
}

function onKey(event: KeyboardEvent): void {
    const count = rows.value.length;

    if (!count) {
        return;
    }

    const current = hover.value < 0 ? count - 1 : hover.value;

    if (event.key === 'ArrowLeft') {
        hover.value = Math.max(0, current - 1);
    } else if (event.key === 'ArrowRight') {
        hover.value = Math.min(count - 1, current + 1);
    } else if (event.key === 'Home') {
        hover.value = 0;
    } else if (event.key === 'End') {
        hover.value = count - 1;
    } else if (event.key === 'Escape') {
        hover.value = -1;
        return;
    } else {
        return;
    }

    event.preventDefault();
}

let observer: ResizeObserver | null = null;

function measure(): void {
    if (box.value) {
        width.value = Math.floor(box.value.clientWidth);
    }
}

onMounted(() => {
    measure();

    if (typeof ResizeObserver !== 'undefined' && box.value) {
        observer = new ResizeObserver(measure);
        observer.observe(box.value);
    }

    store.load();
});

onBeforeUnmount(() => {
    if (observer) {
        observer.disconnect();
    }
});

// A new display currency reloads the page, but the profile or the hidden-account setting can change in place.
watch(() => settingsStore.appSettings.overviewAccountFilterInHomePage, () => {
    hover.value = -1;
}, { deep: true });
</script>

<style>
/* Not scoped: the same card is used in the desktop (Vuetify) and the mobile (Framework7) apps. */
.ebk-trend {
    --ebk-trend-total: #b67443;
    --ebk-trend-invest: #00796b;
    --ebk-trend-up: 0, 121, 107;
    --ebk-trend-down: 200, 50, 50;
    --ebk-trend-grid: rgba(128, 128, 128, 0.22);
    --ebk-trend-muted: rgba(128, 128, 128, 0.95);

    color: inherit;
    padding: 1rem 1.25rem 1rem;
}

.v-theme--dark .ebk-trend,
.theme-dark .ebk-trend,
.dark .ebk-trend {
    --ebk-trend-total: #d9955c;
    --ebk-trend-invest: #4db6ac;
    --ebk-trend-up: 77, 182, 172;
    --ebk-trend-down: 239, 120, 120;
}

.ebk-trend-compact {
    padding: 0.75rem 1rem;
}

.ebk-numeric {
    font-variant-numeric: tabular-nums;
}

.ebk-trend-head {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-start;
    justify-content: space-between;
    gap: 0.75rem 1rem;
}

.ebk-trend-title {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--ebk-trend-muted);
}

.ebk-trend-total {
    margin-top: 0.125rem;
    font-size: 1.75rem;
    font-weight: 600;
    line-height: 1.2;
}

.ebk-trend-compact .ebk-trend-total {
    font-size: 1.5rem;
}

.ebk-trend-sub {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem 0.5rem;
    min-height: 1.25rem;
    font-size: 0.8125rem;
}

.ebk-trend-muted {
    color: var(--ebk-trend-muted);
}

.ebk-trend-up {
    color: rgb(var(--ebk-trend-up));
}

.ebk-trend-down {
    color: rgb(var(--ebk-trend-down));
}

.ebk-trend-ranges {
    display: inline-flex;
    gap: 0.25rem;
}

.ebk-trend-range {
    min-width: 2.75rem;
    min-height: 2rem;
    padding: 0 0.625rem;
    border: 1px solid var(--ebk-trend-grid);
    border-radius: 1rem;
    background: transparent;
    color: inherit;
    font: inherit;
    font-size: 0.8125rem;
    cursor: pointer;
}

.ebk-trend-range:hover:not(:disabled) {
    border-color: var(--ebk-trend-total);
}

.ebk-trend-range:focus-visible,
.ebk-trend-svg:focus-visible {
    outline: 2px solid var(--ebk-trend-total);
    outline-offset: 2px;
}

.ebk-trend-range-on {
    border-color: var(--ebk-trend-total);
    background: var(--ebk-trend-total);
    color: #fff;
}

.ebk-trend-range:disabled {
    cursor: progress;
    opacity: 0.7;
}

.ebk-trend-chart {
    position: relative;
    margin-top: 0.75rem;
}

.ebk-trend-svg {
    display: block;
    touch-action: pan-y;
    transition: opacity 0.2s ease;
}

.ebk-trend-busy {
    opacity: 0.55;
}

.ebk-trend-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    height: 100%;
    text-align: center;
    color: var(--ebk-trend-muted);
}

.ebk-trend-skeleton {
    height: 100%;
    border-radius: 0.5rem;
    background: linear-gradient(90deg, var(--ebk-trend-grid), transparent, var(--ebk-trend-grid));
    background-size: 200% 100%;
    animation: ebk-trend-shimmer 1.4s linear infinite;
}

@keyframes ebk-trend-shimmer {
    from { background-position: 200% 0; }
    to { background-position: -200% 0; }
}

.ebk-trend-grid line {
    stroke: var(--ebk-trend-grid);
    stroke-width: 1;
    shape-rendering: crispEdges;
}

.ebk-trend-axis {
    fill: var(--ebk-trend-muted);
    font-size: 11px;
}

.ebk-trend-area {
    fill: var(--ebk-trend-total);
    opacity: 0.1;
}

.ebk-trend-line {
    fill: none;
    stroke-linecap: round;
    stroke-linejoin: round;
}

.ebk-trend-line-total {
    stroke: var(--ebk-trend-total);
    stroke-width: 2.5;
}

.ebk-trend-line-invest {
    stroke: var(--ebk-trend-invest);
    stroke-width: 2;
}

.ebk-trend-line-estimate {
    stroke-dasharray: 5 4;
}

.ebk-trend-line-cost {
    stroke: var(--ebk-trend-muted);
    stroke-width: 1.5;
    stroke-dasharray: 2 4;
}

.ebk-trend-cursor {
    stroke: var(--ebk-trend-muted);
    stroke-width: 1;
}

.ebk-trend-dot {
    stroke: #fff;
    stroke-width: 2;
}

.ebk-trend-dot-total {
    fill: var(--ebk-trend-total);
}

.ebk-trend-dot-invest {
    fill: var(--ebk-trend-invest);
}

.ebk-trend-legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem 1.5rem;
    margin: 0.75rem 0 0;
    padding: 0;
    list-style: none;
    font-size: 0.8125rem;
}

.ebk-trend-legend li {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
}

.ebk-trend-legend-label {
    color: var(--ebk-trend-muted);
}

.ebk-trend-key {
    display: inline-block;
    width: 1rem;
    height: 0;
    border-top: 3px solid var(--ebk-trend-total);
}

.ebk-trend-key-invest {
    border-top-color: var(--ebk-trend-invest);
}

.ebk-trend-key-cost {
    border-top: 2px dotted var(--ebk-trend-muted);
}

.ebk-trend-note {
    margin: 0.5rem 0 0;
    font-size: 0.75rem;
    color: var(--ebk-trend-muted);
}

@media (prefers-reduced-motion: reduce) {
    .ebk-trend-skeleton {
        animation: none;
    }

    .ebk-trend-svg {
        transition: none;
    }
}
</style>
