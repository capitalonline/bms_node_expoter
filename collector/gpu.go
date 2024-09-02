// +build darwin linux openbsd
// +build !nogpu

package collector

import (
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type gpuCollector struct {
	info                     gpuCache
	gpuDriverVersion         *prometheus.Desc
	total                    *prometheus.Desc
	used                     *prometheus.Desc
	free                     *prometheus.Desc
	utilization              *prometheus.Desc
	temp                     *prometheus.Desc
	memUtilization           *prometheus.Desc //显存使用绿
	maxClock                 *prometheus.Desc //最大时钟频率
	fanSpeed                 *prometheus.Desc //风扇数度 in %
	computeRunningProcesses  *prometheus.Desc //运行计算的进程数量
	graphicsRunningProcesses *prometheus.Desc //运行图像处理的进程数量
	maxPcieLinkWidth         *prometheus.Desc //最大PCIE的连接带宽
	pcieThroughput           *prometheus.Desc //PCIE的吞吐
	performanceState         *prometheus.Desc //性能状态
	powerManagementDefLimit  *prometheus.Desc //电源管理的默认上限
	powerManagementLimit     *prometheus.Desc //电源管理的上限
	powerState               *prometheus.Desc //电源状态
	powerUsage               *prometheus.Desc //电源使用量
	temperatureThreshold     *prometheus.Desc //gpu温度限速阈值
	gpuCount *prometheus.Desc //GPU数量的指标
}

func init() {
	registerCollector("gpu", defaultEnabled, NewGpuCollector)
}

// NewGpuCollector data come from nvidia-smi -q
func NewGpuCollector(logger log.Logger) (Collector, error) {
	info := gpuCache{}

	return &gpuCollector{
		info: info,
		gpuDriverVersion: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "gpuDriverVersion"),
			"GPU driver version",
			gpuGeneralLabelNames, nil,
		),
		memUtilization: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "memUtilization"),
			"Framebuffer memory utilization (in %).",
			gpuLabelNames, nil,
		),
		maxClock: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "maxClock"),
			"GPU Max Clock information.",
			gpuLabelNames, nil,
		),
		fanSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "fanSpeed"),
			"fan speed (in %).",
			gpuLabelNames, nil,
		),
		computeRunningProcesses: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "computeRunningProcesses"),
			"number of running compute processes.",
			gpuLabelNames, nil,
		),
		graphicsRunningProcesses: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "graphicsRunningProcesses"),
			"number of running graphics processes.",
			gpuLabelNames, nil,
		),
		maxPcieLinkWidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "maxPcieLinkWidth"),
			"Max PCIE link width.",
			gpuLabelNames, nil,
		),
		pcieThroughput: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "pcieThroughput"),
			"PCI-E throughput.",
			gpuLabelNames, nil,
		),
		performanceState: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "performanceState"),
			"performance status . 0 is for Maximum Performance.",
			gpuLabelNames, nil,
		),
		powerManagementDefLimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "powerManagementDefLimit"),
			"power management default max value (in Watt).",
			gpuLabelNames, nil,
		),
		powerManagementLimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "powerManagementLimit"),
			"power management max value (in Watt).",
			gpuLabelNames, nil,
		),
		powerState: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "powerState"),
			"power state. 0 stands for The operation was successful..,all other values are abnormal",
			gpuLabelNames, nil,
		),
		powerUsage: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "powerUsage"),
			"current power usage (in Watt).",
			gpuLabelNames, nil,
		),
		temperatureThreshold: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "temperatureThreshold"),
			"GPU will encounter threshold when temperature is above this value (in Watt).",
			gpuLabelNames, nil,
		),
		total: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "total"),
			"Framebuffer memory total (in MiB).",
			gpuLabelNames, nil,
		),
		used: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "used"),
			"Framebuffer memory used (in MiB).",
			gpuLabelNames, nil,
		),
		free: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "free"),
			"Framebuffer memory free (in MiB).",
			gpuLabelNames, nil,
		),
		utilization: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "utilization"),
			"GPU utilization (in %).",
			gpuLabelNames, nil,
		),
		temp: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "temp"),
			"GPU temperature (in C).",
			gpuLabelNames, nil,
		),
		gpuCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, gpuCollectorSubsystem, "gpuCount"),
			"Number of GPUs.",
			gpuGeneralLabelNames, nil,
		),
	}, nil
}

func (this *gpuCollector) Update(ch chan<- prometheus.Metric) error {
	if err := this.updateStat(ch); err != nil {
		return err
	}
	return nil
}

func (this *gpuCollector) updateStat(ch chan<- prometheus.Metric) error {
	stats, err := this.info.Stat()
	if err != nil {
		return err
	}

	for _, gpuStat := range stats {
		ch <- prometheus.MustNewConstMetric(this.total, prometheus.GaugeValue, float64(gpuStat.TotalMem/1024/1024), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.used, prometheus.GaugeValue, float64(gpuStat.UsedMem/1024/1024), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.free, prometheus.GaugeValue, float64(gpuStat.FreeMem/1024/1024), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.utilization, prometheus.GaugeValue, float64(gpuStat.Utilization), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.temp, prometheus.GaugeValue, float64(gpuStat.Temp), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.memUtilization, prometheus.GaugeValue, float64(gpuStat.MemUtilization), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.maxClock, prometheus.GaugeValue, float64(gpuStat.MaxClock), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.fanSpeed, prometheus.GaugeValue, float64(gpuStat.FanSpeed), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.computeRunningProcesses, prometheus.GaugeValue, float64(gpuStat.ComputeRunningProcesses), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.graphicsRunningProcesses, prometheus.GaugeValue, float64(gpuStat.GraphicsRunningProcesses), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.maxPcieLinkWidth, prometheus.GaugeValue, float64(gpuStat.MaxPcieLinkWidth), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.pcieThroughput, prometheus.GaugeValue, float64(gpuStat.PcieThroughput), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.performanceState, prometheus.GaugeValue, float64(gpuStat.PerformanceState), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.powerManagementDefLimit, prometheus.GaugeValue, gpuStat.PowerManagementDefLimit, gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.powerManagementLimit, prometheus.GaugeValue, gpuStat.PowerManagementLimit, gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.powerState, prometheus.GaugeValue, float64(gpuStat.PowerState), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.powerUsage, prometheus.GaugeValue, gpuStat.PowerUsage, gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.temperatureThreshold, prometheus.GaugeValue, float64(gpuStat.TemperatureThreshold), gpuStat.Host, gpuStat.ID, gpuStat.UUID, gpuStat.Types)
		ch <- prometheus.MustNewConstMetric(this.gpuCount, prometheus.GaugeValue, float64(GpuCount), gpuStat.Host,gpuStat.Types,GPUDriverVersion)
		ch <- prometheus.MustNewConstMetric(this.gpuDriverVersion, prometheus.GaugeValue, 1, gpuStat.Host, gpuStat.Types, GPUDriverVersion)
	}
	return nil
}
