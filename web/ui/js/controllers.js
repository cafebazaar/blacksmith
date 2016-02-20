var blacksmithUIControllers = angular.module('blacksmithUIControllers', []);

blacksmithUIControllers.controller('BlacksmithNodesCtrl', ['$scope', 'Node', function ($scope, Node) {
  $scope.sortType     = 'name';
  $scope.sortReverse  = false;
  $scope.searchTerm   = '';
  Node.query().$promise.then(
    function( value ){ $scope.nodes = value },
    function( error ){ $scope.errorMessage = error.data; $scope.nodes = [] }
  );
}]);

blacksmithUIControllers.controller('BlacksmithAboutCtrl', ['$scope', 'Version', function ($scope, Version) {
  Version.query().$promise.then(
    function( value ){ $scope.info = value },
    function( error ){ $scope.errorMessage = error.data; $scope.info = {} }
  );
}]);

blacksmithUIControllers.controller('BlacksmithFilesCtrl', ['$scope', 'UploadedFiles', '$http', function ($scope, UploadedFiles, $http) {
  UploadedFiles.query().$promise.then(
    function( value ){ $scope.files = value },
    function( error ){ $scope.errorMessage = error.data; $scope.files = [] }
  );
  $scope.deleteFile = function (file) {
      if (file.$delete){
        file.$delete(
		      function( value ){},
		      function( error ){ $scope.errorMessage = error.data }
		   );
        var index = $scope.files.indexOf(file);
        if (index > -1) {
          $scope.files.splice(index, 1);
        }
      } else {
        $http.delete('/files?name=' + file.name ).then(
		      function( value ){
		           var index = $scope.files.indexOf(file);
		           if (index > -1) {
		             $scope.files.splice(index, 1);
		           }
					},
		      function( error ){ $scope.errorMessage = error.data }
		    );
      }
    }
}]);
