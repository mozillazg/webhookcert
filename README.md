# webhookcert

[![CI](https://github.com/mozillazg/webhookcert/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/mozillazg/webhookcert/actions/workflows/ci.yml)
[![Coverage Status](https://coveralls.io/repos/github/mozillazg/webhookcert/badge.svg?branch=master)](https://coveralls.io/github/mozillazg/webhookcert?branch=master)

A simple cert solution for writing Kubernetes Webhook Server.

## Feature

* Auto-create certificate for webhook server.
* Reuse certificate from secret.
* Auto patch `caBundle` for the `validatingwebhookconfigurations` and `mutatingwebhookconfigurations` resources.
* Auto restore `caBundle` when the value is updated with invalid value (for example, it was overwritten via `kubectl replace`).
* A checker to check whether the webhook server is started.
* A checker to check whether the webhook server used certificate is expired or not synced.


## Usage

```go
package main

import (
	"github.com/mozillazg/webhookcert/pkg/cert"
	"github.com/mozillazg/webhookcert/pkg/ctlrhelper"
	// ...
)

var (
	namespace = "test"
	secretName = "webhook-test-server-cert"
	serviceName = "webhook-test-server"
	port = 9443
	certDir = "/certs"
	webhookConfigName = "webhook-test-server-config"
)

func main() {
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Port:                   port,
		CertDir:                certDir,
		// ...
	})

	ctx := signals.SetupSignalHandler()
	errC := make(chan error, 2)

	setupWebhook(ctx, mgr, errC)

	go func() {
		if err := mgr.Start(ctx); err != nil {
			errC <- err
		}
	}()

	select {
	case <-errC:
		os.Exit(1)
	case <-ctx.Done():
	}
}

func setupWebhook(ctx context.Context, mgr manager.Manager, errC chan<- error) {
	opt := ctlrhelper.Option{
		Namespace:   namespace,
		SecretName:  secretName,
		ServiceName: serviceName,
		CertDir:     certDir,
		Webhooks: []cert.WebhookInfo{
			{
				Type: cert.ValidatingV1,
				Name: webhookConfigName,
			},
		},
		WebhookServerPort: port,
	}

	h, err := ctlrhelper.NewNewWebhookHelper(opt)
	if err != nil {
		errC <- err
		return
	}

	handler1 := // ...
	handler2 := // ...

	h.Setup(ctx, mgr, func(s *webhook.Server) {
		s.Register("/webhook/path/1", &webhook.Admission{Handler: handler1})
		s.Register("/webhook/path/1", &webhook.Admission{Handler: handler2})
	}, errC)
}

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

## Healthz and Readyz

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 9090
  initialDelaySeconds: 5
  timeoutSeconds: 4
readinessProbe:
  httpGet:
    path: /readyz
    port: 9090
  initialDelaySeconds: 5
  timeoutSeconds: 4
startupProbe:
  httpGet:
    path: /readyz
    port: 9090
  failureThreshold: 24
  periodSeconds: 10
  timeoutSeconds: 4
```
