var nodesServices = angular.module('nodesServices', ['ngResource']);

nodesServices.factory('Node', ['$resource',
  function($resource){
    return $resource('/api/nodes', {}, {
      query: {method:'GET', params:{}, isArray:false}
    });
  }]);