export function useCountUp(target: Ref<number>, duration = 800) {
  const display = ref(0)
  let raf: number

  watch(target, (to, from = 0) => {
    cancelAnimationFrame(raf)
    const start = performance.now()
    const diff = to - from

    function step(now: number) {
      const progress = Math.min((now - start) / duration, 1)
      const ease = 1 - Math.pow(1 - progress, 3)
      display.value = Math.round(from + diff * ease)
      if (progress < 1) raf = requestAnimationFrame(step)
    }

    raf = requestAnimationFrame(step)
  }, { immediate: true })

  onUnmounted(() => cancelAnimationFrame(raf))
  return display
}
