这个版本添加了Nvidia GPU信息的抓取，所以编译的时候需要nvml.h 复制到/usr/local/cuda/include 目录里面

1. 复制依赖
   
   ```bash
   mkdir -p /usr/local/cuda/include
   cp -p nvml.h /usr/local/cuda/include
   ```

2. 定义GOPROXY变量
   
   ```bash
   export GOPROXY=https://goproxy.cn
   ```

3. 编译项目go1.22.4
   
   ```bash
   cd /root/node_exporter
   go get
   go build
   ```

4. 在如果一切正常在目录中会生成node_exporter的可执行二进制文件

5. 运行即可
   
   ```bash
   ./node_exporter --web.listen-address=":9200"
   ```

6. 打包arm64架构的方法
   
   * apt install gcc-aarch64-linux-gnu 
   * env CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC_FOR_TARGET=gcc-aarch64-linux-gnu CC=aarch64-linux-gnu-gcc go build

7. 进程流量监控需要添加 libpcap 库
   
   ```
   sudo apt-get update
   sudo apt-get install libpcap-dev
   ```
