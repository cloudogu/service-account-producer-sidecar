# Configuration and hook contract

This document describes the sidecar's environment variables and the contract its hook scripts
must follow.

## Configuration

The sidecar is configured via environment variables:

| Variable       | Required | Default | Description                                                            |
|----------------|----------|---------|------------------------------------------------------------------------|
| `API_KEY`      | yes      | -       | Static key compared against the `X-CES-SA-API-KEY` request header      |
| `CREATE_HOOK`  | yes      | -       | Path to the executable invoked on `PUT /serviceaccounts`               |
| `DELETE_HOOK`  | yes      | -       | Path to the executable invoked on `DELETE /serviceaccounts/{consumer}` |
| `EXISTS_HOOK`  | yes      | -       | Path to the executable invoked on `HEAD /serviceaccounts/{consumer}`   |
| `ADDR`         | no       | `:8080` | HTTP listen address                                                    |
| `LOG_LEVEL`    | no       | `INFO`  | `DEBUG`, `INFO`, `WARN` or `ERROR`                                     |
| `HOOK_TIMEOUT` | no       | `30s`   | Maximum duration a single hook invocation may run                      |

## Hooks

A hook is any executable file. It is invoked as:

```
<hook> [--param=value...] <consumer>
```

- `--param=value...` are the entries of the request's `params` object, passed as named flags
  (e.g. `--fullAccessRepository=foo --permissions=nx-readonly`). `EXISTS_HOOK` never receives
  params, only the consumer.
- `CREATE_HOOK` additionally receives the request's `behaviorParams` object as
  `--behavior-key=value` flags, appended after the domain param flags, e.g.
  `--permissions=nx-readonly --behavior-rotateServiceAccountNow=true`.
- `<consumer>` is always the last, unnamed argument.

### `CREATE_HOOK` / `DELETE_HOOK`

Exit code `0` means success; any other exit code is reported as a `500` to the caller (stderr is
included in the error message). On the create hook, every line printed to stdout of the form
`key: value` becomes an entry in the credentials map returned as the raw JSON response body. All
other stdout output is ignored, so hooks may log freely.

### `EXISTS_HOOK`

Exit code `0` means the service account exists (`HEAD` responds `200`), exit code `1` means it
does not (`HEAD` responds `404`). Any other exit code, or a failure to execute the hook at all, is
reported as a `500`.
