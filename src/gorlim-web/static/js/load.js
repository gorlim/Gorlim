function getAll() {
	var xhr = new XMLHttpRequest();
  var params = 'needle=' + encodeURIComponent("") 
	 
	xhr.open("POST", '/projects', true)
	xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded')
	xhr.onreadystatechange = function() {
		if (this.readyState != 4) 
			return;
		var array = JSON.parse(this.responseText);
		var list = document.getElementById('search_list');
		while (list.firstChild) {
		  list.removeChild(list.firstChild);
		}
		var length = array.length;
		for (var i = 0 ; i < length; i++) {
		  var li = document.createElement('li');
			li.className = li.className + " list-group-item";
			li.innerHTML = array[i].Origin;
			list.appendChild(li);
		}
		filter();
	};
	xhr.send(params)
}

function create(repo) {
	var xhr = new XMLHttpRequest();
  var params = "type=" + encodeURIComponent("github") +"&repo=" + encodeURIComponent(repo)  
	 
	xhr.open("POST", '/add_project', true)
	xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded')
	xhr.onreadystatechange = function() {
		if (this.readyState != 4) {
			return;
		}

		if (this.status != 200) {
			var err = document.getElementById('myError')
			err.style.visibility = "visible"
			err.innerHTML = this.responseText
			return
		}
		$('#myModal').modal('hide')
		getAll();
	};
	xhr.send(params)
}

function filter() {
	var value = document.getElementById('search_input').value.toLowerCase()
	var nodes = document.getElementById('search_list').childNodes
  var length = nodes.length
	for(var i=0; i<nodes.length; i++) {
		var visible = nodes[i].innerHTML.toLowerCase().indexOf(value) >= 0;
		nodes[i].style.display = visible ? "block" : "none";
	}
}
