/*
 * Implementation of document.registerElementNS
 * ============================================
 * 
 * registerElementNS(namespaceURI, localName, definition) registers a custom
 * element in the prefix identified by namespaceURI. It also register the custom
 * element "{namespaceURI}-{localName}" in the XHTML prefix (it relies on
 * document.registerElement implementation for that). The `definition` parameter
 * must be an object that may contain:
 * 
 *   - `prototype`: replacement of the prototype (modification of __poto__) for
 *     the custom elements
 *   - `initialized`: function called to initialize the custom element
 *   - `attached`: function called when the custom element is attached
 *   - `detached`: function called when the custom element is detached
 * 
 * This library also implements document.registerAttributeNS on the same
 * principles as document.registerElementNS (except the `prototype` in the
 * definition parameter is not used).
 * 
 * This implementation has not been tested a lot. It has been found to work on
 * Firefox 38. Notably, it uses DOM3 XPath that may not be available everywhere.
 * 
 * This implementation is freely inspired from the Polymer implementation of
 * document.registerElement and some code may have found its way here.
 */

(function(){
var elementRegistry = {};
var attributeRegistry = {};
var convertedNodes = new WeakMap()
var attachedNodes = new WeakMap()

function xpathStringLiteral(s) {
    //if (s === undefined)
    //    console.trace(s);
    if (s.indexOf('"')===-1)
        return '"'+s+'"';
    if (s.indexOf("'")===-1)
        return "'"+s+"'";
    return 'concat("'+s.replace(/"/g, '",\'"\',"')+'")';
}

function evaluateXPath(aNode, aExpr) {
    var xpe = new XPathEvaluator();
    var nsResolver = xpe.createNSResolver(aNode.ownerDocument == null ?
        aNode.documentElement : aNode.ownerDocument.documentElement);
    var result = xpe.evaluate(aExpr, aNode, nsResolver, 0, null);
    //console.log(result)
    if (result.resultType == 1) {
        return result.numberValue
    } else if (result.resultType == 2) {
        return result.stringValue
    } else if (result.resultType == 3) {
        return result.booleanValue
    }
    var found = [];
    var res;
    while (res = result.iterateNext())
        found.push(res);
    return found;
}

function lookupPrefix(document, namespaceURI){
    var prefix = document.lookupPrefix(namespaceURI)
    // Fallback for html
    if(prefix === null) {
        var attrs = document.documentElement.attributes;
        for(var i = 0; prefix === null && i < attrs.length; ++i) {
            if (attrs[i].value === namespaceURI &&
                attrs[i].localName.startsWith("xmlns:"))
            {
                prefix = attrs[i].localName.substring(6)
            }
        }
    }
    return prefix
}

function registerElementNS(namespaceURI, qualifiedName, r){
    r.namespaceURI = namespaceURI;
    r.qualifiedName = qualifiedName;
    r.nodeType = document.ELEMENT_NODE
    elementRegistry[namespaceURI + qualifiedName] = r;
    forEachElems(convertElem, this, r);
    var prefix = lookupPrefix(this, namespaceURI)
    if (prefix) {
        this.registerElement(prefix + "-" + qualifiedName, r)
    }
}

function registerAttributeNS(namespaceURI, qualifiedName, r){
    r.namespaceURI = namespaceURI;
    r.qualifiedName = qualifiedName;
    r.nodeType = document.ATTRIBUTE_NODE
    attributeRegistry[namespaceURI + qualifiedName] = r;
}

function forEachChildren(elemFunc, attrFunc, parentNode) {
    if(parentNode.localName && parentNode.namespaceURI) {
        var uri = parentNode.namespaceURI + parentNode.localName;
        var reg = elementRegistry[uri]
        if(reg) {
            elemFunc(parentNode, reg)
        }
    }
    // TODO WTF? r attributeRegistry[r] Somethingf is wrong in one loop
    for(r in elementRegistry) {
        forEachElems(elemFunc, parentNode, r);
    }
    for(r in attributeRegistry) {
        forEachAttr(attrFunc, parentNode, attributeRegistry[r])
    }
}

function forEachElems(func, parentNode, reg) {
    var elems = parentNode.getElementsByTagNameNS(reg.namespaceURI, reg.qualifiedName);
    for(var i = 0; i < elems.length; ++i) {
        func(elems[i], reg);
    }
}

function forEachAttr(func, parentNode, reg) {
    var attributes = evaluateXPath(parentNode, "//@*[namespace-uri(.)="+xpathStringLiteral(reg.namespaceURI)+"][local-name(.)="+xpathStringLiteral(reg.qualifiedName)+"]")
    for(var i = 0; i < attributes.length; ++i) {
        func(attributes[i], reg);
    }
}

function convertElem(elem, reg) {
    if (convertedNodes.get(elem) === reg) {
        return;
    }
    convertedNodes.set(elem, reg)
    if(reg.prototype) {
        elem.__proto__ = reg.prototype;
    }
    if(reg.initialized){
        reg.initialized(elem)
    }
    if(elem.createdCallback) {
        elem.createdCallback();
    }
}

function attachElem(elem, reg){
    if(attachedNodes.has(elem)){
        return;
    }
    attachedNodes.set(elem, reg)
    if(reg.attached){
        reg.attached(elem)
    }
    if(elem.attachedCallback) {
        elem.attachedCallback();
    }
}

function detachElem(elem, reg){
    if(!attachedNodes.has(elem)){
        return;
    }
    if(elem.detachedCallback) {
        elem.detachedCallback();
    }
    if(reg.detached){
        reg.detached(elem)
    }
    attachedNodes.delete(elem)
}

function convertAttrNode(attr, reg) {
    if (convertedNodes.get(attr) === reg) {
        return;
    }
    convertedNodes.set(attr, reg)
    if(reg.initialized){
        reg.initialized(attr)
    }
}

function convertAttr(elem, ns, localName, reg) {
    return convertAttrNode(elem.getAttributeNS(ns, localName), reg);
}

function attachAttrNode(attr, reg) {
    if(attachedNodes.has(attr)){
        return;
    }
    attachedNodes.set(attr, reg)
    if(reg.attached){
        reg.attached(attr)
    }
}

function attachAttr(elem, ns, localName, reg) {
    return attachAttrNode(elem.getAttributeNS(ns, localName), reg);
}

function detachAttrNode(attr, reg) {
    if(!attachedNodes.has(attr)){
        return;
    }
    if(reg.detached){
        reg.detached(attr)
    }
    attachedNodes.delete(attr)
}

function detachAttr(elem, ns, localName, reg) {
    return detachAttrNode(elem.getAttributeNS(ns, localName), reg);
}

function attachedNode(n){
    forEachChildren(convertElem, convertAttrNode, n)
    forEachChildren(attachElem, attachAttrNode, n)
}

function detachedNode(n){
    forEachChildren(detachElem, detachAttrNode, n)
}

function attachedAttr(elem, ns, localName, reg){
    convertAttr(elem, ns, localName, reg)
    attachAttr(elem, ns, localName, reg)
}

function detachedAttr(elem, ns, localName, reg){
    detachAttr(elem, ns, localName, reg)
}

/*
  NOTE: In order to process all mutations, it's necessary to recurse into
  any added nodes. However, it's not possible to determine a priori if a node
  will get its own mutation record. This means
  *nodes can be seen multiple times*.

  Here's an example:

  (1) In this case, recursion is required to see `child`:

      node.innerHTML = '<div><child></child></div>'

  (2) In this case, child will get its own mutation record:

      node.appendChild(div).appendChild(child);
*/
function handler(mutations) {
    for(var i = 0; i < mutations.length; ++i){
        var mx = mutations[i];
        if (mx.type === 'childList') {
            for(var j = 0; j < mx.addedNodes.length; ++j) {
                var n = mx.addedNodes[j];
                if (!n.localName) {
                    continue;
                }
                attachedNode(n);
            };
            for(var j = 0; j < mx.removedNodes.length; ++j) {
                var n = mx.removedNodes[j];
                if (!n.localName) {
                    continue;
                }
                detachedNode(n);
            }
        } else if (mx.type === "attributes") {
            var ns = mx.attributeNamespace
            var localName = mx.attributeName
            var reg = attributeRegistry[ns+localName]
            if (reg && mx.target.hasAttributeNS(ns, localName) && !mx.oldValue) {
                attachedAttr(mx.target, ns, localName, reg)
            } else if(reg && !mx.target.hasAttributeNS(ns, localName)) {
                detachedAttr(mx.target, ns, localName, reg)
            }
        }
    }
};

function observe(document){
    if (document.__XMLBindings_observer) {
        return;
    }
    var observer = new MutationObserver(handler);
    observer.observe(document, {
        childList: true,
        subtree: true,
        attributes: true,
        attributeOldValue: true,
    });
    document.__XMLBindings_observer = observer;
}

function convertDocument(document){
    var domCreateElementNS = document.createElementNS.bind(document);
    var domCreateElement = document.createElement.bind(document);

    function createElementNS(namespace, tag, typeExtension) {
        console.log("createElementNS(" + namespace + ", " + tag + ")");
        return domCreateElementNS(namespace, tag, typeExtension);
    }

    attachedNode(document);
    observe(document);

    document.createElementNS = createElementNS.bind(document);
    document.registerElementNS = registerElementNS.bind(document);
    document.registerAttributeNS = registerAttributeNS.bind(document);
}

convertDocument(document);
})();
