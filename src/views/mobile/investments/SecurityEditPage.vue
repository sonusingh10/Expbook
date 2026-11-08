<template>
    <f7-page @page:beforein="onPageBeforeIn">
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': submitting }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="isEdit ? tt('Edit Security') : tt('Add Security')"></f7-nav-title>
            <f7-nav-right>
                <f7-link icon-f7="checkmark_alt" :aria-label="tt('Save')"
                         :class="{ 'disabled': submitting || !canSave }" @click="save"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-list form strong inset dividers class="margin-vertical-half">
            <f7-list-input type="text" clear-button maxlength="64"
                           :disabled="submitting"
                           :label="tt('Name')"
                           :placeholder="tt('Security name')"
                           v-model:value="name"></f7-list-input>
            <f7-list-input type="text" clear-button maxlength="32" autocapitalize="characters"
                           :disabled="submitting"
                           :label="tt('Symbol')"
                           :placeholder="tt('For example INFY or AAPL')"
                           v-model:value="symbol"></f7-list-input>
            <f7-list-input type="select"
                           :disabled="submitting"
                           :label="tt('Type')"
                           v-model:value="securityType">
                <option :value="item.value" :key="item.value" v-for="item in SECURITY_TYPES">{{ tt(item.name) }}</option>
            </f7-list-input>
            <f7-list-input type="select"
                           :disabled="submitting || (isEdit && hasTrades)"
                           :label="tt('Currency')"
                           v-model:value="currency">
                <option :value="item.currencyCode" :key="item.currencyCode" v-for="item in allCurrencies">{{ item.displayName }} ({{ item.currencyCode }})</option>
            </f7-list-input>
            <f7-list-input type="select"
                           :disabled="submitting"
                           :label="tt('Price source')"
                           v-model:value="priceSource">
                <option :value="item.value" :key="item.value" v-for="item in PRICE_SOURCES">{{ tt(item.name) }}</option>
            </f7-list-input>
            <f7-list-input type="text" clear-button maxlength="64"
                           :disabled="submitting"
                           :label="tt('Price symbol')"
                           :placeholder="priceSource === 'amfi' ? tt('AMFI scheme code') : tt('For example INFY.NS or AAPL')"
                           v-model:value="priceSymbol"
                           v-if="priceSource !== 'manual'"></f7-list-input>
            <f7-list-input type="text" clear-button maxlength="255"
                           :disabled="submitting"
                           :label="tt('Comment')"
                           v-model:value="comment"></f7-list-input>
        </f7-list>

        <f7-block class="no-margin-top" v-if="priceSource !== 'manual'">
            <small>{{ tt('Only the price symbol is sent to the price provider. Quantities and amounts never leave your server.') }}</small>
        </f7-block>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';

import { useUserStore } from '@/stores/user.ts';
import { useInvestmentStore } from '@/stores/investment.ts';

import { SECURITY_TYPES, PRICE_SOURCES } from '@/models/investment.ts';

const props = defineProps<{
    f7route: Router.Route;
    f7router: Router.Router;
}>();

const { tt, getAllCurrencies } = useI18n();
const { showToast } = useI18nUIComponents();

const userStore = useUserStore();
const investmentStore = useInvestmentStore();

const editId = ref<string>('');
const hasTrades = ref<boolean>(false);
const submitting = ref<boolean>(false);
const name = ref<string>('');
const symbol = ref<string>('');
const securityType = ref<number>(1);
const currency = ref<string>(userStore.currentUserDefaultCurrency);
const priceSource = ref<string>('manual');
const priceSymbol = ref<string>('');
const comment = ref<string>('');

const isEdit = computed<boolean>(() => !!editId.value);
const allCurrencies = computed(() => getAllCurrencies(false));
const canSave = computed<boolean>(() => !!name.value.trim() && !!symbol.value.trim() && !!currency.value);

function onPageBeforeIn(): void {
    // opened directly (for example after a browser reload): load the data first
    if (!investmentStore.securities.length) {
        investmentStore.reloadAll().then(applyRoute).catch(() => undefined);
    }

    applyRoute();
}

function applyRoute(): void {
    const id = props.f7route.query['id'];
    editId.value = typeof id === 'string' ? id : '';

    const security = investmentStore.securityMap[editId.value];

    if (security) {
        name.value = security.name;
        symbol.value = security.symbol;
        securityType.value = security.securityType;
        currency.value = security.currency;
        priceSource.value = security.priceSource || 'manual';
        priceSymbol.value = security.priceSymbol;
        comment.value = security.comment;
        hasTrades.value = investmentStore.trades.some(trade => trade.securityId === security.id);
    }
}

function save(): void {
    if (submitting.value || !canSave.value) {
        return;
    }

    submitting.value = true;
    showLoading();

    investmentStore.saveSecurity({
        id: editId.value || undefined,
        name: name.value.trim(),
        symbol: symbol.value.trim(),
        securityType: Number(securityType.value),
        currency: currency.value,
        priceSource: priceSource.value,
        priceSymbol: priceSource.value === 'manual' ? '' : priceSymbol.value.trim(),
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
