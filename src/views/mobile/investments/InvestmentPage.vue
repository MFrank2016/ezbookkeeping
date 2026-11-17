<template>
    <f7-page>
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': loading }" back-link="返回"></f7-nav-left>
            <f7-nav-title title="投资"></f7-nav-title>
            <f7-nav-right :class="{ 'disabled': loading }">
                <f7-link icon-f7="arrow_clockwise" aria-label="刷新" @click="refresh"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <div class="investment-page-container">
            <investment-panel />
        </div>
    </f7-page>
</template>

<script setup lang="ts">
import InvestmentPanel from '@/components/common/InvestmentPanel.vue';

import { useInvestmentPageBase } from '@/views/base/investments/InvestmentPageBase.ts';

const { loading, load, refresh } = useInvestmentPageBase();

load();
</script>

<style scoped>
.investment-page-container {
    padding: 8px;
}
</style>
