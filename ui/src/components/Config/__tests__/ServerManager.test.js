import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ServerManager from '../ServerManager';
import { getServers, getGroups, getNodes, updateServer } from '../../../api';

jest.mock('../../../api', () => ({
  getServers: jest.fn(), getGroups: jest.fn(), getNodes: jest.fn(),
  addServer: jest.fn(), updateServer: jest.fn(), deleteServer: jest.fn(),
}));

beforeAll(() => {
  window.matchMedia = () => ({ matches: false,
    addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {} });
  global.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} };
  const getStyle = window.getComputedStyle;
  window.getComputedStyle = (element) => getStyle(element);
});

test('默认组高级 DoH 能通过真实编辑表单保存，参数和作用域保持', async () => {
  const server = { id: 7, address: 'https://dns.example/dns-query', type: 'https', groups: [],
    exclude_default: false, host_ip: '192.0.2.1', tls_host_verify: 'dns.example',
    extra_options: '-http-host dns.example', node_ids: '[3]' };
  getServers.mockResolvedValue({ data: [server] });
  getGroups.mockResolvedValue({ data: [] });
  getNodes.mockResolvedValue({ data: [{ id: 3, name: '测试节点', host: '192.0.2.3' }] });
  updateServer.mockResolvedValue({ success: true });
  render(<ServerManager />);
  await userEvent.click(await screen.findByRole('button', { name: /编辑/ }));
  expect(screen.getByLabelText('固定连接 IP').value).toBe('192.0.2.1');
  await userEvent.click(screen.getByRole('button', { name: /确\s*定/ }));
  await waitFor(() => expect(updateServer).toHaveBeenCalledWith(7, expect.objectContaining({
    groups: [], exclude_default: false, host_ip: '192.0.2.1', tls_host_verify: 'dns.example', node_ids: '[3]',
  })));
});

test('上游列表默认按分组排列，点击地址表头可按IP排序且不提交配置', async () => {
  jest.clearAllMocks();
  getServers.mockResolvedValue({ data: [
    { id: 1, address: '10.0.0.10', type: 'udp', groups: ['local'] },
    { id: 2, address: '10.0.0.3', type: 'udp', groups: ['ad'] },
    { id: 3, address: '10.0.0.2', type: 'udp', groups: ['local'] },
  ] });
  getGroups.mockResolvedValue({ data: [] });
  getNodes.mockResolvedValue({ data: [] });
  render(<ServerManager />);
  await screen.findByText('10.0.0.10');
  const addresses = () => screen.getAllByRole('row').slice(1).map((row) => row.querySelector('code').textContent);
  expect(addresses()).toEqual(['10.0.0.3', '10.0.0.2', '10.0.0.10']);
  await userEvent.click(screen.getByRole('columnheader', { name: /服务器地址/ }));
  expect(addresses()).toEqual(['10.0.0.2', '10.0.0.3', '10.0.0.10']);
  await userEvent.click(screen.getByRole('columnheader', { name: /服务器地址/ }));
  expect(addresses()).toEqual(['10.0.0.10', '10.0.0.3', '10.0.0.2']);
  expect(updateServer).not.toHaveBeenCalled();
});
