// 排序仅用于表格展示，不修改记录数组或下发顺序。
const collator = new Intl.Collator('zh-CN', { numeric: true, sensitivity: 'base' });

export const compareText = (a, b) => collator.compare(String(a ?? ''), String(b ?? ''));
export const byField = (field) => (a, b) => compareText(a[field], b[field]);
export const byNumber = (field) => (a, b) => (Number(a[field]) || 0) - (Number(b[field]) || 0);
export const domainLabel = (record) => record.is_domain_set ? record.domain_set_name : record.domain;
export const compareDomains = (a, b) => compareText(domainLabel(a), domainLabel(b));

export function compareServerGroups(a, b) {
  const left = a.groups || [];
  const right = b.groups || [];
  if (!left.length && right.length) return 1;
  if (left.length && !right.length) return -1;
  // 以界面首个分组为主键，cn/bootstrap 与 cn 放在同一段。
  return compareText(left[0], right[0]) ||
    compareText([...left].sort(compareText).join(','), [...right].sort(compareText).join(',')) ||
    compareText(a.address, b.address);
}

export function sortableColumns(columns, sorters, defaultKey) {
  return columns.map((column) => sorters[column.key] ? {
    ...column,
    sorter: sorters[column.key],
    ...(column.key === defaultKey ? { defaultSortOrder: 'ascend' } : {}),
  } : column);
}
