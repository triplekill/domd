package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/hakpaang/debinterface"
	ping "github.com/prometheus-community/pro-bing"
)

func showMenu() {
	fmt.Println("\n=== 配置菜单 ===")
	fmt.Println("1. 查看网络接口信息")
	fmt.Println("2. 修改网卡IP地址")
	fmt.Println("3. ping测试")
	fmt.Println("4. 云服务联通性测试")
	fmt.Println("0. 退出")

	fmt.Print("\n请选择操作 [0-4]: ")
}

// func executeCommand(cmd string, args ...string) error {
// 	command := exec.Command(cmd, args...)
// 	command.Stdout = os.Stdout
// 	command.Stderr = os.Stderr
// 	return command.Run()
// }

func setIPAddress() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入网卡名称 (例如 enp1s0): ")
	iface, _ := reader.ReadString('\n')
	iface = strings.TrimSpace(iface)
	if !strings.HasPrefix(iface, "enp1s0") && !strings.HasPrefix(iface, "enp2s0") && !strings.HasPrefix(iface, "enp3s0") && !strings.HasPrefix(iface, "enp4s0") && !strings.HasPrefix(iface, "enp5s0") && !strings.HasPrefix(iface, "enp6s0") {
		fmt.Println(color.RedString("错误: 网卡名称错误"))
		return
	}

	fmt.Print("请输入新的IP地址 (例如 192.168.1.100): ")
	ipaddr, _ := reader.ReadString('\n')
	ipaddr = strings.TrimSpace(ipaddr)
	// 验证IP地址格式是否有效
	ip := net.ParseIP(ipaddr)
	if ip == nil {
		fmt.Printf("\n")
		fmt.Println(color.RedString("错误: 无效的IP地址格式"))
		return
	}
	// 检查是否为私有IP地址
	if !ip.IsPrivate() {
		fmt.Printf("\n")
		fmt.Println(color.YellowString("警告: 输入的不是私有IP地址"))
		return
	}

	fmt.Print("请输入新的子网掩码 (例如 255.255.255.0): ")
	netmask, _ := reader.ReadString('\n')
	netmask = strings.TrimSpace(netmask)
	netmaskIP := net.ParseIP(netmask)
	if netmaskIP == nil {
		fmt.Printf("\n")
		fmt.Println(color.RedString("错误: 无效的子网掩码格式"))
		return
	}

	fmt.Print("请输入新的网关 (例如 192.168.1.1): ")
	gateway, _ := reader.ReadString('\n')
	gateway = strings.TrimSpace(gateway)
	gatewayIP := net.ParseIP(gateway)
	if gatewayIP == nil {
		fmt.Printf("\n")
		fmt.Println(color.RedString("错误: 无效的网关格式"))
		return
	}

	setInterface(iface, ipaddr, netmask, gateway)
	fmt.Printf("\n")
	fmt.Printf(color.GreenString("成功设置 %s 的IP地址为 %s 子网掩码为 %s 网关为 %s\n"), iface, ipaddr, netmask, gateway)
}

func setInterface(iface, ipaddr, netmask, gateway string) {

	var out debinterface.Interfaces
	out.InterfacesPath = "/etc/network/interfaces"
	out.Sources = []string{"/etc/network/interfaces.d/*"}

	deviceInterfaces := debinterface.UnmarshalWith("/etc/network/interfaces")

	for _, adapter := range deviceInterfaces.Adapters {
		if adapter.Name == iface {
			adapter.Address = net.ParseIP(ipaddr)
			adapter.Netmask = net.ParseIP(netmask)
			adapter.Gateway = net.ParseIP(gateway)
		}
		out.Adapters = append(out.Adapters, adapter)
	}

	debinterface.Marshal(&out)
	// 重启网络服务
	cmd := exec.Command("systemctl", "restart", "networking")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("\n")
		fmt.Printf(color.RedString("重启网络服务失败: %v\n"), err)
		return
	}
	fmt.Printf("\n")
	fmt.Println(color.GreenString("网络服务重启成功"))
}

func showInterfaceInfo() {
	deviceInterfaces := debinterface.UnmarshalWith("/etc/network/interfaces")

	for _, adapter := range deviceInterfaces.Adapters {
		fmt.Printf("\n")
		fmt.Printf(color.CyanString("网卡名称: %s  IP地址: %s 子网掩码: %s 网关: %s\n"), adapter.Name, adapter.Address, adapter.Netmask, adapter.Gateway)
	}
}

func checkServer() {
	resp, err := http.Get("https://api.threathunting.com.cn/")
	if err != nil {
		fmt.Printf("\n")
		fmt.Println(color.RedString("错误: 无法连接到服务器:", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("\n")
		fmt.Println(color.GreenString("服务器状态正常 (HTTP 200)\n"))
	} else {
		fmt.Printf("\n")
		fmt.Println(color.RedString("服务器返回异常状态码: %d\n", resp.StatusCode))
	}
}

func pingTest() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入要ping的IP地址 (例如 192.168.1.100): ")
	ipaddr, _ := reader.ReadString('\n')
	ipaddr = strings.TrimSpace(ipaddr)
	gatewayIP := net.ParseIP(ipaddr)
	if gatewayIP == nil {
		fmt.Printf("\n")
		fmt.Println(color.RedString("错误: 无效的IP地址格式"))
		return
	}
	pinger, err := ping.NewPinger(gatewayIP.String())
	if err != nil {
		fmt.Printf("\n")
		fmt.Println(color.RedString("Ping失败: %v\n", err))
		return
	}
	pinger.Count = 4
	pinger.Timeout = time.Second * 5 // Set 5 second timeout
	pinger.SetPrivileged(true)       // Add privileged mode to avoid permission denied
	err = pinger.Run()               // Blocks until finished.
	if err != nil {
		fmt.Printf("\n")
		fmt.Println(color.RedString("Ping失败: %v\n", err))
		return
	}
	stats := pinger.Statistics()

	if stats.PacketsRecv == 0 {
		fmt.Printf("\n")
		fmt.Println(color.RedString("Ping失败: 目标主机不可达\n"))
	} else {
		fmt.Printf("\n")
		fmt.Println(color.GreenString("Ping成功: 目标主机可达\n"))
	}
}
func main() {
	var choice string

	for {
		showMenu()
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			fmt.Println("\n网络接口信息：")
			showInterfaceInfo()
		case "2":
			fmt.Println("\n修改网卡IP地址：")
			setIPAddress()
		case "3":
			fmt.Println("\n当前路由表：")
			pingTest()
		case "4":
			fmt.Println("\n检查云服务器是否正常：")
			checkServer()
		case "0":
			fmt.Println("退出控制台")
			return
		default:
			fmt.Println("无效的选择，请重试")
		}
	}
}
