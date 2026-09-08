import { onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router'
import { useDialog } from '@/features/dialogs/use-dialog'
import { draftProtection } from './draft-protection'

export function useDraftGuard(dirty: () => boolean, saving: () => boolean) {
  const dialog = useDialog()
  const confirmDiscard = draftProtection(dirty, saving, () => dialog.confirm({
      title: 'Discard changes?', content: 'Unsaved changes will be lost.', type: 'warning',
      actions: [{ key: 'cancel', label: 'Keep editing', variant: 'secondary' }, { key: 'discard', label: 'Discard changes', variant: 'danger' }],
    }).then(result => result === 'discard'))
  onBeforeRouteLeave(confirmDiscard)
  onBeforeRouteUpdate((to, from) => to.path === from.path ? true : confirmDiscard())
  return confirmDiscard
}
