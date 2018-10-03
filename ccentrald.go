package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/slvwolf/ccentral/client"
	"github.com/slvwolf/ccentral/plugins"
)

var cc client.CCApi
var ccService *client.CCentralService

func writeInternalError(w http.ResponseWriter, msg string, status int) {
	w.WriteHeader(status)
	w.Write([]byte("{\"error\": \"" + msg + "\"}"))
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
	serviceList, err := cc.GetServiceList()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{\"error\": \"Could not retrieve configuration\"}"))
		return
	}
	v, err := json.Marshal(serviceList)
	if err != nil {
		log.Printf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{\"error\": \"Error marshalling json\"}"))
		return
	}
	w.Write(v)
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

	version, err := cc.SetConfigItem(string(serviceID), string(keyID), string(value))

	if err != nil {
		writeInternalError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Configuration updated: [%v] %v=%v (version: %v)", string(serviceID), string(keyID), string(value), version)
}

func hidePasswordFields(schema map[string]client.SchemaItem, config map[string]client.ConfigItem) {
	for k, v := range schema {
		if v.Type == "password" {
			iv, ok := config[k]
			if ok == true {
				iv.Value = "******"
			}
			config[k] = iv
		}
	}
}

func handleService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	setHeaders(w)
	serviceID := vars["serviceId"]
	schema, err := cc.GetSchema(serviceID)
	if err != nil {
		writeInternalError(w, "Could not retrieve service schema", http.StatusInternalServerError)
		return
	}
	config, err := cc.GetConfig(serviceID)
	if err != nil {
		writeInternalError(w, "Could not retrieve config", http.StatusInternalServerError)
		return
	}
	instances, err := cc.GetInstanceList(serviceID)
	if err != nil {
		log.Printf("Problem getting instances: %v", err)
		writeInternalError(w, "Could not retrieve instances", http.StatusInternalServerError)
		return
	}
	info, err := cc.GetServiceInfoList(serviceID)
	if err != nil {
		log.Printf("Problem getting service info: %v", err)
		writeInternalError(w, "Could not retrieve service info", http.StatusInternalServerError)
		return
	}
	hidePasswordFields(schema, config)
	output, err := json.Marshal(client.NewService(schema, config, instances, info))
	if err != nil {
		writeInternalError(w, "Could not convert to json", http.StatusInternalServerError)
		return
	}
	w.Write(output)
}

func handlePrometheus(w http.ResponseWriter, r *http.Request) {
	enabled, _ := ccService.GetConfigBool("prometheus_enabled")
	if enabled {
		data, err := plugins.GeneratePrometheusPayload(cc, time.Now())
		if err != nil {
			writeInternalError(w, "Failed to generate payload", 500)
			return
		}
		w.Write(data)
	} else {
		w.Write([]byte("# Prometheus metrics are disabled"))
	}
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func main() {
	etcdHost := flag.String("etcd", os.Getenv("ETCD"), "etcd locations and port (Default: http://127.0.0.1:2379)")
	port := flag.String("port", os.Getenv("PORT"), "Port to listen (Default: 3000)")
	presentation := flag.Bool("presentation", false, "Run in presentation mode")

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
	router.HandleFunc("/", handleRoot)
	router.HandleFunc("/{res}", handleRoot)
	router.HandleFunc("/check", handleCheck)
	router.HandleFunc("/{path}/{res}", handleRoot)
	cc = &client.CCService{}
	if !*presentation {
		err := cc.InitCCentral(*etcdHost)
		if err != nil {
			panic("Could not initialize CCentral")
		}
		ccService = client.InitCCentralService(cc, "ccentral")
		ccService.AddSchema("zabbix_enabled", "0", "boolean", "Zabbix Enabled", "Boolean for enabling or disabling Zabbix monitoring for all services")
		ccService.AddSchema("zabbix_host", "localhost", "string", "Zabbix Hostname", "Hostname for Zabbix")
		ccService.AddSchema("zabbix_port", "10051", "integer", "Zabbix Port", "Port for Zabbix")
		ccService.AddSchema("zabbix_interval", "60", "integer", "Zabbix Interval", "Update interval for Zabbix metrics")
		ccService.AddSchema("prometheus_enabled", "0", "boolean", "Prometheus Enabled", "Boolean for enabling or disabling prometheus endpoint (/prometheus)")
		router.HandleFunc("/api/1/services", handleServiceList)
		router.HandleFunc("/api/1/services/{serviceId}", handleService)
		router.HandleFunc("/api/1/services/{serviceId}/keys/{keyId}", handleItem)
		router.HandleFunc("/prometheus", handlePrometheus)
		plugins.StartZabbixUpdater(ccService, cc)
	} else {
		// TODO: User mocked CCApi instead
		log.Printf("Running in PRESENTATION mode")
		router.HandleFunc("/api/1/services", handleMockServiceList)
		router.HandleFunc("/api/1/services/{serviceId}", handleMockService)
		router.HandleFunc("/api/1/services/{serviceId}/keys/{keyId}", handleMockItem)
	}
	log.Printf("Admin UI available at :" + *port)
	err := http.ListenAndServe(":"+*port, router)
	if err != nil {
		log.Fatal(err)
	}
}
