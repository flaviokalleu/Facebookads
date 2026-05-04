<script setup lang="ts">
interface Props {
  variant?: 'success' | 'warning' | 'danger' | 'neutral'
}
const props = withDefaults(defineProps<Props>(), { variant: 'neutral' })

const cls = computed(() => {
  switch (props.variant) {
    case 'success': return 'bg-success-soft text-success'
    case 'warning': return 'bg-warning-soft text-warning'
    case 'danger':  return 'bg-danger-soft text-danger'
    default:        return 'bg-bg-muted text-ink-muted'
  }
})
</script>

<template>
  <span :class="['inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium', cls]">
    <slot />
  </span>
</template>
