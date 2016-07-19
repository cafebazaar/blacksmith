var blacksmithUIControllers = angular.module('blacksmithUIControllers', []);

blacksmithUIControllers.controller('BlacksmithNodesCtrl', ['$scope', 'Nodes', 'Node', 'Flag', function ($scope, Nodes, Node, Flag) {
  $scope.sortType     = 'name';
  $scope.sortReverse  = false;
  $scope.searchTerm   = '';
  $scope.nodeDetails  = {};
  $scope.nodeName     = '';
  $scope.nodeMac      = '';
  $scope.errorMessage = false;
  $scope.getNodes = function () {
    Nodes.query().$promise.then(
      function( value ){ $scope.nodes = value; },
      function( error ){ $scope.errorMessage = error.data; $scope.nodes = []; $scope.nodeDetails = {} }
    );
  };
  $scope.getNodes();

  $scope.getNode = function(nic, name) {
    $scope.nodeMac = nic;
    $scope.nodeName = name;
    $scope.errorMessage = false;
    Node.query({nic: nic}).$promise.then(
      function( value ){
        $scope.nodeDetails = value;
      },
      function( error ){
        $scope.errorMessage = error.data;
        $scope.nodeDetails = {};
        $('#nodeModal').modal('hide');
      }
    );
  };

  $scope.addFlag = function() {
    var name = prompt("Enter flag name", "");
    var value = prompt("Enter flag value", "");
    if(!name || name == "") return;

    Flag.set({mac: $scope.nodeMac, name: name, value: value}).$promise.then(
      function( value ){
        $scope.getNode($scope.nodeMac, $scope.nodeName);
      },
      function( error ){
        $scope.errorMessage = error.data;
        $('#nodeModal').modal('hide');
      }
    );
  };

  $scope.setFlag = function(name, value) {
    Flag.set({mac: $scope.nodeMac, name: name, value: value}).$promise.then(
      function( value ){
        $scope.getNode($scope.nodeMac, $scope.nodeName);
      },
      function( error ){
        $scope.errorMessage = error.data;
        $('#nodeModal').modal('hide');
      }
    );
  };

  $scope.deleteFlag = function(name) {
    Flag.delete({mac: $scope.nodeMac, name: name}).$promise.then(
      function( value ){
        $scope.getNode($scope.nodeMac, $scope.nodeName);
      },
      function( error ){
        $scope.errorMessage = error.data;
        $('#nodeModal').modal('hide');
      }
    );
  };

}]);



blacksmithUIControllers.controller('BlacksmithVariablesCtrl', ['$scope', 'Variable', function ($scope, Variable) {
  $scope.sortType     = 'key';
  $scope.sortReverse  = false;
  $scope.searchTerm   = '';
  $scope.errorMessage = false;

  $scope.getVariables = function () {
    Variable.query().$promise.then(
      function( value ){ $scope.variables = value; },
      function( error ){ $scope.errorMessage = error.data; $scope.variables = []; }
    );
  };
  $scope.getVariables();
  
  $scope.addVariable = function() {
    var key = prompt("Enter variable key", "");
    var value = prompt("Enter variable value", "");
    if(!key || key == "") return;

    Variable.set({key: key, value: value}).$promise.then(
      function( value ){
        $scope.getVariables();
      },
      function( error ){
        $scope.errorMessage = error.data;
      }
    );
  };

  $scope.setVariable = function(key, value) {
    Variable.set({key: key, value: value}).$promise.then(
      function( value ){
        $scope.getVariables();
      },
      function( error ){
        $scope.errorMessage = error.data;
      }
    );
  };

  $scope.deleteVariable = function(key) {
    Variable.delete({key: key}).$promise.then(
      function( value ){
        $scope.getVariables();
      },
      function( error ){
        $scope.errorMessage = error.data;
      }
    );
  };

}]);


blacksmithUIControllers.controller('BlacksmithAboutCtrl', ['$scope', 'Version', function ($scope, Version) {
  Version.query().$promise.then(
    function( value ){
      $scope.info = value
      $scope.uptime = function() {
        return secondsToStr((new Date().getTime())/1000 - $scope.info.serviceStartTime);
      }
    },
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
