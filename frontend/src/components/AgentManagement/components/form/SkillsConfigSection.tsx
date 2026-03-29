/**
 * SkillsConfigCard - 技能配置卡片
 */
import React from 'react';
import { Button, Card, Form, Select, Space, Tag } from 'antd';
import { ThunderboltOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../../types/agent';
import type { Skill } from '../../../../api/skillApi';

interface SkillsConfigCardProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
  skillsOptions: Skill[];
}

export const SkillsConfigCard: React.FC<SkillsConfigCardProps> = ({
  form, editing, editingSections, toggleSectionEdit, handlePatchSection, skillsOptions,
}) => {
  const isEditing = editingSections.skillsConfig;

  const handleSave = () => {
    const skills = form.getFieldValue('skills_list') as string[] || [];
    handlePatchSection('skillsConfig', { skills_list: skills });
  };

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><ThunderboltOutlined /> 技能配置</span>}
      style={{ marginBottom: 8 }}
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Button size="small" type="primary" onClick={handleSave}>保存</Button>
              <Button size="small" onClick={() => toggleSectionEdit('skillsConfig')}>取消</Button>
            </Space>
          ) : (
            <Button size="small" onClick={() => toggleSectionEdit('skillsConfig')}>编辑</Button>
          )
        ) : null
      }
    >
      {!isEditing ? (
        <div>
          {(form.getFieldValue('skills_list') as string[] || []).length === 0 ? (
            <span style={{ color: '#999' }}>未配置技能（不限）</span>
          ) : (
            <Space wrap>
              {(form.getFieldValue('skills_list') as string[] || []).map((s) => (
                <Tag key={s} color="blue">{s}</Tag>
              ))}
            </Space>
          )}
        </div>
      ) : (
        <Form.Item label="Skills（可多选/自定义）" name="skills_list" style={{ marginBottom: 0 }}>
          <Select mode="tags" placeholder="从列表选择或输入添加"
            options={skillsOptions.map((s) => ({
              value: s.name,
              label: s.description ? `${s.name} - ${s.description}` : s.name,
            }))} />
        </Form.Item>
      )}
    </Card>
  );
};

export default SkillsConfigCard;