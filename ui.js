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
            $scope.info = [];
            $http.get('/api/1/services/' + service).then(function(v) {
                $scope.serviceData = {"v": {"title": "Version",
                                      "type": "string",
                                      "description": "Automatically incremented on each configuration change"}};
                $scope.instances = v.data.clients;
                console.log($scope.instances)
                _.each($scope.instances, function(serviceData, serviceId) {
                    _.each(serviceData, function(value, key) {
                        nkey = key;
                        if (key.startsWith("c_")) {
                            nkey = key.substr(2)
                        }
                        $scope.instanceHeaders[key] = nkey;
                    });
                });
                console.log($scope.instanceHeaders);
                $scope.info = v.data.info;
                _.each(v.data.schema, function(v, k) {
                    $scope.serviceData[k] = v;
                    v.value = v.default;
                    v.value_orig = v.default;
                });
                _.each(v.data.config, function(v, k) {
                    if ($scope.serviceData[k] === undefined) {
                        $scope.serviceData[k] = {value_orig: v.value, value: v.value};
                    } else {
                        $scope.serviceData[k].value_orig = v.value;
                        $scope.serviceData[k].value = v.value;
                    }
                });
            });
        };

        $scope.representValue = function(value) {
            if (typeof value === 'object') {
                return value[0];
            }
            return value;
        }

        $scope.saveField = function(key) {
            data = $scope.serviceData[key].value;
            console.log("Saving " + key + " = " + data);
            $http.put('/api/1/services/' + $scope.selectedService + "/keys/" + key, data).then(function(v) {
                $scope.serviceData[key].value_orig =$scope.serviceData[key].value;
            });
        };

        $scope.configChanged = function(config) {
            $scope.serviceData.config[config].newValue = config;
        };
        $scope.loadServices();

    }
]);