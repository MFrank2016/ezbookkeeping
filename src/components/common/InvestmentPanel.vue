<template>
    <div class="investment-panel">
        <div class="ip-card ip-summary" v-if="hasAccounts || loading">
            <div class="ip-summary-item">
                <span class="ip-label">总市值</span>
                <span class="ip-value">{{ totalMarketValueText }}</span>
            </div>
            <div class="ip-summary-item">
                <span class="ip-label">总成本</span>
                <span class="ip-value">{{ totalCostText }}</span>
            </div>
            <div class="ip-summary-item">
                <span class="ip-label">浮动盈亏</span>
                <span class="ip-value" :class="totalFloatProfitClass">{{ totalFloatProfitText }}</span>
            </div>
            <div class="ip-summary-item ip-summary-price">
                <span class="ip-label">最新金价</span>
                <span class="ip-value" :class="{ 'ip-error-text': hasAccounts && !priceAvailable }">{{ priceText }}</span>
                <span class="ip-hint" v-if="priceAvailable">
                    {{ priceSourceText }}<template v-if="priceFetchedAtText"> · {{ priceFetchedAtText }}</template>
                </span>
                <span class="ip-hint ip-error-text" v-else-if="hasAccounts">刷新可重试</span>
            </div>
        </div>

        <div class="ip-card" v-if="lastError && overview">
            <span class="ip-error-text">{{ lastError }}</span>
            <button type="button" class="ip-btn ip-btn-retry" @click="refresh">重试</button>
        </div>

        <div class="ip-card" v-if="loading && !overview">
            <div class="ip-loading">加载中…</div>
        </div>

        <div class="ip-card" v-else-if="lastError && !overview">
            <span class="ip-error-text">{{ lastError }}</span>
            <button type="button" class="ip-btn ip-btn-retry" @click="refresh">重试</button>
        </div>

        <div class="ip-card" v-else-if="!hasAccounts">
            <p class="ip-empty-title">还没有投资账户</p>
            <p class="ip-hint">
                在下方「投资账户配置」里选择一个账户，填写期初克重和期初成本即可开始记录持仓。
                期初克重指开始记账时该账户里已有的黄金克重，期初成本指这些黄金的买入总额。
            </p>
        </div>

        <template v-else>
            <div class="ip-card">
                <h3 class="ip-section-title">持仓</h3>
                <div class="ip-table-wrapper">
                    <table class="ip-table">
                        <thead>
                        <tr>
                            <th>标的</th>
                            <th>账户</th>
                            <th class="ip-align-end">克重</th>
                            <th class="ip-align-end">买入均价</th>
                            <th class="ip-align-end">成本</th>
                            <th class="ip-align-end">市值</th>
                            <th class="ip-align-end">浮动盈亏</th>
                            <th class="ip-align-end">账面余额</th>
                            <th class="ip-align-end">差额</th>
                        </tr>
                        </thead>
                        <tbody>
                        <tr :key="holding.id" v-for="holding in holdings">
                            <td>{{ holding.targetName }}</td>
                            <td>{{ holding.accountName }}</td>
                            <td class="ip-align-end">{{ holding.units }}</td>
                            <td class="ip-align-end">{{ holding.avgCost }}</td>
                            <td class="ip-align-end">{{ holding.cost }}</td>
                            <td class="ip-align-end">{{ holding.marketValue }}</td>
                            <td class="ip-align-end" :class="holding.floatProfitClass">{{ holding.floatProfit }}</td>
                            <td class="ip-align-end">{{ holding.balance }}</td>
                            <td class="ip-align-end" :class="holding.diffClass">
                                {{ holding.diff }} <span class="ip-hint">({{ holding.diffPercent }})</span>
                            </td>
                        </tr>
                        </tbody>
                    </table>
                </div>
                <p class="ip-hint" v-if="!priceAvailable">金价获取失败，市值、浮动盈亏和差额暂不可用。</p>
            </div>

            <div class="ip-card" v-if="pendingTransfers.length > 0">
                <h3 class="ip-section-title">待补克重转账（{{ pendingTransfers.length }}）</h3>
                <p class="ip-hint">
                    以下转账涉及投资账户但还没有填写克重，请到交易记录里找到对应转账并补填克重。
                    存在未补克重的记录时，周估值会暂停。
                </p>
                <ul class="ip-list">
                    <li :key="pending.transactionId" v-for="pending in pendingTransfers">
                        <span>{{ pending.timeText }}</span>
                        <span>{{ pending.directionText }}</span>
                        <span class="ip-list-amount">{{ pending.amount }}</span>
                        <span class="ip-list-comment">{{ pending.comment }}</span>
                    </li>
                </ul>
            </div>

            <div class="ip-card">
                <h3 class="ip-section-title">购入记录</h3>
                <p class="ip-hint" v-if="records.length === 0">暂无购入记录。</p>
                <div class="ip-table-wrapper" v-else>
                    <table class="ip-table">
                        <thead>
                        <tr>
                            <th>时间</th>
                            <th>方向</th>
                            <th class="ip-align-end">金额</th>
                            <th class="ip-align-end">克重</th>
                            <th class="ip-align-end">单价</th>
                            <th class="ip-align-end">手续费</th>
                            <th>备注</th>
                        </tr>
                        </thead>
                        <tbody>
                        <tr :key="record.key" v-for="record in records">
                            <td class="ip-nowrap">{{ record.timeText }}</td>
                            <td>{{ record.directionText }}</td>
                            <td class="ip-align-end">{{ record.amount }}</td>
                            <td class="ip-align-end">{{ record.units }}</td>
                            <td class="ip-align-end">{{ record.unitPrice }}</td>
                            <td class="ip-align-end">{{ record.fee }}</td>
                            <td>{{ record.comment }}</td>
                        </tr>
                        </tbody>
                    </table>
                </div>
                <p class="ip-hint">最多显示最近 50 条，完整记录请到交易记录中查看。</p>
            </div>
        </template>

        <div class="ip-card">
            <h3 class="ip-section-title">投资账户配置</h3>

            <p class="ip-hint" v-if="configurations.length === 0">
                还没有配置投资账户。选择一个账户并填写期初克重和期初成本后，该账户的转账即可记录投资克重。
            </p>

            <ul class="ip-list" v-else>
                <li :key="configuration.id" v-for="configuration in configurations" class="ip-config-row">
                    <template v-if="editingId !== configuration.id">
                        <div class="ip-config-info">
                            <span class="ip-config-name">{{ configuration.accountName }}</span>
                            <span class="ip-hint">期初 {{ configuration.initialUnits }} / {{ configuration.initialCost }}</span>
                            <span class="ip-hint ip-error-text" v-if="!configuration.hasInitialUnits">期初克重为 0，请填写开始记账时该账户已有的克重</span>
                        </div>
                        <div class="ip-config-actions">
                            <button type="button" class="ip-btn" :disabled="submitting" @click="startEdit(configuration)">编辑</button>
                            <button type="button" class="ip-btn ip-btn-danger" :disabled="submitting" @click="remove(configuration)">删除</button>
                        </div>
                    </template>
                    <template v-else>
                        <div class="ip-config-edit">
                            <label class="ip-field">
                                <span class="ip-label">期初克重（克）</span>
                                <input class="ip-input" type="number" min="0" step="0.0001" v-model="editingGrams" />
                            </label>
                            <label class="ip-field">
                                <span class="ip-label">期初成本（元）</span>
                                <input class="ip-input" type="number" min="0" step="0.01" v-model="editingCost" />
                            </label>
                        </div>
                        <div class="ip-config-actions">
                            <button type="button" class="ip-btn ip-btn-primary" :disabled="submitting" @click="submitEdit">保存</button>
                            <button type="button" class="ip-btn" :disabled="submitting" @click="cancelEdit">取消</button>
                        </div>
                    </template>
                </li>
            </ul>

            <div class="ip-add-form">
                <p class="ip-label ip-add-title">添加投资账户</p>
                <p class="ip-hint" v-if="availableAccountOptions.length === 0">没有可配置的账户（已全部配置或还没有账户）。</p>
                <template v-else>
                    <div class="ip-config-edit">
                        <label class="ip-field">
                            <span class="ip-label">账户</span>
                            <select class="ip-input" v-model="addAccountId">
                                <option :key="option.id" :value="option.id" v-for="option in availableAccountOptions">{{ option.name }}</option>
                            </select>
                        </label>
                        <label class="ip-field">
                            <span class="ip-label">标的</span>
                            <input class="ip-input" type="text" value="黄金" disabled />
                        </label>
                        <label class="ip-field">
                            <span class="ip-label">期初克重（克）</span>
                            <input class="ip-input" type="number" min="0" step="0.0001" v-model="addInitialGrams" />
                        </label>
                        <label class="ip-field">
                            <span class="ip-label">期初成本（元）</span>
                            <input class="ip-input" type="number" min="0" step="0.01" v-model="addInitialCost" />
                        </label>
                    </div>
                    <button type="button" class="ip-btn ip-btn-primary ip-btn-add" :disabled="submitting || !addAccountId" @click="submitAdd">添加</button>
                </template>
            </div>

            <p class="ip-error-text" v-if="formError">{{ formError }}</p>
            <p class="ip-hint">删除配置只会移除投资配置，不会删除账户和交易。</p>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useInvestmentPageBase } from '@/views/base/investments/InvestmentPageBase.ts';

const {
    submitting,
    formError,
    addAccountId,
    addInitialGrams,
    addInitialCost,
    editingId,
    editingGrams,
    editingCost,
    overview,
    loading,
    lastError,
    hasAccounts,
    priceAvailable,
    priceText,
    priceFetchedAtText,
    priceSourceText,
    totalMarketValueText,
    totalCostText,
    totalFloatProfitText,
    totalFloatProfitClass,
    holdings,
    records,
    pendingTransfers,
    configurations,
    availableAccountOptions,
    refresh,
    submitAdd,
    startEdit,
    cancelEdit,
    submitEdit,
    remove
} = useInvestmentPageBase();
</script>

<style scoped>
.investment-panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.ip-card {
    padding: 12px 16px;
    border: 1px solid rgba(128, 128, 128, 0.25);
    border-radius: 8px;
}

.ip-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 16px 32px;
}

.ip-summary-item {
    display: flex;
    flex-direction: column;
    min-width: 120px;
}

.ip-summary-price {
    flex: 1 1 220px;
}

.ip-label {
    font-size: 0.8rem;
    opacity: 0.7;
}

.ip-value {
    font-size: 1.25rem;
    line-height: 1.75rem;
}

.ip-hint {
    font-size: 0.8rem;
    opacity: 0.7;
}

.ip-error-text {
    color: rgb(var(--ebk-amount-color-red));
}

.investment-profit-positive {
    color: rgb(var(--ebk-amount-color-green));
}

.investment-profit-negative {
    color: rgb(var(--ebk-amount-color-red));
}

.ip-section-title {
    margin: 0 0 8px;
    font-size: 1rem;
    font-weight: 600;
}

.ip-table-wrapper {
    overflow-x: auto;
}

.ip-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.9rem;
}

.ip-table th,
.ip-table td {
    padding: 6px 8px;
    text-align: left;
    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
    white-space: nowrap;
}

.ip-table th {
    font-weight: 600;
    opacity: 0.7;
}

.ip-align-end {
    text-align: right !important;
}

.ip-nowrap {
    white-space: nowrap;
}

.ip-list {
    margin: 0;
    padding: 0;
    list-style: none;
}

.ip-list > li {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    padding: 6px 0;
    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
    font-size: 0.9rem;
}

.ip-list > li:last-child {
    border-bottom: none;
}

.ip-list-amount {
    font-weight: 600;
}

.ip-list-comment {
    opacity: 0.7;
    word-break: break-all;
}

.ip-config-row {
    align-items: center;
    justify-content: space-between;
}

.ip-config-info {
    display: flex;
    flex-direction: column;
}

.ip-config-name {
    font-weight: 600;
}

.ip-config-actions {
    display: flex;
    gap: 8px;
}

.ip-config-edit {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 16px;
    width: 100%;
}

.ip-field {
    display: flex;
    flex-direction: column;
    min-width: 140px;
    flex: 1 1 140px;
}

.ip-input {
    padding: 4px 8px;
    min-height: 32px;
    border: 1px solid rgba(128, 128, 128, 0.4);
    border-radius: 4px;
    background: transparent;
    color: inherit;
    font-size: 0.9rem;
}

.ip-add-form {
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid rgba(128, 128, 128, 0.2);
}

.ip-add-title {
    margin: 0 0 8px;
}

.ip-btn {
    padding: 4px 12px;
    min-height: 32px;
    border: 1px solid rgba(128, 128, 128, 0.4);
    border-radius: 4px;
    background: transparent;
    color: inherit;
    cursor: pointer;
    font-size: 0.85rem;
}

.ip-btn-retry {
    margin-inline-start: 8px;
}

.ip-btn-add {
    margin-top: 8px;
}

.ip-btn:disabled {
    opacity: 0.5;
    cursor: default;
}

.ip-btn-primary {
    border-color: currentColor;
    font-weight: 600;
}

.ip-btn-danger {
    color: rgb(var(--ebk-amount-color-red));
}

.ip-empty-title {
    margin: 0 0 4px;
    font-weight: 600;
}

.ip-loading {
    opacity: 0.7;
}
</style>
