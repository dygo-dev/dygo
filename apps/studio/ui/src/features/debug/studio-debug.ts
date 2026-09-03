export function isStudioDebugBarAvailable(dev = import.meta.env.DEV): boolean {
  return Boolean(dev)
}
