var apiServices = angular.module('apiServices', ['ngResource']);
apiServices.factory('UploadedFiles', ['$resource',
  function($resource){
    return $resource('/files/', {}, {
      query: {method:'GET', params:{}, isArray:true},
      delete: {method:'DELETE', params:{name: '@name'}, isArray:false}
    });
  }]);
apiServices.factory('Node', ['$resource',
    function($resource){
      return $resource('/api/nodes', {}, {
        query: {method:'GET', params:{}, isArray:true}
      });
  }]);
apiServices.factory('Version', ['$resource',
    function($resource){
      return $resource('/api/version', {}, {
        query: {method:'GET', params:{}, isArray:false}
      });
  }]);
