package collector

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// func execCommandBash(cmd string) (string, error) {
// 	pipeline := exec.Command("/bin/bash", "-c", cmd)
// 	var out bytes.Buffer
// 	var stderr bytes.Buffer
// 	pipeline.Stdout = &out
// 	pipeline.Stderr = &stderr
// 	err := pipeline.Run()
// 	if err != nil {
// 		return stderr.String(), err
// 	}
// 	return out.String(), nil
// }

func top5localPort(pids []string) (map[string]string, error) {
	var (
		pidSocketMap = map[string]string{}
		// pidPortList  = []string{}
	)
	for _, pid := range pids {
		pidSocket, err := execCommandBash("ls -l /proc/" + pid + "/fd | grep socket")
		if err != nil {
			fmt.Errorf("couldn't get pidSocket: %w", err)
			// pidSocketMap[pid] = nil
			continue
		}

		reg := regexp.MustCompile(`socket:\[(\d+)\]`)
		subs := reg.FindAllStringSubmatch(pidSocket, -1)
		socketStr := ""
		for _, sub := range subs {
			if len(sub) > 1 {
				socketStr = socketStr + sub[1] + "|"
			}

		}
		if len(socketStr) > 1 {
			pidPortCmd := `cat /proc/net/{tcp,tcp6,udp,udp6} | grep -E '` + socketStr[0:len(socketStr)-1] + `'`
			pidProtMsg, _ := execCommandBash(pidPortCmd)

			for _, st := range strings.Split(pidProtMsg, "\n") {
				ipPort := strings.Fields(st)
				if len(ipPort) > 1 && len(ipPort[1]) > 5 {
					decimal, err := strconv.ParseUint(ipPort[1][len(ipPort[1])-4:], 16, 0)
					if err != nil {
						fmt.Println("端口16进制转10进制出错:", err)
						continue
					}
					pidSocketMap[strconv.FormatUint(decimal, 10)] = pid
					// pidPortList = append(pidPortList, string(decimal))
				}
			}

		}

	}

	return pidSocketMap, nil
}

func ipMacRelationship() (map[string]int, map[string]string) {
	macNumCmd := `ip a`
	macNumMsg, _ := execCommandBash(macNumCmd)

	macNumMap := map[string]int{}
	nicBegin := false
	macStr := ""
	originalMacNameMap := map[string]string{}
	nicName := ""

	for _, st := range strings.Split(macNumMsg, "\n") {
		if strings.Contains(st, " state UP ") || nicBegin {
			nicBegin = true
			if strings.Contains(st, " ens") || strings.Contains(st, " eno") || strings.Contains(st, " enp") {
				nicLine := strings.Fields(st)
				nicName = nicLine[1]
			}
			if strings.Contains(st, "link/ether") {
				nicLine := strings.Fields(st)
				macStr = nicLine[1]
				originalMacNameMap[nicName] = macStr
			} else if strings.Contains(st, "inet ") {
				nicBegin = false
				macNumMap[macStr] += 1
			} else if strings.Contains(st, "inet6") {
				nicBegin = false
				macNumMap[macStr] += 1
			}
		}
	}

	for _, v := range originalMacNameMap {
		macNumMap[v] -= 1
	}

	for k, _ := range macNumMap {
		macNumMap[k] += 1
	}

	ipMacCmd := `ip a  | grep "inet " -B 1`
	ipMacMsg, _ := execCommandBash(ipMacCmd)

	ipMacMap := map[string]string{}

	currMac := "::"
	for _, st := range strings.Split(ipMacMsg, "\n") {
		if strings.Contains(st, "--") {
			continue
		} else if strings.Contains(st, "link") {
			nicLine := strings.Fields(st)
			currMac = nicLine[1]
		} else if strings.Contains(st, "inet") {
			nicLine := strings.Fields(st)
			ipAddr := strings.Split(nicLine[1], "/")
			ipMacMap[ipAddr[0]] = currMac
		}
	}

	return macNumMap, ipMacMap
}

func getPhysicalNicName() []string {
	nicCmd := `ip a`
	nicMsg, _ := execCommandBash(nicCmd)

	nicName := []string{}

	for _, st := range strings.Split(nicMsg, "\n") {
		if strings.Contains(st, " state UP ") {
			if strings.Contains(st, " ens") || strings.Contains(st, " eno") || strings.Contains(st, " enp") {
				nicLine := strings.Fields(st)
				nicName = append(nicName, nicLine[1])
			}
		}
	}
	return nicName
}

func sumPidFlow(pids []string) (map[string]int, map[string]int, map[string]int, map[string]int, float64) {

	var (
		// top5pid = []string{"1", "2", "3", "4", "5"}
		recFlow = map[string]int{}
		recPkg  = map[string]int{}
		tmtFlow = map[string]int{}
		tmtPkg  = map[string]int{}
		flowSec = float64(0.0)
	)

	for _, pid := range pids {
		recFlow[pid] = 0.0
		recPkg[pid] = 0.0
		tmtFlow[pid] = 0.0
		tmtPkg[pid] = 0.0
	}
	pidSocketMap, _ := top5localPort(pids)
	fmt.Println(pidSocketMap)

	if len(pidSocketMap) == 0 {
		// 进程没有对应的端口，直接返回
		return recFlow, recPkg, tmtFlow, tmtPkg, 1.0
	}

	// ipRegex := `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`
	// re, _ := regexp.Compile(ipRegex)
	// // 查找所有网卡设备
	// devices, err := pcap.FindAllDevs()
	// if err != nil {
	// 	fmt.Errorf("couldn't find nic : %w", err)
	// }
	// networkInterfaceNames := []string{}
	// for _, d := range devices {
	// 	// fmt.Println("Name:", d.Name)
	// 	// fmt.Println("Description:", d.Description)
	// 	for _, address := range d.Addresses {
	// 		if re.MatchString(address.IP.String()) && address.IP.String() != "127.0.0.1" {
	// 			networkInterfaceNames = append(networkInterfaceNames, d.Name)
	// 		}
	// 	}
	// }
	// fmt.Println(networkInterfaceNames)
	networkInterfaceNames := getPhysicalNicName()
	for _, deviceName := range networkInterfaceNames {

		// fmt.Println(ipMacMap)
		// macNumMap, ipMacMap := ipMacRelationship()
		// fmt.Println(ipMacMap)
		// fmt.Println(macNumMap)
		handle, err := pcap.OpenLive(deviceName, 65536, true, 5)
		if err != nil {
			fmt.Errorf("couldn't get pcap: %w", err)
			return recFlow, recPkg, tmtFlow, tmtPkg, 1.0
		}
		defer handle.Close()

		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

		endTime := time.Date(2000, time.April, 10, 12, 0, 0, 0, time.UTC)
		beginTime := endTime
		// payloadLen := 0
		for packet := range packetSource.Packets() {
			if beginTime.Equal(endTime) {
				beginTime = packet.Metadata().Timestamp
			} else {
				endTime = packet.Metadata().Timestamp
			}
			// 判断数据包是否为TCP数据包，可解析源端口、目的端口、seq序列号、tcp标志位等
			tcpLayer := packet.Layer(layers.LayerTypeTCP)
			if tcpLayer != nil {
				tcp, _ := tcpLayer.(*layers.TCP)
				// SrcPort, DstPort, Seq, Ack, DataOffset, Window, Checksum, Urgent
				// Bool flags: FIN, SYN, RST, PSH, ACK, URG, ECE, CWR, NS
				srcPort := strconv.FormatUint(uint64(tcp.SrcPort), 10)
				dstPort := strconv.FormatUint(uint64(tcp.DstPort), 10)

				if pidSocketMap[srcPort] != "" {
					// fmt.Printf("源端口为 %d 目的端口为 %d\n", tcp.SrcPort, tcp.DstPort)
					ipLayer := packet.Layer(layers.LayerTypeIPv4) //这里抓取ipv4的数据包
					// ip, _ := ipLayer.(*layers.IPv4)
					// fmt.Printf("源ip为 %d 目的ip为 %d\n", ip.SrcIP, ip.DstIP)
					// if macNumMap[ipMacMap[ip.SrcIP.String()]] != 0 {
					// 	nicNum = macNumMap[ipMacMap[ip.SrcIP.String()]]
					// }
					if ipLayer != nil {
						tmtFlow[pidSocketMap[srcPort]] += packet.Metadata().Length
						tmtPkg[pidSocketMap[srcPort]] += 1
					}
				} else if pidSocketMap[dstPort] != "" {

					ipLayer := packet.Layer(layers.LayerTypeIPv4) //这里抓取ipv4的数据包
					// ip, _ := ipLayer.(*layers.IPv4)
					// fmt.Printf("源ip为 %d 目的ip为 %d\n", ip.SrcIP, ip.DstIP)
					// if macNumMap[ipMacMap[ip.DstIP.String()]] != 0 {
					// 	nicNum = macNumMap[ipMacMap[ip.DstIP.String()]]
					// }

					if ipLayer != nil {
						recFlow[pidSocketMap[dstPort]] += packet.Metadata().Length
						recPkg[pidSocketMap[dstPort]] += 1
					}
				}

			}

			udpLayer := packet.Layer(layers.LayerTypeUDP)
			if udpLayer != nil {
				udp, _ := udpLayer.(*layers.UDP)

				udpSrcPort := strconv.FormatUint(uint64(udp.SrcPort), 10)
				udpDstPort := strconv.FormatUint(uint64(udp.DstPort), 10)

				if pidSocketMap[udpSrcPort] != "" {

					// fmt.Printf("源端口为 %d 目的端口为 %d\n", udp.SrcPort, udp.DstPort)
					ipLayer := packet.Layer(layers.LayerTypeIPv4) //这里抓取ipv4的数据包
					// ip, _ := ipLayer.(*layers.IPv4)
					// fmt.Printf("源ip为 %d 目的ip为 %d\n", ip.SrcIP, ip.DstIP)
					// if macNumMap[ipMacMap[ip.SrcIP.String()]] != 0 {
					// 	nicNum = macNumMap[ipMacMap[ip.SrcIP.String()]]
					// }
					if ipLayer != nil {
						tmtFlow[pidSocketMap[udpSrcPort]] += packet.Metadata().Length
						tmtPkg[pidSocketMap[udpSrcPort]] += 1
					}
				} else if pidSocketMap[udpDstPort] != "" {

					ipLayer := packet.Layer(layers.LayerTypeIPv4) //这里抓取ipv4的数据包
					// ip, _ := ipLayer.(*layers.IPv4)
					// fmt.Printf("源ip为 %d 目的ip为 %d\n", ip.SrcIP, ip.DstIP)
					// if macNumMap[ipMacMap[ip.DstIP.String()]] != 0 {
					// 	nicNum = macNumMap[ipMacMap[ip.DstIP.String()]]
					// }
					if ipLayer != nil {
						recFlow[pidSocketMap[udpDstPort]] += packet.Metadata().Length
						recPkg[pidSocketMap[udpDstPort]] += 1
					}
				}
			}

			flowSec = endTime.Sub(beginTime).Seconds()
			if flowSec >= 1 {
				fmt.Println(flowSec)
				break
			}
		}
	}

	return recFlow, recPkg, tmtFlow, tmtPkg, 1.0
}

func sumFlowTest() {

	top5pid := []string{"7304"}
	recFlow, recPkg, tmtFlow, tmtPkg, _ := sumPidFlow(top5pid)
	fmt.Println("== 接受流量 ==")
	for k, v := range recFlow {
		fmt.Println(k)
		fmt.Println(v)
	}
	fmt.Println("== 接收包  ==")
	for k, v := range recPkg {
		fmt.Println(k)
		fmt.Println(v)
	}
	fmt.Println("====")
	for k, v := range tmtFlow {
		fmt.Println(k)
		fmt.Println(v)
	}
	fmt.Println("====")
	for k, v := range tmtPkg {
		fmt.Println(k)
		fmt.Println(v)
	}
	fmt.Println("====")

}
