# service-account-producer-sidecar

A generic sidecar that implements the HTTP API expected by the [Cloudogu service-account-operator](https://github.com/cloudogu/service-account-operator)'s producer client (`internal/producer/http_client.go`) by executing configurable shell-script hooks:

| Method       | Path                          | Auth required? | Purpose                                |
|--------------|-------------------------------|----------------|----------------------------------------|
| `PUT`        | `/serviceaccounts`            | yes            | Create or update a service account     |
| `DELETE`     | `/serviceaccounts/{consumer}` | yes            | Remove a service account               |
| `HEAD`       | `/serviceaccounts/{consumer}` | yes            | Check whether a service account exists |
| `GET`/`HEAD` | `/serviceaccounts`            | **no**         | Readiness check                        |

It carries no Dogu-specific logic itself: the hook scripts that actually create/delete/lookup a service account (e.g. calling a Dogu's REST API, reading/writing config via `doguctl`, ...) are supplied by the consuming Dogu/component.

## Configuration

The sidecar is configured entirely via environment variables:

| Variable       | Required | Default | Description                                                            |
|----------------|----------|---------|------------------------------------------------------------------------|
| `API_KEY`      | yes      | -       | Static key compared against the `X-CES-SA-API-KEY` request header      |
| `CREATE_HOOK`  | yes      | -       | Path to the executable invoked on `PUT /serviceaccounts`               |
| `DELETE_HOOK`  | yes      | -       | Path to the executable invoked on `DELETE /serviceaccounts/{consumer}` |
| `EXISTS_HOOK`  | yes      | -       | Path to the executable invoked on `HEAD /serviceaccounts/{consumer}`   |
| `ADDR`         | no       | `:8080` | HTTP listen address                                                    |
| `LOG_LEVEL`    | no       | `INFO`  | `DEBUG`, `INFO`, `WARN` or `ERROR`                                     |
| `HOOK_TIMEOUT` | no       | `30s`   | Maximum duration a single hook invocation may run                      |

## Hook contract

A hook is any executable file. It is invoked as:

```
<hook> [--param=value...] <consumer>
```

- `--param=value...` are the entries of the request's `params` object (`map[string]string`, matching the operator's HTTP client), passed as named long-flags, sorted by key for deterministic invocations (e.g. `--fullAccessRepository=foo --permissions=nx-readonly`). `EXISTS_HOOK` never receives params, only the consumer.
- `CREATE_HOOK` additionally receives the request's `behaviorParams` object (the operator's `producer.BehaviorParams`, e.g. `rotateServiceAccountNow`) as `--behavior-key=value` flags, appended after the domain param flags and sorted by key the same way - e.g. `--permissions=nx-readonly --behavior-rotateServiceAccountNow=true`.
- `<consumer>` is always the last, unnamed argument.
- The sidecar itself has no opinion on what a hook does with these flags - it is deliberately kept free of any assumptions about a specific Dogu script's own CLI convention. If the underlying script expects a different shape (e.g. Nexus's `create-sa.sh`/`remove-sa.sh` expect bare positional `key=value` parameters, no `--`), the hook is expected to be a small wrapper that translates `--key=value` into whatever the real script needs, so that script doesn't have to change.

**`CREATE_HOOK`/`DELETE_HOOK`:** exit code `0` means success; any other exit code is reported as a `500` to the caller (stderr is included in the error message). On the create hook, every line printed to stdout of the form `key: value` becomes an entry in the credentials map returned as the raw JSON response body (matching what the operator's HTTP client decodes: `map[string]string`, not wrapped in an object). All other stdout output is ignored, so hooks may log freely.

**`EXISTS_HOOK`:** follows the classic Unix `grep` convention instead - exit code `0` means the service account exists (`HEAD` responds `200`), exit code `1` means it does not (`HEAD` responds `404`). Any other exit code, or a failure to execute the hook at all, is reported as a `500`.

This mirrors the existing `create-sa.sh`/`remove-sa.sh` convention used by several Cloudogu Dogus, so those scripts can be used as hooks with little to no changes. An equivalent `exists`-style hook (e.g. checking `doguctl config service_accounts/<consumer>`) needs to be added per Dogu.

---
## What is the Cloudogu EcoSystem?
The Cloudogu EcoSystem is an open platform, which lets you choose how and where your team creates great software. Each service or tool is delivered as a Dogu, a Docker container. Each Dogu can easily be integrated in your environment just by pulling it from our registry.

We have a growing number of ready-to-use Dogus, e.g. SCM-Manager, Jenkins, Nexus Repository, SonarQube, Redmine and many more. Every Dogu can be tailored to your specific needs. Take advantage of a central authentication service, a dynamic navigation, that lets you easily switch between the web UIs and a smart configuration magic, which automatically detects and responds to dependencies between Dogus.

The Cloudogu EcoSystem is open source and it runs either on-premises or in the cloud. The Cloudogu EcoSystem is developed by Cloudogu GmbH under [AGPL-3.0-only](https://spdx.org/licenses/AGPL-3.0-only.html).

## License
Copyright © 2020 - present Cloudogu GmbH
This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, version 3.
This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more details.
You should have received a copy of the GNU Affero General Public License along with this program. If not, see https://www.gnu.org/licenses/.
See [LICENSE](LICENSE) for details.


---
MADE WITH :heart:&nbsp;FOR DEV ADDICTS. [Legal notice / Imprint](https://cloudogu.com/en/imprint/?mtm_campaign=ecosystem&mtm_kwd=imprint&mtm_source=github&mtm_medium=link)
