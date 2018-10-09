package plugins_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slvwolf/ccentral/client"
	"github.com/slvwolf/ccentral/plugins"
)

type mockApi struct {
	serviceName string
	keyName     string
}

func newMockApi(serviceName string, keyName string) *mockApi {
	return &mockApi{serviceName: serviceName, keyName: keyName}
}

type mockUnix struct {
}

func (*mockUnix) Unix() int64 {
	return 100
}

func (*mockApi) GetServiceInfoList(serviceID string) (map[string]string, error) {
	return nil, nil
}

func (m *mockApi) GetServiceList() (client.ServiceList, error) {
	var s []string
	s = append(s, m.serviceName)
	result := client.ServiceList{Services: s}
	return result, nil
}

func (m *mockApi) GetInstanceList(serviceID string) (map[string]map[string]interface{}, error) {
	var counterArr []interface{}
	counterArr = append(counterArr, float64(1), float64(2))
	instances := make(map[string]map[string]interface{})
	instances["i1"] = make(map[string]interface{})
	instances["i1"][m.keyName] = counterArr
	return instances, nil
}

func (*mockApi) GetSchema(serviceID string) (map[string]client.SchemaItem, error) {
	return nil, nil
}

func (*mockApi) GetConfig(serviceID string) (map[string]client.ConfigItem, error) {
	return nil, nil
}

func TestResultFormatting(t *testing.T) {
	api := newMockApi("service1", "c_one")
	unix := &(mockUnix{})
	data, error := plugins.GeneratePrometheusPayload(api, unix)
	assert.Nil(t, error)
	assert.Equal(t, "# TYPE cc_service1_instances gauge\ncc_service1_instances 1 100\n# TYPE cc_service1_c_one gauge\ncc_service1_c_one 2 100\n", string(data))
}

func TestResultFormattingCleansServiceName(t *testing.T) {
	api := newMockApi("service-1%#", "c_one")
	unix := &(mockUnix{})
	data, error := plugins.GeneratePrometheusPayload(api, unix)
	assert.Nil(t, error)
	assert.Equal(t, "# TYPE cc_service1_instances gauge\ncc_service1_instances 1 100\n# TYPE cc_service1_c_one gauge\ncc_service1_c_one 2 100\n", string(data))
}

func TestResultFormattingCleansKeys(t *testing.T) {
	api := newMockApi("service1", "c_--one#")
	unix := &(mockUnix{})
	data, error := plugins.GeneratePrometheusPayload(api, unix)
	assert.Nil(t, error)
	assert.Equal(t, "# TYPE cc_service1_instances gauge\ncc_service1_instances 1 100\n# TYPE cc_service1_c_one gauge\ncc_service1_c_one 2 100\n", string(data))
}
