# Service-Account-Producer-Sidecar einbinden

Dieses Dokument beschreibt, wie der service-account-producer-sidecar in das Helm-Chart eines Dogus
oder einer Komponente eingebunden wird, damit dieser als `ServiceAccountProducer` für den
[service-account-operator](https://github.com/cloudogu/service-account-operator) fungieren kann.
Die vollständige HTTP-API- und Hook-Referenz steht im [README](../../README.md).

## Voraussetzungen

- Bestehende Logik zum Anlegen, Löschen und Prüfen eines Service-Accounts in der Zielanwendung
  (Skripte, ein API-Client o. Ä.), verfügbar als Executables (Shell-Skripts)
- Ein Helm-Chart für das Dogu/die Komponente
- Der `service-account-operator` läuft im Ziel-Cluster

## 1. Hook-Skripte im Sidecar-Container bereitstellen

Das Sidecar-Image ist generisch und enthält kein anwendungsspezifisches Tooling. Die Skripte, die
Create/Delete/Exists implementieren, über einen Init-Container aus dem eigenen Anwendungs-Image in
ein mit dem Sidecar geteiltes Volume kopieren:

```yaml
initContainers:
  - name: sa-manager-hooks-init
    image: "<anwendungs-image>"
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

## 2. Sidecar-Container konfigurieren

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

`CREATE_HOOK`, `DELETE_HOOK` und `EXISTS_HOOK` sollten auf Wrapper-Skripte zeigen (siehe Schritt 3),
nicht direkt auf die eigenen Skripte der Anwendung, außer diese entsprechen bereits dem
[Hook-Contract](../../README.md#hooks) des Sidecars.

`/serviceaccounts` ist der unauthentifizierte Readiness-Endpunkt des Sidecars.

## 3. Wrapper-Skripte für Create, Delete und Exists ergänzen

Falls die bestehenden Skripte der Anwendung eine andere Argumentform erwarten, als der Sidecar
liefert (z. B. positionale `key=value`-Argumente statt benannter `--key=value`-Flags), pro Hook
einen dünnen Wrapper ergänzen, der die Argumente übersetzt und dann das echte Skript aufruft. Das
echte Skript selbst muss dafür nicht geändert werden:

```bash
#!/bin/bash
set -o errexit -o nounset -o pipefail

# Der Consumer ist immer das letzte, unbenannte Argument.
CONSUMER="${@: -1}"
# Die übrigen --key=value-Flags in die Argumente übersetzen, die das echte Skript erwartet.

exec /hooks/create-sa.sh "${PLAIN_PARAMS[@]}" "${CONSUMER}"
```

### Idempotentes Create/Rotate

Bestehende Create-Skripte sind oft nicht idempotent: Sie legen bei jedem Aufruf einen neuen Account an, ohne zu prüfen, ob für den Consumer schon einer existiert. 
Bei wiederholten Aufrufen (Parameter-Änderung, oder Rotation) bleiben dadurch zuvor angelegte Accounts zurück. 
Die fehlende Prüfung im Create-Wrapper ergänzen:

- Prüfen, ob für den Consumer bereits ein Account existiert (z. B. über eine gespeicherte Kennung).
- Existiert einer und wurde keine Rotation angefordert: mit `0` beenden, ohne etwas auf stdout
  auszugeben. Der Sidecar meldet dann `204 No Content`, und der Operator lässt das bestehende
  Secret unverändert.
- Existiert einer und wurde eine Rotation angefordert (`--behavior-rotateServiceAccountNow=true`):
  zuerst den bestehenden Account löschen, dann einen neuen anlegen.
- Existiert keiner: wie gewohnt einen neuen Account anlegen.

## 4. API-Key in einem Secret ablegen

Den Key einmal generieren und über `helm upgrade` hinweg stabil halten, mit Helms `lookup`-Funktion:

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

## 5. Sidecar-Port freigeben und Zugriff erlauben

Den Port des Sidecars über den bestehenden Service der Anwendung freigeben, oder über einen
eigenen. Falls der Ziel-Namespace eine Default-Deny-NetworkPolicy hat, eine explizite Regel
ergänzen, die Ingress von den `service-account-operator`-Pods erlaubt:

```yaml
ingress:
  - from:
      - podSelector:
          matchLabels:
            k8s.cloudogu.com/component.name: service-account-operator
    ports:
      - port: 8080
```

## 6. ServiceAccountProducer registrieren

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

## 7. Integration testen

Den Sidecar per Port-Forward erreichbar machen und die API direkt aufrufen:

```bash
kubectl port-forward <pod> 8080:8080
API_KEY=$(kubectl get secret sa-manager-auth -o jsonpath='{.data.apiKey}' | base64 -d)

curl -X HEAD -H "X-CES-SA-API-KEY: $API_KEY" localhost:8080/serviceaccounts/some-consumer   # 404, falls noch nicht vorhanden
curl -X PUT  -H "X-CES-SA-API-KEY: $API_KEY" -d '{"consumer":"some-consumer","params":{}}' localhost:8080/serviceaccounts
curl -X DELETE -H "X-CES-SA-API-KEY: $API_KEY" localhost:8080/serviceaccounts/some-consumer
```

Anschließend eine `ServiceAccountRequest` gegen den registrierten Producer anlegen, um den
vollständigen Pfad vom Operator über den Sidecar bis zu den Hook-Skripten durchzuspielen.
