package main

// func init() {

// 	config := newConfig()

// 	if config.GraphiteEnabled {
// 		Graphite, err = graphite.NewGraphite(config.Graphite.Host, config.Graphite.Port)
// 	} else {
// 		Graphite = graphite.NewGraphiteNop(config.Graphite.Host, config.Graphite.Port)
// 	}
// 	// if you couldn't connect to graphite, use a nop
// 	if err != nil {
// 		Graphite = graphite.NewGraphiteNop(config.Graphite.Host, config.Graphite.Port)
// 	}

// 	log.Printf("Loaded Graphite connection: %#v", Graphite)
// 	Graphite.SimpleSend("stats.graphite_loaded", "1")
// }

// func graphitePollLoop(service) {
// 	for {
// 		enabled, _ := service.GetConfigBool("graphite_enabled")
// 		if enabled {
// 			serviceList, err := GetServiceList()
// 			if err != nil {
// 				log.Printf("WARN Could not retrieve service list")
// 			}
// 			for _, serviceID := range serviceList.Services {
// 				log.Printf("Handling service %v", serviceID)
// 				instances, err := GetInstanceList(serviceID)
// 				if err == nil {
// 					count := len(instances)
// 				}
// 			}
// 			log.Printf("Sent total of %v records to Graphite", 0)
// 		}
// 		interval, _ := service.GetConfigInt("graphite_interval")
// 		time.Sleep(time.Duration(interval) * time.Second)
// 	}
// }

// func startGraphiteUpdater(service *CCentralService) {
// 	go pollLoop(service)
// }
