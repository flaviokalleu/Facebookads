// useToast — global toast queue. Shared across the app via a single ref.

export type ToastKind = 'success' | 'info' | 'warning' | 'danger'

export interface Toast {
  id: string
  kind: ToastKind
  title: string
  description?: string
  duration?: number // ms, default 4000
}

// Singleton state, defined at module level so every composable call shares it.
const toasts = ref<Toast[]>([])
let counter = 0

export function useToast() {
  function push(t: Omit<Toast, 'id'>): string {
    const id = `t-${++counter}`
    const full: Toast = { id, duration: 4000, ...t }
    toasts.value.push(full)
    if (full.duration && full.duration > 0) {
      setTimeout(() => dismiss(id), full.duration)
    }
    return id
  }

  function dismiss(id: string) {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  return {
    toasts,
    dismiss,
    success: (title: string, description?: string) => push({ kind: 'success', title, description }),
    info:    (title: string, description?: string) => push({ kind: 'info',    title, description }),
    warning: (title: string, description?: string) => push({ kind: 'warning', title, description }),
    error:   (title: string, description?: string) => push({ kind: 'danger',  title, description, duration: 6000 }),
    push,
  }
}
