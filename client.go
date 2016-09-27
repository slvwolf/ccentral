package main

import (
	"errors"
	"strconv"
	"time"
)

// CCentralService is base struct for CCentral services
type CCentralService struct {
	CheckIntervalSeconds int64
	lastCheck            int64
	servideID            string
	config               map[string]ConfigItem
	schema               map[string]SchemaItem
}

// InitCCentralService returns service struct for easier configuration access
func InitCCentralService(serviceID string) *CCentralService {
	service := CCentralService{servideID: serviceID, schema: make(map[string]SchemaItem), config: make(map[string]ConfigItem)}
	return &service
}

// AddSchema adds a single schema item into configuration
func (s *CCentralService) AddSchema(configID string, defaultValue string, valueType string, title string, description string) {
	i := SchemaItem{Default: defaultValue, Type: valueType, Title: title, Description: description}
	s.schema[configID] = i
}

// UpdateConfig updates configuration CheckIntervalSeconds has passed since last check
func (s *CCentralService) UpdateConfig() error {
	if time.Now().Unix()-s.CheckIntervalSeconds > s.lastCheck {
		return s.ForceUpdateConfig()
	}
	return nil
}

// ForceUpdateConfig will force configuration update
func (s *CCentralService) ForceUpdateConfig() error {
	if s.lastCheck == 0 {
		err := SetSchema(s.servideID, s.schema)
		if err != nil {
			s.lastCheck = time.Now().Unix()
			return err
		}
	}
	config, err := GetConfig(s.servideID)
	s.lastCheck = time.Now().Unix()
	if err != nil {
		return err
	}
	s.config = config
	return nil
}

// GetConfig returns single configuration option
func (s *CCentralService) GetConfig(configID string) (string, error) {
	s.UpdateConfig()
	defaultItem, ok := s.schema[configID]
	if !ok {
		return "", errors.New("Schema has not been defined for option " + configID)
	}
	valueItem, ok := s.config[configID]
	if ok {
		if len(valueItem.Value) > 0 {
			return valueItem.Value, nil
		}
	}
	return defaultItem.Default, nil
}

// GetConfigBool returns boolean value of the configuration options
func (s *CCentralService) GetConfigBool(configID string) (bool, error) {
	value, err := s.GetConfig(configID)
	if err != nil {
		return false, err
	}
	bValue, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}
	return bValue, nil
}

// GetConfigInt returns integer value of the configuration options
func (s *CCentralService) GetConfigInt(configID string) (int, error) {
	value, err := s.GetConfig(configID)
	if err != nil {
		return 0, err
	}
	iValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return iValue, nil
}
