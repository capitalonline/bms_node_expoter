// +build !nogpu

package collector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"github.com/prometheus/node_exporter/nvml"
	"math"
	"os"
	"os/exec"
	"strconv"
)

var hostname string
var GPUDriverVersion string

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	if err := nvml.Init(); err != nil {
		fmt.Printf("nvml error: %+v", err)
		return
	}
	defer nvml.Shutdown()
	//获取驱动版本
	ver, err := nvml.SystemGetDriverVersion()
	if err != nil {
		failedMsg("SystemGetDriverVersion", err)
	} else {
		fmt.Printf("SystemGetDriverVersion: %s\n", ver)
		GPUDriverVersion = ver

	}

}

type gpuInfo struct {
	TotalMem                 uint64  //总的显存，单位是Byte
	UsedMem                  uint64  //使用的显存,单位是Byte
	FreeMem                  uint64  //剩余的显存，单位是Byte
	Utilization              uint    //显卡使用率,单位是%
	MemUtilization           uint    //显存使用率，单位%
	MaxClock                 uint    //最大时钟频率
	FanSpeed                 uint    //风扇数度 in %
	ComputeRunningProcesses  int     //运行计算的进程数量
	GraphicsRunningProcesses int     //运行图像处理的进程数量
	MaxPcieLinkWidth         uint    //最大PCIE的连接带宽
	PcieThroughput           uint    //PCIE的吞吐
	PerformanceState         uint    //性能状态
	PowerManagementDefLimit  float64 //电源管理的默认上限
	PowerManagementLimit     float64 //电源管理的上限
	PowerState               uint    //电源状态
	PowerUsage               float64 //电源使用量
	TemperatureThreshold     uint    //gpu温度限速阈值
	Temp                     uint    //温度
	gpuCount				 uint    //gpu数量
	Host                     string
	UUID                     string
	ID                       string
	Types                    string
}

type gpuCache struct{}
var GpuCount uint;

func (this gpuCache) Stat() ([]gpuInfo, error) {
	// lockSuccess := locker.Lock()
	// defer locker.Unlock()
	// if !lockSuccess {
	// 	fmt.Println("获取采集GPU locker失败")
	// 	return nil, nil
	// }
	var (
		result []gpuInfo
		// err    error
	)

	if err := nvml.Init(); err != nil {
		fmt.Printf("nvml error: %+v", err)
		return nil, err
	}
	defer nvml.Shutdown()

	result = []gpuInfo{}
	var tmp gpuInfo
	num, err := nvml.DeviceGetCount()
	GpuCount=num
	if err != nil {
		GpuCount=0
		failedMsg("DeviceGetCount", err)
	}

	for i := uint(0); i < num; i++ {
		//fmt.Println("============")
		tmp = gpuInfo{}
		dev, err := nvml.DeviceGetHandleByIndex(i)
		if err != nil {
			failedMsg("DeviceGetHandleByIndex", err)
		}

		//获取显卡的编号
		minor, err := dev.DeviceGetMinorNumber()
		if err != nil {
			failedMsg("DeviceGetMinorNumber", err)
		} else {
			tmp.ID = strconv.Itoa(int(minor))
		}
		//获取GPU里面计算运行的进程数量
		processes, err := dev.DeviceGetComputeRunningProcesses(32)
		if err != nil {
			failedMsg("DeviceGetComputeRunningProcesses", err)
		} else {
			tmp.ComputeRunningProcesses = len(processes)
			//for _, proc := range processes {
			//	fmt.Printf("\tpid: %d, usedMemory: %d", proc.Pid, proc.UsedGPUMemory)
			//}
		}

		//获取频率抑制的原因
		//reasons, err := dev.DeviceGetCurrentClocksThrottleReasons()
		//if err != nil {
		//	failedMsg("DeviceGetCurrentClocksThrottleReasons", err)
		//} else {
		//	fmt.Printf("DeviceGetCurrentClocksThrottleReasons: %d\n", len(reasons))
		//	for _, reason := range reasons {
		//		fmt.Printf("\tReason: %+v\n", reason)
		//	}
		//}

		//nvidia-smi 里面的Display.A，是否允许显示？
		//display, err := dev.DeviceGetDisplayMode()
		//if err != nil {
		//	failedMsg("DeviceGetDisplayMode", err)
		//} else {
		//	fmt.Printf("DeviceGetDisplayMode: %+v\n", display)
		//}

		//电源的上限
		//powerLimit, err := dev.DeviceGetEnforcedPowerLimit()
		//if err != nil {
		//	failedMsg("DeviceGetEnforcedPowerLimit", err)
		//} else {
		//	fmt.Printf("DeviceGetEnforcedPowerLimit: %d\n", powerLimit)
		//}

		//风扇的速度，in %
		speed, err := dev.DeviceGetFanSpeed()
		if err != nil {
			failedMsg("DeviceGetFanSpeed", err)
		} else {
			tmp.FanSpeed = speed
		}

		//显卡的运行程序?
		gRunningProcs, err := dev.GetGraphicsRunningProcesses(10)
		if err != nil {
			failedMsg("GetGraphicsRunningProcesses", err)
		} else {
			tmp.GraphicsRunningProcesses = len(gRunningProcs)
			//
			//for _, proc := range gRunningProcs {
			//	fmt.Printf("\t%d %d\n", proc.Pid, proc.UsedGPUMemory)
			//}
		}

		//最大时钟频率?
		maxClock, err := dev.DeviceGetMaxClockInfo(nvml.CLOCK_MEM)
		if err != nil {
			failedMsg("DeviceGetMaxClockInfo", err)
		} else {
			tmp.MaxClock = maxClock
		}

		//PCIE的带宽
		maxWidth, err := dev.DeviceGetMaxPcieLinkWidth()
		if err != nil {
			failedMsg("DeviceGetMaxPcieLinkWidth", err)
		} else {
			tmp.MaxPcieLinkWidth = maxWidth
		}

		//显存的使用情况
		memFree, memUsed, memTotal, err := dev.DeviceGetMemoryInfo()
		if err != nil {
			failedMsg("DeviceGetMemoryInfo", err)
		} else {
			tmp.TotalMem = memTotal
			tmp.FreeMem = memFree
			tmp.UsedMem = memUsed

		}

		//显卡名称
		name, err := dev.DeviceGetName()
		if err != nil {
			failedMsg("DeviceGetName", err)
		} else {
			//fmt.Printf("DeviceGetName: %s\n", name)
			tmp.Types = name
		}

		//pcie的吞吐
		throughput, err := dev.DeviceGetPcieThroughput(nvml.PCIE_UTIL_RX_BYTES)
		if err != nil {
			failedMsg("DeviceGetPcieThroughput", err)
		} else {
			tmp.PcieThroughput = throughput
		}

		//性能状态
		performState, err := dev.DeviceGetPerformanceState()
		if err != nil {
			failedMsg("DeviceGetPerformanceState", err)
		} else {
			tmp.PerformanceState = performState
		}

		//电源的管理默认最大值
		powerManagementDefLimit, err := dev.DeviceGetPowerManagementDefaultLimit()
		if err != nil {
			failedMsg("DeviceGetPowerManagementDefaultLimit", err)
		} else {
			tmp.PowerManagementDefLimit = float64(powerManagementDefLimit/1000) / math.Pow10(0)
		}
		//电源的管理最大值
		powerManagementLimit, err := dev.DeviceGetPowerManagementLimit()
		if err != nil {
			failedMsg("DeviceGetPowerManagementLimit", err)
		} else {
			tmp.PowerManagementLimit = float64(powerManagementLimit/1000) / math.Pow10(0)
		}
		//电源使用，值/1000 = 多少瓦，56255/1000 = 56W
		powerUsage, err := dev.DeviceGetPowerUsage()
		if err != nil {
			failedMsg("DeviceGetPowerUsage", err)
		} else {
			tmp.PowerUsage = float64(powerUsage/1000) / math.Pow10(0)
		}
		//管理的上下限
		//minLimit, maxLimit, err := dev.DeviceGetPowerManagementLimitConstraints()
		//if err != nil {
		//	failedMsg("DeviceGetPowerManagementLimitConstraints", err)
		//} else {
		//	fmt.Printf("DeviceGetPowerManagementLimitConstraints: %d, %d\n", minLimit, maxLimit)
		//}
		//是否电源管理模式
		//powerManagementMode, err := dev.DeviceGetPowerManagementMode()
		//if err != nil {
		//	failedMsg("DeviceGetPowerManagementMode", err)
		//} else {
		//	fmt.Printf("DeviceGetPowerManagementMode: %+v\n", powerManagementMode)
		//}
		//电源状态
		powerState, err := dev.DeviceGetPowerState()
		if err != nil {
			failedMsg("DeviceGetPowerState", err)
		} else {
			//fmt.Printf("DeviceGetPowerState: %d\n", powerState)
			tmp.PowerState = powerState
		}

		//GPU温度
		temper, err := dev.DeviceGetTemperature()
		if err != nil {
			failedMsg("DeviceGetTemperature", err)
		} else {
			tmp.Temp = temper
		}
		//GPU温度限速阈值
		temperThreshold, err := dev.DeviceGetTemperatureThreshold(nvml.TEMPERATURE_THRESHOLD_SLOWDOWN)
		if err != nil {
			failedMsg("DeviceGetTemperatureThreshold", err)
		} else {
			tmp.TemperatureThreshold = temperThreshold
		}

		//gpu的UUID
		uuid, err := dev.DeviceGetUUID()
		if err != nil {
			failedMsg("DeviceGetUUID", err)
		} else {
			tmp.UUID = uuid
		}
		util, err := dev.DeviceGetUtilizationRates()
		if err != nil {
			failedMsg("DeviceGetUtilizationRates", err)
		} else {
			tmp.Utilization = util.GPU
			//util.Memory是内存的使用率
			tmp.MemUtilization = util.Memory
		}
		tmp.Host = hostname
		result = append(result, tmp)
	}

	//for n, x := range strings.Split(data, "\n") {
	//	// fmt.Println(n, n%15, n/15, x)
	//	log.Debug(n, n%15, n/15, x)
	//	if n%15 == 0 && n != 0 {
	//		result = append(result, tmp)
	//		tmp = gpuInfo{}
	//		tmp.Host = hostname
	//		tmp.Types = strings.TrimSpace(x)
	//	} else if n%15 == 0 && n == 0 {
	//		tmp = gpuInfo{}
	//		tmp.Host = hostname
	//		tmp.Types = strings.TrimSpace(x)
	//	} else if n%15 == 1 {
	//		tmp.UUID = strings.TrimSpace(x)
	//	} else if n%15 == 2 {
	//		tmp.Count = strings.TrimSpace(x)
	//	} else if n%15 == 3 {
	//		tmp.TotalMem, _ = strconv.ParseFloat(strings.TrimSpace(strings.Split(x, " ")[0]), 64)
	//	} else if n%15 == 4 {
	//		tmp.UsedMem, _ = strconv.ParseFloat(strings.TrimSpace(strings.Split(x, " ")[0]), 64)
	//	} else if n%15 == 5 {
	//		tmp.FreeMem, _ = strconv.ParseFloat(strings.TrimSpace(strings.Split(x, " ")[0]), 64)
	//	} else if n%15 == 9 {
	//		tmp.Utilization, _ = strconv.ParseFloat(strings.TrimSpace(strings.Split(x, " ")[0]), 64)
	//	} else if n%15 == 14 {
	//		tmp.Temp, _ = strconv.ParseFloat(strings.TrimSpace(strings.Split(x, " ")[0]), 64)
	//	}
	//}
	return result, nil
}

func execCommand(cmd string) (string, error) {
	pipeline := exec.Command("/bin/sh", "-c", cmd)
	var out bytes.Buffer
	var stderr bytes.Buffer
	pipeline.Stdout = &out
	pipeline.Stderr = &stderr
	err := pipeline.Run()
	if err != nil {
		return stderr.String(), err
	}
	return out.String(), nil
}

func test() {
	tmp := gpuCache{}
	x, err := tmp.Stat()
	if err != nil {
		panic(err)
	}

	xxx, _ := json.Marshal(x)
	fmt.Println(string(xxx))
}
