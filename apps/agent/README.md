<div align="center">
    <h1 align="center">Vault</h1>
    <h5>Secure storage and encryption for per-tenant data encryption keys</h5>
</div>

<div align="center">
  <a href="https://go.unkey.com">unkey.com</a>
</div>
<br/>




## S3 layout

We will use a single bucket for all tenants. A default name for the bucket is `vault-{env}`, ie. `vault-production`.

Objects are namespaced by version and tenant: `/v1/{TENANT_ID}/{DEK_ID}`

Each tenant may have multiple DEKs and we will have a special `LATEST` object for the latest DEK used for decryption

```
tenant_1/
├─ dek_1
├─ dek_2
├─ LATEST
tenant_2/
├─ dek_5
├─ dek_6
├─ LATEST

```