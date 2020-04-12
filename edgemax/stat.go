package edgemax

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// A Stat is a statistic provided by an EdgeMAX device.  Type assertions
// can be used to determine the specific type of a Stat, and to access
// a Stat's fields.
type Stat interface {
	StatType() StatType
}

// A StatType is a type of Stat.  StatType values can be used to retrieve
// certain types of Stats from Client.Stats
type StatType string

const (
	// StatTypeDPIStats retrieves EdgeMAX deep packet inspection statistics.
	StatTypeDPIStats StatType = "export"

	// StatTypeSystemStats retrieves EdgeMAX system statistics, including
	// system uptime, CPU utilization, and memory utilization.
	StatTypeSystemStats StatType = "system-stats"

	// StatTypeInterfaces retrieves EdgeMAX network interface statistics.
	StatTypeInterfaces StatType = "interfaces"
)

// SystemStats is a Stat which contains system statistics for an EdgeMAX
// device.
type SystemStats struct {
	CPU    int
	Uptime time.Duration
	Memory int
}

var _ Stat = &SystemStats{}

// StatType implements the Stats interface.
func (ss *SystemStats) StatType() StatType {
	return StatTypeSystemStats
}

// UnmarshalJSON unmarshals JSON into a SystemStats.
func (ss *SystemStats) UnmarshalJSON(b []byte) error {
	var v struct {
		CPU    string `json:"cpu"`
		Uptime string `json:"uptime"`
		Mem    string `json:"mem"`
	}

	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	cpu, err := strconv.Atoi(v.CPU)
	if err != nil {
		return err
	}

	uptime, err := strconv.Atoi(v.Uptime)
	if err != nil {
		return err
	}

	memory, err := strconv.Atoi(v.Mem)
	if err != nil {
		return err
	}

	*ss = SystemStats{
		CPU:    cpu,
		Uptime: time.Duration(uptime) * time.Second,
		Memory: memory,
	}

	return nil
}

// Interfaces is a slice of Interface values, which contains information about
// network interfaces for EdgeMAX devices.
type Interfaces []*Interface

var _ Stat = &Interfaces{}

// StatType implements the Stats interface.
func (i Interfaces) StatType() StatType {
	return StatTypeInterfaces
}

// An Interface is an EdgeMAX network interface.
type Interface struct {
	Name            string
	Up              bool
	Autonegotiation bool
	Duplex          string
	Speed           int
	MAC             net.HardwareAddr
	MTU             int
	Addresses       []net.IP
	Stats           InterfaceStats
}

// InterfaceStats contains network interface data transmission statistics.
type InterfaceStats struct {
	ReceivePackets  int
	TransmitPackets int
	ReceiveBytes    int
	TransmitBytes   int
	ReceiveErrors   int
	TransmitErrors  int
	ReceiveDropped  int
	TransmitDropped int
	Multicast       int
	ReceiveBPS      int
	TransmitBPS     int
}

// UnmarshalJSON unmarshals JSON into an Interfaces.
func (i *Interfaces) UnmarshalJSON(b []byte) error {
	var v map[string]struct {
		Up        string      `json:"up"`
		Autoneg   string      `json:"autoneg"`
		Duplex    string      `json:"duplex"`
		Speed     string      `json:"speed"`
		MAC       string      `json:"mac"`
		MTU       string      `json:"mtu"`
		Addresses interface{} `json:"addresses"`
		Stats     struct {
			RXPackets string `json:"rx_packets"`
			TXPackets string `json:"tx_packets"`
			RXBytes   string `json:"rx_bytes"`
			TXBytes   string `json:"tx_bytes"`
			RXErrors  string `json:"rx_errors"`
			TXErrors  string `json:"tx_errors"`
			RXDropped string `json:"rx_dropped"`
			TXDropped string `json:"tx_dropped"`
			Multicast string `json:"multicast"`
			RXBPS     string `json:"rx_bps"`
			TXBPS     string `json:"tx_bps"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	is := make(Interfaces, 0, len(v))
	for k, vv := range v {
		ss := []string{
			vv.Speed,
			vv.MTU,
			vv.Stats.RXPackets,
			vv.Stats.TXPackets,
			vv.Stats.RXBytes,
			vv.Stats.TXBytes,
			vv.Stats.RXErrors,
			vv.Stats.TXErrors,
			vv.Stats.RXDropped,
			vv.Stats.TXDropped,
			vv.Stats.Multicast,
			vv.Stats.RXBPS,
			vv.Stats.TXBPS,
		}

		ints := make([]int, 0, len(ss))
		for _, str := range ss {
			if str == "" {
				ints = append(ints, 0)
				continue
			}

			v, err := strconv.Atoi(str)
			if err != nil {
				return err
			}

			ints = append(ints, v)
		}

		var mac net.HardwareAddr
		if vv.MAC != "" {
			var err error
			mac, err = net.ParseMAC(vv.MAC)
			if err != nil {
				return err
			}
		}

		ips := make([]net.IP, 0)

		switch reflect.ValueOf(vv.Addresses).Kind() {
		case reflect.Slice:
			for _, ip := range vv.Addresses.([]interface{}) {
				ip, _, err := net.ParseCIDR(ip.(string))
				if err != nil {
					return err
				}
				ips = append(ips, ip)
			}
		case reflect.String:
			v := vv.Addresses.(string)
			if v != "" {
				ip, _, err := net.ParseCIDR(v)
				if err != nil {
					return err
				}
				ips = append(ips, ip)
			}
		}

		is = append(is, &Interface{
			Name:            k,
			Up:              vv.Up == "true",
			Autonegotiation: vv.Autoneg == "true",
			Duplex:          vv.Duplex,
			Speed:           ints[0],
			MAC:             mac,
			MTU:             ints[1],
			Addresses:       ips,
			Stats: InterfaceStats{
				ReceivePackets:  ints[2],
				TransmitPackets: ints[3],
				ReceiveBytes:    ints[4],
				TransmitBytes:   ints[5],
				ReceiveErrors:   ints[6],
				TransmitErrors:  ints[7],
				ReceiveDropped:  ints[8],
				TransmitDropped: ints[9],
				Multicast:       ints[10],
				ReceiveBPS:      ints[11],
				TransmitBPS:     ints[12],
			},
		})
	}

	sort.Sort(byInterfaceName(is))
	*i = is
	return nil
}

// byInterfaceName is used to sort Interfaces by network interface name.
type byInterfaceName []*Interface

func (b byInterfaceName) Len() int               { return len(b) }
func (b byInterfaceName) Less(i int, j int) bool { return b[i].Name < b[j].Name }
func (b byInterfaceName) Swap(i int, j int)      { b[i], b[j] = b[j], b[i] }

// DPIStats is a slice of DPIStat values, and contains Deep Packet Inspection
// stats from an EdgeMAX device.
type DPIStats []*DPIStat

// A DPIStat contains Deep Packet Inspection stats from an EdgeMAX device, for
// an individual client and traffic type.
type DPIStat struct {
	IP            net.IP
	Type          string
	Category      string
	ReceiveBytes  int
	ReceiveRate   int
	TransmitBytes int
	TransmitRate  int
}

// StatType implements the Stats interface.
func (d DPIStats) StatType() StatType {
	return StatTypeDPIStats
}

// UnmarshalJSON unmarshals JSON into a DPIStats.
func (d *DPIStats) UnmarshalJSON(b []byte) error {
	var v map[string]map[string]struct {
		RXBytes string `json:"rx_bytes"`
		RXRate  string `json:"rx_rate"`
		TXBytes string `json:"tx_bytes"`
		TXRate  string `json:"tx_rate"`
	}

	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	var out DPIStats
	for statIP := range v {
		ip := net.ParseIP(statIP)
		if ip == nil {
			continue
		}

		for statType, stats := range v[statIP] {
			nameCat := strings.SplitN(statType, "|", 2)
			if len(nameCat) != 2 {
				return fmt.Errorf("invalid stat type: %q", statType)
			}

			rxBytes, err := strconv.Atoi(stats.RXBytes)
			if err != nil {
				return err
			}

			rxRate, err := strconv.Atoi(stats.RXRate)
			if err != nil {
				return err
			}

			txBytes, err := strconv.Atoi(stats.TXBytes)
			if err != nil {
				return err
			}

			txRate, err := strconv.Atoi(stats.TXRate)
			if err != nil {
				return err
			}

			out = append(out, &DPIStat{
				IP:            ip,
				Type:          nameCat[0],
				Category:      nameCat[1],
				ReceiveBytes:  rxBytes,
				ReceiveRate:   rxRate,
				TransmitBytes: txBytes,
				TransmitRate:  txRate,
			})
		}
	}

	sort.Sort(byIPAndType(out))
	*d = out
	return nil
}

// byIPAndType is used to sort Interfaces by network interface name.
type byIPAndType []*DPIStat

func (b byIPAndType) Len() int { return len(b) }
func (b byIPAndType) Less(i int, j int) bool {
	less := ipLess(b[i].IP, b[j].IP)
	if less {
		return true
	}

	if b[i].Type < b[j].Type {
		return true
	}

	return false
}
func (b byIPAndType) Swap(i int, j int) { b[i], b[j] = b[j], b[i] }

func ipLess(a net.IP, b net.IP) bool {
	// Need 4-byte IPv4 representation where possible
	if ip4 := a.To4(); ip4 != nil {
		a = ip4
	}
	if ip4 := b.To4(); ip4 != nil {
		b = ip4
	}

	switch {
	// IPv4 addresses should appear before IPv6 addresses
	case len(a) == net.IPv4len && len(b) == net.IPv6len:
		return true
	case len(a) == net.IPv6len && len(b) == net.IPv4len:
		return false
	case a == nil && b == nil:
		return false
	}

	for i := range a {
		if a[i] < b[i] {
			return true
		}
	}

	return false
}
