(function(){
var TEMPLATE_NS = "tag:mildred.fr,2015-05:template#";

var instance = Object.create(HTMLElement.prototype, {
});

instance.createdCallback = function(){
    var observer = new MutationObserver(handler.bind(this));
    observer.observe(document, {
        attributes: true
    });
    //this.compute();
    waitDataSetChanges.bind(this)();

    function handler(mutations) {
        for(var i = 0; i < mutations.length; ++i){
            var mx = mutations[i];
            if (mx.type !== "attributes") continue;
            if (mx.target !== this) continue;
            if (mx.attributeName !== "data-set") continue;
            return waitDataSetChanges.bind(this)();
        }
    }

    var removeListener

    function waitDataSetChanges(){
        if (!this.hasAttribute("data-set")) return;
        if (!this.getAttribute("data-set") === "") return;
        if(removeListener) removeListener();
        var data_set = this.ownerDocument.getElementById(this.getAttribute("data-set"))
        var computeFun = compute.bind(this, data_set)
        if (data_set.ready) {
            computeFun()
        }
        data_set.addEventListener("data", computeFun)
        removeListener = function(){
            data_set.removeEventListener("data", computeFun)
        }
    }

    function compute(data_set){
        var template = this.ownerDocument.getElementById(this.getAttribute("template"))
        var data = data_set.data
        //console.log({template: template, data_set: data_set, data: data});
        this.innerHTML = ""
        this.appendChild(template.content.cloneNode(true))
        process.bind(this)(data, {})
    }
}

// compute when dataset event fires
instance.compute = function() {
    if (!this.hasAttribute("data-set")) return;
    if (!this.getAttribute("data-set") === "") return;
    var template = this.ownerDocument.getElementById(this.getAttribute("template"))
    var data_set = this.ownerDocument.getElementById(this.getAttribute("data-set"))
    var data = data_set.data
    //console.log({template: template, data_set: data_set, data: data});
    this.innerHTML = ""
    this.appendChild(template.content.cloneNode(true))
    process.bind(this)(data, {})
}

function process(context, vars){
    var newContext = false
    if(this.hasAttributeNS(TEMPLATE_NS, "select")) {
        var code = this.getAttributeNS(TEMPLATE_NS, "select")
        //console.log("eval("+code+"):")
        //console.log({vars: vars, context:context})
		try {
	        with(vars) context = eval(code)
			//console.log({element: this, vars: vars, ok: true, parent: this.parentElement});
		} catch (e) {
			console.warn(e);
			console.log({element: this, vars: vars, ok: false, parent: this.parentElement});
		}
        this.removeAttributeNS(TEMPLATE_NS, "select")
        newContext = true
    } else if (this.namespaceURI == TEMPLATE_NS && this.hasAttributeNS("", "select")) {
        var code = this.getAttributeNS("", "select")
		try {
	        with(vars) context = eval(code)			
			//console.log({element: this, vars: vars, ok: true, parent: this.parentElement});
		} catch (e) {
			console.warn(e);
			console.log({element: this, vars: vars, ok: false, parent: this.parentElement});
			context = e
		}
        this.removeAttributeNS("", "select")
        newContext = true
    }
	//console.group("Process");
	//console.log({process: this, context: context})
    if (this.namespaceURI == TEMPLATE_NS && this.localName == "value") {
        var newnode = this.ownerDocument.createTextNode(context);
        this.parentElement.insertBefore(newnode, this);
        this.remove(); 
    } else if (newContext && context instanceof Array) {
        for(var i = 0; i < context.length; ++i) {
            var n = this.cloneNode(true)
			//console.log("Context dataset " + i + "/" + (context.length-1));
            follow.call(n, context[i], vars);
            this.parentElement.insertBefore(n, this)
        }
        this.remove()
    } else {
        follow.call(this, context, vars);
    }
	//console.groupEnd();

    function follow(context, vars){
		//console.log({elem: this, html: this.outerHTML, hasVar: this.hasAttributeNS(TEMPLATE_NS, "variable")});
        if(this.hasAttributeNS(TEMPLATE_NS, "variable")) {
			var varName = this.getAttributeNS(TEMPLATE_NS, "variable");
			vars = {__proto__: vars, [varName]: context}
            this.removeAttributeNS(TEMPLATE_NS, "variable");
			//console.log("Follow " + this.children.length + " children (using "+varName+")");
			//console.log({[varName]: context});
        //} else {
			//console.trace();
			//console.log("Follow " + this.children.length + " children");
		}
		//console.log({follow: this, children: this.children, vars: vars})
		var child = this.firstElementChild
		//var i = 0, max = this.children.length;
		while (child) {
			var nextChild = child.nextElementSibling;
			//console.group("Follow child " + (++i) + "/" + max);
			//console.log({follow_process: child, vars: vars})
            process.call(child, context, vars)
			//console.groupEnd();
			child = nextChild;
		}
		//console.log(i + " children followed");
    }
}

document.registerElementNS(TEMPLATE_NS, "instance", {
    prototype: instance
    /*
    initialized: function(elem){
        console.log("init")
        console.log(elem)
    },
    attached: function(elem){
        console.log("attached")
        console.log(elem)
        //elem.compute()
    },
    detached: function(elem){
        console.log("detached")
        console.log(elem)
    }*/
})


document.registerAttributeNS("tag:mildred.fr,2015-05:template#", "for", {
    //prototype: proto
    initialized: function(elem){
        console.log("init for")
        console.log(elem)
    },
    attached: function(elem){
        console.log("attached for")
        console.log(elem)
    },
    detached: function(elem){
        console.log("detached for")
        console.log(elem)
    }
})


})();