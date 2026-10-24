# 在管理台本机执行定时脚本

自定义脚本配置增加可选字段 `execution_mode`。省略或设置为 `remote` 时继续沿用远程节点执行方式；设置为 `local` 时，在管理服务所在环境执行，结果与错误写入现有任务历史。

```json
{
  "execution_mode": "local",
  "node_ids": [],
  "script": "exec python3 /persistent/tools/update.py --config /persistent/private/config.json --apply",
  "working_dir": "/persistent/tools",
  "timeout": 900,
  "env_vars": {},
  "run_as_user": ""
}
```

通过任务管理页面的配置 JSON 可查看和修改这些字段。本机任务不能同时指定远程节点；脚本使用管理服务已有的系统用户，不执行提权或用户切换。运行所需文件应对该用户开放最小权限，敏感配置保持 `600`，并存储在持久化目录。

脚本遇到非零退出状态、超时或取消时，任务会标记为失败，标准输出与错误输出保留在执行历史中。需要超时控制的长期脚本应使用 `exec` 启动实际程序，避免只取消外层 shell。调度表达式使用包含秒的六段格式，例如 `0 30 6 * * *`。

可用以下命令执行不依赖数据库或远程节点的测试：

```sh
cd backend
go test services/custom_script_local.go services/custom_script_local_test.go
```
