<script setup lang="ts">
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS, CategoryScale, LinearScale, PointElement,
  LineElement, Title, Tooltip, Legend, Filler,
} from 'chart.js'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

const props = defineProps<{
  labels: string[]
  spend: number[]
  leads: number[]
}>()

const chartData = computed(() => ({
  labels: props.labels,
  datasets: [
    {
      label: 'Spend ($)',
      data: props.spend,
      borderColor: '#3B82F6',
      backgroundColor: 'rgba(59,130,246,0.08)',
      borderWidth: 2,
      fill: true,
      tension: 0.4,
      pointRadius: 3,
      pointHoverRadius: 5,
      yAxisID: 'ySpend',
    },
    {
      label: 'Leads',
      data: props.leads,
      borderColor: '#10B981',
      backgroundColor: 'rgba(16,185,129,0.08)',
      borderWidth: 2,
      fill: true,
      tension: 0.4,
      pointRadius: 3,
      pointHoverRadius: 5,
      yAxisID: 'yLeads',
    },
  ],
}))

const options = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index' as const, intersect: false },
  plugins: {
    legend: { labels: { color: '#94A3B8', font: { size: 11 }, boxWidth: 12 } },
    tooltip: {
      backgroundColor: '#0A1628',
      borderColor: '#1E2D45',
      borderWidth: 1,
      titleColor: '#E2E8F0',
      bodyColor: '#94A3B8',
    },
  },
  scales: {
    x: { grid: { color: '#1E2D45' }, ticks: { color: '#64748B', font: { size: 10 } } },
    ySpend: {
      position: 'left' as const,
      grid: { color: '#1E2D45' },
      ticks: { color: '#64748B', font: { size: 10 }, callback: (v: string | number) => `$${v}` },
    },
    yLeads: {
      position: 'right' as const,
      grid: { drawOnChartArea: false },
      ticks: { color: '#64748B', font: { size: 10 } },
    },
  },
}
</script>

<template>
  <div class="h-56">
    <Line :data="chartData" :options="options" />
  </div>
</template>
