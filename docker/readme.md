# Unkey API: minimal Dockerized implementation

This contains an example implementation of Unkey's API using SQLite and Hono. It's designed to be run on a single long-running server. Keys are persisted in a single SQLite file. Since this does not contain dependencies on third-party services such as Planetscale, it's possible to self-host this in a single Docker container. 

A minimal subset of the Unkey API is exposed, with just two routes:

- `/v1/keys/createKey` to create a new key, and
- `v1/keys/verifyKey` to verify a key.

This is a single-tenant implementation, so keys are not associated with an API. Other limitations:

- Rate limiting is not supported
- Key expiration is not supported
- 'Enabled' is not supported

Otherwise, the key creation and verification API matches that of the [main Unkey API](https://unkey.dev/docs/api-reference/keys/create). Example requests:

### Key creation

```bash

curl --request POST \
     --url http://localhost:3000/v1/keys/createKey \
     --header 'Authorization: Bearer <token>' \
     --header 'Content-Type: application/json' \
     --data '{
        "name": "myKey",
        "ownerId": "userId123",
        "remaining": 20,
        "meta": { "roles" : "admin" }
     }'
```

### Key verification

```bash

curl --request POST \
      --url http://localhost:3000/v1/keys.verifyKey \
      --header 'Content-Type: application/json' \
      --data '{
        "key": "sk_1234"
      }'

```

# Table schema 

- `start`: stores the first 9 characters of the key, to help as an identified without exposing the whole thing
- `id`: the key ID, helpfully prefixed with `key_`
- `hash`: the md5 hash of the key
- `name`: optionally give a key a name as an identifier
- `ownerId`: optionally associate the key with a user ID 
- `meta`: optionally add additional metadata to the key
- `createdAt`: unix timestamp of key creation datetime
- `remaining`: optional: a key can be created with a fixed number of uses, in which case the `remaining` field will decrement each time it is verified.