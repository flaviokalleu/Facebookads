<script setup lang="ts">
interface Props { modelValue: number }
defineProps<Props>()
const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()

const options = [
  { value: 1,  label: 'Hoje' },
  { value: 7,  label: '7 dias' },
  { value: 14, label: '14 dias' },
  { value: 30, label: '30 dias' },
]
</script>

<template>
  <div class="inline-flex items-center gap-1 rounded-full bg-bg-muted p-1">
    <button
      v-for="o in options"
      :key="o.value"
      type="button"
      :class="[
        'rounded-full px-3 py-1 text-xs font-medium transition',
        modelValue === o.value
          ? 'bg-bg text-ink shadow-sm'
          : 'text-ink-muted hover:text-ink',
      ]"
      @click="emit('update:modelValue', o.value)"
    >
      {{ o.label }}
    </button>
  </div>
</template>
