var aghajoonUIControllers = angular.module('aghajoonUIControllers', []);

aghajoonUIControllers.controller('AghajoonNodesCtrl', ['$scope', 'Node', function ($scope, Node) {
	Node.query().$promise.then(
		function( value ){ $scope.nodes = value },
		function( error ){ $scope.errorMessage = error.data; $scope.nodes = [] }
 );
}]);

aghajoonUIControllers.controller('AghajoonFilesCtrl', ['$scope', 'UploadedFiles', '$http', function ($scope, UploadedFiles, $http) {
    UploadedFiles.query().$promise.then(
      function( value ){ $scope.files = value },
      function( error ){ $scope.errorMessage = error.data; $scope.files = [] }
   );
    $scope.deleteFile = function (file)
    {
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
