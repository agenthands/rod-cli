// on_dom_node_inserted.js - onDOMNodeInserted recipe for rod-cli
// Hook: onDOMNodeInserted -> CDP event proto.DOMChildNodeInserted
// Logs the node name of every node inserted into the DOM.
// Read results with: plugin run getInsertedNodes

var insertedNodes = [];

// onDOMNodeInserted fires each time a child node is inserted into the DOM.
// event.ParentNodeID is the parent; event.Node is the newly inserted node.
function onDOMNodeInserted(event) {
  if (event && event.Node) {
    insertedNodes.push(event.Node.NodeName);
  }
}

// getInsertedNodes returns all collected node names as JSON.
function getInsertedNodes() {
  return JSON.stringify(insertedNodes);
}
