# Contributing

Thanks for working on eve.

## Local Verification

Run the checks that match your change:

For normal product or CLI development:

```sh
go test ./...
npm --prefix ui test
npm --prefix ui run build
npm --prefix npm/eve test
npm --prefix npm/eve run pack:check
```

Use `npm --prefix ui ci` before the first local UI build.

For documentation-site changes only:

```sh
npm --prefix site ci
npm --prefix site run build
```

## Product Changes

For completed product changes in this repository:

1. Commit the implementation changes to Git.
2. Record the product change with eve using `eve add` and `eve commit`.
3. Commit the generated `.eve/` record to Git.

Include the verification command and result in the eve record.

Tag releases also publish `@nhestrompia/eve` to npm. Maintainers must configure
the npm automation token as the `NPM_TOKEN` GitHub Actions secret.

## Pull Requests

Keep pull requests focused. Include:

- What changed
- How it was verified
- Any risks, follow-ups, or intentionally deferred work

Do not commit secrets, machine-local credentials, or private customer data.
