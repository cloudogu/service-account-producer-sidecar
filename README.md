# service-account-producer-sidecar

A generic sidecar that implements the HTTP API expected by the [Cloudogu service-account-operator](https://github.com/cloudogu/service-account-operator)'s producer client (`internal/producer/http_client.go`) by executing configurable shell-script hooks:

| Method       | Path                          | Auth required? | Purpose                                |
|--------------|-------------------------------|----------------|----------------------------------------|
| `PUT`        | `/serviceaccounts`            | yes            | Create or update a service account     |
| `DELETE`     | `/serviceaccounts/{consumer}` | yes            | Remove a service account               |
| `HEAD`       | `/serviceaccounts/{consumer}` | yes            | Check whether a service account exists |
| `GET`/`HEAD` | `/serviceaccounts`            | no             | Readiness check                        |

It carries no Dogu-specific logic itself: the hook scripts that actually create/delete/lookup a service account (e.g. calling a Dogu's REST API, reading/writing config via `doguctl`, ...) are supplied by the consuming Dogu/component.

### Where do I find the configuration and hook contract reference?

- [Konfiguration und Hook-Contract](docs/operations/hooks_de.md)
- [Configuration and hook contract](docs/operations/hooks_en.md)

### Where do I find the integration guide?

- [Service-Account-Producer-Sidecar einbinden](docs/operations/integration_de.md)
- [Integrate the service-account-producer-sidecar](docs/operations/integration_en.md)

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
