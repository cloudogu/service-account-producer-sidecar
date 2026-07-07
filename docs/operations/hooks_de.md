# Konfiguration und Hook-Contract

Dieses Dokument beschreibt die Umgebungsvariablen des Sidecars und den Contract, den seine
Hook-Skripte einhalten müssen. 

## Konfiguration

Der Sidecar wird über Umgebungsvariablen konfiguriert:

| Variable       | Pflicht | Default | Beschreibung                                                                      |
|----------------|---------|---------|-----------------------------------------------------------------------------------|
| `API_KEY`      | ja      | -       | Statischer Key, geprüft gegen den Header `X-CES-SA-API-KEY`                       |
| `CREATE_HOOK`  | ja      | -       | Pfad zum Executable, das bei `PUT /serviceaccounts` aufgerufen wird               |
| `DELETE_HOOK`  | ja      | -       | Pfad zum Executable, das bei `DELETE /serviceaccounts/{consumer}` aufgerufen wird |
| `EXISTS_HOOK`  | ja      | -       | Pfad zum Executable, das bei `HEAD /serviceaccounts/{consumer}` aufgerufen wird   |
| `ADDR`         | nein    | `:8080` | HTTP-Listen-Adresse                                                               |
| `LOG_LEVEL`    | nein    | `INFO`  | `DEBUG`, `INFO`, `WARN` oder `ERROR`                                              |
| `HOOK_TIMEOUT` | nein    | `30s`   | Maximale Laufzeit eines einzelnen Hook-Aufrufs                                    |

## Hooks

Ein Hook ist eine beliebige Executable-Datei. Sie wird aufgerufen als:

```
<hook> [--param=value...] <consumer>
```

- `--param=value...` sind die Einträge des `params`-Objekts aus dem Request, übergeben als
  benannte Flags (z. B. `--fullAccessRepository=foo --permissions=nx-readonly`). Der `EXISTS_HOOK`
  erhält nie Params, nur den Consumer.
- Der `CREATE_HOOK` erhält zusätzlich das `behaviorParams`-Objekt des Requests als
  `--behavior-key=value`-Flags, angehängt nach den Domain-Param-Flags, z. B.
  `--permissions=nx-readonly --behavior-rotateServiceAccountNow=true`.
- `<consumer>` ist immer das letzte, unbenannte Argument.

### `CREATE_HOOK` / `DELETE_HOOK`

Exit-Code `0` bedeutet Erfolg; jeder andere Exit-Code wird dem Aufrufer als `500` gemeldet (stderr
ist in der Fehlermeldung enthalten). Beim Create-Hook wird jede auf stdout ausgegebene Zeile der
Form `key: value` zu einem Eintrag in der Credentials-Map, die als rohes JSON im Response-Body
zurückgegeben wird. Alle anderen stdout-Ausgaben werden ignoriert, Hooks dürfen also frei loggen.

### `EXISTS_HOOK`

Exit-Code `0` bedeutet, der Service-Account existiert (`HEAD` antwortet `200`), Exit-Code `1`
bedeutet, er existiert nicht (`HEAD` antwortet `404`). Jeder andere Exit-Code, oder ein
Ausführungsfehler des Hooks selbst, wird als `500` gemeldet.
