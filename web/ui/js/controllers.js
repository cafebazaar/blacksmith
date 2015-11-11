var aghajoonUIApp = angular.module('aghajoonUIApp', ['nodesServices', 'aghajoonUIControllers']);

var aghajoonUIControllers = angular.module('aghajoonUIControllers', []);

aghajoonUIControllers.controller('AghajoonUICtrl', ['$scope', 'Node', function($scope, Node) {
  $scope.nodes = Node.query();
  $scope.orderProp = 'IP';
}]);