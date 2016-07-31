var apiServices = angular.module('apiServices', ['ngResource']);
apiServices.factory('UploadedFiles', ['$resource',
  function($resource){
    return $resource('/files/', {}, {
      query: {method:'GET', params:{}, isArray:true},
      delete: {method:'DELETE', params:{id: '@id'}, isArray:false}
    });
  }]);
apiServices.factory('Nodes', ['$resource',
    function($resource){
      return $resource('/api/nodes', {}, {
        query: {method:'GET', params:{}, isArray:true}
      });
  }]);
apiServices.factory('Node', ['$resource',
    function($resource){
      return $resource('/api/node/:nic', {nic: '@nic'}, {
        query: {method:'GET', params:{nic: '@nic'}, isArray:false},
        setIPMI: {method: 'PUT', isArray:false}
      });
  }]);
apiServices.factory('Flag', ['$resource',
    function($resource){
      return $resource('/api/flag/:name', {name: '@name'}, {
        set: {method:'PUT', params:{name: '@name', mac: '@mac', value: '@value'}, isArray:false},
        delete: {method:'DELETE', params:{name: '@name', mac: '@mac'}, isArray:false}
      });
  }]);
  
  apiServices.factory('Variable', ['$resource',
    function($resource){
      return $resource('/api/variables/:key', {key: '@key'}, {  
        query: {method:'GET', params:{}, isArray:false},
        set: {method:'PUT', params:{key: '@key', value: '@value'}, isArray:false},
        delete: {method:'DELETE', params:{key: '@key'}, isArray:false}
      });
  }]);
  
apiServices.factory('Version', ['$resource',
    function($resource){
      return $resource('/api/version', {}, {
        query: {method:'GET', params:{}, isArray:false}
      });
  }]);
