<script setup lang="ts">
interface Props {
  modelValue?: string
  label?: string
  error?: string
  type?: string
  placeholder?: string
  hint?: string
  autocomplete?: string
}

defineProps<Props>()
defineEmits<{ (e: 'update:modelValue', v: string): void }>()
</script>

<template>
  <label class="block">
    <span v-if="label" class="mb-1.5 block text-sm font-medium text-ink">{{ label }}</span>
    <input
      :value="modelValue"
      :type="type || 'text'"
      :placeholder="placeholder"
      :autocomplete="autocomplete"
      class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm text-ink placeholder:text-ink-faint focus:border-accent focus:shadow-focus focus:outline-none"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <span v-if="hint && !error" class="mt-1 block text-xs text-ink-muted">{{ hint }}</span>
    <span v-if="error" class="mt-1 block text-xs text-danger">{{ error }}</span>
  </label>
</template>
