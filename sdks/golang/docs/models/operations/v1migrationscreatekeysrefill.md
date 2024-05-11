# V1MigrationsCreateKeysRefill

Unkey enables you to refill verifications for each key at regular intervals.


## Fields

| Field                                                                                                  | Type                                                                                                   | Required                                                                                               | Description                                                                                            |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `Interval`                                                                                             | [operations.V1MigrationsCreateKeysInterval](../../models/operations/v1migrationscreatekeysinterval.md) | :heavy_check_mark:                                                                                     | Unkey will automatically refill verifications at the set interval.                                     |
| `Amount`                                                                                               | *int64*                                                                                                | :heavy_check_mark:                                                                                     | The number of verifications to refill for each occurrence is determined individually for each key.     |