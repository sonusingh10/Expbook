<template>
    <v-dialog width="640" :persistent="submitting" v-model="showState">
        <one-column-dialog-layout :disabled="submitting"
                                  :title="isEdit ? tt('Edit Security') : tt('Add Security')"
                                  :cancel-button-title="tt('Cancel')"
                                  @cancel="cancel">
            <template #toolbar>
                <v-btn class="mx-2" density="comfortable" variant="outlined"
                       :disabled="submitting || !canSave" @click="confirm">
                    {{ tt('Save') }}
                    <v-progress-circular indeterminate size="22" class="ms-2" v-if="submitting"></v-progress-circular>
                </v-btn>
            </template>

            <template #content>
                <v-form class="mt-2" @submit.prevent="confirm">
                    <v-row>
                        <v-col cols="12" md="6">
                            <v-text-field autofocus persistent-placeholder
                                          :disabled="submitting"
                                          :label="tt('Name')"
                                          :placeholder="tt('Security name')"
                                          maxlength="64"
                                          v-model="name" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <v-text-field persistent-placeholder
                                          :disabled="submitting"
                                          :label="tt('Symbol')"
                                          :placeholder="tt('For example INFY or AAPL')"
                                          maxlength="32"
                                          v-model="symbol" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <v-select item-title="title" item-value="value"
                                      :disabled="submitting"
                                      :label="tt('Type')"
                                      :items="securityTypeItems"
                                      v-model="securityType" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <currency-select :disabled="submitting || (isEdit && hasTrades)"
                                             :label="tt('Currency')"
                                             :placeholder="tt('Currency')"
                                             v-model="currency" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <v-select item-title="title" item-value="value"
                                      :disabled="submitting"
                                      :label="tt('Price source')"
                                      :items="priceSourceItems"
                                      v-model="priceSource" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <v-text-field persistent-placeholder
                                          :disabled="submitting || priceSource === 'manual'"
                                          :label="tt('Price symbol')"
                                          :placeholder="priceSymbolPlaceholder"
                                          maxlength="64"
                                          v-model="priceSymbol" />
                        </v-col>
                        <v-col cols="12">
                            <v-text-field persistent-placeholder
                                          :disabled="submitting"
                                          :label="tt('Comment')"
                                          maxlength="255"
                                          v-model="comment" />
                        </v-col>
                        <v-col cols="12" v-if="priceSource !== 'manual'">
                            <p class="text-body-small text-medium-emphasis">
                                {{ tt('Only the price symbol is sent to the price provider. Quantities and amounts never leave your server.') }}
                            </p>
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

import { useI18n } from '@/locales/helpers.ts';

import { useUserStore } from '@/stores/user.ts';
import { useInvestmentStore } from '@/stores/investment.ts';

import {
    SECURITY_TYPES,
    PRICE_SOURCES,
    type SecurityInfoResponse
} from '@/models/investment.ts';

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();

const userStore = useUserStore();
const investmentStore = useInvestmentStore();

const showState = ref<boolean>(false);
const submitting = ref<boolean>(false);
const editId = ref<string>('');
const hasTrades = ref<boolean>(false);
const name = ref<string>('');
const symbol = ref<string>('');
const securityType = ref<number>(1);
const currency = ref<string>(userStore.currentUserDefaultCurrency);
const priceSource = ref<string>('manual');
const priceSymbol = ref<string>('');
const comment = ref<string>('');

const snackbar = useTemplateRef<SnackBarType>('snackbar');

const isEdit = computed<boolean>(() => !!editId.value);

const securityTypeItems = computed(() => SECURITY_TYPES.map(item => ({ value: item.value, title: tt(item.name) })));
const priceSourceItems = computed(() => PRICE_SOURCES.map(item => ({ value: item.value, title: tt(item.name) })));

const priceSymbolPlaceholder = computed<string>(() => {
    if (priceSource.value === 'yahoo') {
        return tt('For example INFY.NS or AAPL');
    } else if (priceSource.value === 'amfi') {
        return tt('AMFI scheme code');
    }

    return '';
});

const canSave = computed<boolean>(() => !!name.value.trim() && !!symbol.value.trim() && !!currency.value);

let resolveFunc: ((security: SecurityInfoResponse) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

function open(security?: SecurityInfoResponse, securityHasTrades?: boolean): Promise<SecurityInfoResponse> {
    showState.value = true;
    submitting.value = false;
    editId.value = security ? security.id : '';
    hasTrades.value = !!securityHasTrades;
    name.value = security ? security.name : '';
    symbol.value = security ? security.symbol : '';
    securityType.value = security ? security.securityType : 1;
    currency.value = security ? security.currency : userStore.currentUserDefaultCurrency;
    priceSource.value = security ? (security.priceSource || 'manual') : 'manual';
    priceSymbol.value = security ? security.priceSymbol : '';
    comment.value = security ? security.comment : '';

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function confirm(): void {
    if (submitting.value || !canSave.value) {
        return;
    }

    submitting.value = true;

    investmentStore.saveSecurity({
        id: editId.value || undefined,
        name: name.value.trim(),
        symbol: symbol.value.trim(),
        securityType: securityType.value,
        currency: currency.value,
        priceSource: priceSource.value,
        priceSymbol: priceSource.value === 'manual' ? '' : priceSymbol.value.trim(),
        comment: comment.value.trim()
    }).then(security => {
        submitting.value = false;
        resolveFunc?.(security);
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
