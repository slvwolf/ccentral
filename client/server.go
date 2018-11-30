package client

// TODO: Support for API V.1
// TODO: Add support for "integer" type
// TODO: Add support for "float" type
// TODO: Add support for "list" type
// TODO: Add support for "boolean" type
// TODO: Add support for "password" type

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/pkg/errors"
)

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

// CCApi - Interface for CCentral methods
type CCApi interface {
	GetServiceInfoList(serviceID string) (map[string]string, error)
	GetServiceList() (ServiceList, error)
	InitCCentral(etcdHost string) error
	GetInstanceList(serviceID string) (map[string]map[string]interface{}, error)
	SetConfigItem(serviceID string, keyID string, value string) (string, error)
	GetSchema(serviceID string) (map[string]SchemaItem, error)
	SetSchema(serviceID string, schema map[string]SchemaItem) error
	GetConfig(serviceID string) (map[string]ConfigItem, error)
}

type CCInit interface {
	InitCCentral(etcdHost string) error
}

type CCClientApi interface {
	SetSchema(serviceID string, schema map[string]SchemaItem) error
	GetConfig(serviceID string) (map[string]ConfigItem, error)
}

type CCClientReadApi interface {
	GetConfig(serviceID string) (map[string]ConfigItem, error)
}

type CCClientWriteApi interface {
	SetSchema(serviceID string, schema map[string]SchemaItem) error
}

type CCServerReadApi interface {
	GetServiceInfoList(serviceID string) (map[string]string, error)
	GetServiceList() (ServiceList, error)
	GetInstanceList(serviceID string) (map[string]map[string]interface{}, error)
	GetSchema(serviceID string) (map[string]SchemaItem, error)
	GetConfig(serviceID string) (map[string]ConfigItem, error)
}

type CCServerWriteApi interface {
	SetConfigItem(serviceID string, keyID string, value string) (string, error)
	SetSchema(serviceID string, schema map[string]SchemaItem) error
}

// Service is a container for all service data
type Service struct {
	Schema    map[string]SchemaItem             `json:"schema"`
	Config    map[string]ConfigItem             `json:"config"`
	Instances map[string]map[string]interface{} `json:"clients"`
	Info      map[string]string                 `json:"info"`
}

// CCService - ...
type CCService struct {
	etcd client.KeysAPI
}

// NewSchemaItem - Represents single entry in schema which contains metadata for ConfigItem
func NewSchemaItem(defaultValue, itemType, title, description string) *SchemaItem {
	return &SchemaItem{Default: defaultValue, Type: itemType, Title: title, Description: description}
}

// NewConfigItem - Value container for configuration entry (based on SchemaItem)
func NewConfigItem(value string, changed int64) *ConfigItem {
	return &ConfigItem{Value: value, Changed: changed}
}

// InitCCentral will initialize everything required for CCentral usage
func (cc *CCService) InitCCentral(etcdHost string) error {
	var err error
	log.Printf("Connecting to %s", etcdHost)
	cfg := client.Config{Endpoints: []string{etcdHost}}
	e, err := client.New(cfg)
	if err != nil {
		return errors.Wrap(err, "Could not initialize CCentral")
	}
	cc.etcd = client.NewKeysAPI(e)
	return nil
}

// GetServiceList returns list of available services
func (cc *CCService) GetServiceList() (ServiceList, error) {
	resp, err := cc.etcd.Get(context.Background(), "/ccentral/services/", nil)
	if err != nil {
		return ServiceList{}, errors.Wrap(err, "Could not get service list")
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
func (cc *CCService) GetInstanceList(serviceID string) (map[string]map[string]interface{}, error) {
	instances := make(map[string]map[string]interface{})
	resp, err := cc.etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/clients", nil)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			log.Printf("No instances found for service %v", serviceID)
			return instances, nil
		}
		return nil, errors.Wrap(err, "Could not get instance list")
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
	version := config["v"]
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
func (cc *CCService) SetConfigItem(serviceID string, keyID string, value string) (string, error) {
	config, err := cc.GetConfig(serviceID)
	if err != nil {
		return "", errors.Wrap(err, "Could not retrieve service configuration")
	}
	config[keyID] = ConfigItem{
		Value:   value,
		Changed: time.Now().Unix(),
	}

	version := incrementVersion(config)

	output, err := json.Marshal(config)
	if err != nil {
		return "", errors.Wrap(err, "Could not convert to JSON")
	}

	_, err = cc.etcd.Set(context.Background(), "/ccentral/services/"+serviceID+"/config", string(output), nil)
	if err != nil {
		return "", errors.Wrap(err, "Could not update configuration")
	}
	return version, nil
}

// GetSchema returns configuration schema
func (cc *CCService) GetSchema(serviceID string) (map[string]SchemaItem, error) {
	resp, err := cc.etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/schema", nil)
	if err != nil {
		return nil, err
	}
	v := make(map[string]SchemaItem)
	err = json.Unmarshal([]byte(resp.Node.Value), &v)
	return v, err
}

// SetSchema writes the new schema to the etcd
func (cc *CCService) SetSchema(serviceID string, schema map[string]SchemaItem) error {
	data, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	_, err = cc.etcd.Set(context.Background(), "/ccentral/services/"+serviceID+"/schema", string(data), nil)
	return err
}

// GetServiceInfoList returns list of service shared service information reported by the clients
func (cc *CCService) GetServiceInfoList(serviceID string) (map[string]string, error) {
	info := make(map[string]string)
	resp, err := cc.etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/info", nil)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			log.Printf("No service info found for service %v", serviceID)
			return info, nil
		}
		return nil, errors.Wrap(err, "Could not get service info list")
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
func (cc *CCService) GetConfig(serviceID string) (map[string]ConfigItem, error) {
	v := make(map[string]ConfigItem)
	resp, err := cc.etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/config", nil)
	if err != nil {
		// Most likely new service that has only schema setup, just ignore the missing configuration
		if strings.Contains(err.Error(), "100: Key not found") {
			return v, nil
		}
		return nil, errors.Wrap(err, "Configuration could not be loaded")
	}
	err = json.Unmarshal([]byte(resp.Node.Value), &v)
	return v, err
}
