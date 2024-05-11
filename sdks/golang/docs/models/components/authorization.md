# Authorization

Perform RBAC checks


## Fields

| Field                                                                        | Type                                                                         | Required                                                                     | Description                                                                  | Example                                                                      |
| ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `Permissions`                                                                | [*components.Permissions](../../models/components/permissions.md)            | :heavy_minus_sign:                                                           | A query for which permissions you require                                    | {<br/>"or": [<br/>{<br/>"and": [<br/>"dns.record.read",<br/>"dns.record.update"<br/>]<br/>},<br/>"admin"<br/>]<br/>} |