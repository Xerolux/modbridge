<!--
  One rendering of a calibration report, used both right after a run and when
  an older report is reopened. A measurement costs up to 90 seconds during
  which the proxy serves nobody, so the stored report has to be as readable as
  the fresh one — two divergent renderings of the same numbers would be a way
  to get them wrong.
-->
<template>
    <div v-if="report" class="space-y-2">
        <div class="overflow-x-auto">
            <table class="w-full text-xs">
                <thead>
                    <tr class="text-[var(--text-muted)]">
                        <th class="text-left font-normal">{{ $t('control.form.calibrateGap') }}</th>
                        <th class="text-right font-normal">{{ $t('control.form.calibrateErrors') }}</th>
                        <th class="text-right font-normal">p50</th>
                        <th class="text-right font-normal">p95</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="step in report.gap_steps || []" :key="step.gap_ms">
                        <td>{{ step.gap_ms }} ms</td>
                        <td class="text-right" :class="step.errors ? 'text-[var(--danger)]' : ''">
                            {{ step.errors }}/{{ step.requests }}
                        </td>
                        <td class="text-right">{{ step.p50_ms }} ms</td>
                        <td class="text-right">{{ step.p95_ms }} ms</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-if="(report.connection_steps || []).length" class="overflow-x-auto">
            <table class="w-full text-xs">
                <thead>
                    <tr class="text-[var(--text-muted)]">
                        <th class="text-left font-normal">{{ $t('control.form.calibrateConnections') }}</th>
                        <th class="text-right font-normal">{{ $t('control.form.calibrateErrors') }}</th>
                        <th class="text-right font-normal">p95</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="step in report.connection_steps" :key="step.connections">
                        <td>{{ step.connections }}</td>
                        <td class="text-right" :class="step.errors ? 'text-[var(--danger)]' : ''">
                            {{ step.errors }}/{{ step.requests }}
                        </td>
                        <td class="text-right">{{ step.p95_ms }} ms</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <p v-for="(note, i) in report.notes || []" :key="i" class="text-xs text-[var(--text-muted)]">{{ note }}</p>

        <p v-if="report.duration_ms" class="text-xs text-[var(--text-muted)]">
            {{ $t('control.form.calibrateDone', { seconds: Math.round(report.duration_ms / 1000) }) }}
        </p>

        <div v-if="report.recommended" class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-1">
            <span class="text-xs">
                {{ report.recommended.min_request_gap_ms }} ms ·
                {{ report.recommended.max_target_conns }} ·
                {{ report.recommended.read_timeout }} s
            </span>
            <Button
                v-if="showApply"
                :label="$t('control.form.calibrateApply')"
                size="small"
                class="w-full sm:w-auto shrink-0"
                @click="$emit('apply', report)"
            />
        </div>
    </div>
</template>

<script setup>
import Button from 'primevue/button';

defineProps({
    report: { type: Object, default: null },
    showApply: { type: Boolean, default: false }
});
defineEmits(['apply']);
</script>
