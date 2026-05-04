<script setup lang="ts">
import { Loader2 } from 'lucide-vue-next'

interface Props {
  variant?: 'primary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
  type?: 'button' | 'submit' | 'reset'
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  loading: false,
  type: 'button',
  disabled: false,
})

const variantClass = computed(() => {
  switch (props.variant) {
    case 'primary':
      return 'bg-accent text-white hover:bg-accent-hover focus:shadow-focus'
    case 'ghost':
      return 'border border-border text-ink hover:bg-bg-muted'
    case 'danger':
      return 'text-danger hover:bg-danger-soft'
  }
  return ''
})

const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm': return 'px-3 py-1.5 text-xs'
    case 'lg': return 'px-5 py-3 text-base'
    default:   return 'px-4 py-2.5 text-sm'
  }
})
</script>

<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :class="[
      'inline-flex items-center justify-center gap-2 rounded-lg font-medium transition outline-none disabled:cursor-not-allowed disabled:opacity-60',
      variantClass,
      sizeClass,
    ]"
  >
    <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
    <slot />
  </button>
</template>
