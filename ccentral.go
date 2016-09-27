package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/client"
)

var etcd client.KeysAPI

// ServiceList contains list of serviceIDs
type ServiceList struct {
	Services []string `json:"services"`
}

// SchemaItem describes single configuration schema
type SchemaItem struct {
	Default     string `json:"default"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ConfigItem contains value and timestamp when the value was last changed
type ConfigItem struct {
	Value   string `json:"value"`
	Changed int64  `json:"changed"`
}

// InstanceItem contains required fields for each instance
type InstanceItem struct {
	Version   string  `json:"v"`
	Timestamp float64 `json:"ts"`
}

// InitCCentral will initialize everything required for CCentral usage
func InitCCentral(etcdHost string) error {
	var err error
	log.Printf("Connecting to %s", etcdHost)
	cfg := client.Config{Endpoints: []string{etcdHost}}
	e, err := client.New(cfg)
	if err != nil {
		log.Printf("Could not initialize CCentral: %v", err)
		return err
	}
	etcd = client.NewKeysAPI(e)
	return nil
}

// GetServiceList returns list of available services
func GetServiceList() (ServiceList, error) {
	resp, err := etcd.Get(context.Background(), "/ccentral/services/", nil)
	if err != nil {
		log.Printf(err.Error())
		return ServiceList{}, err
	}
	response := ServiceList{Services: make([]string, 0, resp.Node.Nodes.Len())}
	for _, v := range resp.Node.Nodes {
		keys := strings.Split(v.Key, "/")
		last := keys[len(keys)-1:][0]
		response.Services = append(response.Services, last)
	}
	return response, nil
}

// GetInstanceList returns full information of each running service instance
func GetInstanceList(serviceID string) (map[string]map[string]interface{}, error) {
	instances := make(map[string]map[string]interface{})
	resp, err := etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/clients", nil)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			log.Printf("No instances found for service %v", serviceID)
			return instances, nil
		}
		log.Printf(err.Error())
		return nil, err
	}

	for _, v := range resp.Node.Nodes {
		i := make(map[string]interface{})
		keys := strings.Split(v.Key, "/")
		last := keys[len(keys)-1:][0]
		err = json.Unmarshal([]byte(v.Value), &i)

		if err != nil {
			log.Printf("Could not unmarshal following: %v", v.Value)
		}
		instances[last] = i
	}
	return instances, nil
}

func incrementVersion(config map[string]ConfigItem) string {
	version, ok := config["v"]
	if !ok {
		version = ConfigItem{}
	}
	value, err := strconv.Atoi(version.Value)
	if err != nil {
		value = 1
	}
	version.Value = strconv.Itoa(value + 1)
	version.Changed = time.Now().Unix()
	config["v"] = version
	return version.Value
}

// SetConfigItem allows changing the service configuration
func SetConfigItem(serviceID string, keyID string, value string) (string, error) {
	config, err := GetConfig(serviceID)
	if err != nil {
		return "", errors.New("Could not retrieve service configuration: " + err.Error())
	}
	i := ConfigItem{}
	i.Value = value
	i.Changed = time.Now().Unix()
	config[keyID] = i

	version := incrementVersion(config)

	output, err := json.Marshal(config)
	if err != nil {
		return "", errors.New("Could not convert to json: " + err.Error())
	}

	_, err = etcd.Set(context.Background(), "/ccentral/services/"+serviceID+"/config", string(output), nil)
	if err != nil {
		return "", errors.New("Could not update configuration: " + err.Error())
	}
	return version, nil
}

// GetSchema returns configuration schema
func GetSchema(serviceID string) (map[string]SchemaItem, error) {
	resp, err := etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/schema", nil)
	if err != nil {
		return nil, err
	}
	v := make(map[string]SchemaItem)
	json.Unmarshal([]byte(resp.Node.Value), &v)
	return v, nil
}

// SetSchema writes the new schema to the etcd
func SetSchema(serviceID string, schema map[string]SchemaItem) error {
	data, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	_, err = etcd.Set(context.Background(), "/ccentral/services/"+serviceID+"/schema", string(data), nil)
	if err != nil {
		return err
	}
	return nil
}

// GetServiceInfoList returns list of service shared service information reported by the clients
func GetServiceInfoList(serviceID string) (map[string]string, error) {
	info := make(map[string]string)
	resp, err := etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/info", nil)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			log.Printf("No service info found for service %v", serviceID)
			return info, nil
		}
		log.Printf(err.Error())
		return nil, err
	}
	for _, v := range resp.Node.Nodes {
		keys := strings.Split(v.Key, "/")
		last := keys[len(keys)-1:][0]
		if err != nil {
			log.Printf("Could not unmarshal following: %v", v.Value)
		}
		info[last] = v.Value
	}
	return info, nil
}

// GetConfig returns full listing of service configuration
func GetConfig(serviceID string) (map[string]ConfigItem, error) {
	resp, err := etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/config", nil)
	if err != nil {
		// Most likely new service that has only schema setup, just ignore the missing configuration
		if strings.Contains(err.Error(), "100: Key not found") {
			v := make(map[string]ConfigItem)
			return v, nil
		}
		log.Printf("Configuration could not be loaded, %v", err)
		return nil, err
	}
	v := make(map[string]ConfigItem)
	json.Unmarshal([]byte(resp.Node.Value), &v)
	return v, nil
}
