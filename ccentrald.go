package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// Service is a container for all service data
type service struct {
	Schema    map[string]SchemaItem             `json:"schema"`
	Config    map[string]ConfigItem             `json:"config"`
	Instances map[string]map[string]interface{} `json:"clients"`
	Info      map[string]string                 `json:"info"`
}

func newService(schema map[string]SchemaItem, config map[string]ConfigItem, instances map[string]map[string]interface{}, info map[string]string) *service {
	return &service{Schema: schema, Config: config, Instances: instances, Info: info}
}

func writeInternalError(w http.ResponseWriter, msg string, status int) {
	w.WriteHeader(status)
	fmt.Fprintf(w, "{\"error\": \""+msg+"\"}")
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-type", "application/json")
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]
	res := vars["res"]
	if path == "" || path == "/" {
		res = "index2.html"
		path = ""
	}
	if path == "js" {
		w.Header().Add("Content-Type", "application/javascript; charset=utf-8")
	}
	if path == "css" {
		w.Header().Add("Content-Type", "text/css")
	}
	w.WriteHeader(200)
	path = "web/" + path + "/" + res
	body, _ := ioutil.ReadFile(path)
	w.Write(body)
}

func handleServiceList(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)
	serviceList, err := GetServiceList()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{\"error\": \"Could not retrieve configuration\"}")
		return
	}
	v, err := json.Marshal(serviceList)
	if err != nil {
		log.Printf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{\"error\": \"Error marshalling json\"}")
		return
	}
	fmt.Fprintf(w, string(v))
}

func handleItem(w http.ResponseWriter, r *http.Request) {
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

	version, err := SetConfigItem(string(serviceID), string(keyID), string(value))

	if err != nil {
		writeInternalError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Configuration updated: [%v] %v=%v (version: %v)", string(serviceID), string(keyID), string(value), version)
}

func handleService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	setHeaders(w)
	serviceID := vars["serviceId"]
	schema, err := GetSchema(serviceID)
	if err != nil {
		writeInternalError(w, "Could not retrieve service schema", http.StatusInternalServerError)
		return
	}
	config, err := GetConfig(serviceID)
	if err != nil {
		writeInternalError(w, "Could not retrieve config", http.StatusInternalServerError)
		return
	}
	instances, err := GetInstanceList(serviceID)
	if err != nil {
		log.Printf("Problem getting instances: %v", err)
		writeInternalError(w, "Could not retrieve instances", http.StatusInternalServerError)
		return
	}
	info, err := GetServiceInfoList(serviceID)
	if err != nil {
		log.Printf("Problem getting service info: %v", err)
		writeInternalError(w, "Could not retrieve service info", http.StatusInternalServerError)
		return
	}
	output, err := json.Marshal(newService(schema, config, instances, info))
	if err != nil {
		writeInternalError(w, "Could not convert to json", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, string(output))
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func main() {
	etcdHost := flag.String("etcd", os.Getenv("ETCD"), "etcd locations and port (Default: http://127.0.0.1:2379)")
	port := flag.String("port", os.Getenv("PORT"), "Port to listen (Default: 3000)")
	flag.Parse()
	if *etcdHost == "" {
		*etcdHost = "http://127.0.0.1:2379"
	}
	if *port == "" {
		*port = "3000"
	}

	log.Printf(`
_________ _________                __                .__
\_   ___ \\_   ___ \  ____   _____/  |_____________  |  |
/    \  \//    \  \/_/ __ \ /    \   __\_  __ \__  \ |  |
\     \___\     \___\  ___/|   |  \  |  |  | \// __ \|  |__
 \______  /\______  /\___  >___|  /__|  |__|  (____  /____/
        \/        \/     \/     \/                 \/      `)

	router := mux.NewRouter().StrictSlash(true)
	InitCCentral(*etcdHost)
	service := InitCCentralService("ccentral")
	service.AddSchema("zabbix_enabled", "false", "string", "Zabbix Enabled", "Boolean for enabling or disabling Zabbix monitoring for all services")
	service.AddSchema("zabbix_host", "localhost", "string", "Zabbix Hostname", "Hostname for Zabbix")
	service.AddSchema("zabbix_port", "10051", "string", "Zabbix Hostname", "Port for Zabbix")
	service.AddSchema("zabbix_interval", "60", "string", "Zabbix Interval", "Update interval for Zabbix metrics")
	router.HandleFunc("/{res}", handleRoot)
	router.HandleFunc("/check", handleCheck)
	router.HandleFunc("/{path}/{res}", handleRoot)
	router.HandleFunc("/api/1/services", handleServiceList)
	router.HandleFunc("/api/1/services/{serviceId}", handleService)
	router.HandleFunc("/api/1/services/{serviceId}/keys/{keyId}", handleItem)
	startZabbixUpdater(service)
	log.Printf("Admin UI available at :" + *port)
	err := http.ListenAndServe(":"+*port, router)
	if err != nil {
		log.Fatal(err)
	}
}
