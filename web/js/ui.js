var m = angular.module('cCentral', []);

m.controller('MainCtrl', ['$scope', '$http',
    function($scope, $http) {

        $scope.selectedService = "";
        $scope.serviceData = null;
        $scope.services = [];
        $scope.instances = [];
        $scope.instanceHeaders = {};
        $scope.info = [];

        $scope.loadServices = function() {
            $http.get('/api/1/services').then(function(v) {
                console.log(v);
                $scope.services = v.data.services;
            });
        };

        $scope.selectService = function(service) {
            $scope.selectedService = service;
            $scope.serviceData = null;
            $scope.instances = [];
            $scope.instanceHeaders = {};
            $scope.instanceTags = {};
            $scope.info = [];
            $http.get('/api/1/services/' + service).then(function(v) {
                $scope.serviceData = {
                    "v": {
                        "title": "Version",
                        "type": "string",
                        "description": "Automatically incremented on each configuration change"
                    }
                };

                $scope.info = v.data.info;
                _.each(v.data.schema, function(v, k) {
                    $scope.serviceData[k] = v;
                    v.value = v.default;
                    v.value_orig = v.default;
                });
                _.each(v.data.config, function(v, k) {
                    if ($scope.serviceData[k] === undefined) {
                        $scope.serviceData[k] = {
                            value_orig: v.value,
                            value: v.value
                        };
                    } else {
                        $scope.serviceData[k].value_orig = v.value;
                        $scope.serviceData[k].value = v.value;
                    }
                });
                $scope.instances = v.data.clients;
                $scope.instanceTotals = {};

                _.each($scope.instances, function(serviceData, serviceId) {
                    $scope.instanceTags[serviceId] = [];
                    _.each(serviceData, function(value, key) {
                        nkey = key;
                        if (key.startsWith("c_")) {
                            nkey = key.substr(2) + " 1/min";
                            if ($scope.instanceTotals[nkey] === undefined) {
                                $scope.instanceTotals[nkey] = 0;
                            }
                            if (value !== undefined && value.length > 0) {
                                $scope.instanceTotals[nkey] += parseInt(value[value.length - 1]);
                            }
                        }
                        if (key.startsWith("k_")) {
                            nkey = key.substr(2);
                        }
                        if (key === "ts") {
                            if (value < (new Date()).getTime() / 1000 - 60) {
                                $scope.instanceTags[serviceId].push({"text": "Expired timestamp", "type": "warning"});
                            }
                        } else if (key === "v") {
                            if (value != $scope.serviceData.v.value) {
                                $scope.instanceTags[serviceId].push({"text": "Old version ( v." + value + " )", "type": "danger"});
                            }
                        } else {
                            $scope.instanceHeaders[key] = nkey;
                        }
                    });
                    if ($scope.instanceTags[serviceId].length === 0) {
                        $scope.instanceTags[serviceId].push({"text": "Ok", "type": "success"});
                    }
                });
            });
        };

        $scope.representValue = function(key, value) {
            if (key.startsWith("c_")) {
                if (value === undefined || value.length === 0) {
                    return "N/A";
                }
                return value[value.length - 1];
            }
            return value;
        };

        $scope.saveField = function(key) {
            data = $scope.serviceData[key].value;
            console.log("Saving " + key + " = " + data);
            $http.put('/api/1/services/' + $scope.selectedService + "/keys/" + key, data).then(function(v) {
                $scope.serviceData[key].value_orig = $scope.serviceData[key].value;
            });
        };

        $scope.configChanged = function(config) {
            $scope.serviceData.config[config].newValue = config;
        };
        $scope.loadServices();

    }
]);