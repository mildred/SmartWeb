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
    console.log({template: template, data_set: data_set, data: data});
    this.innerHTML = ""
    this.appendChild(template.content.cloneNode(true))
    process.bind(this)(data, {})
}

function process(context, vars){
    vars = {__proto__: vars}
    var newContext = false
    if(this.hasAttributeNS(TEMPLATE_NS, "select")) {
        var code = this.getAttributeNS(TEMPLATE_NS, "select")
        //console.log("eval("+code+"):")
        //console.log({vars: vars, context:context})
        with(vars) context = eval(code)
        this.removeAttributeNS(TEMPLATE_NS, "select")
        newContext = true
    } else if (this.namespaceURI == TEMPLATE_NS && this.hasAttributeNS("", "select")) {
        var code = this.getAttributeNS("", "select")
        with(vars) context = eval(code)
        this.removeAttributeNS("", "select")
        newContext = true
    }
    if (this.namespaceURI == TEMPLATE_NS && this.localName == "value") {
        var newnode = this.ownerDocument.createTextNode(context);
        this.parentElement.insertBefore(newnode, this);
        this.remove(); 
    } else if (newContext && context instanceof Array) {
        for(var i = 0; i < context.length; ++i) {
            var n = this.cloneNode(true)
            follow.bind(n)(context[i]);
            this.parentElement.insertBefore(n, this)
        }
        this.remove()
    } else {
        follow.bind(this)(context);
    }

    function follow(context){
        if(this.hasAttributeNS(TEMPLATE_NS, "variable")) {
            vars[this.getAttributeNS(TEMPLATE_NS, "variable")] = context
            this.removeAttributeNS(TEMPLATE_NS, "variable")
        }
        for(var i = 0; i < this.children.length; ++i) {
            process.bind(this.children[i])(context, vars)
        }
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