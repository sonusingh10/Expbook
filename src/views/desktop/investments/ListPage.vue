<template>
    <main-page-layout>
        <template #content>
            <v-row class="match-height">
                <v-col cols="12">
                    <v-card variant="flat">
                        <template #title>
                            <div class="title-and-toolbar d-flex align-center flex-wrap ga-2">
                                <span>{{ tt('Investments') }}</span>
                                <v-btn density="compact" color="default" variant="text" class="ms-1"
                                       :aria-label="tt('Refresh prices')" :icon="true"
                                       :disabled="loading || refreshing || !hasRefreshableSecurities"
                                       :loading="refreshing" @click="refreshPrices">
                                    <template #loader>
                                        <v-progress-circular indeterminate size="20"/>
                                    </template>
                                    <v-icon :icon="mdiRefresh" size="24" />
                                    <v-tooltip activator="parent">{{ hasRefreshableSecurities ? tt('Refresh prices') : tt('No securities use an online price source') }}</v-tooltip>
                                </v-btn>
                                <v-spacer />
                                <v-btn variant="text" color="default" :prepend-icon="mdiPlus" :disabled="loading" @click="addSecurity">{{ tt('Add Security') }}</v-btn>
                                <v-btn variant="outlined" color="default" :prepend-icon="mdiFileUploadOutline" :disabled="loading" @click="importCsv">{{ tt('Import CSV') }}</v-btn>
                                <v-btn variant="flat" color="primary" :prepend-icon="mdiPlus" :disabled="loading || !securities.length" @click="addTrade()">{{ tt('Add Trade') }}</v-btn>
                            </div>
                        </template>

                        <v-card-text class="pt-0">
                            <v-alert type="error" variant="tonal" class="mb-4" v-if="loadError">
                                <div class="d-flex align-center">
                                    <span>{{ loadError }}</span>
                                    <v-spacer />
                                    <v-btn variant="text" size="small" @click="reload">{{ tt('Retry') }}</v-btn>
                                </div>
                            </v-alert>

                            <!-- empty state -->
                            <div class="ebk-empty text-center py-12" v-if="!loading && !loadError && !securities.length">
                                <v-icon :icon="mdiChartLine" size="40" class="mb-3 text-medium-emphasis" />
                                <h2 class="text-title-large mb-2">{{ tt('Track what you own') }}</h2>
                                <p class="text-body-medium text-medium-emphasis mx-auto mb-5 ebk-empty-text">
                                    {{ tt('Add a security and record your buys, sells and dividends, or import them from a CSV file. Holdings, gains and allocation are calculated for you.') }}
                                </p>
                                <v-btn color="primary" variant="flat" class="me-2" :prepend-icon="mdiPlus" @click="addSecurity">{{ tt('Add Security') }}</v-btn>
                                <v-btn color="default" variant="outlined" :prepend-icon="mdiFileUploadOutline" @click="importCsv">{{ tt('Import CSV') }}</v-btn>
                            </div>

                            <template v-else-if="loading || securities.length">
                                <portfolio-trend-card class="ebk-trend-in-page" /> <!-- Expbook -->

                                <!-- totals -->
                                <!-- one flat strip of figures: no card inside the page card -->
                                <div class="ebk-totals mb-4">
                                    <div class="ebk-total" v-for="card in totalCards" :key="card.key">
                                        <div class="text-label-medium text-medium-emphasis">{{ card.label }}</div>
                                        <v-skeleton-loader type="text" class="skeleton-no-margin mt-2" v-if="loading && !summary" />
                                        <template v-else>
                                            <div class="ebk-numeric mt-1" :class="[card.large ? 'text-headline-small' : 'text-title-large', card.className]">
                                                <v-icon size="18" :icon="card.icon" class="me-1" v-if="card.icon" />{{ card.value }}
                                            </div>
                                            <div class="text-body-small text-medium-emphasis ebk-numeric" v-if="card.sub">{{ card.sub }}</div>
                                        </template>
                                    </div>
                                </div>

                                <v-alert type="warning" color="warning-strong" variant="tonal" density="compact" class="mb-3" v-if="summary && !summary.ratesAvailable">
                                    {{ tt('Exchange rates are unavailable right now, so holdings in other currencies are left out of the totals.') }}
                                </v-alert>
                                <v-alert type="warning" color="warning-strong" variant="tonal" density="compact" class="mb-3" v-else-if="summary && summary.unconvertedCurrencies.length">
                                    {{ tt('No exchange rate for') }} {{ summary.unconvertedCurrencies.join(', ') }}. {{ tt('These holdings are left out of the totals.') }}
                                </v-alert>
                                <v-alert type="warning" color="warning-strong" variant="tonal" density="compact" class="mb-3" v-if="missingPriceCount > 0">
                                    <template v-if="missingPriceCount === 1">{{ tt('One holding has no price yet, so its value and gain are not counted. Set a price or choose an online price source.') }}</template>
                                    <template v-else>{{ missingPriceCount }} {{ tt('holdings have no price yet, so their value is not counted. Set a price or choose an online price source.') }}</template>
                                </v-alert>

                                <v-tabs class="mb-4" v-model="activeTab">
                                    <v-tab value="holdings">{{ tt('Holdings') }}</v-tab>
                                    <v-tab value="trades">{{ tt('Trades') }}</v-tab>
                                    <v-tab value="allocation">{{ tt('Allocation') }}</v-tab>
                                </v-tabs>

                                <v-window class="disable-tab-transition" v-model="activeTab">
                                    <!-- holdings -->
                                    <v-window-item value="holdings">
                                        <v-table class="ebk-table" density="comfortable" :hover="!loading">
                                            <thead>
                                            <tr>
                                                <th>{{ tt('Security') }}</th>
                                                <th class="text-end">{{ tt('Quantity') }}</th>
                                                <th class="text-end">{{ tt('Average cost') }}</th>
                                                <th class="text-end">{{ tt('Price') }}</th>
                                                <th class="text-end">{{ tt('Value') }}</th>
                                                <th class="text-end">{{ tt('Gain') }}</th>
                                                <th class="text-end">{{ tt('XIRR') }}</th>
                                                <th class="ebk-actions-col"><span class="sr-only">{{ tt('Actions') }}</span></th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            <tr :key="n" v-for="n in (loading && !summary ? [1, 2, 3, 4] : [])">
                                                <td colspan="8"><v-skeleton-loader type="text" /></td>
                                            </tr>
                                            <tr v-if="!loading && summary && !summary.holdings.length">
                                                <td colspan="8" class="text-medium-emphasis">{{ tt('No holdings yet') }}</td>
                                            </tr>
                                            <tr :key="holding.securityId" v-for="holding in holdings">
                                                <td>
                                                    <div class="font-weight-medium">{{ holding.name }}</div>
                                                    <div class="text-body-small text-medium-emphasis">
                                                        {{ holding.symbol }} · {{ typeName(holding.securityType) }} · {{ holding.currency }}
                                                    </div>
                                                </td>
                                                <td class="text-end ebk-numeric">{{ qty(holding.quantity) }}</td>
                                                <td class="text-end ebk-numeric">{{ isHeld(holding) ? price(holding.averageCost) : '-' }}</td>
                                                <td class="text-end ebk-numeric">
                                                    <template v-if="holding.priceAvailable">
                                                        <div>{{ price(holding.price) }}</div>
                                                        <div class="text-body-small" :class="isStale(holding) ? 'text-warning-strong' : 'text-medium-emphasis'">
                                                            {{ isStale(holding) ? tt('Stale') + ' · ' : '' }}{{ formatDate(holding.priceTime) }}
                                                        </div>
                                                    </template>
                                                    <v-btn size="small" variant="text" color="default" v-else-if="isHeld(holding)" @click="setPrice(holding.securityId)">{{ tt('Set price') }}</v-btn>
                                                    <span v-else>-</span>
                                                </td>
                                                <td class="text-end ebk-numeric">
                                                    <template v-if="holding.priceAvailable && isHeld(holding)">{{ money(holding.value, holding.currency) }}</template>
                                                    <span v-else>-</span>
                                                    <div class="text-body-small text-warning-strong" v-if="holding.warning === 'no_exchange_rate'">{{ tt('No exchange rate') }}</div>
                                                </td>
                                                <td class="text-end ebk-numeric">
                                                    <template v-if="holding.priceAvailable && isHeld(holding)">
                                                        <div :class="signClass(holding.unrealised)">
                                                            <v-icon size="14" :icon="signIcon(holding.unrealised)" v-if="signIcon(holding.unrealised)" />
                                                            {{ signedMoney(holding.unrealised, holding.currency) }}
                                                        </div>
                                                        <div class="text-body-small" :class="signClass(holding.unrealised)">{{ formatSignedPercent(holding.unrealisedPercent, false) }}</div>
                                                    </template>
                                                    <span v-else>-</span>
                                                </td>
                                                <td class="text-end ebk-numeric">{{ formatSignedPercent(holding.xirr, true) }}</td>
                                                <td class="text-end">
                                                    <v-btn density="comfortable" variant="text" color="default" :icon="true" :aria-label="tt('Actions')">
                                                        <v-icon :icon="mdiDotsVertical" size="20" />
                                                        <v-menu activator="parent">
                                                            <v-list density="compact">
                                                                <v-list-item :prepend-icon="mdiPlus" :title="tt('Add Trade')" @click="addTrade(holding.securityId)" />
                                                                <v-list-item :prepend-icon="mdiCurrencyUsd" :title="tt('Set price')" @click="setPrice(holding.securityId)" />
                                                                <v-list-item :prepend-icon="mdiPencilOutline" :title="tt('Edit Security')" @click="editSecurity(holding.securityId)" />
                                                                <v-list-item :prepend-icon="mdiDeleteOutline" :title="tt('Delete')" @click="removeSecurity(holding.securityId)" />
                                                            </v-list>
                                                        </v-menu>
                                                    </v-btn>
                                                </td>
                                            </tr>
                                            </tbody>
                                        </v-table>
                                        <p class="text-body-small text-medium-emphasis mt-3">
                                            {{ tt('Average cost includes fees. Gains are unrealised and shown in each security\'s own currency; totals are converted to') }} {{ displayCurrencyCode }}.
                                        </p>
                                    </v-window-item>

                                    <!-- trades -->
                                    <v-window-item value="trades">
                                        <v-table class="ebk-table" density="comfortable" :hover="!loading">
                                            <thead>
                                            <tr>
                                                <th>{{ tt('Date') }}</th>
                                                <th>{{ tt('Security') }}</th>
                                                <th>{{ tt('Type') }}</th>
                                                <th class="text-end">{{ tt('Quantity') }}</th>
                                                <th class="text-end">{{ tt('Price') }}</th>
                                                <th class="text-end">{{ tt('Amount') }}</th>
                                                <th class="text-end">{{ tt('Fees') }}</th>
                                                <th class="ebk-actions-col"><span class="sr-only">{{ tt('Actions') }}</span></th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            <tr v-if="!loading && !trades.length">
                                                <td colspan="8" class="text-medium-emphasis">{{ tt('No trades yet') }}</td>
                                            </tr>
                                            <tr :key="trade.id" v-for="trade in trades">
                                                <td class="ebk-numeric">{{ formatDate(trade.time) }}</td>
                                                <td>
                                                    <div>{{ securityMap[trade.securityId]?.name }}</div>
                                                    <div class="text-body-small text-medium-emphasis">{{ securityMap[trade.securityId]?.symbol }}</div>
                                                </td>
                                                <td>{{ tradeTypeName(trade.tradeType) }}</td>
                                                <td class="text-end ebk-numeric">{{ trade.tradeType === 3 ? '-' : qty(trade.quantity) }}</td>
                                                <td class="text-end ebk-numeric">{{ trade.tradeType === 3 ? '-' : price(trade.price) }}</td>
                                                <td class="text-end ebk-numeric">{{ tradeAmount(trade) }}</td>
                                                <td class="text-end ebk-numeric">{{ trade.fees ? money(trade.fees, securityMap[trade.securityId]?.currency) : '-' }}</td>
                                                <td class="text-end text-no-wrap">
                                                    <v-btn density="comfortable" variant="text" color="default" :icon="true" :aria-label="tt('Edit')" @click="editTrade(trade)">
                                                        <v-icon :icon="mdiPencilOutline" size="20" />
                                                        <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                                    </v-btn>
                                                    <v-btn density="comfortable" variant="text" color="default" :icon="true" :aria-label="tt('Delete')" @click="removeTrade(trade)">
                                                        <v-icon :icon="mdiDeleteOutline" size="20" />
                                                        <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
                                                    </v-btn>
                                                </td>
                                            </tr>
                                            </tbody>
                                        </v-table>
                                    </v-window-item>

                                    <!-- allocation -->
                                    <v-window-item value="allocation">
                                        <v-row>
                                            <v-col cols="12" md="6">
                                                <h3 class="text-title-medium mb-3">{{ tt('By type') }}</h3>
                                                <p class="text-body-medium text-medium-emphasis" v-if="!allocationByType.length">{{ tt('Allocation appears once your holdings have prices.') }}</p>
                                                <div class="ebk-bar-row" :key="item.key" v-for="item in allocationByType">
                                                    <div class="d-flex text-body-medium">
                                                        <span>{{ allocationTypeName(item.key) }}</span>
                                                        <v-spacer />
                                                        <span class="ebk-numeric">{{ baseMoney(item.value) }} · {{ item.percent.toFixed(2) }}%</span>
                                                    </div>
                                                    <div class="ebk-bar"><div class="ebk-bar-fill" :style="{ width: item.percent + '%' }"></div></div>
                                                </div>
                                            </v-col>
                                            <v-col cols="12" md="6">
                                                <h3 class="text-title-medium mb-3">{{ tt('By currency') }}</h3>
                                                <p class="text-body-medium text-medium-emphasis" v-if="!allocationByCurrency.length">{{ tt('Allocation appears once your holdings have prices.') }}</p>
                                                <div class="ebk-bar-row" :key="item.key" v-for="item in allocationByCurrency">
                                                    <div class="d-flex text-body-medium">
                                                        <span>{{ item.key }}</span>
                                                        <v-spacer />
                                                        <span class="ebk-numeric">{{ baseMoney(item.value) }} · {{ item.percent.toFixed(2) }}%</span>
                                                    </div>
                                                    <div class="ebk-bar"><div class="ebk-bar-fill" :style="{ width: item.percent + '%' }"></div></div>
                                                </div>
                                            </v-col>
                                        </v-row>
                                    </v-window-item>
                                </v-window>
                            </template>
                        </v-card-text>
                    </v-card>
                </v-col>
            </v-row>
        </template>
    </main-page-layout>

    <security-edit-dialog ref="securityDialog" />
    <trade-edit-dialog ref="tradeDialog" />
    <import-dialog ref="importDialog" />
    <price-dialog ref="priceDialog" />

    <confirm-dialog ref="confirmDialog" />
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';
import SecurityEditDialog from './dialogs/SecurityEditDialog.vue';
import PortfolioTrendCard from '@/expbook/portfolio-trend/PortfolioTrendCard.vue'; // Expbook
import TradeEditDialog from './dialogs/TradeEditDialog.vue';
import ImportDialog from './dialogs/ImportDialog.vue';
import PriceDialog from './dialogs/PriceDialog.vue';

import { computed, nextTick, ref, useTemplateRef } from 'vue';
import { storeToRefs } from 'pinia';
import Decimal from 'decimal.js';

import { useI18n } from '@/locales/helpers.ts';
import { useInvestmentDisplayCurrency } from '@/expbook/display-currency/investments.ts'; // Expbook


import { useInvestmentStore } from '@/stores/investment.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import {
    SECURITY_TYPES,
    TRADE_TYPES,
    ALLOCATION_TYPE_NAMES,
    type InvestmentHolding,
    type InvestmentTradeInfoResponse
} from '@/models/investment.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';
import {
    combineNetWorth,
    getValueSign,
    withExplicitSign,
    formatSignedPercent,
    formatPrice,
    formatQuantity,
    unrealisedPercentOfPricedCost,
    isPriceStale
} from '@/lib/investments.ts';

import {
    mdiRefresh,
    mdiPlus,
    mdiFileUploadOutline,
    mdiChartLine,
    mdiDotsVertical,
    mdiPencilOutline,
    mdiDeleteOutline,
    mdiCurrencyUsd,
    mdiArrowTopRight,
    mdiArrowBottomRight
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;
type SecurityDialogType = InstanceType<typeof SecurityEditDialog>;
type TradeDialogType = InstanceType<typeof TradeEditDialog>;
type ImportDialogType = InstanceType<typeof ImportDialog>;
type PriceDialogType = InstanceType<typeof PriceDialog>;

const { tt, formatAmountToLocalizedNumeralsWithCurrency, formatNumberToLocalizedNumerals, getCurrentDecimalSeparator } = useI18n();

const investmentStore = useInvestmentStore();
const accountsStore = useAccountsStore();
const settingsStore = useSettingsStore();
const exchangeRatesStore = useExchangeRatesStore();
const { securities, summary, trades, securityMap } = storeToRefs(investmentStore);

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');
const securityDialog = useTemplateRef<SecurityDialogType>('securityDialog');
const tradeDialog = useTemplateRef<TradeDialogType>('tradeDialog');
const importDialog = useTemplateRef<ImportDialogType>('importDialog');
const priceDialog = useTemplateRef<PriceDialogType>('priceDialog');

const activeTab = ref<string>('holdings');
const loading = ref<boolean>(true);
const refreshing = ref<boolean>(false);
const loadError = ref<string>('');

const holdings = computed<InvestmentHolding[]>(() => summary.value ? summary.value.holdings : []);
const allocationByType = computed(() => summary.value ? summary.value.allocationByType : []);
const allocationByCurrency = computed(() => summary.value ? summary.value.allocationByCurrency : []);
const hasRefreshableSecurities = computed<boolean>(() => securities.value.some(s => s.priceSource && s.priceSource !== 'manual'));
const missingPriceCount = computed<number>(() => holdings.value.filter(h => h.warning === 'no_price').length);

function money(minor: number, currency?: string): string {
    return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(minor), currency || summary.value?.baseCurrency);
}

// Prices and average costs read as 2 to 4 decimals, quantities keep their fractions; stored values are unchanged.
function price(value: string): string {
    return formatPrice(value, number => formatNumberToLocalizedNumerals(number), getCurrentDecimalSeparator());
}

function qty(value: string): string {
    return formatQuantity(value, number => formatNumberToLocalizedNumerals(number), getCurrentDecimalSeparator());
}

function signedMoney(minor: number, currency?: string): string {
    return withExplicitSign(money(minor, currency), minor);
}

// Expbook: totals come from the server in the profile currency; show them in the display currency.
const { toDisplay, displayCurrencyCode } = useInvestmentDisplayCurrency();

function baseMoney(minor: number): string {
    const shown = toDisplay(minor, summary.value?.baseCurrency);
    return shown ? money(shown.minor, shown.currency) : money(minor);
}

function signedBaseMoney(minor: number): string {
    return withExplicitSign(baseMoney(minor), minor);
}

function signClass(minor: number): string {
    const sign = getValueSign(minor);
    return sign === 'positive' ? 'text-income' : (sign === 'negative' ? 'text-expense' : '');
}

function signIcon(minor: number): string {
    const sign = getValueSign(minor);
    return sign === 'positive' ? mdiArrowTopRight : (sign === 'negative' ? mdiArrowBottomRight : '');
}

function isHeld(holding: InvestmentHolding): boolean {
    return holding.quantity !== '0' && holding.quantity !== '';
}

function isStale(holding: InvestmentHolding): boolean {
    return isHeld(holding) && isPriceStale(holding.priceTime, Math.floor(Date.now() / 1000));
}

// Dates are calendar dates (stored as midday UTC), so they are shown in UTC to keep the same day everywhere.
function formatDate(unixTime: number): string {
    return unixTime ? new Date(unixTime * 1000).toISOString().slice(0, 10) : '-';
}

function typeName(value: number): string {
    return tt(SECURITY_TYPES.find(item => item.value === value)?.name || 'Other');
}

function tradeTypeName(value: number): string {
    return tt(TRADE_TYPES.find(item => item.value === value)?.name || 'Buy');
}

function allocationTypeName(key: string): string {
    return tt(ALLOCATION_TYPE_NAMES[key] || 'Other');
}

function tradeAmount(trade: InvestmentTradeInfoResponse): string {
    const currency = securityMap.value[trade.securityId]?.currency;

    if (trade.tradeType === 3) {
        return money(trade.amount, currency);
    }

    // gross amount = quantity x price, rounded half up to minor units (display only; the server is the source of truth)
    const gross = new Decimal(trade.quantity).times(trade.price).times(100).toDecimalPlaces(0, Decimal.ROUND_HALF_UP).toNumber();

    return money(gross, currency);
}

// Net worth = account balances (converted to the default currency) + investment value.
const netWorth = computed<{ total: string, accounts: string }>(() => {
    if (!summary.value) {
        return { total: '-', accounts: '' };
    }

    const currency = displayCurrencyCode.value;
    const accountsNet = accountsStore.getNetAssets(settingsStore.appSettings.showAccountBalance, settingsStore.appSettings.totalAmountExcludeAccountIds);
    const invested = toDisplay(summary.value.totals.value, summary.value.baseCurrency);

    // Without a rate the investments cannot be added to the accounts, so no total is shown rather than a wrong one.
    if (!invested || invested.currency !== currency) {
        return { total: '-', accounts: formatAmountToLocalizedNumeralsWithCurrency(accountsNet, currency) };
    }

    const combined = combineNetWorth(accountsNet, parseBigDecimal(invested.minor));

    return {
        total: formatAmountToLocalizedNumeralsWithCurrency(combined, currency),
        accounts: formatAmountToLocalizedNumeralsWithCurrency(accountsNet, currency)
    };
});

const totalCards = computed(() => {
    const totals = summary.value ? summary.value.totals : null;
    const unrealisedPercent = totals ? unrealisedPercentOfPricedCost(totals.unrealised, totals.pricedCost) : null;
    const hasUnpriced = missingPriceCount.value > 0;

    return [
        {
            key: 'netWorth',
            label: tt('Net worth'),
            value: netWorth.value.total,
            sub: totals ? tt('Accounts') + ' ' + netWorth.value.accounts + ' · ' + tt('Investments') + ' ' + baseMoney(totals.value) : '',
            large: true,
            className: '',
            icon: ''
        },
        {
            key: 'value',
            label: tt('Current value'),
            value: totals ? baseMoney(totals.value) : '-',
            sub: totals ? tt('Invested') + ' ' + baseMoney(totals.invested) : '',
            large: false,
            className: '',
            icon: ''
        },
        {
            key: 'gain',
            label: tt('Unrealised gain'),
            value: totals ? signedBaseMoney(totals.unrealised) : '-',
            sub: formatSignedPercent(unrealisedPercent, false) + (hasUnpriced && unrealisedPercent !== null ? ' ' + tt('on priced holdings only') : ''),
            large: false,
            className: totals ? signClass(totals.unrealised) : '',
            icon: totals ? signIcon(totals.unrealised) : ''
        },
        {
            key: 'realised',
            label: tt('Realised gain and dividends'),
            value: totals ? signedBaseMoney(totals.realised + totals.dividends) : '-',
            sub: totals ? tt('Dividends') + ' ' + baseMoney(totals.dividends) : '',
            large: false,
            className: totals ? signClass(totals.realised + totals.dividends) : '',
            icon: totals ? signIcon(totals.realised + totals.dividends) : ''
        },
        {
            key: 'xirr',
            label: tt('Annualised return (XIRR)'),
            value: totals ? formatSignedPercent(totals.xirr, true) : '-',
            sub: totals && totals.xirr === null ? tt('Needs at least a year of history or a sale') : '',
            large: false,
            className: '',
            icon: ''
        }
    ];
});

function reload(): void {
    loading.value = true;
    loadError.value = '';

    // net worth also needs accounts and exchange rates; if either fails the card simply shows an incomplete total
    accountsStore.loadAllAccounts({ force: false }).catch(() => undefined);
    exchangeRatesStore.getLatestExchangeRates({ silent: true, force: false }).catch(() => undefined);

    investmentStore.reloadAll().then(() => {
        loading.value = false;
    }).catch(error => {
        loading.value = false;
        loadError.value = tt(error && error.message ? error.message : 'Unable to retrieve portfolio summary');
    });
}

function refreshPrices(): void {
    refreshing.value = true;

    investmentStore.refreshPrices().then(results => {
        refreshing.value = false;
        const failed = results.filter(r => !r.ok);

        if (!failed.length) {
            snackbar.value?.showMessage('Prices have been updated');
        } else {
            const names = failed.map(r => r.symbol + ': ' + r.message).join('; ');
            snackbar.value?.showError({ message: tt('Some prices could not be updated') + ' - ' + names });
        }
    }).catch(error => {
        refreshing.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

// Keyboard users must land back where they were when a dialog closes (Vuetify only does this for its own activators).
function openDialog(open: () => Promise<unknown> | undefined): void {
    const opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;

    open()?.catch(() => undefined).finally(() => {
        nextTick(() => {
            if (opener && opener.isConnected) {
                opener.focus();
            }
        });
    });
}

function addSecurity(): void {
    openDialog(() => securityDialog.value?.open());
}

function editSecurity(securityId: string): void {
    const security = securityMap.value[securityId];

    if (security) {
        const hasTrades = trades.value.some(trade => trade.securityId === securityId);
        openDialog(() => securityDialog.value?.open(security, hasTrades));
    }
}

function addTrade(securityId?: string): void {
    openDialog(() => tradeDialog.value?.open({ securityId }));
}

function editTrade(trade: InvestmentTradeInfoResponse): void {
    openDialog(() => tradeDialog.value?.open({ trade }));
}

function setPrice(securityId: string): void {
    const security = securityMap.value[securityId];

    if (security) {
        openDialog(() => priceDialog.value?.open(security));
    }
}

function importCsv(): void {
    openDialog(() => importDialog.value?.open());
}

function removeSecurity(securityId: string): void {
    confirmDialog.value?.open('Are you sure you want to delete this security? Securities with trades cannot be deleted.').then(() => {
        investmentStore.deleteSecurity(securityId).then(() => {
            snackbar.value?.showMessage('Security has been deleted');
        }).catch(error => {
            if (!error.processed) {
                snackbar.value?.showError(error);
            }
        });
    }).catch(() => undefined);
}

function removeTrade(trade: InvestmentTradeInfoResponse): void {
    confirmDialog.value?.open('Are you sure you want to delete this trade?').then(() => {
        investmentStore.deleteTrade(trade.id).then(() => {
            snackbar.value?.showMessage('Trade has been deleted');
        }).catch(error => {
            if (!error.processed) {
                snackbar.value?.showError(error);
            }
        });
    }).catch(() => undefined);
}

reload();
</script>

<style scoped>
.ebk-numeric {
    font-variant-numeric: tabular-nums;
}

.ebk-empty-text {
    max-width: 52ch;
}

.ebk-actions-col {
    width: 56px;
}

.ebk-table th {
    white-space: nowrap;
}

.ebk-bar-row {
    margin-bottom: 1rem;
}

.ebk-totals {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    border-block: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.ebk-total {
    padding: 1rem 1.25rem;
    border-inline-start: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.ebk-total:first-child {
    border-inline-start: 0;
    padding-inline-start: 0;
}

.ebk-bar {
    height: 8px;
    margin-top: 0.375rem;
    border-radius: var(--ebk-radius-sm);
    background: rgba(var(--v-theme-on-surface), 0.14);
    overflow: hidden;
}

.ebk-bar-fill {
    height: 100%;
    border-radius: var(--ebk-radius-sm);
    background: rgb(var(--v-theme-primary));
    transition: width 150ms ease;
}

.sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
}
</style>
