<template>
    <v-dialog width="640" :persistent="submitting" v-model="showState">
        <one-column-dialog-layout :disabled="submitting"
                                  :title="isEdit ? tt('Edit Trade') : tt('Add Trade')"
                                  :cancel-button-title="tt('Cancel')"
                                  @cancel="cancel">
            <template #toolbar>
                <v-btn class="mx-2" density="comfortable" variant="outlined"
                       :disabled="submitting || !!validationMessage" @click="confirm">
                    {{ tt('Save') }}
                    <v-progress-circular indeterminate size="22" class="ms-2" v-if="submitting"></v-progress-circular>
                </v-btn>
            </template>

            <template #content>
                <v-form class="mt-2" @submit.prevent="confirm">
                    <v-row>
                        <v-col cols="12" md="7">
                            <v-select item-title="title" item-value="value"
                                      :disabled="submitting || isEdit"
                                      :label="tt('Security')"
                                      :placeholder="tt('Select a security')"
                                      :no-data-text="tt('Add a security first')"
                                      :items="securityItems"
                                      v-model="securityId" />
                        </v-col>
                        <v-col cols="12" md="5">
                            <v-text-field type="date" persistent-placeholder
                                          :disabled="submitting"
                                          :label="tt('Date')"
                                          v-model="date" />
                        </v-col>
                        <v-col cols="12">
                            <v-btn-toggle mandatory divided density="comfortable" class="w-100"
                                          :disabled="submitting" v-model="tradeType">
                                <v-btn class="flex-grow-1" :value="item.value" :key="item.value"
                                       v-for="item in tradeTypeItems">{{ item.title }}</v-btn>
                            </v-btn-toggle>
                        </v-col>

                        <template v-if="tradeType !== 3">
                            <v-col cols="12" md="6">
                                <v-text-field inputmode="decimal" persistent-placeholder class="ebk-numeric-field"
                                              :disabled="submitting"
                                              :label="tt('Quantity')"
                                              :error-messages="quantityError"
                                              placeholder="0"
                                              v-model="quantity" />
                            </v-col>
                            <v-col cols="12" md="6">
                                <v-text-field inputmode="decimal" persistent-placeholder class="ebk-numeric-field"
                                              :disabled="submitting"
                                              :label="tt('Price per unit')"
                                              :suffix="currency"
                                              :error-messages="priceError"
                                              placeholder="0.00"
                                              v-model="price" />
                            </v-col>
                        </template>
                        <v-col cols="12" md="6" v-else>
                            <v-text-field inputmode="decimal" persistent-placeholder class="ebk-numeric-field"
                                          :disabled="submitting"
                                          :label="tt('Dividend amount')"
                                          :suffix="currency"
                                          :error-messages="amountError"
                                          placeholder="0.00"
                                          v-model="amount" />
                        </v-col>
                        <v-col cols="12" md="6" v-if="tradeType !== 3">
                            <v-text-field inputmode="decimal" persistent-placeholder class="ebk-numeric-field"
                                          :disabled="submitting"
                                          :label="tt('Fees')"
                                          :suffix="currency"
                                          :error-messages="feesError"
                                          placeholder="0.00"
                                          v-model="fees" />
                        </v-col>
                        <v-col cols="12" md="6" v-if="tradeType !== 3 && totalText">
                            <div class="text-label-medium text-medium-emphasis">{{ tradeType === 1 ? tt('Total cost') : tt('Net proceeds') }}</div>
                            <div class="text-title-medium ebk-numeric">{{ totalText }}</div>
                        </v-col>
                        <v-col cols="12">
                            <v-text-field persistent-placeholder
                                          :disabled="submitting"
                                          :label="tt('Comment')"
                                          maxlength="255"
                                          v-model="comment" />
                        </v-col>
                    </v-row>
                </v-form>
            </template>
        </one-column-dialog-layout>
    </v-dialog>

    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';

import { computed, ref, useTemplateRef } from 'vue';
import Decimal from 'decimal.js';

import { useI18n } from '@/locales/helpers.ts';

import { useInvestmentStore } from '@/stores/investment.ts';

import {
    TRADE_TYPES,
    type InvestmentTradeInfoResponse
} from '@/models/investment.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';
import {
    isValidQuantityOrPrice,
    majorToMinorUnits,
    minorToMajorString
} from '@/lib/investments.ts';

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

const investmentStore = useInvestmentStore();

const showState = ref<boolean>(false);
const submitting = ref<boolean>(false);
const editId = ref<string>('');
const accountId = ref<string>('0');
const securityId = ref<string>('');
const tradeType = ref<number>(1);
const date = ref<string>('');
const quantity = ref<string>('');
const price = ref<string>('');
const amount = ref<string>('');
const fees = ref<string>('');
const comment = ref<string>('');

const snackbar = useTemplateRef<SnackBarType>('snackbar');

const isEdit = computed<boolean>(() => !!editId.value);
const tradeTypeItems = computed(() => TRADE_TYPES.map(item => ({ value: item.value, title: tt(item.name) })));

const securityItems = computed(() => investmentStore.securities.map(security => ({
    value: security.id,
    title: security.name + ' (' + security.symbol + ', ' + security.currency + ')'
})));

const currency = computed<string>(() => investmentStore.securityMap[securityId.value]?.currency || '');

// the field-level messages only appear once the user typed something
const quantityError = computed<string>(() => quantity.value && !isValidQuantityOrPrice(quantity.value) ? tt('Enter a positive number with up to 8 decimals, no thousands separators') : '');
const priceError = computed<string>(() => price.value && !isValidQuantityOrPrice(price.value) ? tt('Enter a positive number with up to 8 decimals, no thousands separators') : '');
const amountError = computed<string>(() => amount.value && (majorToMinorUnits(amount.value) === null || majorToMinorUnits(amount.value) === 0) ? tt('Enter a positive amount with up to 2 decimals') : '');
const feesError = computed<string>(() => fees.value && majorToMinorUnits(fees.value) === null ? tt('Enter an amount with up to 2 decimals') : '');

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

function toIsoDate(unixTime: number): string {
    return new Date(unixTime * 1000).toISOString().slice(0, 10);
}

// Trades are calendar dates: store midday UTC so the day is the same in every timezone.
function dateToUnixTime(isoDate: string): number {
    const [year, month, day] = isoDate.split('-').map(Number);
    return Math.floor(Date.UTC(year as number, (month as number) - 1, day as number, 12) / 1000);
}

let resolveFunc: ((trade: InvestmentTradeInfoResponse) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

function open(options?: { trade?: InvestmentTradeInfoResponse, securityId?: string }): Promise<InvestmentTradeInfoResponse> {
    const trade = options?.trade;

    showState.value = true;
    submitting.value = false;
    editId.value = trade ? trade.id : '';
    accountId.value = trade ? trade.accountId : '0';
    securityId.value = trade ? trade.securityId : (options?.securityId || '');
    tradeType.value = trade ? trade.tradeType : 1;
    date.value = trade ? toIsoDate(trade.time) : new Date().toISOString().slice(0, 10);
    quantity.value = trade && trade.tradeType !== 3 ? trade.quantity : '';
    price.value = trade && trade.tradeType !== 3 ? trade.price : '';
    amount.value = trade && trade.tradeType === 3 ? minorToMajorString(trade.amount) : '';
    fees.value = trade && trade.fees ? minorToMajorString(trade.fees) : '';
    comment.value = trade ? trade.comment : '';

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function confirm(): void {
    if (submitting.value || validationMessage.value) {
        return;
    }

    const isDividend = tradeType.value === 3;
    submitting.value = true;

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
    }).then(trade => {
        submitting.value = false;
        resolveFunc?.(trade);
        showState.value = false;
    }).catch(error => {
        submitting.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function cancel(): void {
    rejectFunc?.();
    showState.value = false;
}

defineExpose({
    open
});
</script>

<style scoped>
.ebk-numeric,
.ebk-numeric-field :deep(input) {
    font-variant-numeric: tabular-nums;
}

.ebk-numeric-field :deep(input) {
    text-align: right;
}
</style>
