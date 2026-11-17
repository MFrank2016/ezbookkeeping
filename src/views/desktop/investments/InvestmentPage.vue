<template>
    <main-page-layout no-navbar>
        <template #content>
            <v-row>
                <v-col cols="12">
                    <v-card min-height="780">
                        <template #title>
                            <div class="title-and-toolbar d-flex align-center">
                                <span>投资</span>
                                <v-btn class="ms-3" density="compact" color="default" variant="text"
                                       aria-label="刷新" :icon="true" :loading="loading" @click="refresh">
                                    <template #loader>
                                        <v-progress-circular indeterminate size="20"/>
                                    </template>
                                    <v-icon :icon="mdiRefresh" size="24" />
                                    <v-tooltip activator="parent">刷新</v-tooltip>
                                </v-btn>
                            </div>
                        </template>

                        <v-card-text>
                            <investment-panel />
                        </v-card-text>
                    </v-card>
                </v-col>
            </v-row>
        </template>
    </main-page-layout>
</template>

<script setup lang="ts">
import InvestmentPanel from '@/components/common/InvestmentPanel.vue';

import { useInvestmentPageBase } from '@/views/base/investments/InvestmentPageBase.ts';

import { mdiRefresh } from '@mdi/js';

const { loading, load, refresh } = useInvestmentPageBase();

load();
</script>
