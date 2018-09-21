# Configuration Central

Simple centralized configuration management and real-time monitoring for services over etcd.

## WebUI
 
### Building

- Install dependencies `make vendor_get`
- Build the app `make`
- Run it `./ccentral`

### Usage

	Usage of ./ccentral:
	  -etcd string
			etcd locations and port (Default: http://127.0.0.1:2379)
	  -port string
			Port to listen (Default: 3000)

Parameters also work from environvent variables (`ETCD`, `PORT`)

## Client

### Configuration Field Types

| Type     | Description                                          |
| -------- | ---------------------------------------------------- |
| string   | Plain string                                         |
| password | String field which wont be showed in UI              |
| integer  | Integer                                              |
| float    | Floaf                                                |
| list     | List, stored internally in JSON ["field1", "field2"] |
| boolean  | 1 or 0                                               |

### Etcd Keys

#### /ccentral/services/`SERVICE_ID`/schema

- `default` : Default value
- `type` : Field type, currently supported "string"
- `title` : Title (for WebUI)
- `description` : Description (for WebUI)

#### /ccentral/services/`SERVICE_ID`/config

- `value` : Configuration value
- `ts` : Last changed

#### /ccentral/services/`SERVICE_ID`/clients/`CLIENT_ID`

- `v` : Configuration version
- `cv` : Client library version 
- `ts` : Last update in epoch seconds
- `av` : API version
- `hostname` : Client hostname
- `lv` : Language version
- `started` : Epoch timestamp in seconds
- `uinterval` : Reporting interval
- `k_` : Prefix for custom keys
