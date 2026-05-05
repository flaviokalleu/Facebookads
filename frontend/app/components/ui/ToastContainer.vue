<script setup lang="ts">
import { CheckCircle2, AlertTriangle, AlertCircle, Info, X } from 'lucide-vue-next'
import { useToast, type Toast } from '~/composables/useToast'

const { toasts, dismiss } = useToast()

const styles: Record<Toast['kind'], { icon: any; ring: string; bg: string; iconColor: string }> = {
  success: { icon: CheckCircle2,   ring: 'ring-success/20',  bg: 'bg-success-soft',  iconColor: 'text-success' },
  info:    { icon: Info,           ring: 'ring-accent/20',   bg: 'bg-accent-soft',   iconColor: 'text-accent' },
  warning: { icon: AlertTriangle,  ring: 'ring-warning/20',  bg: 'bg-warning-soft',  iconColor: 'text-warning' },
  danger:  { icon: AlertCircle,    ring: 'ring-danger/20',   bg: 'bg-danger-soft',   iconColor: 'text-danger' },
}
</script>

<template>
  <Teleport to="body">
    <div class="pointer-events-none fixed inset-0 z-[100] flex flex-col items-end justify-end gap-2 p-4 sm:p-6">
      <TransitionGroup
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="opacity-0 translate-y-2"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition duration-150 ease-in"
        leave-from-class="opacity-100"
        leave-to-class="opacity-0 translate-x-4"
      >
        <div
          v-for="t in toasts"
          :key="t.id"
          :class="[
            'pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-xl border bg-bg p-3 shadow-lg ring-1',
            styles[t.kind].ring,
          ]"
          role="status"
        >
          <div :class="['rounded-full p-1', styles[t.kind].bg]">
            <component :is="styles[t.kind].icon" :class="['h-4 w-4', styles[t.kind].iconColor]" />
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-ink">{{ t.title }}</p>
            <p v-if="t.description" class="mt-0.5 text-xs text-ink-muted">{{ t.description }}</p>
          </div>
          <button
            type="button"
            class="rounded-md p-1 text-ink-faint hover:bg-bg-muted hover:text-ink"
            @click="dismiss(t.id)"
            aria-label="Fechar"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
