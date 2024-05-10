# HTTPMetadata


## Fields

| Field                                                   | Type                                                    | Required                                                | Description                                             |
| ------------------------------------------------------- | ------------------------------------------------------- | ------------------------------------------------------- | ------------------------------------------------------- |
| `Response`                                              | [*http.Response](https://pkg.go.dev/net/http#Response)  | :heavy_check_mark:                                      | Raw HTTP response; suitable for custom response parsing |
| `Request`                                               | [*http.Request](https://pkg.go.dev/net/http#Request)    | :heavy_check_mark:                                      | Raw HTTP request; suitable for debugging                |