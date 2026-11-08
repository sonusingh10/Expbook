<template>
    <f7-page ptr @ptr:refresh="reload" @page:afterin="onPageAfterIn">
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': loading }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="tt('Investments')"></f7-nav-title>
            <f7-nav-right :class="{ 'navbar-compact-icons': true, 'disabled': loading }">
                <f7-link icon-f7="ellipsis" :aria-label="tt('More')" @click="showMoreActionSheet = true"></f7-link>
                <f7-link icon-f7="plus" :aria-label="tt('Add Trade')" :class="{ 'disabled': !securities.length }" @click="addTrade()"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <!-- error -->
        <f7-block class="text-align-center" v-if="loadError && !loading">
            <p>{{ loadError }}</p>
            <f7-button small outline @click="reload()">{{ tt('Retry') }}</f7-button>
        </f7-block>

        <!-- empty -->
        <f7-block class="text-align-center margin-top" v-else-if="!loading && !securities.length">
            <f7-icon f7="chart_line_uptrend_xyaxis" size="40" class="margin-bottom" style="opacity: 0.5"></f7-icon>
            <h2 class="no-margin-top">{{ tt('Track what you own') }}</h2>
            <p class="ebk-mobile-empty-text">{{ tt('Add a security and record your buys, sells and dividends. Holdings, gains and allocation are calculated for you.') }}</p>
            <f7-button fill round @click="addSecurity">{{ tt('Add Security') }}</f7-button>
        </f7-block>

        <template v-else>
            <!-- totals -->
            <f7-list strong inset dividers class="margin-vertical-half">
                <f7-list-item class="list-item-with-header-and-title list-item-no-item-after" :header="tt('Current value')">
                    <template #title>
                        <span class="ebk-mobile-large ebk-mobile-numeric" v-if="summary">{{ money(summary.totals.value) }}</span>
                        <span v-else>…</span>
                    </template>
                </f7-list-item>
                <f7-list-item :title="tt('Invested')" :after="summary ? money(summary.totals.invested) : ''"></f7-list-item>
                <f7-list-item :title="tt('Unrealised gain')">
                    <template #after>
                        <span class="ebk-mobile-numeric" :class="signClass(summary ? summary.totals.unrealised : 0)" v-if="summary">
                            {{ signedMoney(summary.totals.unrealised) }}<template v-if="unrealisedPercent !== null"> ({{ formatSignedPercent(unrealisedPercent, false) }})</template>
                        </span>
                    </template>
                </f7-list-item>
                <f7-list-item :title="tt('Realised gain and dividends')">
                    <template #after>
                        <span class="ebk-mobile-numeric" :class="signClass(realisedAndDividends)" v-if="summary">{{ signedMoney(realisedAndDividends) }}</span>
                    </template>
                </f7-list-item>
                <f7-list-item :title="tt('Annualised return (XIRR)')">
                    <template #after>
                        <span class="ebk-mobile-numeric" v-if="summary">{{ formatSignedPercent(summary.totals.xirr, true) }}</span>
                    </template>
                </f7-list-item>
            </f7-list>

            <f7-block class="no-margin-vertical ebk-mobile-warning" v-if="summary && !summary.ratesAvailable">
                <small>{{ tt('Exchange rates are unavailable right now, so holdings in other currencies are left out of the totals.') }}</small>
            </f7-block>
            <f7-block class="no-margin-vertical ebk-mobile-warning" v-else-if="summary && summary.unconvertedCurrencies.length">
                <small>{{ tt('No exchange rate for') }} {{ summary.unconvertedCurrencies.join(', ') }}. {{ tt('These holdings are left out of the totals.') }}</small>
            </f7-block>
            <f7-block class="no-margin-vertical ebk-mobile-warning" v-if="missingPriceCount > 0">
                <small>
                    <template v-if="missingPriceCount === 1">{{ tt('One holding has no price yet, so its value and gain are not counted. Set a price or choose an online price source.') }}</template>
                    <template v-else>{{ missingPriceCount }} {{ tt('holdings have no price yet, so their value is not counted. Set a price or choose an online price source.') }}</template>
                </small>
            </f7-block>

            <f7-segmented strong tag="p" class="margin-horizontal margin-top">
                <f7-button :active="activeTab === 'holdings'" @click="activeTab = 'holdings'">{{ tt('Holdings') }}</f7-button>
                <f7-button :active="activeTab === 'trades'" @click="activeTab = 'trades'">{{ tt('Trades') }}</f7-button>
            </f7-segmented>

            <!-- holdings -->
            <f7-list media-list strong inset dividers class="margin-vertical" v-if="activeTab === 'holdings'">
                <f7-list-item v-if="summary && !holdings.length" :title="tt('No holdings yet')"></f7-list-item>
                <f7-list-item link="#" no-chevron
                              :key="holding.securityId"
                              v-for="holding in holdings"
                              @click="openHoldingActions(holding.securityId)">
                    <template #title>{{ holding.name }}</template>
                    <template #subtitle>
                        <span class="ebk-mobile-numeric">{{ holding.symbol }} · {{ qty(holding.quantity) }}</span>
                    </template>
                    <template #after>
                        <div class="text-align-right ebk-mobile-numeric">
                            <div v-if="holding.priceAvailable && isHeld(holding)">{{ money(holding.value, holding.currency) }}</div>
                            <div v-else-if="isHeld(holding)"><small class="ebk-mobile-warning">{{ tt('Set price') }}</small></div>
                            <small :class="signClass(holding.unrealised)" v-if="holding.priceAvailable && isHeld(holding)">
                                {{ signedMoney(holding.unrealised, holding.currency) }} · {{ formatSignedPercent(holding.unrealisedPercent, false) }}
                            </small>
                            <small class="ebk-mobile-warning" v-else-if="holding.warning === 'no_exchange_rate'">{{ tt('No exchange rate') }}</small>
                        </div>
                    </template>
                    <template #text>
                        <small class="ebk-mobile-numeric" :class="isStale(holding) ? 'ebk-mobile-warning' : 'ebk-mobile-muted'" v-if="holding.priceAvailable">
                            {{ tt('Price') }} {{ price(holding.price) }} {{ holding.currency }} · {{ isStale(holding) ? tt('Stale') + ' · ' : '' }}{{ formatDate(holding.priceTime) }}
                        </small>
                    </template>
                </f7-list-item>
            </f7-list>

            <!-- trades -->
            <f7-list media-list strong inset dividers class="margin-vertical" v-if="activeTab === 'trades'">
                <f7-list-item v-if="!loading && !trades.length" :title="tt('No trades yet')"></f7-list-item>
                <f7-list-item swipeout link="#" no-chevron
                              :id="'trade_' + trade.id"
                              :key="trade.id"
                              v-for="trade in trades"
                              @click="editTrade(trade.id)">
                    <template #title>{{ tradeTypeName(trade.tradeType) }} · {{ securityMap[trade.securityId]?.symbol }}</template>
                    <template #subtitle>
                        <span class="ebk-mobile-numeric">{{ formatDate(trade.time) }}</span>
                    </template>
                    <template #after>
                        <span class="ebk-mobile-numeric">{{ tradeAmount(trade) }}</span>
                    </template>
                    <template #text>
                        <small class="ebk-mobile-numeric ebk-mobile-muted" v-if="trade.tradeType !== 3">{{ qty(trade.quantity) }} × {{ price(trade.price) }}</small>
                    </template>
                    <f7-swipeout-actions :left="textDirection === TextDirection.RTL" :right="textDirection === TextDirection.LTR">
                        <f7-swipeout-button color="red" class="padding-horizontal" :aria-label="tt('Delete')" @click="confirmRemoveTrade(trade.id)">
                            <f7-icon f7="trash"></f7-icon>
                        </f7-swipeout-button>
                    </f7-swipeout-actions>
                </f7-list-item>
            </f7-list>
        </template>

        <f7-actions close-by-outside-click close-on-escape :opened="showMoreActionSheet" @actions:closed="showMoreActionSheet = false">
            <f7-actions-group>
                <f7-actions-button @click="addSecurity">{{ tt('Add Security') }}</f7-actions-button>
                <f7-actions-button :class="{ 'disabled': !hasRefreshableSecurities || refreshing }" @click="refreshPrices">{{ tt('Refresh prices') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>

        <f7-actions close-by-outside-click close-on-escape :opened="showHoldingActionSheet" @actions:closed="showHoldingActionSheet = false">
            <f7-actions-group>
                <f7-actions-label>{{ selectedSecurity ? selectedSecurity.name : '' }}</f7-actions-label>
                <f7-actions-button @click="addTrade(selectedSecurityId)">{{ tt('Add Trade') }}</f7-actions-button>
                <f7-actions-button @click="setPrice">{{ tt('Set price') }}</f7-actions-button>
                <f7-actions-button @click="editSecurity">{{ tt('Edit Security') }}</f7-actions-button>
                <f7-actions-button color="red" @click="confirmRemoveSecurity">{{ tt('Delete') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>

        <f7-actions close-by-outside-click close-on-escape :opened="showDeleteActionSheet" @actions:closed="showDeleteActionSheet = false">
            <f7-actions-group>
                <f7-actions-label>{{ deleteMessage }}</f7-actions-label>
                <f7-actions-button color="red" @click="runDelete">{{ tt('Delete') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { storeToRefs } from 'pinia';
import Decimal from 'decimal.js';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';

import { useInvestmentStore } from '@/stores/investment.ts';

import { TextDirection } from '@/core/text.ts';
import {
    TRADE_TYPES,
    type InvestmentHolding,
    type InvestmentTradeInfoResponse
} from '@/models/investment.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';
import {
    getValueSign,
    withExplicitSign,
    formatSignedPercent,
    formatPrice,
    formatQuantity,
    unrealisedPercentOfPricedCost,
    isPriceStale,
    isValidQuantityOrPrice
} from '@/lib/investments.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt, getCurrentLanguageTextDirection, formatAmountToLocalizedNumeralsWithCurrency, formatNumberToLocalizedNumerals, getCurrentDecimalSeparator } = useI18n();
const { showToast, showPrompt } = useI18nUIComponents();

const investmentStore = useInvestmentStore();
const { securities, summary, trades, securityMap } = storeToRefs(investmentStore);

const loading = ref<boolean>(true);
const refreshing = ref<boolean>(false);
const loadError = ref<string>('');
const activeTab = ref<string>('holdings');
const showMoreActionSheet = ref<boolean>(false);
const showHoldingActionSheet = ref<boolean>(false);
const showDeleteActionSheet = ref<boolean>(false);
const selectedSecurityId = ref<string>('');
const deleteMessage = ref<string>('');
let pendingDelete: (() => void) | null = null;

const textDirection = computed<TextDirection>(() => getCurrentLanguageTextDirection());
const holdings = computed<InvestmentHolding[]>(() => summary.value ? summary.value.holdings : []);
const selectedSecurity = computed(() => securityMap.value[selectedSecurityId.value]);
const hasRefreshableSecurities = computed<boolean>(() => securities.value.some(s => s.priceSource && s.priceSource !== 'manual'));
const realisedAndDividends = computed<number>(() => summary.value ? summary.value.totals.realised + summary.value.totals.dividends : 0);
const missingPriceCount = computed<number>(() => holdings.value.filter(h => h.warning === 'no_price').length);
const unrealisedPercent = computed<number | null>(() => {
    const totals = summary.value?.totals;
    return totals ? unrealisedPercentOfPricedCost(totals.unrealised, totals.pricedCost) : null;
});

// Prices and average costs read as 2 to 4 decimals, quantities keep their fractions; stored values are unchanged.
function price(value: string): string {
    return formatPrice(value, number => formatNumberToLocalizedNumerals(number), getCurrentDecimalSeparator());
}

function qty(value: string): string {
    return formatQuantity(value, number => formatNumberToLocalizedNumerals(number), getCurrentDecimalSeparator());
}

function money(minor: number, currency?: string): string {
    return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(minor), currency || summary.value?.baseCurrency);
}

function signedMoney(minor: number, currency?: string): string {
    return withExplicitSign(money(minor, currency), minor);
}

function signClass(minor: number): string {
    const sign = getValueSign(minor);
    return sign === 'positive' ? 'text-income' : (sign === 'negative' ? 'text-expense' : '');
}

function isHeld(holding: InvestmentHolding): boolean {
    return holding.quantity !== '0' && holding.quantity !== '';
}

function isStale(holding: InvestmentHolding): boolean {
    return isHeld(holding) && isPriceStale(holding.priceTime, Math.floor(Date.now() / 1000));
}

function formatDate(unixTime: number): string {
    return unixTime ? new Date(unixTime * 1000).toISOString().slice(0, 10) : '-';
}

function tradeTypeName(value: number): string {
    return tt(TRADE_TYPES.find(item => item.value === value)?.name || 'Buy');
}

function tradeAmount(trade: InvestmentTradeInfoResponse): string {
    const currency = securityMap.value[trade.securityId]?.currency;

    if (trade.tradeType === 3) {
        return money(trade.amount, currency);
    }

    const gross = new Decimal(trade.quantity).times(trade.price).times(100).toDecimalPlaces(0, Decimal.ROUND_HALF_UP).toNumber();

    return money(gross, currency);
}

function reload(done?: () => void): void {
    loading.value = true;
    loadError.value = '';

    if (!done) {
        showLoading();
    }

    investmentStore.reloadAll().then(() => {
        done?.();
        loading.value = false;
        hideLoading();
    }).catch(error => {
        done?.();
        loading.value = false;
        hideLoading();
        loadError.value = tt(error && error.message ? error.message : 'Unable to retrieve portfolio summary');
    });
}

function onPageAfterIn(): void {
    // coming back from an edit page: refresh quietly
    if (!loading.value) {
        investmentStore.reloadAll().catch(() => undefined);
    }
}

function refreshPrices(): void {
    refreshing.value = true;
    showLoading();

    investmentStore.refreshPrices().then(results => {
        refreshing.value = false;
        hideLoading();
        const failed = results.filter(r => !r.ok);
        showToast(failed.length ? tt('Some prices could not be updated') + ': ' + failed.map(r => r.symbol).join(', ') : 'Prices have been updated');
    }).catch(error => {
        refreshing.value = false;
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function addSecurity(): void {
    props.f7router.navigate('/investment/security/edit');
}

function editSecurity(): void {
    props.f7router.navigate('/investment/security/edit?id=' + selectedSecurityId.value);
}

function addTrade(securityId?: string): void {
    props.f7router.navigate('/investment/trade/edit' + (securityId ? '?securityId=' + securityId : ''));
}

function editTrade(tradeId: string): void {
    props.f7router.navigate('/investment/trade/edit?id=' + tradeId);
}

function openHoldingActions(securityId: string): void {
    selectedSecurityId.value = securityId;
    showHoldingActionSheet.value = true;
}

function setPrice(): void {
    const security = selectedSecurity.value;

    if (!security) {
        return;
    }

    showPrompt(tt('Latest price per unit') + ' (' + security.currency + ')', security.latestPrice || '', value => {
        const price = value.trim();

        if (!isValidQuantityOrPrice(price)) {
            showToast('Enter a positive number with up to 8 decimals, no thousands separators');
            return;
        }

        showLoading();

        investmentStore.setSecurityPrice(security.id, price).then(() => {
            hideLoading();
        }).catch(error => {
            hideLoading();

            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    });
}

function confirmRemoveSecurity(): void {
    const id = selectedSecurityId.value;
    deleteMessage.value = tt('Are you sure you want to delete this security? Securities with trades cannot be deleted.');
    pendingDelete = () => {
        showLoading();

        investmentStore.deleteSecurity(id).then(() => {
            hideLoading();
            showToast('Security has been deleted');
        }).catch(error => {
            hideLoading();

            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    };
    showDeleteActionSheet.value = true;
}

function confirmRemoveTrade(tradeId: string): void {
    deleteMessage.value = tt('Are you sure you want to delete this trade?');
    pendingDelete = () => {
        showLoading();

        investmentStore.deleteTrade(tradeId).then(() => {
            hideLoading();
            showToast('Trade has been deleted');
        }).catch(error => {
            hideLoading();

            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    };
    showDeleteActionSheet.value = true;
}

function runDelete(): void {
    showDeleteActionSheet.value = false;
    pendingDelete?.();
    pendingDelete = null;
}

reload();
</script>

<style scoped>
.ebk-mobile-numeric {
    font-variant-numeric: tabular-nums;
}

.ebk-mobile-muted {
    color: var(--ebk-muted-text-color);
}

.ebk-mobile-warning {
    color: var(--ebk-warning-text-color);
}

.ebk-mobile-large {
    font-size: 1.75rem;
    font-weight: 600;
    line-height: 1.2;
}

.ebk-mobile-empty-text {
    max-width: 36ch;
    margin-inline: auto;
}
</style>
