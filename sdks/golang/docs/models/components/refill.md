# Refill

Unkey allows you to refill remaining verifications on a key on a regular interval.


## Fields

| Field                                                             | Type                                                              | Required                                                          | Description                                                       | Example                                                           |
| ----------------------------------------------------------------- | ----------------------------------------------------------------- | ----------------------------------------------------------------- | ----------------------------------------------------------------- | ----------------------------------------------------------------- |
| `Interval`                                                        | [components.Interval](../../models/components/interval.md)        | :heavy_check_mark:                                                | Determines the rate at which verifications will be refilled.      | daily                                                             |
| `Amount`                                                          | *int64*                                                           | :heavy_check_mark:                                                | Resets `remaining` to this value every interval.                  | 100                                                               |
| `LastRefillAt`                                                    | **float64*                                                        | :heavy_minus_sign:                                                | The unix timestamp in miliseconds when the key was last refilled. | 100                                                               |