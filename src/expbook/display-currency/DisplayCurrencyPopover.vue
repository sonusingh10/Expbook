<template>
    <f7-popover class="display-currency-popover">
        <f7-list dividers>
            <f7-list-item group-title :title="tt('Show amounts in')"></f7-list-item>
            <f7-list-item link="#" no-chevron popover-close
                          :key="choice.code"
                          :title="choice.code"
                          :after="choice.name"
                          :class="{ 'display-currency-selected': choice.selected }"
                          @click="pick(choice.code)"
                          v-for="choice in choices">
                <template #media>
                    <f7-icon f7="checkmark" v-if="choice.selected"></f7-icon>
                    <span class="display-currency-media-spacer" v-else></span>
                </template>
            </f7-list-item>
            <f7-list-item :footer="ratesAsOf" v-if="ratesAsOf"></f7-list-item>
            <f7-list-item class="text-color-red" :title="tt('No exchange rate for this currency yet')" v-if="!store.hasRate"></f7-list-item>
            <f7-list-item class="text-color-red" :title="tt('Unable to save your choice. Try again.')" v-if="failed"></f7-list-item>
        </f7-list>
    </f7-popover>
</template>

<script setup lang="ts">
import { useDisplayCurrencyPicker } from './picker.ts';

const { store, choices, ratesAsOf, failed, pick, tt } = useDisplayCurrencyPicker(false);
</script>

<style>
.display-currency-popover {
    max-height: 70vh;
    overflow-y: auto;
}

.display-currency-link {
    font-weight: 600;
}

.display-currency-media-spacer {
    display: inline-block;
    width: 22px;
}

.display-currency-selected .item-title {
    font-weight: 600;
}
</style>
