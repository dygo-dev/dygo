export function recordsToCSV(rows: Array<Record<string, unknown>>, columns: Array<{ key: string; label: string; formatValue?: (value: unknown) => string }>): string {
  const values = [columns.map((column) => column.label), ...rows.map((row) => columns.map((column) => column.formatValue ? column.formatValue(row[column.key]) : String(row[column.key] ?? '')))]
  return values.map((line) => line.map(csvCell).join(',')).join('\r\n') + '\r\n'
}

export function parseCSV(text: string): string[][] {
  const rows: string[][] = []
  let row: string[] = []
  let cell = ''
  let quoted = false
  for (let index = 0; index < text.length; index += 1) {
    const char = text[index]
    if (quoted && char === '"' && text[index + 1] === '"') { cell += '"'; index += 1; continue }
    if (char === '"') { quoted = !quoted; continue }
    if (!quoted && char === ',') { row.push(cell); cell = ''; continue }
    if (!quoted && (char === '\n' || char === '\r')) {
      if (char === '\r' && text[index + 1] === '\n') index += 1
      row.push(cell); cell = ''
      if (row.some((value) => value !== '')) rows.push(row)
      row = []
      continue
    }
    cell += char
  }
  if (cell !== '' || row.length > 0) { row.push(cell); rows.push(row) }
  return rows
}

function csvCell(value: string): string {
  return /[",\r\n]/.test(value) ? `"${value.replaceAll('"', '""')}"` : value
}
