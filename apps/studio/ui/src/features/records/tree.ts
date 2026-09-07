import type { RecordData } from './records.api'
import type { TreeNode } from './tree.api'

export type TreeItem = { key: string; label: string; record?: RecordData; children?: TreeItem[]; contextOnly?: boolean; pathUnavailable?: boolean; action?: () => void }

export function treeRecord(node: TreeNode, labelField?: string): TreeItem {
  const label = labelField ? node.record[labelField] : undefined
  return { key: `record:${node.record.name}`, label: typeof label === 'string' && label.trim() ? label : String(node.record.name), record: node.record, children: node.hasChildren ? [] : undefined, contextOnly: !node.matched, pathUnavailable: node.pathUnavailable }
}

// Context is display-only. Missing paths remain explicitly unavailable, never fabricated roots.
export function searchTree(matches: TreeNode[], context: RecordData[], parentField: string, labelField?: string): TreeItem[] {
  const nodes = new Map<string, TreeItem>()
  for (const record of context) nodes.set(String(record.name), treeRecord({ record, hasChildren: false, matched: false, pathUnavailable: false }, labelField))
  for (const match of matches) nodes.set(String(match.record.name), treeRecord({ ...match, hasChildren: false }, labelField))
  const roots: TreeItem[] = []
  for (const node of nodes.values()) {
    const parent = node.pathUnavailable ? undefined : nodes.get(String(node.record?.[parentField] ?? ''))
    if (parent) (parent.children ??= []).push(node)
    else roots.push(node)
  }
  return roots
}
