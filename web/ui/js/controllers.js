var aghajoonUIControllers = angular.module('aghajoonUIControllers', []);

aghajoonUIControllers.controller('AghajoonNodesCtrl', ['$scope', 'Node', function ($scope, Node) {
	Node.query().$promise.then(
		function( value ){ $scope.nodes = value },
		function( error ){ errorMessage = error }
 );
	$scope.errorMessage = '';
}]);

aghajoonUIControllers.controller('AghajoonFilesCtrl', ['$scope', 'UploadedFiles', '$http', function ($scope, UploadedFiles, $http) {
    $scope.files = UploadedFiles.query().$promise.then(
      function( value ){ $scope.files = value },
      function( error ){ errorMessage = error }
   );
		$scope.errorMessage = '';
    $scope.deleteFile = function (file)
    {
      if (file.$delete){
        file.$delete(
		      function( value ){},
		      function( error ){ errorMessage = error }
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
		      function( error ){ errorMessage = error }
		    );
      }
    }
}]);
