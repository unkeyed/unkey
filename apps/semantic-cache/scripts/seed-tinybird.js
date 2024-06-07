fetch("https://api.tinybird.co/v0/events?name=semantic_cache__v6", {
  method: "POST",
  body: JSON.stringify({
    requestId: "id123",
    time: 1717600107824,
    latency: 100,
    gatewayId: "lgw_S7y7TdiEr2YbY8XRUoFJremy5UG",
    workspaceId: "ws_44ytdBUAzAwh6mdNmvgFZbcs5F8P",
    stream: true,
    tokens: 100,
    cache: true,
    model: "gpt-4",
    query: "write a poem about API keys",
    vector: [],
    response: "Hidden lines of code, \nSilent keys unlock the doors — \nSecrets of the web.",
  }),
  headers: {
    Authorization:
      "Bearer p.eyJ1IjogIjY1YzM2ZWZiLTIwMmYtNGE0OC1iOTdjLWQyZWNjNjVhNTNjNiIsICJpZCI6ICJiZjM0NTA2OS0zOWQ5LTRjYjQtOWU2OS00NGUwYjc5MjAyZDgiLCAiaG9zdCI6ICJldV9zaGFyZWQifQ.K1k7mIJ4YPZwqp2nVmKc2ZfKsbCpEP50gzdzWvhcRU8",
  },
})
  .then((res) => res.json())
  .then((data) => console.log(data));
