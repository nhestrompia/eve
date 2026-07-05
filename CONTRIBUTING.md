# Contributing

Thanks for working on EVE.

## Local Verification

Run the checks that match your change:

```sh
go test ./...
npm --prefix ui test
npm --prefix ui run build
npm --prefix site run build
```

Use `npm --prefix ui ci` and `npm --prefix site ci` before the first local build.

## Product Changes

For completed product changes in this repository:

1. Commit the implementation changes to Git.
2. Record the product change with EVE using `eve add` and `eve commit`.
3. Commit the generated `.eve/` record to Git.

Include the verification command and result in the EVE record.

## Pull Requests

Keep pull requests focused. Include:

- What changed
- How it was verified
- Any risks, follow-ups, or intentionally deferred work

Do not commit secrets, machine-local credentials, or private customer data.
