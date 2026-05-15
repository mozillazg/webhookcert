package ctlrhelper

import (
	"testing"
	"time"

	"github.com/mozillazg/webhookcert/pkg/cert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
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
		DynamicClient:       dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
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
	if opt.KubeClient != nil {
		t.Fatal("KubeClient should not be initialized when SkipSecretReadWrite is true")
	}
	if opt.HealthzCheckName != defaultHealthzCheckName {
		t.Fatalf("HealthzCheckName = %q, want %q", opt.HealthzCheckName, defaultHealthzCheckName)
	}
	if opt.ReadyzCheckName != defaultReadyzCheckName {
		t.Fatalf("ReadyzCheckName = %q, want %q", opt.ReadyzCheckName, defaultReadyzCheckName)
	}
}

func TestOption_ValidateAndFillDefaultValues_requireSecretName(t *testing.T) {
	opt := Option{
		Namespace:         "default",
		ServiceName:       "webhook",
		CertDir:           "/certs",
		WebhookServerPort: 9443,
		KubeClient:        kubernetesfake.NewSimpleClientset(),
		DynamicClient:     dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	}

	err := opt.ValidateAndFillDefaultValues()
	if err == nil {
		t.Fatal("ValidateAndFillDefaultValues() error = nil, want error")
	}
	if err.Error() != "the SecretName field can not be empty" {
		t.Fatalf("ValidateAndFillDefaultValues() error = %q, want SecretName error", err.Error())
	}
}

func TestOption_ValidateAndFillDefaultValues_customCheckNames(t *testing.T) {
	opt := Option{
		Namespace:           "default",
		ServiceName:         "webhook",
		CertDir:             "/certs",
		WebhookServerPort:   9443,
		SkipSecretReadWrite: true,
		HealthzCheckName:    "webhook-cert",
		ReadyzCheckName:     "webhook-server",
		DynamicClient:       dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	}

	if err := opt.ValidateAndFillDefaultValues(); err != nil {
		t.Fatalf("ValidateAndFillDefaultValues() error = %v", err)
	}
	if opt.HealthzCheckName != "webhook-cert" {
		t.Fatalf("HealthzCheckName = %q, want webhook-cert", opt.HealthzCheckName)
	}
	if opt.ReadyzCheckName != "webhook-server" {
		t.Fatalf("ReadyzCheckName = %q, want webhook-server", opt.ReadyzCheckName)
	}
}

func TestWebhookHelper_ReadyChannels(t *testing.T) {
	opt := Option{
		Namespace:           "default",
		ServiceName:         "webhook",
		CertDir:             "/certs",
		WebhookServerPort:   9443,
		SkipSecretReadWrite: true,
		DynamicClient:       dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	}
	h, err := NewNewWebhookHelper(opt)
	if err != nil {
		t.Fatalf("NewNewWebhookHelper() error = %v", err)
	}

	if h.EnsureCertFinished() == nil {
		t.Fatal("EnsureCertFinished() = nil")
	}
	if h.WebhookReady() == nil {
		t.Fatal("WebhookReady() = nil")
	}
}

func TestNewWebhookHelperOrDie_initializesReadyChannelsAndCheckNames(t *testing.T) {
	opt := Option{
		Namespace:           "default",
		ServiceName:         "webhook",
		CertDir:             "/certs",
		WebhookServerPort:   9443,
		SkipSecretReadWrite: true,
		DynamicClient:       dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	}
	h := NewWebhookHelperOrDie(opt)

	if h.EnsureCertFinished() == nil {
		t.Fatal("EnsureCertFinished() = nil")
	}
	if h.WebhookReady() == nil {
		t.Fatal("WebhookReady() = nil")
	}
	if h.opt.HealthzCheckName != defaultHealthzCheckName {
		t.Fatalf("HealthzCheckName = %q, want %q", h.opt.HealthzCheckName, defaultHealthzCheckName)
	}
	if h.opt.ReadyzCheckName != defaultReadyzCheckName {
		t.Fatalf("ReadyzCheckName = %q, want %q", h.opt.ReadyzCheckName, defaultReadyzCheckName)
	}
	if h.opt.TimeoutForEnsureCertReady != defaultTimeoutForEnsureCertReady {
		t.Fatalf("TimeoutForEnsureCertReady = %s, want %s", h.opt.TimeoutForEnsureCertReady, defaultTimeoutForEnsureCertReady)
	}
	if h.opt.TimeoutForCheckServerStarted != defaultTimeoutForCheckServerStarted {
		t.Fatalf("TimeoutForCheckServerStarted = %s, want %s", h.opt.TimeoutForCheckServerStarted, defaultTimeoutForCheckServerStarted)
	}
	if h.opt.TimeoutForCheckServerCert != defaultTimeoutForCheckServerCert {
		t.Fatalf("TimeoutForCheckServerCert = %s, want %s", h.opt.TimeoutForCheckServerCert, defaultTimeoutForCheckServerCert)
	}
}

func TestWebhookHelper_markWebhookReadyWhenStarted_error(t *testing.T) {
	oldBackoff := defaultBackoffForCheckServerStarted
	defaultBackoffForCheckServerStarted = wait.Backoff{
		Steps:    1,
		Duration: time.Millisecond,
	}
	defer func() {
		defaultBackoffForCheckServerStarted = oldBackoff
	}()

	h := &WebhookHelper{
		opt: Option{
			TimeoutForCheckServerStarted: time.Millisecond,
		},
		webhookReady: make(chan struct{}),
	}
	errC := make(chan error, 1)

	h.markWebhookReadyWhenStarted(&cert.WebhookCert{}, "127.0.0.1:1", errC)

	select {
	case err := <-errC:
		if err == nil {
			t.Fatal("got nil error")
		}
	default:
		t.Fatal("expected startup error")
	}

	select {
	case <-h.WebhookReady():
		t.Fatal("WebhookReady() was closed after startup failure")
	default:
	}
}
