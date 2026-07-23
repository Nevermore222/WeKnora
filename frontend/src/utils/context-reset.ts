const aborters = new Set<AbortController>()
const resets = new Set<() => void | Promise<void>>()

export function registerContextAborter(controller: AbortController) {
  aborters.add(controller)
  return () => aborters.delete(controller)
}

export function registerContextReset(reset: () => void | Promise<void>) {
  resets.add(reset)
  return () => resets.delete(reset)
}

export async function resetActiveContext() {
  for (const controller of aborters) {
    controller.abort('context-switch')
  }
  aborters.clear()

  for (const reset of Array.from(resets)) {
    await reset()
  }
}

export function resetContextRegistriesForTest() {
  aborters.clear()
  resets.clear()
}
