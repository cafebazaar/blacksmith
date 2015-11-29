
var filesServices = angular.module('filesServices', ['ngResource']);
filesServices.factory('UploadedFiles', ['$resource',
  function($resource){
    return $resource('/files/', {}, {
      query: {method:'GET', params:{}, isArray:true},
      delete: {method:'DELETE', params:{name: '@name'}, isArray:false}
    });
  }]);

var nodesServices = angular.module('nodesServices', ['ngResource']);
nodesServices.factory('Node', ['$resource',
    function($resource){
      return $resource('/api/nodes', {}, {
        query: {method:'GET', params:{}, isArray:false}
      });
  }]);

var etcdEndpointsServices = angular.module('etcdEndpointsServices', ['ngResource']);
nodesServices.factory('EtcdEndpoints', ['$resource',
    function($resource){
      return $resource('/api/etcd-endpoints', {}, {
        query: {method:'GET', params:{}, isArray:true}
      });
  }]);
