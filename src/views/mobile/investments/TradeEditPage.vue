<template>
    <f7-page @page:beforein="onPageBeforeIn">
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': submitting }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="isEdit ? tt('Edit Trade') : tt('Add Trade')"></f7-nav-title>
            <f7-nav-right>
                <f7-link icon-f7="checkmark_alt" :aria-label="tt('Save')"
                         :class="{ 'disabled': submitting || !!validationMessage }" @click="save"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-segmented strong tag="p" class="margin-horizontal margin-top">
            <f7-button :active="tradeType === item.value" :key="item.value"
                       v-for="item in TRADE_TYPES" @click="tradeType = item.value">{{ tt(item.name) }}</f7-button>
        </f7-segmented>

        <f7-list form strong inset dividers class="margin-vertical-half">
            <f7-list-input type="select"
                           :disabled="submitting || isEdit"
                           :label="tt('Security')"
                           v-model:value="securityId">
                <option value="">{{ tt('Select a security') }}</option>
                <option :value="security.id" :key="security.id" v-for="security in securities">{{ security.name }} ({{ security.symbol }}, {{ security.currency }})</option>
            </f7-list-input>
            <f7-list-input type="date"
                           :disabled="submitting"
                           :label="tt('Date')"
                           v-model:value="date"></f7-list-input>

            <template v-if="tradeType !== 3">
                <f7-list-input type="text" inputmode="decimal" clear-button
                               :disabled="submitting"
                               :label="tt('Quantity')"
                               placeholder="0"
                               :error-message-force="!!quantity && !isValidQuantityOrPrice(quantity)"
                               :error-message="tt('Enter a positive number with up to 8 decimals, no thousands separators')"
                               v-model:value="quantity"></f7-list-input>
                <f7-list-input type="text" inputmode="decimal" clear-button
                               :disabled="submitting"
                               :label="tt('Price per unit') + (currency ? ' (' + currency + ')' : '')"
                               placeholder="0.00"
                               :error-message-force="!!price && !isValidQuantityOrPrice(price)"
                               :error-message="tt('Enter a positive number with up to 8 decimals, no thousands separators')"
                               v-model:value="price"></f7-list-input>
                <f7-list-input type="text" inputmode="decimal" clear-button
                               :disabled="submitting"
                               :label="tt('Fees') + (currency ? ' (' + currency + ')' : '')"
                               placeholder="0.00"
                               :error-message-force="!!fees && majorToMinorUnits(fees) === null"
                               :error-message="tt('Enter an amount with up to 2 decimals')"
                               v-model:value="fees"></f7-list-input>
            </template>
            <f7-list-input type="text" inputmode="decimal" clear-button
                           :disabled="submitting"
                           :label="tt('Dividend amount') + (currency ? ' (' + currency + ')' : '')"
                           placeholder="0.00"
                           :error-message-force="!!amount && (majorToMinorUnits(amount) === null || majorToMinorUnits(amount) === 0)"
                           :error-message="tt('Enter a positive amount with up to 2 decimals')"
                           v-model:value="amount"
                           v-else></f7-list-input>

            <f7-list-input type="text" clear-button maxlength="255"
                           :disabled="submitting"
                           :label="tt('Comment')"
                           v-model:value="comment"></f7-list-input>
        </f7-list>

        <f7-block class="no-margin-top" v-if="tradeType !== 3 && totalText">
            <small>{{ tradeType === 1 ? tt('Total cost') : tt('Net proceeds') }}</small>
            <div class="ebk-trade-total">{{ totalText }}</div>
        </f7-block>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import Decimal from 'decimal.js';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';

import { useInvestmentStore } from '@/stores/investment.ts';

import { TRADE_TYPES } from '@/models/investment.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';
import {
    isValidQuantityOrPrice,
    majorToMinorUnits,
    minorToMajorString
} from '@/lib/investments.ts';

const props = defineProps<{
    f7route: Router.Route;
    f7router: Router.Router;
}>();

const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const { showToast } = useI18nUIComponents();

const investmentStore = useInvestmentStore();

const editId = ref<string>('');
const accountId = ref<string>('0');
const submitting = ref<boolean>(false);
const securityId = ref<string>('');
const tradeType = ref<number>(1);
const date = ref<string>('');
const quantity = ref<string>('');
const price = ref<string>('');
const amount = ref<string>('');
const fees = ref<string>('');
const comment = ref<string>('');

const securities = computed(() => investmentStore.securities);
const isEdit = computed<boolean>(() => !!editId.value);
const currency = computed<string>(() => investmentStore.securityMap[securityId.value]?.currency || '');

const validationMessage = computed<string>(() => {
    if (!securityId.value) {
        return tt('Select a security');
    }

    if (!date.value) {
        return tt('Select a date');
    }

    if (tradeType.value === 3) {
        const minor = majorToMinorUnits(amount.value);
        return minor === null || minor === 0 ? tt('Enter a positive amount with up to 2 decimals') : '';
    }

    if (!isValidQuantityOrPrice(quantity.value) || !isValidQuantityOrPrice(price.value)) {
        return tt('Enter quantity and price');
    }

    if (fees.value && majorToMinorUnits(fees.value) === null) {
        return tt('Enter an amount with up to 2 decimals');
    }

    return '';
});

const totalText = computed<string>(() => {
    if (!isValidQuantityOrPrice(quantity.value) || !isValidQuantityOrPrice(price.value) || !currency.value) {
        return '';
    }

    const feeMinor = fees.value ? (majorToMinorUnits(fees.value) ?? 0) : 0;
    const gross = new Decimal(quantity.value).times(price.value).times(100).toDecimalPlaces(0, Decimal.ROUND_HALF_UP).toNumber();
    const total = tradeType.value === 1 ? gross + feeMinor : gross - feeMinor;

    return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(total), currency.value);
});

// Trades are calendar dates stored as midday UTC, so the day is the same in every timezone.
function dateToUnixTime(isoDate: string): number {
    const [year, month, day] = isoDate.split('-').map(Number);
    return Math.floor(Date.UTC(year as number, (month as number) - 1, day as number, 12) / 1000);
}

function onPageBeforeIn(): void {
    // opened directly (for example after a browser reload): load the data first
    if (!investmentStore.securities.length) {
        investmentStore.reloadAll().then(applyRoute).catch(() => undefined);
    }

    applyRoute();
}

function applyRoute(): void {
    const id = props.f7route.query['id'];
    const presetSecurityId = props.f7route.query['securityId'];
    const trade = typeof id === 'string' ? investmentStore.trades.find(item => item.id === id) : undefined;

    editId.value = trade ? trade.id : '';
    accountId.value = trade ? trade.accountId : '0';
    securityId.value = trade ? trade.securityId : (typeof presetSecurityId === 'string' ? presetSecurityId : '');
    tradeType.value = trade ? trade.tradeType : 1;
    date.value = trade ? new Date(trade.time * 1000).toISOString().slice(0, 10) : new Date().toISOString().slice(0, 10);
    quantity.value = trade && trade.tradeType !== 3 ? trade.quantity : '';
    price.value = trade && trade.tradeType !== 3 ? trade.price : '';
    amount.value = trade && trade.tradeType === 3 ? minorToMajorString(trade.amount) : '';
    fees.value = trade && trade.fees ? minorToMajorString(trade.fees) : '';
    comment.value = trade ? trade.comment : '';
}

function save(): void {
    if (submitting.value || validationMessage.value) {
        return;
    }

    const isDividend = tradeType.value === 3;
    submitting.value = true;
    showLoading();

    investmentStore.saveTrade({
        id: editId.value || undefined,
        securityId: securityId.value,
        tradeType: tradeType.value,
        time: dateToUnixTime(date.value),
        quantity: isDividend ? '' : quantity.value.trim(),
        price: isDividend ? '' : price.value.trim(),
        amount: isDividend ? (majorToMinorUnits(amount.value) ?? 0) : 0,
        fees: isDividend ? 0 : (fees.value ? (majorToMinorUnits(fees.value) ?? 0) : 0),
        accountId: accountId.value,
        comment: comment.value.trim()
    }).then(() => {
        submitting.value = false;
        hideLoading();
        props.f7router.back();
    }).catch(error => {
        submitting.value = false;
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}
</script>

<style scoped>
.ebk-trade-total {
    font-size: 1.25rem;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
}
</style>
