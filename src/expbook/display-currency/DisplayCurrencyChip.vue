<template>
    <v-menu location="bottom end" max-height="480" width="280">
        <template #activator="{ props }">
            <v-btn class="ms-2 display-currency-chip" variant="tonal" density="comfortable" rounded="pill"
                   :color="store.isOverridden ? 'primary' : 'default'"
                   :aria-label="tt('Show amounts in another currency')"
                   :loading="saving"
                   v-bind="props">
                <v-icon :icon="mdiSwapHorizontal" size="18" class="me-1" />
                {{ store.currency }}
                <v-tooltip activator="parent">{{ tt('Show amounts in another currency') }}</v-tooltip>
            </v-btn>
        </template>
        <v-list density="compact">
            <v-list-subheader>{{ tt('Show amounts in') }}</v-list-subheader>
            <v-list-item :key="choice.code" :active="choice.selected" color="primary"
                         @click="pick(choice.code)"
                         v-for="choice in choices">
                <v-list-item-title>
                    <span class="font-weight-medium">{{ choice.code }}</span>
                    <span class="text-medium-emphasis ms-2">{{ choice.name }}</span>
                </v-list-item-title>
                <template #append v-if="choice.selected">
                    <v-icon :icon="mdiCheck" size="18" />
                </template>
            </v-list-item>
            <v-divider class="my-1" />
            <div class="px-4 py-2 text-caption text-medium-emphasis">
                <div v-if="failed" class="text-error">{{ tt('Unable to save your choice. Try again.') }}</div>
                <div v-if="!store.hasRate" class="text-error">{{ tt('No exchange rate for this currency yet') }}</div>
                <div v-if="ratesAsOf">{{ ratesAsOf }}</div>
            </div>
        </v-list>
    </v-menu>
</template>

<script setup lang="ts">
import { mdiCheck, mdiSwapHorizontal } from '@mdi/js';

import { useDisplayCurrencyPicker } from './picker.ts';

const { store, choices, ratesAsOf, saving, failed, pick, tt } = useDisplayCurrencyPicker();
</script>
