# Table schema 

- `start`: stores the first 9 characters of the key, to help as an identified without exposing the whole thing
- `id`: the key ID, helpfully prefixed with `key_`
- `hash`: the md5 hash of the key
- `name`: optionally give a key a name as an identifier
- `ownerId`: optionally associate the key with a user ID 
- `meta`: optionally add additional metadata to the key
- `createdAt`: unix timestamp of key creation datetime
- `remaining`: optional: a key can be created with a fixed number of uses, in which case the `remaining` field will decrement each time it is verified.