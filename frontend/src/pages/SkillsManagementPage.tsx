/**
 * Skills 管理页面
 * 显示所有可用技能及其详情
 */
import React, { useCallback, useEffect, useState } from 'react';
import { Card, Drawer, Grid, Space, Spin, Table, Tag, Typography, Button, Descriptions, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { RobotOutlined, ReloadOutlined } from '@ant-design/icons';
import { listSkills, getSkill, type Skill } from '../api/skillApi';

const { useBreakpoint } = Grid;
const { Title } = Typography;

export const SkillsManagementPage: React.FC = () => {
  const screens = useBreakpoint();
  const [items, setItems] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false);
  const [selectedSkill, setSelectedSkill] = useState<Skill | null>(null);
  const [skillContent, setSkillContent] = useState<string>('');
  const [loadingDetail, setLoadingDetail] = useState(false);

  const fetchList = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listSkills();
      setItems(data);
    } catch (_error) {
      message.error('获取技能列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const openDetail = useCallback(async (skill: Skill) => {
    setSelectedSkill(skill);
    setDetailDrawerOpen(true);
    setLoadingDetail(true);
    try {
      const detail = await getSkill(skill.name);
      setSkillContent(detail.content);
    } catch (_error) {
      setSkillContent('无法加载技能内容');
    } finally {
      setLoadingDetail(false);
    }
  }, []);

  const closeDetail = useCallback(() => {
    setDetailDrawerOpen(false);
    setSelectedSkill(null);
    setSkillContent('');
  }, []);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const columns: ColumnsType<Skill> = [
      {
            title: '操作',
            key: 'action',
            render: (_: unknown, record: Skill) => (
              <Button
                onClick={() => openDetail(record)} type="link" size="small" style={{ padding: 0 }}
              >
                查看
              </Button>
            ),
              width: 100,
              fixed: 'left' as const
          },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (name: string) => <strong>{name}</strong>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 100,
      render: (source: string) => (
        <Tag color={source === 'builtin' ? 'blue' : 'green'}>
          {source === 'builtin' ? '内置' : '工作区'}
        </Tag>
      ),
    },
    {
      title: '状态',
      key: 'available',
      width: 100,
      render: (_: unknown, record: Skill) => (
        record.available === false ? (
          <Tag color="red">不可用</Tag>
        ) : (
          <Tag color="success">可用</Tag>
        )
      ),
    },
    {
      title: '需求',
      dataIndex: 'requires',
      key: 'requires',
      width: 200,
      ellipsis: true,
      render: (requires: string) => requires ? <Tag color="orange">{requires}</Tag> : '-',
    }
  ];

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={<Title level={screens.xs ? 4 : 3} style={{ margin: 0 }}>技能管理</Title>}
        extra={
          <Space>
            <Button onClick={fetchList} icon={<ReloadOutlined />} loading={loading}>
              刷新
            </Button>
          </Space>
        }
      >
        <Table<Skill>
          rowKey="name"
          loading={loading}
          dataSource={items}
          columns={columns}
          size={screens.xs ? 'small' : 'middle'}
          pagination={{ pageSize: 10, showSizeChanger: true }}
          scroll={{ x: 800 }}
        />
      </Card>

      <Drawer
        title={<Space><RobotOutlined /> 技能详情: {selectedSkill?.name}</Space>}
        placement="right"
        open={detailDrawerOpen}
        onClose={closeDetail}
        width={screens.xs ? '100%' : 720}
        destroyOnClose
      >
        {selectedSkill && (
          <div>
            <Descriptions bordered column={1} size="small" style={{ marginBottom: 16 }}>
              <Descriptions.Item label="名称">
                <strong>{selectedSkill.name}</strong>
              </Descriptions.Item>
              <Descriptions.Item label="来源">
                <Tag color={selectedSkill.source === 'builtin' ? 'blue' : 'green'}>
                  {selectedSkill.source === 'builtin' ? '内置' : '工作区'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                {selectedSkill.available === false ? (
                  <Tag color="red">不可用 ({selectedSkill.requires})</Tag>
                ) : (
                  <Tag color="success">可用</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="描述">
                {selectedSkill.description || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="路径">
                {selectedSkill.path || '-'}
              </Descriptions.Item>
            </Descriptions>

            <Title level={5}>技能内容</Title>
            {loadingDetail ? (
              <div style={{ textAlign: 'center', padding: 40 }}>
                <Spin />
              </div>
            ) : (
              <Card styles={{ body: { padding: 0 } }}>
                <pre style={{
                  margin: 0,
                  padding: 16,
                  overflow: 'auto',
                  maxHeight: 500,
                  fontSize: 12,
                  fontFamily: 'monospace',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                }}>
                  {skillContent}
                </pre>
              </Card>
            )}
          </div>
        )}
      </Drawer>
    </div>
  );
};
