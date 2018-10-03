package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"time"

	"github.com/gorilla/mux"
	"github.com/slvwolf/ccentral/client"
)

func handleMockService(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	setHeaders(w)
	//serviceID := vars["serviceId"]
	schema := make(map[string]client.SchemaItem)
	schema["example-str-set"] = *client.NewSchemaItem("default", "string", "Configuration SET (String)", "Configuration with some configuration set")
	schema["example-str-unset"] = *client.NewSchemaItem("default", "string", "Configuration UNSET (String)", "Configuration with default values")
	schema["example-password-set"] = *client.NewSchemaItem("default", "password", "Configuration SET (Password)", "Configuration with some configuration set")
	schema["example-password-unset"] = *client.NewSchemaItem("default", "password", "Configuration UNSET (Password)", "Configuration with default values")
	config := make(map[string]client.ConfigItem)
	config["example-str-set"] = *client.NewConfigItem("Value is set", 0)
	config["example-old-conf"] = *client.NewConfigItem("This config should not be shown", 0)
	config["example-password-set"] = *client.NewConfigItem("Value is set", 0)
	instances := make(map[string]map[string]interface{})
	i := make(map[string]interface{})
	instances["1234"] = i
	i["started"] = fmt.Sprintf("%v", time.Now().Unix())
	info := make(map[string]string)
	hidePasswordFields(schema, config)
	output, err := json.Marshal(client.NewService(schema, config, instances, info))
	if err != nil {
		writeInternalError(w, "Could not convert to json", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, string(output))
}

func handleMockServiceList(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)
	response := client.ServiceList{Services: make([]string, 0, 1)}
	response.Services = append(response.Services, "example")
	v, err := json.Marshal(response)
	if err != nil {
		log.Printf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{\"error\": \"Error marshalling json\"}")
		return
	}
	fmt.Fprintf(w, string(v))
}

func handleMockItem(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)
	vars := mux.Vars(r)
	serviceID := vars["serviceId"]
	keyID := vars["keyId"]
	if r.Method != http.MethodPut {
		writeInternalError(w, "Allowed methods are: PUT", http.StatusBadRequest)
		return
	}

	value, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeInternalError(w, "Could not read body", http.StatusInternalServerError)
		return
	}

	log.Printf("Configuration updated: [%v] %v=%v", string(serviceID), string(keyID), string(value))
}
