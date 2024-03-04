# Websocket for AKL analyzers

wss://807843.xyz
- first 6 numbers of the AKL discord server ID

### Websocket urls
- cmini: `~/python/cmini`
- a200: `~/python/a200`
- oxeylyzer: `~/rust/oxeylyzer`
- genkey: `~/go/genkey`

### How to use
1. open up your browser console
2. paste this code snippet and edit the url

```javascript
let webSocket = new WebSocket("wss://807843.xyz");
webSocket.onopen = function() {
  console.log("Connected to websocket!");
};

webSocket.onmessage = function(event) {
  const message = event.data;
  console.log("Received message: " + message);
};

function send(msg) {
  if (webSocket.readyState === WebSocket.OPEN) {
    webSocket.send(msg);
  }
}
```

3. call `send("your message goes here")`
   to send a message to the websocket
