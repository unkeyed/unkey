# ErrUnauthorized

Although the HTTP standard specifies "unauthorized", semantically this response means "unauthenticated". That is, the client must authenticate itself to get the requested response.


## Fields

| Field                                                                            | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `Error`                                                                          | [sdkerrors.ErrUnauthorizedError](../../models/sdkerrors/errunauthorizederror.md) | :heavy_check_mark:                                                               | N/A                                                                              |