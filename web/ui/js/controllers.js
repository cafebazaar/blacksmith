var blacksmithUIControllers = angular.module('blacksmithUIControllers', []);

blacksmithUIControllers.controller('BlacksmithMachinesCtrl', ['$scope', 'Machines', 'MachineVariable', function ($scope, Machines, MachineVariable) {
  $scope.sortType     = 'name';
  $scope.sortReverse  = false;
  $scope.searchTerm   = '';
  $scope.machineDetails  = {};
  $scope.machineName     = '';
  $scope.machineMac      = '';
  $scope.IPMInode     = '';
  $scope.errorMessage = false;
  $scope.getMachines = function () {
    Machines.query().$promise.then(
      function( value ){ $scope.machines = value; },
      function( error ){ $scope.errorMessage = error.data; $scope.machines = []; $scope.machinesDetails = {} }
    );
  };
  $scope.getMachines();

  $scope.getMachine = function(nic, name, IPMInode) {
    $scope.machineMac = nic;
    $scope.machineName = name;
    $scope.IPMInode = IPMInode;
    $scope.errorMessage = false;
    MachineVariable.query({mac: nic}).$promise.then(
      function( value ){
        $scope.machineDetails = value;
      },
      function( error ){
        $scope.errorMessage = error.data;
        $scope.machineDetails = {};
        $('#machineModal').modal('hide');
      }
    );
  };

  $scope.addMachineVariable = function() {
    var name = prompt("Enter variable name", "");
    var value = prompt("Enter variable value", "");
    if(!name || name == "") return;

    MachineVariable.set({mac: $scope.machineMac, name: name, value: value}).$promise.then(
      function( value ){
        $scope.getMachine($scope.machineMac, $scope.machineName);
      },
      function( error ){
        $scope.errorMessage = error.data;
        $('#machineModal').modal('hide');
      }
    );
  };

  $scope.setIPMI = function (nic, IPMInode) {
    Machines.setIPMI({nic: nic, machine: nic, IPMImachine: IPMImachine}).$promise.then(
        function (value) {},
        function (error) {
          $scope.errorMessage = error.data;
          $('#machineModal').modal('hide');
        }

    );
  };

  $scope.setMachineVariable = function(name, value) {
    MachineVariable.set({mac: $scope.machineMac, name: name, value: value}).$promise.then(
      function( value ){
        $scope.getMachine($scope.machineMac, $scope.machineName);
      },
      function( error ){
        $scope.errorMessage = error.data;
        $('#machineModal').modal('hide');
      }
    );
  };

  $scope.deleteMachineVariable = function(name) {
    MachineVariable.delete({mac: $scope.machineMac, name: name}).$promise.then(
      function( value ){
        $scope.getMachine($scope.machineMac, $scope.machineName);
      },
      function( error ){
        $scope.errorMessage = error.data;
        $('#machineModal').modal('hide');
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
        $http.delete('/files?id=' + file.id ).then(
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
