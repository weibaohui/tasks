import React from 'react';
import { Card, Tabs } from 'antd';
import { TokenManagement } from '../components/TokenManagement/TokenManagement';

const { TabPane } = Tabs;

export const SettingsPage: React.FC = () => {
  return (
    <Card>
      <Tabs defaultActiveKey="tokens">
        <TabPane tab="API Token" key="tokens">
          <TokenManagement />
        </TabPane>
      </Tabs>
    </Card>
  );
};

export default SettingsPage;
