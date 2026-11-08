<template>
    <v-dialog width="420" :persistent="submitting" v-model="showState">
        <one-column-dialog-layout :disabled="submitting"
                                  :title="tt('Set Price')"
                                  :cancel-button-title="tt('Cancel')"
                                  @cancel="cancel">
            <template #toolbar>
                <v-btn class="mx-2" density="comfortable" variant="outlined"
                       :disabled="submitting || !valid" @click="confirm">
                    {{ tt('Save') }}
                    <v-progress-circular indeterminate size="22" class="ms-2" v-if="submitting"></v-progress-circular>
                </v-btn>
            </template>

            <template #content>
                <v-form class="mt-2" @submit.prevent="confirm">
                    <p class="text-body-medium mb-4">{{ securityName }}</p>
                    <v-text-field autofocus inputmode="decimal" persistent-placeholder class="ebk-price-input"
                                  :disabled="submitting"
                                  :label="tt('Latest price per unit')"
                                  :suffix="currency"
                                  :error-messages="price && !valid ? tt('Enter a positive number with up to 8 decimals, no thousands separators') : ''"
                                  placeholder="0.00"
                                  v-model="price" />
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

import { useInvestmentStore } from '@/stores/investment.ts';

import type { SecurityInfoResponse } from '@/models/investment.ts';

import { isValidQuantityOrPrice } from '@/lib/investments.ts';

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();

const investmentStore = useInvestmentStore();

const showState = ref<boolean>(false);
const submitting = ref<boolean>(false);
const securityId = ref<string>('');
const securityName = ref<string>('');
const currency = ref<string>('');
const price = ref<string>('');

const snackbar = useTemplateRef<SnackBarType>('snackbar');

const valid = computed<boolean>(() => isValidQuantityOrPrice(price.value));

let resolveFunc: (() => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

function open(security: SecurityInfoResponse): Promise<void> {
    showState.value = true;
    submitting.value = false;
    securityId.value = security.id;
    securityName.value = security.name + ' (' + security.symbol + ')';
    currency.value = security.currency;
    price.value = security.latestPrice || '';

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function confirm(): void {
    if (submitting.value || !valid.value) {
        return;
    }

    submitting.value = true;

    investmentStore.setSecurityPrice(securityId.value, price.value.trim()).then(() => {
        submitting.value = false;
        resolveFunc?.();
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
.ebk-price-input :deep(input) {
    font-variant-numeric: tabular-nums;
    text-align: right;
}
</style>
