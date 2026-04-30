<script setup lang="ts">
import { Doughnut } from 'vue-chartjs'
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from 'chart.js'

ChartJS.register(ArcElement, Tooltip, Legend)

const props = defineProps<{
  scaling: number
  healthy: number
  atRisk: number
  underperforming: number
}>()

const total = computed(() => props.scaling + props.healthy + props.atRisk + props.underperforming)

const chartData = computed(() => ({
  labels: ['Scaling', 'Healthy', 'At Risk', 'Underperforming'],
  datasets: [{
    data: [props.scaling, props.healthy, props.atRisk, props.underperforming],
    backgroundColor: ['#10B981', '#3B82F6', '#F59E0B', '#EF4444'],
    borderColor: '#0A1628',
    borderWidth: 3,
    hoverOffset: 4,
  }],
}))

const options = {
  responsive: true,
  maintainAspectRatio: false,
  cutout: '68%',
  plugins: {
    legend: {
      position: 'bottom' as const,
      labels: { color: '#94A3B8', font: { size: 10 }, boxWidth: 10, padding: 10 },
    },
    tooltip: {
      backgroundColor: '#0A1628',
      borderColor: '#1E2D45',
      borderWidth: 1,
      titleColor: '#E2E8F0',
      bodyColor: '#94A3B8',
    },
  },
}
</script>

<template>
  <div class="relative h-52">
    <Doughnut :data="chartData" :options="options" />
    <div class="absolute inset-0 flex items-center justify-center pointer-events-none" style="padding-bottom: 2.5rem">
      <div class="text-center">
        <p class="text-primary text-2xl font-bold font-mono">{{ total }}</p>
        <p class="text-muted text-xs">Campaigns</p>
      </div>
    </div>
  </div>
</template>
