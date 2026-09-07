export function draftProtection(dirty: () => boolean, busy: () => boolean, confirm: () => Promise<boolean>) {
  let pending: Promise<boolean> | undefined
  function confirmDiscard() {
    if (busy()) return Promise.resolve(false)
    if (!dirty()) return Promise.resolve(true)
    pending ??= confirm().finally(() => { pending = undefined })
    return pending
  }
  return confirmDiscard
}
