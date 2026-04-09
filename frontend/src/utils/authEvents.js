const listeners = []

export function onUnauthorized(cb) {
  listeners.push(cb)
}

export function emitUnauthorized() {
  listeners.forEach(cb => cb())
}
