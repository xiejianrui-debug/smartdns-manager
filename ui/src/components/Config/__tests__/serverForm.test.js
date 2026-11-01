import { nodeScope, serverFormValues, serverPayload } from '../serverForm';

test('高级 DoH 默认组与节点作用域在编辑保存后保持', () => {
  const record = { address: 'https://dns.example/dns-query', type: 'https', groups: [],
    exclude_default: false, node_ids: '[3]', host_ip: '192.0.2.1',
    tls_host_verify: 'dns.example', extra_options: '-http-host dns.example' };
  expect(serverPayload(serverFormValues(record))).toEqual(record);
});

test('全局与显式节点作用域不同，错误作用域不能静默扩大', () => {
  expect(nodeScope('')).toEqual([]);
  expect(nodeScope('[1,3]')).toEqual([1, 3]);
  for (const invalid of ['oops', 'null', '{}', '[0]', '[-1]']) {
    expect(() => nodeScope(invalid)).toThrow();
  }
});
