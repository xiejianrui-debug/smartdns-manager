# 上游高级参数与安全同步

上游管理支持固定连接 IP、TLS 证书校验域名、其他高级参数，以及生效节点。分组留空且不排除默认组时，上游参与默认查询；生效节点留空表示所有节点。

例如，DoH 上游可生成：

```conf
server-https https://dns.google/dns-query -host-ip 8.8.8.8 -tls-host-verify dns.google
```

类型、URL、固定 IP 和证书名称分别保存。编辑现有记录时，未提交的高级字段会保留；显式提交空字符串才清空。历史 `options` 中的参数会被读取到高级字段，其余参数保留在“其他高级参数”中。

## 同步前预览

节点的“DNS 服务器”标签页提供“预览完整同步”。预览只读取节点文件和数据库，显示生成内容及可能丢失的有效指令，不写文件、不重启。完整同步会再次检查；若存在丢失则拒绝写入。

配置解析和生成保留未建模指令、引用文件、注释、规则组和重复监听项。旧数据库记录同步时还会保留节点额外的上游分组及高级选项，避免历史数据缺项抹掉有效配置。规则中的重复同值参数不会在反复同步时累积。

如果用户确实希望删除或替换已生效参数，应先审阅差异，通过节点配置编辑器明确修改后再同步。编辑器保存携带读取时的校验和；文件已被其他人修改时会拒绝旧内容覆盖。

## API

- `GET /api/sync/node/:id/preview`：返回 `before`、`after`、`losses`、`safe`、`changed` 和 SHA256 `checksum`。
- `POST /api/sync/node/:id/full`：可提交预览返回的 `checksum`；节点文件变化时拒绝执行。请求等待同步完成，不会仅返回异步受理成功。
- `POST /api/nodes/:id/config`：保存 `raw_content` 时必须提交该节点 `GET config` 返回的 MD5 `checksum`。旧页面需要刷新后重新读取。
- 上游创建及更新支持 `host_ip`、`tls_host_verify`、`extra_options`、`node_ids`。其中 `node_ids` 是 JSON 数组字符串，例如 `"[3]"`；`"[]"` 表示所有节点。

同步仅写文件，仍需对目标节点重启加载。文件中引用的外部文件由节点管理，预览保留引用，不递归修改其内容。

## 验证

在 `backend` 运行 `go test ./...`。现场快照可放在仓库外，通过 `SMARTDNS_CONFIG_AUDIT_DIR` 指向目录后运行 `go test ./services -run TestExternalConfigSnapshots -v`。该可选测试只做离线生成比较，不连接节点；目录包含 `node-<id>.conf` 快照、表示计划状态的 `planned-servers.json` 和 `addresses.json`。不要将现场配置或数据库凭据提交到仓库。
