var apiServices = angular.module('apiServices', ['ngResource']);

apiServices.factory('Machines', ['$resource',
  function($resource){
    return $resource('/api/machines', {}, {
      query: {method:'GET', params:{}, isArray:true}
    });
}]);

apiServices.factory('Variable', ['$resource',
  function($resource){
    return $resource('/api/variables/:name', {}, {
      query: {method:'GET', params:{}, isArray:false},
      set: {method:'PUT', params:{name: '@name', value: '@value'}, isArray:false},
      delete: {method:'DELETE', params:{name: '@name'}, isArray:false}
    });
}]);

apiServices.factory('MachineVariable', ['$resource',
  function($resource){
    return $resource('/api/machines/:mac/variables/:name', {}, {
      query: {method: 'GET', params: {mac: '@mac'}, isArray: false},  
      set: {method:'PUT', params:{name: '@name', mac: '@mac', value: '@value'}, isArray:false},
      delete: {method:'DELETE', params:{name: '@name', mac: '@mac'}, isArray:false}
    });
}]);

apiServices.factory('MachineConfigure', ['$resource',
  function ($resource) {
    return $resource('/api/machines/:mac', {}, {
      delete: {method:'DELETE', params:{name: '@mac'}, isArray:false}
    });
}]);

apiServices.factory('Version', ['$resource',
  function($resource){
    return $resource('/api/version', {}, {
      query: {method:'GET', params:{}, isArray:false}
    });
}]);
