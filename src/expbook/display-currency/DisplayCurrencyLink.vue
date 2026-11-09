<template>
    <f7-link class="display-currency-link" popover-open=".display-currency-popover"
             :aria-label="tt('Show amounts in another currency')">
        {{ store.currency }}
    </f7-link>
</template>

<script setup lang="ts">
import { useDisplayCurrencyPicker } from './picker.ts';

// The matching list lives in DisplayCurrencyPopover, which must sit outside the navbar.
const { store, tt } = useDisplayCurrencyPicker(true);
</script>
