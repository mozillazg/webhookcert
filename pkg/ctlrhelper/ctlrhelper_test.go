package ctlrhelper

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

func TestOption_ValidateAndFillDefaultValues_skipSecretReadWrite(t *testing.T) {
	opt := Option{
		Namespace:           "default",
		ServiceName:         "webhook",
		CertDir:             "/certs",
		WebhookServerPort:   9443,
		SkipSecretReadWrite: true,
		dynamicClient:       dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	}

	if err := opt.ValidateAndFillDefaultValues(); err != nil {
		t.Fatalf("ValidateAndFillDefaultValues() error = %v", err)
	}
	if opt.SecretName != "" {
		t.Fatalf("SecretName = %q, want empty", opt.SecretName)
	}
	if opt.DnsName != "webhook.default.svc" {
		t.Fatalf("DnsName = %q, want webhook.default.svc", opt.DnsName)
	}
	if opt.kubeClient != nil {
		t.Fatal("kubeClient should not be initialized when SkipSecretReadWrite is true")
	}
}

func TestOption_ValidateAndFillDefaultValues_requireSecretName(t *testing.T) {
	opt := Option{
		Namespace:         "default",
		ServiceName:       "webhook",
		CertDir:           "/certs",
		WebhookServerPort: 9443,
		kubeClient:        kubernetesfake.NewSimpleClientset(),
		dynamicClient:     dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	}

	err := opt.ValidateAndFillDefaultValues()
	if err == nil {
		t.Fatal("ValidateAndFillDefaultValues() error = nil, want error")
	}
	if err.Error() != "the SecretName field can not be empty" {
		t.Fatalf("ValidateAndFillDefaultValues() error = %q, want SecretName error", err.Error())
	}
}
