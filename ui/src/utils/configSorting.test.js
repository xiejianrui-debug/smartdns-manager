import { compareServerGroups, compareDomains, byField, byNumber, sortableColumns } from './configSorting';

test('同主分组连续排列，默认组最后，IP 按数字排序且不修改原数组', () => {
  const records = [
    { address: '10.0.0.10', groups: ['local'] },
    { address: 'dns.example', groups: [] },
    { address: '10.0.0.3', groups: ['cn', 'bootstrap'] },
    { address: '10.0.0.2', groups: ['local'] },
    { address: '10.0.0.4', groups: ['cn'] },
  ];
  const copy = JSON.stringify(records);
  const sorted = [...records].sort(compareServerGroups);
  expect(sorted.map((row) => row.groups[0])).toEqual(['cn', 'cn', 'local', 'local', undefined]);
  expect(sorted.slice(2, 4).map((row) => row.address)).toEqual(['10.0.0.2', '10.0.0.10']);
  expect(JSON.stringify(records)).toBe(copy);
});

test('域名集按显示名称排序；缺失字段及数字正常比较', () => {
  expect(compareDomains({ is_domain_set: true, domain_set_name: 'ad' }, { domain: 'local' })).toBeLessThan(0);
  expect(byField('name')({}, { name: 'a' })).toBeLessThan(0);
  expect(byNumber('priority')({ priority: 2 }, { priority: 10 })).toBeLessThan(0);
});

test('仅可排序的表头添加比较器，默认排序唯一，操作列保持原样', () => {
  const columns = [{ key: 'name' }, { key: 'action' }];
  const sorted = sortableColumns(columns, { name: byField('name') }, 'name');
  expect(sorted[0].defaultSortOrder).toBe('ascend');
  expect(sorted[1]).toBe(columns[1]);
  expect(columns[0].sorter).toBeUndefined();
});
