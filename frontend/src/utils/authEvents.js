const listeners = new Set()

export function onUnauthorized(cb) {
  listeners.add(cb)
  return () => listeners.delete(cb)
}

export function emitUnauthorized() {
  listeners.forEach((cb) => cb())
}
