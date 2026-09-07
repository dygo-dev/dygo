# Encrypted Secrets

dygo stores environment secrets as encrypted YAML files in the repository.

The encrypted files are safe to commit. The local `.dygo/secrets/master.key` is ignored by git and required to decrypt, edit, validate, or rotate secrets.

Database credentials use the same model. Local development should store `DATABASE_URL` in the `development` encrypted secrets file, not in plaintext config.

## Storage Model

Committed files:

```txt
config/secrets/development.yml.age
config/secrets/staging.yml.age
config/secrets/production.yml.age
```

Ignored local files:

```txt
.dygo/secrets/master.key
.dygo/secrets/tmp/
```

dygo uses `filippo.io/age` with one hybrid age identity in `.dygo/secrets/master.key`. The public encryption recipient is derived from that key when dygo writes encrypted files, so separate recipient files are not needed.

Do not commit `.dygo/secrets/master.key`. Anyone with that file can decrypt every environment secret file in the project.

## Environments

Use full environment names only:

- `development`
- `staging`
- `production`

Do not use short forms like `dev` or `prod`.

## Commands

Secret commands discover the dygo project root before reading or writing `.dygo/secrets/master.key`, `config/secrets/`, and `.dygo/secrets/tmp/`, so they can be run from nested directories inside a project.

`dygo new <name>` runs the same initialization for new projects and seeds `DATABASE_URL` in development secrets. Run `dygo secret init` directly only for an existing project that does not have encrypted secrets yet.

Initialize secrets:

```sh
dygo secret init
```

This creates `.dygo/secrets/master.key` and missing encrypted files for `development`, `staging`, and `production`.

Edit development secrets:

```sh
dygo secret edit
```

Without `--editor`, dygo opens `nano`.

Edit another environment:

```sh
dygo secret edit --env staging
```

Choose an editor explicitly:

```sh
dygo secret edit --editor nano
dygo secret edit --env staging --editor "code --wait"
```

Print one secret value for scripts:

```sh
dygo secret get DATABASE_URL
dygo secret get database.url --env staging
```

Validate secrets and config references:

```sh
dygo secret validate
dygo secret validate --env staging
```

Rotate the project master key:

```sh
dygo secret rotate-key
dygo secret rotate-key --yes
```

`rotate-key` prints the rotation plan and prompts before writing unless `--yes` is passed. It decrypts every environment with the existing master key, stages and verifies the rotated key and encrypted files, replaces files in a recoverable order, and then re-encrypts every environment file for the new key.

## Decrypted Shape

The encrypted file decrypts to a plain YAML mapping, similar to Rails credentials:

```yaml
DATABASE_URL: postgres://local
STRIPE_SECRET_KEY: sk_test_example
```

Nested YAML is allowed:

```yaml
database:
  url: postgres://local
stripe:
  secret_key: sk_test_example
```

Secret references use either root keys or dot-separated paths:

```txt
DATABASE_URL
database.url
stripe.secret_key
```

Secret references must be non-empty and cannot contain empty path segments.

```txt
database..url
.DATABASE_URL
DATABASE_URL.
```

## Adding Database Credentials

Open the development secrets file:

```sh
dygo secret edit
```

Add `DATABASE_URL`:

```yaml
DATABASE_URL: postgres://user:password@127.0.0.1:5432/dygo
```

Then validate:

```sh
dygo secret validate
```

## Manifest References

Manifests should reference secret names, not raw values.

```yaml
env:
  DATABASE_URL:
    secret: DATABASE_URL
```

`dygo secret validate --env <environment>` checks existing YAML under `config/` for this shape and fails when a referenced secret is missing.

The project database config also references secrets:

```yaml
database:
  url:
    secret: DATABASE_URL
```

Nested references work the same way:

```yaml
database:
  url:
    secret: database.url
```

## Boundaries

Secrets can only be changed through `dygo secret edit` and read through `dygo secret get`. There are no public `set`, `show`, `list`, or `remove` commands.

`.dygo/secrets/master.key` is intentionally project-local for now. Sharing it, backing it up, and injecting it into deployment environments are operational concerns outside this first implementation.

dygo still uses one local root key for development, staging, and production. Per-environment recipients, KMS, Vault, and other external production secret providers are coming soon.

## Record encryption keys

The `secret` Entity field uses the existing credential store. Its dedicated
Record keys are a structured `_record_encryption` mapping inside
`config/secrets/<environment>.yml.age`. The existing master key protects this
file. No separate plaintext Record key file is created. Do not edit this mapping
by hand or copy its contents into source code.

Initialize the existing credential store with `dygo secret init` if necessary.
Then initialize the Record key for the selected environment:

```sh
dygo secret record-key init --env development --dry-run
dygo secret record-key init --env development --yes
```

Initialization preserves existing keys. Invalid key configuration produces an
error; initialization does not replace it. Projects without `secret` fields do
not need Record keys. The CLI supplies the selected environment's key provider
to server, worker, and other runtime command contexts. Missing keys prevent
secret encryption and decryption. Ordinary reads do not decrypt secrets.

### Rotate a Record key

1. Stop all servers, workers, and other database writers for the environment.
2. Back up the database, encrypted environment YAML, and existing master key.
   Keep the master key in secure storage separate from the database backup.
3. Review the target, then run the rotation command:

```sh
dygo secret record-key rotate --env production --offline --dry-run
dygo secret record-key rotate --env production --offline --yes
```

4. Wait for `Record key rotate complete`. Restart servers and workers.

`--offline` confirms that you stopped the writers; it does not stop them for you.
An advisory lock prevents two Record key commands from running together.
Rotation saves the new key in encrypted YAML before it rewrites any Record.
It commits batches of 100 values and verifies all live metadata-defined secret
columns, including Single Entities and collection rows. It does not run
business Hooks or change Record timestamps.

If rotation stops, keep writers stopped and run the same command again. The
saved rotation state reuses the new key and skips values already encrypted with
it. A missing old key or corrupt ciphertext stops rotation. Restore the affected
key or value from a backup before retrying; do not replace the entire key ring.

Old identities remain in encrypted YAML after rotation so older database backups
remain readable. Automatic key deletion is not supported. Back up the updated
encrypted YAML after rotation.

`dygo secret rotate-key` changes the credential store's master key and re-encrypts
the environment YAML files. `dygo secret record-key rotate` changes the Record
data key and re-encrypts database values. These are separate operations. Do not
run credential edits or master-key rotation during Record-key rotation.
