export function nodeScope(value) {
  const ids = typeof value === 'string' ? (value.trim() ? JSON.parse(value) : []) : (value ?? []);
  if (!Array.isArray(ids) || ids.some((id) => !Number.isInteger(id) || id <= 0)) {
    throw new Error('节点作用域无效');
  }
  return ids;
}

export function serverFormValues(record) {
  return {
    ...record,
    groups: record.groups || [],
    node_ids: nodeScope(record.node_ids),
    host_ip: record.host_ip || '',
    tls_host_verify: record.tls_host_verify || '',
    extra_options: record.extra_options || '',
  };
}

export function serverPayload(values) {
  return { ...values, node_ids: JSON.stringify(nodeScope(values.node_ids)) };
}
