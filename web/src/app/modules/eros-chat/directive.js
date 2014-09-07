'use strict';

/* Chat directives */

angular.module('erosApp.chat', [])
.directive('erosChat', ['$rootScope', function($rootScope){
	return {
		restrict: 'A',
		templateUrl: 'modules/eros-chat/chat.tpl.html',
		controller: ['$scope',  function($scope){
			$scope.selectRoom = function(room){
				if(typeof room == "object"){
					$scope.$parent.selectedRoom = room
				}else{
					$scope.$parent.selectedRoom = $scope.rooms[Object.keys($scope.rooms)[0]]
				}
				$rootScope.$emit("chat_room","selectedRoom")
			}

			$scope.sendChatMessage = function(target, message) {
				if(message[0] === "@"){
					target = message.split(" ",1)[0].split("@")[1]
					eros.chat.sendToPriv(target, message.split(" ").slice(1).join(" "), false)
				}else if(typeof ($scope.selectedRoom.priv) == 'object'){
					eros.chat.sendToPriv($scope.selectedRoom.priv.name, message, false)
				}else{
					eros.chat.sendToRoom(target, message);
				}
				// $scope.chatMessage = "real"
			}

			$scope.setDefaultRoom = function(room){
				if(typeof $scope.$parent.selectedRoom == 'undefined'){
					$scope.selectRoom(room)
					$scope.defaultRoom = room
				}
			}

			$scope.addUserMsg = function(user){
				var newmessage;
				if(typeof $scope.chatMessage == "undefined" || $scope.chatMessage.length == 0){
					newmessage = "@" + user + " "
				}else{
					newmessage = $scope.chatMessage + " @" + user + " "
				}
				$scope.$parent.chatMessage = newmessage

				document.getElementById("chat-input").childNodes[0].focus()
		
				
			}

			$scope.updateChatInput = function(message){
				// Replace user names
				$scope.chatMessage;
			}
		}]
	}
}])
.directive('roomUsers', ['$rootScope', '$animate', function($rootScope, $animate){
	return{
		restrict: 'AC',
		controller: ['$scope', function($scope){
			// $scope.room = ''
			$scope.users;
		}],
		link: function($scope, $elem, $attrs, $controller){
			$rootScope.$on('chat_room', function(){
				var room = $scope.$parent.selectedRoom.room
				if(room){
					if(typeof(room.users) === "function"){
						$scope.roomusers=room.users()
					}else{
						$scope.roomusers=[]
					}
				}
			})
		}

	}
}])
.directive('username', function(){
	return {
		restrict: 'A',
		template: 'template',
		link: function($scope, $elem, $attrs, $controller){

		}
	}
})
.directive('chatMessage', function(){
	return {
		restrict: 'A',
		link: function($scope, $elem, $attrs, $controller){
			$elem.html($scope.message.message.replace(/@\w+/g, '<span username>'+'$&'+'</span>'))
		}
	}
})
.directive('erosUser', function(){
	return {
		restrict: 'A',
		link: function($scope, $elem, $attrs){
			$attrs.$observe('erosUser', function(value){
				if(value != ""){
					$scope.user = eros.user(value)
				}
			})
		}
	}
})
;