export const DATE_PRESETS = [
  { label: 'Today',      value: 'today' },
  { label: 'Yesterday',  value: 'yesterday' },
  { label: 'Last 3 days',value: 'last_3d' },
  { label: 'Last 7 days',value: 'last_7d' },
  { label: 'Last 14 days',value: 'last_14d' },
  { label: 'Last 30 days',value: 'last_30d' },
  { label: 'Last 90 days',value: 'last_90d' },
  { label: 'This month', value: 'this_month' },
  { label: 'Last month', value: 'last_month' },
]

export function useDateRange(initial = 'last_7d') {
  const preset = ref(initial)
  const label = computed(() => DATE_PRESETS.find(p => p.value === preset.value)?.label ?? preset.value)
  return { preset, label, DATE_PRESETS }
}
