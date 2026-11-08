<template>
    <v-dialog width="860" :persistent="busy" v-model="showState">
        <one-column-dialog-layout :disabled="busy"
                                  :title="tt('Import Trades from CSV')"
                                  :cancel-button-title="tt('Close')"
                                  @cancel="close">
            <template #toolbar>
                <v-btn class="mx-1" density="comfortable" variant="text"
                       :disabled="busy" @click="downloadTemplate">
                    {{ tt('Download template') }}
                </v-btn>
                <v-btn class="mx-1" density="comfortable" variant="outlined"
                       :disabled="busy || !csv.trim()" @click="preview">
                    {{ tt('Preview') }}
                    <v-progress-circular indeterminate size="20" class="ms-2" v-if="busy && !result"></v-progress-circular>
                </v-btn>
                <v-btn class="mx-2" color="primary" density="comfortable" variant="flat"
                       :disabled="busy || !canImport" @click="runImport">
                    {{ importButtonText }}
                    <v-progress-circular indeterminate size="20" class="ms-2" v-if="importing"></v-progress-circular>
                </v-btn>
            </template>

            <template #content>
                <p class="text-body-medium mb-3">
                    {{ tt('Columns: date, type (buy, sell, dividend), symbol are required. Buys and sells need quantity and price (or amount). Dividends need amount. Importing the same file again never creates duplicates.') }}
                </p>

                <v-file-input accept=".csv,text/csv,text/plain" density="comfortable" prepend-icon=""
                              :prepend-inner-icon="mdiPaperclip"
                              :disabled="busy"
                              :label="tt('Choose a CSV file')"
                              :error-messages="fileError"
                              @update:model-value="onFileSelected" />

                <v-textarea auto-grow rows="6" max-rows="12" class="ebk-csv-input"
                            :disabled="busy"
                            :label="tt('Or paste CSV text')"
                            :placeholder="IMPORT_CSV_HEADER"
                            v-model="csv"
                            @update:model-value="onCsvEdited" />

                <div class="mt-2" v-if="result">
                    <div class="d-flex flex-wrap align-center ga-4 mb-3" role="status">
                        <span class="text-title-small ebk-numeric">{{ counts.valid }} {{ tt('ready to import') }}</span>
                        <span class="text-body-medium text-medium-emphasis ebk-numeric">{{ counts.duplicate }} {{ tt('already imported') }}</span>
                        <span class="text-body-medium ebk-numeric" :class="counts.error ? 'text-error' : 'text-medium-emphasis'">{{ counts.error }} {{ tt('with errors') }}</span>
                    </div>

                    <v-alert type="success" variant="tonal" density="compact" class="mb-3" v-if="!result.dryRun">
                        {{ tt('Import finished') }}: {{ result.created }} {{ tt('created') }}, {{ result.skipped }} {{ tt('skipped') }}, {{ result.failed }} {{ tt('failed') }}
                    </v-alert>

                    <v-table density="compact" class="ebk-import-table" v-if="result.rows.length">
                        <thead>
                        <tr>
                            <th class="text-end">{{ tt('Line') }}</th>
                            <th>{{ tt('Symbol') }}</th>
                            <th>{{ tt('Status') }}</th>
                            <th>{{ tt('Details') }}</th>
                        </tr>
                        </thead>
                        <tbody>
                        <tr :key="row.line" v-for="row in result.rows">
                            <td class="text-end ebk-numeric">{{ row.line }}</td>
                            <td>{{ row.symbol }}</td>
                            <td>
                                <span class="d-inline-flex align-center">
                                    <v-icon size="16" class="me-1" :icon="statusIcon(row.status)" :color="row.status === 'error' ? 'error' : undefined" />
                                    {{ statusText(row.status) }}
                                </span>
                            </td>
                            <td class="text-body-small">{{ row.message }}</td>
                        </tr>
                        </tbody>
                    </v-table>
                </div>
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

import type { InvestmentImportResponse } from '@/models/investment.ts';

import {
    IMPORT_CSV_EXAMPLE,
    IMPORT_CSV_HEADER,
    countImportRows
} from '@/lib/investments.ts';

import {
    mdiPaperclip,
    mdiCheckCircleOutline,
    mdiContentCopy,
    mdiAlertCircleOutline
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;

const MAX_FILE_BYTES = 2 * 1024 * 1024;

const { tt } = useI18n();

const investmentStore = useInvestmentStore();

const showState = ref<boolean>(false);
const busy = ref<boolean>(false);
const importing = ref<boolean>(false);
const csv = ref<string>('');
const fileError = ref<string>('');
const result = ref<InvestmentImportResponse | null>(null);

const snackbar = useTemplateRef<SnackBarType>('snackbar');

const counts = computed(() => countImportRows(result.value ? result.value.rows : []));
const canImport = computed<boolean>(() => !!result.value && result.value.dryRun && counts.value.valid > 0);
const importButtonText = computed<string>(() => canImport.value ? tt('Import') + ' ' + counts.value.valid : tt('Import'));

function statusText(status: string): string {
    if (status === 'valid') {
        return tt('Ready');
    } else if (status === 'duplicate') {
        return tt('Already imported');
    }

    return tt('Error');
}

function statusIcon(status: string): string {
    if (status === 'valid') {
        return mdiCheckCircleOutline;
    } else if (status === 'duplicate') {
        return mdiContentCopy;
    }

    return mdiAlertCircleOutline;
}

function onFileSelected(files: File | File[] | null | undefined): void {
    fileError.value = '';
    const file = Array.isArray(files) ? files[0] : files;

    if (!file) {
        return;
    }

    if (file.size > MAX_FILE_BYTES) {
        fileError.value = tt('The file is larger than 2 MB');
        return;
    }

    const reader = new FileReader();

    reader.onload = () => {
        csv.value = typeof reader.result === 'string' ? reader.result : '';
        result.value = null;
    };

    reader.onerror = () => {
        fileError.value = tt('Unable to read this file');
    };

    reader.readAsText(file);
}

// editing the text invalidates the previous preview, so an old preview can never be imported
function onCsvEdited(): void {
    result.value = null;
}

function downloadTemplate(): void {
    const blob = new Blob([IMPORT_CSV_EXAMPLE + '\n'], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'expbook-trades-template.csv';
    link.click();
    URL.revokeObjectURL(url);
}

function preview(): void {
    busy.value = true;
    result.value = null;

    investmentStore.importTrades(csv.value, true).then(response => {
        busy.value = false;
        result.value = response;
    }).catch(error => {
        busy.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function runImport(): void {
    busy.value = true;
    importing.value = true;

    investmentStore.importTrades(csv.value, false).then(response => {
        busy.value = false;
        importing.value = false;
        result.value = response;
    }).catch(error => {
        busy.value = false;
        importing.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function open(): Promise<void> {
    showState.value = true;
    busy.value = false;
    importing.value = false;
    csv.value = '';
    fileError.value = '';
    result.value = null;

    return Promise.resolve();
}

function close(): void {
    showState.value = false;
}

defineExpose({
    open
});
</script>

<style scoped>
.ebk-numeric {
    font-variant-numeric: tabular-nums;
}

.ebk-csv-input :deep(textarea) {
    font-variant-numeric: tabular-nums;
    font-size: 0.8125rem;
}

.ebk-import-table {
    max-height: 320px;
    overflow-y: auto;
}
</style>
