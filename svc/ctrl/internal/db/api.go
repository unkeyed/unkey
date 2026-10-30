package db

// Api preserves the model name while queries omit the legacy ip_whitelist column
// and the schema retains it for deployment safety. Remove this alias when the
// column is dropped and sqlc generates Api again.
type Api = FindApiByIDRow
