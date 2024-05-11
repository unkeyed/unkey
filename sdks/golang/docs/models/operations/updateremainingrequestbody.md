# UpdateRemainingRequestBody


## Fields

| Field                                                             | Type                                                              | Required                                                          | Description                                                       | Example                                                           |
| ----------------------------------------------------------------- | ----------------------------------------------------------------- | ----------------------------------------------------------------- | ----------------------------------------------------------------- | ----------------------------------------------------------------- |
| `KeyID`                                                           | *string*                                                          | :heavy_check_mark:                                                | The id of the key you want to modify                              | key_123                                                           |
| `Op`                                                              | [operations.Op](../../models/operations/op.md)                    | :heavy_check_mark:                                                | The operation you want to perform on the remaining count          |                                                                   |
| `Value`                                                           | *int64*                                                           | :heavy_check_mark:                                                | The value you want to set, add or subtract the remaining count by | 1                                                                 |