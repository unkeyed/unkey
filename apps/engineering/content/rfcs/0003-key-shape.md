---
title: 0003 Key Shape
authors:
  - Andreas Thomas
date: 2024-01-17
---


[https://unkey.slack.com/archives/C05SYU6P2DC/p1705420411688469](https://unkey.slack.com/archives/C05SYU6P2DC/p1705420411688469)

This is the current shape of our keys:`{prefix?}_{version}{length}{base58_randomness}`
I chose this for backwards compatibility if we ever wanted to encode something into the key, for example a semi-public identifier, think `client_id` and `client_secret` in OAuth or other implementations. If the version would match, we knew it was a key with separate client_id and client_secret and could split and parse it correctly.

However there is a problem with this.

We are encoding the version as well as the byte-length at the *beginning* of the key, this makes all of our keys look very similar when you’re only seeing the first few characters.

Other complications come from the fact that we want to be able to onboard other keys without the end user noticing, which means we can not force them to reroll their key, but need to accept it as is, we can only change the hash function to our own (sha256). This means, we can never enforce a strict schema on keys, we can only try to shift the key landscape, by asking users to reroll.

Example:

Resend have keys in the shape of `re_{user_id}_{secret}_{checksum}`

When we want to win them as a customer, we do not want to ask all of their users to create new keys, existing keys must keep working without our customer having to maintain their own system as well as unkey.

We do this by adding their hashes to our db and marking them as well as noting how we derive the hash from the key (see https://linear.app/unkey/issue/ENG-119/migrating-keys-to-unkey).  When they now create new keys through unkey, they’ll have our own keyshape, but old keys do not change.
At this point when we receive a key, we do not know what shape it’ll be, it could be a resend shape, it could be one of our own, or it could be something entirely different from another customer.

## Possible Solution

Having a version encoded is still a good idea I think, but there’s nothing stopping us from moving it to the end, instead of parsing a key from the start, we can just start reading from the back.

## Other thoughts

- We should also think of adding a checksum, which would allow us to clearly identify “this key was created by unkey”, as well as it would help with github secret scanning stuff.
- A problem with github secret scanning is, that we allow our users to choose their prefix, so it’s not as simple as matching `/^unkey_.{16}$/`, we’ll have to look a bit deeper, but a checksum may help.
- our future key shape could look like this:

    ```tsx
    {prefix}_{base58_randomness}{meta}{version}_{checksum}
    ```

    I don’t really like the second delimiter, but I haven’t thought deeply enough how we can match that in a way that doesn’t accidentally match a migrated key
