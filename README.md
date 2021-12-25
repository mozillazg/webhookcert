# webhookcert

[![CI](https://github.com/mozillazg/webhookcert/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/mozillazg/webhookcert/actions/workflows/ci.yml)
[![Coverage Status](https://coveralls.io/repos/github/mozillazg/webhookcert/badge.svg?branch=master)](https://coveralls.io/github/mozillazg/webhookcert?branch=master)

A simple cert solution for writing Kubernetes Webhook Server.

## Usage

```
go get github.com/mozillazg/webhookcert/pkg/cert
```

```go
import "github.com/mozillazg/webhookcert/pkg/cert"

	kubeclient := kubernetes.NewForConfigOrDie(config.GetConfigOrDie())
	dyclient := dynamic.NewForConfigOrDie(config.GetConfigOrDie())
	webhooks := []cert.WebhookInfo{
		{
			Type: cert.ValidatingV1,
			Name: validatingName,
		},
	}
	webhookcert := cert.NewWebhookCert(cert.CertOption{
		CAName:          caName,
		CAOrganizations: []string{caOrganization},
		Hosts:        []string{dnsName},
		CommonName:      dnsName,
		CertDir:         certDir,
		SecretInfo: cert.SecretInfo{
			Name:      secretName,
			Namespace: namespace,
		},
	}, webhooks, kubeclient, dyclient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute * 3)
	defer cancel()
	err := webhookcert.EnsureCertReady(ctx)
	if err != nil {
		klog.Fatalf("ensure cert ready failed: %+v", err)
	}
```

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
    resourceNames:
      - <validating_name>
    verbs:
      - get
      - update
  - apiGroups:
      - admissionregistration.k8s.io
    resources:
      - validatingwebhookconfigurations
    verbs:
      - watch
```
