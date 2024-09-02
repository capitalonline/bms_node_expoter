# bms_node_expoter

裸金属基于node_expoter二次开发适配

**手动安装：**

1. 将node_exporter 存放于/usr/local/bin/ 路径下

2. 将node-exporter.service 存放于 /usr/lib/systemd/system/路径下

3. 启动服务和设置自启动，确保任务为running状态即可
   
   ```
   systemctl start node-exporter.service
   
   systemctl enable node-exporter.service
   
   systemctl status node-exporter.service
   ```


