```yaml
services:
  - hostname: lifeline
    type: go@1
    ports:
      - port: 1999
    envVariables:
      GOOGLE_JSON_CREDENTIALS: |-
        <your-service-account-credentials-json-content>
    buildFromGit: https://github.com/tikinang/lifeline
    enableSubdomainAccess: true
    minContainers: 1
    maxContainers: 1
```