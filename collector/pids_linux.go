// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !nopids
// +build !nopids

package collector

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

var (
	pidsLabelNames = []string{"pid", "cmd"}
)

type pidsCollector struct {
	fs                       procfs.FS
	pidsCpuUtilization       *prometheus.Desc // 进程CPU利用率    %
	pidsMemUtilization       *prometheus.Desc // 进程的内存消耗占比   %
	pidsThreadNum            *prometheus.Desc // 进程所使用的线程数   Count
	pidsFdUsed               *prometheus.Desc // 进程所使用的文件描述符数    Count
	pidsReadDiskCount        *prometheus.Desc // 进程读取磁盘的次数   Count
	pidsWriteDiskCount       *prometheus.Desc // 进程写入磁盘的次数   Count
	pidsReadDiskBytes        *prometheus.Desc // 进程读取磁盘的字节数  Bytes
	pidsWriteDiskBytes       *prometheus.Desc // 进程写入磁盘的字节数  Bytes
	pidsNetworkReceiveBytes  *prometheus.Desc // 进程接收的网络字节数  Bytes/s
	pidsNetworkTransmitBytes *prometheus.Desc // 进程发送的网络字节数  Bytes/s
	pidsNetworkReceivePkg    *prometheus.Desc // 进程接收的网络包数量  Count
	pidsNetworktransmitPkg   *prometheus.Desc // 进程发送的网络包数量  Count
	voluntaryCtxtSwitches    *prometheus.Desc // 进程切换上下文数
	nonvoluntaryCtxtSwitches *prometheus.Desc // 进程切换上下文数
	logger                   log.Logger
}

func init() {
	registerCollector("pids", defaultEnabled, NewPidsStatCollector)
}

func getCpuUtilizationTop5() (map[string][]string, []string, error) {
	top5Msg, err := execCommand("top -b -n 1 -o %CPU | grep -v top| head -10 | tail -5")
	if err != nil {
		fmt.Errorf("couldn't get diskInode: %w", err)
		return nil, nil, err
	}
	return parseTopCmd(top5Msg)
}

func parseTopCmd(str string) (map[string][]string, []string, error) {
	var (
		pidsMsg  = map[string][]string{}
		pidsList = []string{}
	)

	for _, pidLine := range strings.Split(str, "\n") {

		parts := strings.Fields(pidLine)
		if len(parts) != 0 {
			dev := parts[0]
			pidsList = append(pidsList, parts[0])
			pidsMsg[dev] = parts[1:]
		}

	}

	return pidsMsg, pidsList, nil
}

// NewPidsStatCollector returns a new Collector exposing process data read from the proc filesystem.
func NewPidsStatCollector(logger log.Logger) (Collector, error) {
	fs, err := procfs.NewFS(*procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}
	subsystem := "pids"
	return &pidsCollector{
		fs: fs,
		pidsCpuUtilization: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "pidsCpuUtilization"),
			"CPU utilization of processes",
			pidsLabelNames, nil,
		),
		pidsMemUtilization: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "pidsMemUtilization"),
			"Process memory usage rate",
			pidsLabelNames, nil,
		),
		pidsThreadNum: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "pidsThreadNum"),
			"The number of threads used by the process.",
			pidsLabelNames, nil,
		),
		pidsFdUsed: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pidsFdUsed"),
			"The number of file descriptors used by the process",
			pidsLabelNames, nil,
		),
		voluntaryCtxtSwitches: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "voluntaryCtxtSwitches"),
			"voluntaryCtxtSwitches",
			pidsLabelNames, nil,
		),
		nonvoluntaryCtxtSwitches: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "nonvoluntaryCtxtSwitches"),
			"nonvoluntaryCtxtSwitches",
			pidsLabelNames, nil,
		),
		pidsReadDiskCount: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pidsReadDiskCount"),
			"pidsReadDiskCount",
			pidsLabelNames, nil,
		),
		pidsWriteDiskCount: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pidsWriteDiskCount"),
			"pidsWriteDisk",
			pidsLabelNames, nil,
		),
		pidsReadDiskBytes: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pidsReadDiskBytes"),
			"pidsReadDiskBytes",
			pidsLabelNames, nil,
		),
		pidsWriteDiskBytes: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pidsWriteDiskBytes"),
			"pidsWriteDiskBytes",
			pidsLabelNames, nil,
		),
		pidsNetworkReceiveBytes: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pidsNetworkReceiveBytes"),
			"pidsNetworkReceiveBytes",
			pidsLabelNames, nil,
		),
		pidsNetworkTransmitBytes: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pidsNetworkTransmitBytes"),
			"pidsNetworkTransmit",
			pidsLabelNames, nil,
		),
		pidsNetworkReceivePkg: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pidsNetworkReceivePkg"),
			"pidsNetworkReceivePkg",
			pidsLabelNames, nil,
		),
		pidsNetworktransmitPkg: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pidsNetworktransmitPkg"),
			"pidsNetworktransmitPkg",
			pidsLabelNames, nil,
		),
		logger: logger,
	}, nil
}

func (c *pidsCollector) getPidIoFile(pidStr string) (map[string]string, error) {
	file, err := os.Open(procFilePath(pidStr + "/io"))

	if err != nil {
		return nil, err
	}
	defer file.Close()

	return c.parsePidIoFile(file)
}

func (c *pidsCollector) parsePidIoFile(r io.Reader) (map[string]string, error) {
	var (
		pidIoMap = map[string]string{
			"rchar": "0",
			"wchar": "0",
			"syscr": "0",
			"syscw": "0",
			// "read_bytes":   "0",
			// "writes_bytes": "0",
		}
		scanner = bufio.NewScanner(r)
	)

	for scanner.Scan() {
		textLine := scanner.Text()
		parts := strings.Fields(textLine)
		if strings.Contains(textLine, "rchar") {
			pidIoMap["rchar"] = parts[1]
		}
		if strings.Contains(textLine, "wchar") {
			pidIoMap["wchar"] = parts[1]
		}
		if strings.Contains(textLine, "syscr") {
			pidIoMap["syscr"] = parts[1]
		}
		if strings.Contains(textLine, "syscw") {
			pidIoMap["syscw"] = parts[1]
		}
	}

	return pidIoMap, scanner.Err()
}

func (c *pidsCollector) getPidStatusFile(pidStr string) (map[string]string, error) {
	file, err := os.Open(procFilePath(pidStr + "/status"))
	// Threads  voluntary_ctxt_switches nonvoluntary_ctxt_switches
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return c.parsePidStatus(file)
}

func (c *pidsCollector) parsePidStatus(r io.Reader) (map[string]string, error) {
	var (
		pidMap = map[string]string{
			"Threads":                    "0",
			"voluntary_ctxt_switches":    "0",
			"nonvoluntary_ctxt_switches": "0",
		}
		scanner = bufio.NewScanner(r)
	)

	for scanner.Scan() {
		textLine := scanner.Text()
		parts := strings.Fields(textLine)
		if strings.Contains(textLine, "Threads") {
			pidMap["Threads"] = parts[1]
		}else if strings.Contains(textLine, "nonvoluntary_ctxt_switches") {
			pidMap["nonvoluntary_ctxt_switches"] = parts[1]
		}else if strings.Contains(textLine, "voluntary_ctxt_switches") {
			pidMap["voluntary_ctxt_switches"] = parts[1]
		}
	}

	return pidMap, scanner.Err()
}

func (c *pidsCollector) getFdNum(pidStr string) (float64, error) {
	fdFiles, err := os.ReadDir(procFilePath(pidStr + "/fd"))
	if err != nil {
		fmt.Errorf("getFdNum err:", err)
		return 0, err
	}
	return float64(len(fdFiles)), err
}

func (c *pidsCollector) Update(ch chan<- prometheus.Metric) error {
	pidsStats, pidsList, err := getCpuUtilizationTop5()
	if err != nil {
		return fmt.Errorf("couldn't get pidsStats: %w", err)
	}
	recFlow, recPkg, tmtFlow, tmtPkg, flowSec := sumPidFlow(pidsList)

	for ppid, stats := range pidsStats {

		cmd_name := stats[10]

		pCpuUtil, _ := strconv.ParseFloat(stats[7], 64)
		pMemUtil, _ := strconv.ParseFloat(stats[8], 64)

		ch <- prometheus.MustNewConstMetric(c.pidsCpuUtilization, prometheus.GaugeValue, pCpuUtil, ppid, cmd_name)
		ch <- prometheus.MustNewConstMetric(c.pidsMemUtilization, prometheus.GaugeValue, pMemUtil, ppid, cmd_name)

		// 从/proc/pid/status 获取结果
		pidThreadContxt, err := c.getPidStatusFile(ppid)
		if err != nil {
			fmt.Errorf("couldn't get pidThreadContxt: %w", err)
		} else {
			threadNum, _ := strconv.ParseFloat(pidThreadContxt["Threads"], 64)
			vCtxtSwitch, _ := strconv.ParseFloat(pidThreadContxt["voluntary_ctxt_switches"], 64)
			uvCtxtSwitch, _ := strconv.ParseFloat(pidThreadContxt["nonvoluntary_ctxt_switches"], 64)
			ch <- prometheus.MustNewConstMetric(c.pidsThreadNum, prometheus.GaugeValue, threadNum, ppid, cmd_name)
			ch <- prometheus.MustNewConstMetric(c.voluntaryCtxtSwitches, prometheus.CounterValue, vCtxtSwitch, ppid, cmd_name)
			ch <- prometheus.MustNewConstMetric(c.nonvoluntaryCtxtSwitches, prometheus.CounterValue, uvCtxtSwitch, ppid, cmd_name)
		}

		// 从/proc/pid/fd 获取结果
		pidFdNum, err := c.getFdNum(ppid)
		if err != nil {
			fmt.Errorf("couldn't get pidFdNum: %w", err)
		} else {
			ch <- prometheus.MustNewConstMetric(c.pidsFdUsed, prometheus.GaugeValue, pidFdNum, ppid, cmd_name)
		}

		// 从/proc/pid/io 获取结果
		pidsIoMap, err := c.getPidIoFile(ppid)
		if err != nil {
			fmt.Errorf("couldn't get pidIo: %w", err)
		} else {
			rchar, _ := strconv.ParseFloat(pidsIoMap["rchar"], 64)
			wchar, _ := strconv.ParseFloat(pidsIoMap["wchar"], 64)
			syscr, _ := strconv.ParseFloat(pidsIoMap["syscr"], 64)
			syscw, _ := strconv.ParseFloat(pidsIoMap["syscw"], 64)
			ch <- prometheus.MustNewConstMetric(c.pidsReadDiskBytes, prometheus.CounterValue, rchar, ppid, cmd_name)
			ch <- prometheus.MustNewConstMetric(c.pidsWriteDiskBytes, prometheus.CounterValue, wchar, ppid, cmd_name)
			ch <- prometheus.MustNewConstMetric(c.pidsReadDiskCount, prometheus.CounterValue, syscr, ppid, cmd_name)
			ch <- prometheus.MustNewConstMetric(c.pidsWriteDiskCount, prometheus.CounterValue, syscw, ppid, cmd_name)
		}

		pidsNetworkReceiveBytes := float64(recFlow[ppid])/flowSec
		pidsNetworkReceivePkg := float64(recPkg[ppid])/flowSec
		pidsNetworkTransmitBytes := float64(tmtFlow[ppid])/flowSec
		pidsNetworktransmitPkg := float64(tmtPkg[ppid])/flowSec
		ch <- prometheus.MustNewConstMetric(c.pidsNetworkReceiveBytes, prometheus.GaugeValue, pidsNetworkReceiveBytes, ppid, cmd_name)
		ch <- prometheus.MustNewConstMetric(c.pidsNetworkReceivePkg, prometheus.GaugeValue, pidsNetworkReceivePkg, ppid, cmd_name)
		ch <- prometheus.MustNewConstMetric(c.pidsNetworkTransmitBytes, prometheus.GaugeValue, pidsNetworkTransmitBytes, ppid, cmd_name)
		ch <- prometheus.MustNewConstMetric(c.pidsNetworktransmitPkg, prometheus.GaugeValue, pidsNetworktransmitPkg, ppid, cmd_name)

		// // 从/proc/pid/net/dev
		// pidNetStatus, err := c.getPidNetDevFile(ppid)
		// if err != nil {
		// 	fmt.Errorf("couldn't get pidNetStatus: %w", err)
		// } else {
		// 	for netDevName, netData := range pidNetStatus {
		// 		pidsNetworkReceiveBytes, _ := strconv.ParseFloat(netData[0], 64)
		// 		pidsNetworkReceivePkg, _ := strconv.ParseFloat(netData[1], 64)
		// 		pidsNetworkTransmitBytes, _ := strconv.ParseFloat(netData[2], 64)
		// 		pidsNetworktransmitPkg, _ := strconv.ParseFloat(netData[3], 64)
		// 		ch <- prometheus.MustNewConstMetric(c.pidsNetworkReceiveBytes, prometheus.GaugeValue, pidsNetworkReceiveBytes, ppid, cmd_name, netDevName)
		// 		ch <- prometheus.MustNewConstMetric(c.pidsNetworkReceivePkg, prometheus.GaugeValue, pidsNetworkReceivePkg, ppid, cmd_name, netDevName)
		// 		ch <- prometheus.MustNewConstMetric(c.pidsNetworkTransmitBytes, prometheus.GaugeValue, pidsNetworkTransmitBytes, ppid, cmd_name, netDevName)
		// 		ch <- prometheus.MustNewConstMetric(c.pidsNetworktransmitPkg, prometheus.GaugeValue, pidsNetworktransmitPkg, ppid, cmd_name, netDevName)
		// 	}
		// }
	}

	return nil
}
