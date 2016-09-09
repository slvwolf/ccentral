package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

type serviceList struct {
	Services []string `json:"services"`
}

type schemaItem struct {
	Default     string `json:"default"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type item struct {
	Value   string `json:"value"`
	Changed int64  `json:"changed"`
}

type service struct {
	Schema    map[string]schemaItem   `json:"schema"`
	Config    map[string]item         `json:"config"`
	Instances map[string]instanceItem `json:"clients"`
	Info      map[string]string       `json:"info"`
}

type instanceItem struct {
	Version   string  `json:"v"`
	Timestamp float64 `json:"ts"`
}

var etcd client.KeysAPI

func newService(schema map[string]schemaItem, config map[string]item, instances map[string]instanceItem, info map[string]string) *service {
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
	if path == "" || path == "/" {
		path = "index.html"
	}
	body, _ := ioutil.ReadFile(path)
	fmt.Fprintf(w, string(body))
}

func handleServiceList(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)
	resp, err := etcd.Get(context.Background(), "/ccentral/services/", nil)
	if err != nil {
		log.Printf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{\"error\": \"Could not retrieve configuration\"}")
		return
	}
	response := serviceList{Services: make([]string, 0, resp.Node.Nodes.Len())}
	for _, v := range resp.Node.Nodes {
		keys := strings.Split(v.Key, "/")
		last := keys[len(keys)-1:][0]
		response.Services = append(response.Services, last)
	}
	v, err := json.Marshal(response)
	if err != nil {
		log.Printf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{\"error\": \"Error marshalling json\"}")
		return
	}
	fmt.Fprintf(w, string(v))
}

func getInstanceList(serviceId string) (map[string]instanceItem, error) {
	instances := make(map[string]instanceItem)
	resp, err := etcd.Get(context.Background(), "/ccentral/services/"+serviceId+"/clients", nil)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			log.Printf("No instances found for service %v", serviceId)
			return instances, nil
		}
		log.Printf(err.Error())
		return nil, err
	}

	for _, v := range resp.Node.Nodes {
		i := instanceItem{}
		keys := strings.Split(v.Key, "/")
		last := keys[len(keys)-1:][0]
		err = json.Unmarshal([]byte(v.Value), &i)
		if err != nil {
			log.Printf("Could not unmarshal following: %v", v.Value)
			i.Version = "problem"
			i.Timestamp = 0
		}
		instances[last] = i
	}
	return instances, nil
}

func getServiceInfoList(serviceID string) (map[string]string, error) {
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

func getSchema(serviceID string) (map[string]schemaItem, error) {
	resp, err := etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/schema", nil)
	if err != nil {
		return nil, err
	}
	v := make(map[string]schemaItem)
	json.Unmarshal([]byte(resp.Node.Value), &v)
	return v, nil
}

func getConfig(serviceID string) (map[string]item, error) {
	resp, err := etcd.Get(context.Background(), "/ccentral/services/"+serviceID+"/config", nil)
	if err != nil {
		// Most likely new service that has only schema setup, just ignore the missing configuration
		if strings.Contains(err.Error(), "100: Key not found") {
			v := make(map[string]item)
			return v, nil
		} else {
			log.Printf("Configuration could not be loaded, %v", err)
			return nil, err
		}
	}
	v := make(map[string]item)
	json.Unmarshal([]byte(resp.Node.Value), &v)
	return v, nil
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

	config, err := getConfig(serviceID)
	if err != nil {
		writeInternalError(w, "Could not retrieve service configuration", http.StatusInternalServerError)
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeInternalError(w, "Could not read body", http.StatusInternalServerError)
		return
	}

	i := item{}
	i.Value = string(data)
	i.Changed = time.Now().Unix()
	config[keyID] = i

	incrementVersion(config)

	output, err := json.Marshal(config)
	if err != nil {
		writeInternalError(w, "Could not convert to json", http.StatusInternalServerError)
		return
	}

	log.Printf("Configuration updated: [%v] %v=%v (version: %v)", string(serviceID), string(keyID), string(data), config["v"].Value)

	_, err = etcd.Set(context.Background(), "/ccentral/services/"+serviceID+"/config", string(output), nil)
	if err != nil {
		writeInternalError(w, "Could not update configuration", http.StatusInternalServerError)
		return
	}
}

func incrementVersion(config map[string]item) {
	version, ok := config["v"]
	if !ok {
		version = item{}
	}
	value, err := strconv.Atoi(version.Value)
	if err != nil {
		value = 1
	}
	version.Value = strconv.Itoa(value + 1)
	version.Changed = time.Now().Unix()
	config["v"] = version
}

func handleService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	setHeaders(w)
	serviceId := vars["serviceId"]
	schema, err := getSchema(serviceId)
	if err != nil {
		writeInternalError(w, "Could not retrieve service schema", http.StatusInternalServerError)
		return
	}
	config, err := getConfig(serviceId)
	if err != nil {
		writeInternalError(w, "Could not retrieve config", http.StatusInternalServerError)
		return
	}
	instances, err := getInstanceList(serviceId)
	if err != nil {
		log.Printf("Problem getting instances: %v", err)
		writeInternalError(w, "Could not retrieve instances", http.StatusInternalServerError)
		return
	}
	info, err := getServiceInfoList(serviceId)
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

func initEtcd(etcdHost string) {
	var err error
	log.Printf("Connecting to %s", etcdHost)
	cfg := client.Config{Endpoints: []string{etcdHost}}
	e, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	etcd = client.NewKeysAPI(e)
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
	initEtcd(*etcdHost)
	router.HandleFunc("/", handleRoot)
	router.HandleFunc("/check", handleCheck)
	router.HandleFunc("/{path}", handleRoot)
	router.HandleFunc("/api/1/services", handleServiceList)
	router.HandleFunc("/api/1/services/{serviceId}", handleService)
	router.HandleFunc("/api/1/services/{serviceId}/keys/{keyId}", handleItem)
	err := http.ListenAndServe(":"+*port, router)
	if err != nil {
		log.Fatal(err)
	}
}
