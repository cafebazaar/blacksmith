var blacksmithUIControllers = angular.module('blacksmithUIControllers', []);

blacksmithUIControllers.controller('BlacksmithMachinesCtrl', ['$scope', 'Machines', 'MachineVariable', 'MachineConfigure', function ($scope, Machines, MachineVariable, MachineConfigure) {
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

  $scope.deleteMachine = function(machine, nic) {
    if (!confirm("Are you sure about removal of this machine? This action is not easily undoable."))
      return;

    $scope.machineMac = nic;
    $scope.errorMessage = false;
    MachineConfigure.delete({mac: nic}).$promise.then(
      function( value ){
        $scope.machines.splice($scope.machines.indexOf(machine), 1);
      },
      function( error ){
        $scope.errorMessage = error.data;
      }
    );
  };

  $scope.addMachineVariable = function() {
    var name = prompt("Enter variable name", "");
    var value = prompt("Enter variable value", "");
    if(!name || name == "") return;

    $scope.setMachineVariable(name, value);
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
  $scope.sortType     = 'name';
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
    var name = prompt("Enter variable name", "");
    var value = prompt("Enter variable value", "");
    if(!name || name == "") return;

    $scope.setVariable(name, value);
  };

  $scope.setVariable = function(name, value) {
    Variable.set({name: name, value: value}).$promise.then(
      function( value ){
        $scope.getVariables();
      },
      function( error ){
        $scope.errorMessage = error.data;
      }
    );
  };

  $scope.deleteVariable = function(name) {
    Variable.delete({name: name}).$promise.then(
      function( value ){
        $scope.getVariables();
      },
      function( error ){
        $scope.errorMessage = error.data;
      }
    );
  };

}]);


blacksmithUIControllers.controller('BlacksmithAboutCtrl', ['$scope', 'Version', 'Variable', function ($scope, Version, Variable) {
  Version.query().$promise.then(
    function (value) {
      $scope.info = value;
      $scope.uptime = function() {
        return secondsToStr((new Date().getTime())/1000 - $scope.info.serviceStartTime);
      }
    },
    function (error) { $scope.errorMessage = error.data; $scope.info = {}; }
  );
  Variable.query().$promise.then(
    function (value) {
      $scope.activeWorkspaceHash = value.activeWorkspaceHash;
    },
    function (error) { $scope.errorMessage = error.data; $scope.info = {}; }
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
