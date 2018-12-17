package zabbix

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/slvwolf/ccentral/client"
	"github.com/slvwolf/ccentral/plugins"
)

type metric struct {
	Host  string `json:"host"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Clock int64  `json:"clock"`
}

func newMetric(host, key, value string, clock ...int64) *metric {
	m := &metric{Host: host, Key: key, Value: value}
	if m.Clock = time.Now().Unix(); len(clock) > 0 {
		m.Clock = int64(clock[0])
	}
	return m
}

func (m *metric) String() string {
	return fmt.Sprintf("%v/%v=%v", m.Host, m.Key, m.Value)
}

type packet struct {
	Request string    `json:"request"`
	Data    []*metric `json:"data"`
	Clock   int64     `json:"clock"`
}

func newPacket(data []*metric, clock ...int64) *packet {
	p := &packet{Request: `sender data`, Data: data}
	if p.Clock = time.Now().Unix(); len(clock) > 0 {
		p.Clock = int64(clock[0])
	}
	return p
}

func (p *packet) len() []byte {
	dataSize := make([]byte, 8)
	JSONData, _ := json.Marshal(p)
	binary.LittleEndian.PutUint32(dataSize, uint32(len(JSONData)))
	return dataSize
}

type sender struct {
	Host string
	Port int
}

func newSender(host string, port int) *sender {
	s := &sender{Host: host, Port: port}
	return s
}

func (s *sender) connect() (*net.TCPConn, error) {
	iaddr, err := s.getAddr()
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, iaddr)
	if err != nil {
		return conn, err
	}
	return conn, nil
}

func (s *sender) getAddr() (*net.TCPAddr, error) {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	iaddr, err := net.ResolveTCPAddr("tcp", addr)

	if err != nil {
		fmt.Printf("Connection failed: %s", err.Error())
		return iaddr, err
	}

	return iaddr, nil
}

func (s *sender) getHeader() []byte {
	return []byte("ZBXD\x01")
}

func (s *sender) read(conn *net.TCPConn) ([]byte, error) {
	res := make([]byte, 1024)
	res, err := ioutil.ReadAll(conn)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *sender) send(packet *packet) ([]byte, error) {
	conn, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	dataPacket, _ := json.Marshal(packet)
	buffer := append(s.getHeader(), packet.len()...)
	buffer = append(buffer, dataPacket...)

	_, err = conn.Write(buffer)
	if err != nil {
		fmt.Printf("Error while sending the data: %s", err.Error())
	}

	res, err := s.read(conn)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func sendZabbix(service *client.CCentralService, metrics []*metric) {
	packet := newPacket(metrics)
	hostname, _ := service.GetConfig("zabbix_host")
	port, _ := service.GetConfigInt("zabbix_port")
	z := newSender(hostname, port)
	_, err := z.send(packet)
	if err != nil {
		log.Printf("Failed to send data to Zabbix: %v", err.Error())
	}
}

func pollLoop(service *client.CCentralService, cc client.CCServerReadApi) {
	for {
		enabled, _ := service.GetConfigBool("zabbix_enabled")
		if enabled {
			var metrics []*metric
			serviceList, err := cc.GetServiceList()
			if err != nil {
				log.Printf("WARN Could not retrieve service list")
			}
			for _, serviceID := range serviceList.Services {
				log.Printf("Handling service %v", serviceID)
				instances, err := cc.GetInstanceList(serviceID)
				if err == nil {
					count := len(instances)
					counters := make(map[string]int)
					key := fmt.Sprintf("%s.%s", serviceID, "instances")
					metric := newMetric("ccentral", key, strconv.Itoa(count))
					metrics = append(metrics, metric)
					log.Printf("Zabbix: %v", metric)
					for _, instance := range instances {
						log.Printf("Collecting counters for %v", serviceID)
						counters = plugins.CollectInstanceCounters(instance, counters)
					}
					for key, value := range counters {
						zabbixKey := fmt.Sprintf("%s.%s", serviceID, key)
						metric := newMetric("ccentral", zabbixKey, strconv.Itoa(value))
						metrics = append(metrics, metric)
						log.Printf("Zabbix: %v", metric)
					}
				}
			}
			sendZabbix(service, metrics)
			log.Printf("Sent total of %v records to Zabbix", len(metrics))
		}
		interval, _ := service.GetConfigInt("zabbix_interval")
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

// StartZabbixUpdater - Start zabbix poll loop
func StartZabbixUpdater(service *client.CCentralService, cc client.CCServerReadApi) {
	go pollLoop(service, cc)
}
