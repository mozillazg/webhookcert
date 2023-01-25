# webhookcert

[![CI](https://github.com/mozillazg/webhookcert/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/mozillazg/webhookcert/actions/workflows/ci.yml)
[![Coverage Status](https://coveralls.io/repos/github/mozillazg/webhookcert/badge.svg?branch=master)](https://coveralls.io/github/mozillazg/webhookcert?branch=master)

A simple cert solution for writing Kubernetes Webhook Server.

## Feature

* Auto-create certificate for webhook server.
* Reuse certificate from secret.
* Auto patch `caBundle` for the `validatingwebhookconfigurations` and `mutatingwebhookconfigurations` resources.
* Auto restore `caBundle` when the value was invalid (for example, it was overwritten via `kubectl apply`).
* A checker to check whether the webhook server is started.
* A checker to check whether the webhook server used certificate is expired or not synced.


## Usage

```
go get github.com/mozillazg/webhookcert/pkg/cert
```

Real world example: [main.go](https://github.com/mozillazg/echo-k8s-webhook/blob/master/main.go)

## Permissions

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: <name>
  namespace: <namespace>
rules:
  - apiGroups:
      - ""
    resources:
      - secrets
    verbs:
      - create
  - apiGroups:
      - ""
    resources:
      - secrets
    resourceNames:
      - <cert_secret_name>
    verbs:
      - get
      - update

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: <name>
rules:
  - apiGroups:
      - admissionregistration.k8s.io
    resources:
      - validatingwebhookconfigurations
      - mutatingwebhookconfigurations
    resourceNames:
      - <validating_name>
      - <mutating_name>
    verbs:
      - get
      - update
  - apiGroups:
      - admissionregistration.k8s.io
    resources:
      - validatingwebhookconfigurations
      - mutatingwebhookconfigurations
    verbs:
      - watch
```
