import React, { useState, useEffect } from "react";
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  message,
  Popconfirm,
  Switch,
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SyncOutlined,
} from "@ant-design/icons";
import {
  getServers,
  addServer,
  updateServer,
  deleteServer,
  getGroups,
  getNodes,
} from "../../api";
import dayjs from "dayjs";
import { nodeScope, serverFormValues, serverPayload } from "./serverForm";

const { Option } = Select;

const ServerManager = () => {
  const [servers, setServers] = useState([]);
  const [groups, setGroups] = useState([]);
  const [nodes, setNodes] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingServer, setEditingServer] = useState(null);
  const [form] = Form.useForm();

  useEffect(() => {
    loadServers();
    loadGroups();
    getNodes().then((res) => setNodes((res.data || []).map(({ id, name, host }) => ({ id, name, host })))).catch(() => message.error("加载节点作用域失败"));
  }, []);

  const loadGroups = async () => {
    try {
      const response = await getGroups();
      setGroups(response.data || []);
    } catch (error) {
      message.error("加载分组失败");
    }
  };

  const loadServers = async () => {
    try {
      setLoading(true);
      const response = await getServers();
      setServers(response.data || []);
    } catch (error) {
      message.error("加载DNS服务器失败");
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    setEditingServer(null);
    form.resetFields();
    form.setFieldsValue({ type: "udp", groups: [], node_ids: [], exclude_default: false, host_ip: "", tls_host_verify: "", extra_options: "" });
    setModalVisible(true);
  };

  const handleEdit = (record) => {
    try {
      form.setFieldsValue(serverFormValues(record));
    } catch (error) {
      message.error("该记录的节点作用域无效，请先核对配置");
      return;
    }
    setEditingServer(record);
    setModalVisible(true);
  };

  const handleDelete = async (id) => {
    try {
      await deleteServer(id);
      message.success("删除成功");
      loadServers();
    } catch (error) {
      message.error("删除失败");
    }
  };

  const handleSubmit = async () => {
    try {
      const values = serverPayload(await form.validateFields());

      if (editingServer) {
        await updateServer(editingServer.id, values);
        message.success("更新成功");
      } else {
        await addServer(values);
        message.success("添加成功");
      }

      setModalVisible(false);
      loadServers();
    } catch (error) {
      message.error("操作失败");
    }
  };

  const getTypeColor = (type) => {
    const colors = {
      https: "green",
      tls: "blue",
      tcp: "orange",
      udp: "default",
    };
    return colors[type] || "default";
  };


  const columns = [
    {
      title: "服务器地址",
      dataIndex: "address",
      key: "address",
      width: 300,
      render: (text) => <code>{text}</code>,
    },
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      width: 100,
      render: (type) => (
        <Tag color={getTypeColor(type)}>{type?.toUpperCase()}</Tag>
      ),
    },
    {
      title: "分组",
      dataIndex: "groups",
      key: "groups",
      width: 200,
      render: (groups) => (
        <>
          {groups?.map((group) => (
            <Tag key={group} color="cyan">
              {group}
            </Tag>
          ))}
          {(!groups || groups.length === 0) && "默认组"}
        </>
      ),
    },
    {
      title: "排除默认组",
      dataIndex: "exclude_default",
      key: "exclude_default",
      width: 120,
      render: (exclude) => (
        <Tag color={exclude ? "warning" : "default"}>
          {exclude ? "是" : "否"}
        </Tag>
      ),
    },
    {
      title: "生效节点", key: "scope", width: 210,
      render: (_, row) => {
        try { const ids = nodeScope(row.node_ids); return ids.length ? ids.map((id) => nodes.find((n) => n.id === id)?.name || `节点 ${id}`).join("、") : "所有节点"; }
        catch { return <Tag color="error">作用域无效</Tag>; }
      },
    },
    {
      title: "连接与证书", key: "advanced", width: 250,
      render: (_, row) => <div><div>IP：{row.host_ip || "自动解析"}</div><div>校验：{row.tls_host_verify || "随上游地址"}</div></div>,
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 180,
      render: (time) => dayjs(time).format("YYYY-MM-DD HH:mm:ss"),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 150,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div
        style={{
          marginBottom: 16,
          display: "flex",
          justifyContent: "space-between",
        }}
      >
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            添加DNS服务器
          </Button>
          <Button icon={<SyncOutlined />} onClick={loadServers}>
            刷新
          </Button>
        </Space>
        <span style={{ color: "#666" }}>共 {servers.length} 个服务器</span>
      </div>

      <Table
        columns={columns}
        dataSource={servers}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1750 }}
        pagination={{
          pageSize: 20,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条记录`,
        }}
      />

      <Modal
        title={editingServer ? "编辑DNS服务器" : "添加DNS服务器"}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={720}
        okText="确定"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="address"
            label="服务器地址"
            rules={[{ required: true, message: "请输入服务器地址" }]}
            extra="支持格式: IP地址、域名、https://、tls://"
          >
            <Input placeholder="例如: 8.8.8.8 或 https://dns.google/dns-query" />
          </Form.Item>

          <Form.Item
            name="type"
            label="服务器类型"
            rules={[{ required: true, message: "请选择服务器类型" }]}
          >
            <Select placeholder="选择服务器类型">
              <Option value="udp">UDP</Option>
              <Option value="tcp">TCP</Option>
              <Option value="tls">TLS</Option>
              <Option value="https">HTTPS</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="groups"
            label="服务器分组"
            extra="留空时仅参与默认组；也可以选择或输入自定义分组"
          >
            <Select
              mode="tags"
              placeholder="选择分组"
              options={groups.map((g) => ({
                label: (
                  <Space>
                    <Tag color={g.color}>{g.name}</Tag>
                  </Space>
                ),
                value: g.name,
              }))}
            />
          </Form.Item>

          <Form.Item
            name="exclude_default"
            label="排除默认组"
            valuePropName="checked"
            extra="开启后仅供指定分组使用"
          >
            <Switch />
          </Form.Item>
          <Form.Item name="host_ip" label="固定连接 IP" extra="可选。直接连接此 IP，保留上游 URL 的域名。">
            <Input placeholder="例如：8.8.8.8 或 1.1.1.1" allowClear />
          </Form.Item>
          <Form.Item name="tls_host_verify" label="TLS 证书校验域名" extra="可选。明确要求证书匹配的域名，主要用于 DoH/DoT。">
            <Input placeholder="例如：dns.google" allowClear />
          </Form.Item>
          <Form.Item name="extra_options" label="其他高级参数" extra="保留未单独列出的 SmartDNS 参数，例如 -http-host。">
            <Input placeholder="可选，仅填写额外参数" />
          </Form.Item>
          <Form.Item name="node_ids" label="生效节点" extra="留空应用所有节点；指定节点后只向这些节点同步。">
            <Select mode="multiple" allowClear placeholder="所有节点" options={nodes.map((n) => ({value:n.id,label:`${n.name} (${n.host})`}))} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ServerManager;
