# service-account-producer-sidecar

A generic sidecar that implements the [Cloudogu service-account-operator](https://github.com/cloudogu/service-account-operator) HTTP `ServiceAccountProducer` API (`PUT /serviceaccounts/`, `DELETE /serviceaccounts/{consumer}`) by executing configurable shell-script hooks.

It carries no Dogu-specific logic itself: the two hook scripts that actually create/delete a service account (e.g. calling a Dogu's REST API, writing config via `doguctl`, ...) are supplied by the consuming Dogu/component, typically mounted from a ConfigMap.

## Configuration

The sidecar is configured entirely via environment variables:

| Variable       | Required | Default | Description                                                        |
|----------------|----------|---------|----------------------------------------------------------------------|
| `API_KEY`      | yes      | -       | Static key compared against the `X-CES-SA-API-KEY` request header    |
| `CREATE_HOOK`  | yes      | -       | Path to the executable invoked on `PUT /serviceaccounts/`            |
| `DELETE_HOOK`  | yes      | -       | Path to the executable invoked on `DELETE /serviceaccounts/{consumer}` |
| `ADDR`         | no       | `:8080` | HTTP listen address                                                   |
| `LOG_LEVEL`    | no       | `INFO`  | `DEBUG`, `INFO`, `WARN` or `ERROR`                                   |
| `HOOK_TIMEOUT` | no       | `30s`   | Maximum duration a single hook invocation may run                    |

## Hook contract

A hook is any executable file. It is invoked as:

```
<hook> [param...] <consumer>
```

- `param...` are the elements of the request's `params` array, passed through unchanged and in order (e.g. `permissions=nx-readonly`).
- `<consumer>` is always the last argument.
- Exit code `0` means success; any other exit code is reported as a `500` to the caller (stderr is included in the error message).
- On the create hook, every line the hook prints to stdout of the form `key: value` becomes an entry in the `credentials` map returned to the caller. All other stdout output is ignored, so hooks may log freely.

This mirrors the existing `create-sa.sh`/`remove-sa.sh` convention used by several Cloudogu Dogus, so those scripts can be used as hooks with little to no changes.

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
