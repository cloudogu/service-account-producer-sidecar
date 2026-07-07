# Integrate the service-account-producer-sidecar

This document describes how to integrate the service-account-producer-sidecar into a Dogu or
component's Helm chart, so it can act as a `ServiceAccountProducer` for the
[service-account-operator](https://github.com/cloudogu/service-account-operator). See the
[README](../../README.md) for the full HTTP API and hook reference.

## Prerequisites

- Existing logic to create, delete and check for a service account in the target application
  (scripts, an API client, or similar), available as executables (shell scripts)
- A Helm chart for the Dogu/component
- The `service-account-operator` running in the target cluster

## 1. Provide the hook scripts in the sidecar container

The sidecar image is plain Alpine and does not contain any application-specific tooling. Copy the
scripts that implement create/delete/exists into a volume shared with the sidecar, using an init
container based on the application's own image:

```yaml
initContainers:
  - name: sa-manager-hooks-init
    image: "<application-image>"
    command: ["sh", "-c", "cp /create-sa.sh /remove-sa.sh /hooks-src/*.sh /shared/ && chmod 0555 /shared/*.sh"]
    volumeMounts:
      - name: sa-manager-hooks
        mountPath: /shared
containers:
  - name: sa-manager
    volumeMounts:
      - name: sa-manager-hooks
        mountPath: /hooks
        readOnly: true
volumes:
  - name: sa-manager-hooks
    emptyDir: {}
```

If the hook scripts call `doguctl` (as most Cloudogu Dogu hook scripts do), copy that binary in
from the application image the same way - the sidecar image does not provide it:

```yaml
command: ["sh", "-c", "cp /usr/local/bin/doguctl /create-sa.sh /remove-sa.sh /hooks-src/*.sh /shared/ && chmod 0555 /shared/*"]
```

## 2. Configure the sidecar container

```yaml
- name: sa-manager
  image: registry.example.com/service-account-producer-sidecar:0.1.2
  ports:
    - name: sa-manager
      containerPort: 8080
  env:
    - name: CREATE_HOOK
      value: /hooks/sa-hook-create.sh
    - name: DELETE_HOOK
      value: /hooks/sa-hook-remove.sh
    - name: EXISTS_HOOK
      value: /hooks/sa-hook-exists.sh
    - name: API_KEY
      valueFrom:
        secretKeyRef:
          name: sa-manager-auth
          key: apiKey
    - name: LOG_LEVEL
      value: INFO
    # Only needed if a hook script calls a binary copied into /hooks (e.g. doguctl) via a bare
    # command name rather than an absolute path - extends the default Alpine PATH.
    - name: PATH
      value: "/hooks:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
  volumeMounts:
    - name: sa-manager-hooks
      mountPath: /hooks
      readOnly: true
  readinessProbe:
    httpGet:
      path: /serviceaccounts
      port: sa-manager
  livenessProbe:
    httpGet:
      path: /serviceaccounts
      port: sa-manager
```

`CREATE_HOOK`, `DELETE_HOOK` and `EXISTS_HOOK` should point at wrapper scripts (see step 3), not
directly at the application's own scripts, unless those scripts already match the sidecar's
[hook contract](../../README.md#hooks).

`/serviceaccounts` is the sidecar's unauthenticated readiness endpoint.

## 3. Add wrapper scripts for create, delete and exists

If the application's existing scripts expect a different argument shape than the sidecar provides
(e.g. positional `key=value` arguments instead of named `--key=value` flags), add a thin wrapper
per hook that translates the arguments and then calls the real script. The real script itself does
not need to change:

```bash
#!/bin/bash
set -o errexit -o nounset -o pipefail

# The consumer is always the last, unnamed argument.
CONSUMER="${@: -1}"
# Translate the remaining --key=value flags into the arguments the real script expects.

exec /hooks/create-sa.sh "${PLAIN_PARAMS[@]}" "${CONSUMER}"
```

### Idempotent create/rotate

Existing create scripts are often not idempotent: they create a new account on every call without
checking whether one already exists for the consumer. Called repeatedly (a parameter change, or a
rotation), this leaves previously created accounts behind. Add the missing check in the create
wrapper:

- Look up whether an account already exists for the consumer (e.g. via a stored identifier).
- If one exists and no rotation was requested, exit `0` without printing anything to stdout. The
  sidecar then reports `204 No Content`, and the operator leaves the existing secret untouched.
- If one exists and rotation was requested (`--behavior-rotateServiceAccountNow=true`), delete the
  existing account first, then create a new one.
- If none exists, create a new account as usual.

## 4. Store the API key in a secret

Generate the key once and keep it stable across `helm upgrade` using Helm's `lookup` function:

```yaml
{{- $existing := lookup "v1" "Secret" .Release.Namespace "sa-manager-auth" -}}
{{- $apiKey := "" -}}
{{- if and $existing $existing.data (index $existing.data "apiKey") -}}
{{- $apiKey = index $existing.data "apiKey" | b64dec -}}
{{- else -}}
{{- $apiKey = randAlphaNum 32 -}}
{{- end }}
apiVersion: v1
kind: Secret
metadata:
  name: sa-manager-auth
stringData:
  apiKey: {{ $apiKey | quote }}
```

## 5. Expose the sidecar port and allow access

Expose the sidecar's port through the application's existing Service, or a dedicated one. If the
target namespace has a default-deny NetworkPolicy, add an explicit rule allowing ingress from the
`service-account-operator` pods:

```yaml
ingress:
  - from:
      - podSelector:
          matchLabels:
            k8s.cloudogu.com/component.name: service-account-operator
    ports:
      - port: 8080
```

## 6. Register the ServiceAccountProducer

```yaml
apiVersion: k8s.cloudogu.com/v2
kind: ServiceAccountProducer
spec:
  producer: <producer-name>
  http:
    endpoint: "http://<service-name>.<namespace>.svc:8080/serviceaccounts"
    authSecret:
      name: sa-manager-auth
      key: apiKey
    params:
      attributes:
        permissions:
          type: string
          description: "..."
    return:
      username:
        description: "..."
      password:
        description: "..."
```

## 7. Test the integration

Port-forward the sidecar and call its API directly:

```bash
kubectl port-forward <pod> 8080:8080
API_KEY=$(kubectl get secret sa-manager-auth -o jsonpath='{.data.apiKey}' | base64 -d)

curl -X HEAD -H "X-CES-SA-API-KEY: $API_KEY" localhost:8080/serviceaccounts/some-consumer   # 404 if it does not exist yet
curl -X PUT  -H "X-CES-SA-API-KEY: $API_KEY" -d '{"consumer":"some-consumer","params":{}}' localhost:8080/serviceaccounts
curl -X DELETE -H "X-CES-SA-API-KEY: $API_KEY" localhost:8080/serviceaccounts/some-consumer
```

Then create a `ServiceAccountRequest` against the registered producer to exercise the full path
from the operator through the sidecar to the hook scripts.
