package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slvwolf/ccentral/client"
)

type mockApi struct {
	serviceName string
	keyName     string
	testData    interface{}
}

func newMockApi(serviceName string, keyName string, testData interface{}) *mockApi {
	return &mockApi{serviceName: serviceName, keyName: keyName, testData: testData}
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
	instances := make(map[string]map[string]interface{})
	instances["i1"] = make(map[string]interface{})
	instances["i1"][m.keyName] = m.testData
	return instances, nil
}

func (*mockApi) GetSchema(serviceID string) (map[string]client.SchemaItem, error) {
	return nil, nil
}

func (*mockApi) GetConfig(serviceID string) (map[string]client.ConfigItem, error) {
	return nil, nil
}

func createCounterArray() interface{} {
	var array []interface{}
	array = append(array, float64(1), float64(2))
	return array
}

func createHistogram() interface{} {
	var array []interface{}
	array = append(array, float64(75), float64(95), float64(99), float64(50))
	return array
}

func TestResultFormatting(t *testing.T) {
	api := newMockApi("service1", "c_one", createCounterArray())
	unix := &mockUnix{}
	data, err := GeneratePrometheusPayload(api, unix)
	assert.NoError(t, err)
	assert.Equal(t, "# TYPE cc_service1_instances gauge\ncc_service1_instances 1 100000\n# TYPE cc_service1_c_one gauge\ncc_service1_c_one 2 100000\n", string(data))
}

func TestResultFormattingCleansServiceName(t *testing.T) {
	api := newMockApi("service-1%#", "c_one", createCounterArray())
	unix := &mockUnix{}
	data, err := GeneratePrometheusPayload(api, unix)
	assert.NoError(t, err)
	assert.Equal(t, "# TYPE cc_service1_instances gauge\ncc_service1_instances 1 100000\n# TYPE cc_service1_c_one gauge\ncc_service1_c_one 2 100000\n", string(data))
}

func TestResultFormattingCleansKeys(t *testing.T) {
	api := newMockApi("service1", "c_--one#", createCounterArray())
	unix := &mockUnix{}
	data, err := GeneratePrometheusPayload(api, unix)
	assert.NoError(t, err)
	assert.Equal(t, "# TYPE cc_service1_instances gauge\ncc_service1_instances 1 100000\n# TYPE cc_service1_c_one gauge\ncc_service1_c_one 2 100000\n", string(data))
}

func TestHistogramFormatting(t *testing.T) {
	api := newMockApi("service1", "h_api_calls", createHistogram())
	unix := &mockUnix{}
	data, err := GeneratePrometheusPayload(api, unix)
	assert.NoError(t, err)
	assert.Equal(t, "# TYPE cc_service1_instances gauge\n"+
		"cc_service1_instances 1 100000\n"+
		"# TYPE cc_service1_h_api_calls gauge\n"+
		"cc_service1_h_api_calls{percentile=\"75\"} 75 100000\n"+
		"cc_service1_h_api_calls{percentile=\"95\"} 95 100000\n"+
		"cc_service1_h_api_calls{percentile=\"99\"} 99 100000\n"+
		"cc_service1_h_api_calls{percentile=\"median\"} 50 100000\n", string(data))
}
