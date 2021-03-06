var blacksmithUIApp = angular.module('blacksmithUIApp', ["xeditable", 'ngRoute', 'blacksmithUIControllers', 'apiServices']);

blacksmithUIApp.config(['$routeProvider', '$locationProvider',
  function($routeProvider, $locationProvider) {
    $routeProvider.
      when('/ui/machines/', {
        templateUrl: 'static/partials/machines-list.html',
        controller: 'BlacksmithMachinesCtrl'
      }).
      when('/ui/variables/', {
        templateUrl: 'static/partials/variables-list.html',
        controller: 'BlacksmithVariablesCtrl'
      }).
      when('/ui/about/', {
        templateUrl: 'static/partials/about.html',
        controller: 'BlacksmithAboutCtrl'
      }).
      otherwise({
        redirectTo: '/ui/machines/'
      });

      $locationProvider.html5Mode(true);
  }]);

blacksmithUIApp
.filter('custom', function() {
  return function(input, search) {
    if (!input) return input;
    if (!search) return input;
    var expected = ('' + search).toLowerCase();
    var result = {};
    angular.forEach(input, function(value, key) {
      var actual = ('' + key).toLowerCase();
      if (actual.indexOf(expected) !== -1) {
        result[key] = value;
      }
    });
    return result;
  }
});

blacksmithUIApp.directive('dragAndDrop', function() {
    return {
      restrict: 'A',
      link: function($scope, elem, attr) {
        elem.bind('dragenter', function(e) {
          e.stopPropagation();
          e.preventDefault();
          // still can't get this one to behave:
          // http://stackoverflow.com/q/15419839/740318
          $scope.$apply(function () {
            $scope.divClass = 'on-drag-enter';
          });
        });
        elem.bind('dragleave', function(e) {
          e.stopPropagation();
          e.preventDefault();
          $scope.divClass = '';
        });
        elem.bind('dragover', function (e) {
          e.stopPropagation();
          e.preventDefault();
          e.dataTransfer.dropEffect = 'copy';
        });
        elem.bind('drop', function(e) {
          var droppedFiles = e.dataTransfer.files;
          // It's as though the following two methods never occur
          e.stopPropagation();
          e.preventDefault();


          $.each (droppedFiles, function(inn, droppedFile) {

              var index = -1;
              for (file of $scope.files)
              {
                if (file.name == droppedFile.name)
                  index = $scope.files.indexOf(file) + 1;
              }
              if (index == -1) {
                index = $scope.files.push($.extend({}, droppedFile));
              }

              var newFile = $scope.files[index - 1];

              var form = new FormData();
              var xhr = new XMLHttpRequest;

              // Additional POST variables required by the API script
              form.append('file', droppedFile);

              xhr.upload.onprogress = function(e) {
                  // Event listener for when the file is uploading
                    var percentCompleted;
                    if (e.lengthComputable) {
                      $scope.$apply(function () {
                        percentCompleted = Math.round(e.loaded / e.total * 100);
                        if (percentCompleted < 1) {
                            // .uploadStatus will get rendered for the user via the template
                            newFile.uploadStatus =  'Uploading ...';
                            newFile.uploadProgress = percentCompleted;
                        } else if (percentCompleted == 100) {
                            newFile.uploadStatus = 'Saving...';
                            newFile.uploadProgress = percentCompleted;
                        } else {
                            newFile.uploadStatus = 'Uploading ...';
                            newFile.uploadProgress = percentCompleted;
                        }
                      });
                    }

              };

              xhr.upload.onload = function(e) {
                  // Event listener for when the file completed uploading
                  $scope.$apply(function () {
                      newFile.uploadStatus = ''
                      newFile.uploadProgress = 100;
                    });

              };

              xhr.open('POST', '/upload/');
              xhr.send(form);

          });
        });
      }
    };
  });
