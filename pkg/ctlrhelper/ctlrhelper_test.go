package ctlrhelper

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/mozillazg/webhookcert/pkg/cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func writeTempKubeconfig(t *testing.T) string {
	t.Helper()
	content := []byte(`
apiVersion: v1
clusters:
- cluster:
    server: https://example.com
  name: dummy
contexts:
- context:
    cluster: dummy
    user: dummy
  name: dummy
current-context: dummy
users:
- name: dummy
  user:
    token: dummy
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "kubeconfig")
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func TestOption_ValidateAndFillDefaultValues_required(t *testing.T) {
	tests := []struct {
		name string
		opt  Option
	}{
		{name: "missing secret", opt: Option{}},
		{name: "missing namespace", opt: Option{SecretName: "s"}},
		{name: "missing service", opt: Option{SecretName: "s", Namespace: "ns"}},
		{name: "missing cert dir", opt: Option{SecretName: "s", Namespace: "ns", ServiceName: "svc"}},
		{name: "missing port", opt: Option{SecretName: "s", Namespace: "ns", ServiceName: "svc", CertDir: "/tmp"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opt.ValidateAndFillDefaultValues()
			assert.Error(t, err)
		})
	}
}

func TestOption_ValidateAndFillDefaultValues_defaults(t *testing.T) {
	kubeconfig := writeTempKubeconfig(t)
	t.Setenv("KUBECONFIG", kubeconfig)

	kc := k8sfake.NewSimpleClientset()
	dc := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	opt := Option{
		SecretName:                   "s",
		Namespace:                    "ns",
		ServiceName:                  "svc",
		CertDir:                      "/tmp",
		WebhookServerPort:            9443,
		kubeClient:                   kc,
		dynamicClient:                dc,
		TimeoutForCheckServerCert:    0,
		TimeoutForCheckServerStarted: 0,
		TimeoutForEnsureCertReady:    0,
	}

	err := opt.ValidateAndFillDefaultValues()
	require.NoError(t, err)
	assert.Equal(t, "svc.ns.svc", opt.DnsName)
	assert.Equal(t, []string{"svc"}, opt.Organizations)
	assert.Equal(t, []string{"svc.ns.svc"}, opt.Hosts)
	assert.Equal(t, defaultTimeoutForEnsureCertReady, opt.TimeoutForEnsureCertReady)
	assert.Equal(t, defaultTimeoutForCheckServerCert, opt.TimeoutForCheckServerCert)
	assert.Equal(t, defaultTimeoutForCheckServerStarted, opt.TimeoutForCheckServerStarted)
	assert.Same(t, kc, opt.kubeClient)
	assert.Same(t, dc, opt.dynamicClient)
}

func TestOption_ValidateAndFillDefaultValues_keepProvided(t *testing.T) {
	kubeconfig := writeTempKubeconfig(t)
	t.Setenv("KUBECONFIG", kubeconfig)

	opt := Option{
		SecretName:                   "s",
		Namespace:                    "ns",
		ServiceName:                  "svc",
		CertDir:                      "/tmp",
		WebhookServerPort:            9443,
		DnsName:                      "custom.svc",
		Organizations:                []string{"org"},
		Hosts:                        []string{"h1", "h2"},
		TimeoutForEnsureCertReady:    time.Second,
		TimeoutForCheckServerCert:    time.Second * 2,
		TimeoutForCheckServerStarted: time.Second * 3,
	}

	err := opt.ValidateAndFillDefaultValues()
	require.NoError(t, err)
	assert.Equal(t, "custom.svc", opt.DnsName)
	assert.Equal(t, []string{"org"}, opt.Organizations)
	assert.Equal(t, []string{"h1", "h2"}, opt.Hosts)
	assert.Equal(t, time.Second, opt.TimeoutForEnsureCertReady)
	assert.Equal(t, time.Second*2, opt.TimeoutForCheckServerCert)
	assert.Equal(t, time.Second*3, opt.TimeoutForCheckServerStarted)
	assert.NotNil(t, opt.kubeClient)
	assert.NotNil(t, opt.dynamicClient)
}

func TestNewNewWebhookHelper(t *testing.T) {
	kubeconfig := writeTempKubeconfig(t)
	t.Setenv("KUBECONFIG", kubeconfig)

	opt := Option{
		SecretName:        "s",
		Namespace:         "ns",
		ServiceName:       "svc",
		CertDir:           "/tmp",
		WebhookServerPort: 9443,
		kubeClient:        k8sfake.NewSimpleClientset(),
		dynamicClient:     dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	}
	h, err := NewNewWebhookHelper(opt)
	require.NoError(t, err)
	assert.NotNil(t, h)
	assert.NotNil(t, h.ensureCertFinished)
	assert.NotNil(t, h.webhookReady)
}

func TestNewNewWebhookHelper_error(t *testing.T) {
	_, err := NewNewWebhookHelper(Option{})
	assert.Error(t, err)
}

func TestNewNewWebhookHelperOrDie_exit(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		NewNewWebhookHelperOrDie(Option{})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestNewNewWebhookHelperOrDie_exit", "--", "helper")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && !exitErr.Success() {
		return
	}
	t.Fatalf("process should have exited with non-zero code")
}

func TestNewNewWebhookHelperOrDie_success(t *testing.T) {
	kubeconfig := writeTempKubeconfig(t)
	t.Setenv("KUBECONFIG", kubeconfig)
	opt := Option{
		SecretName:        "s",
		Namespace:         "ns",
		ServiceName:       "svc",
		CertDir:           "/tmp",
		WebhookServerPort: 9443,
		kubeClient:        k8sfake.NewSimpleClientset(),
		dynamicClient:     dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	}
	h := NewNewWebhookHelperOrDie(opt)
	assert.NotNil(t, h)
}

func TestNewWebhookHelperOrDie_defaults(t *testing.T) {
	kubeconfig := writeTempKubeconfig(t)
	t.Setenv("KUBECONFIG", kubeconfig)
	opt := Option{
		SecretName:        "s",
		Namespace:         "ns",
		ServiceName:       "svc",
		CertDir:           "/tmp",
		WebhookServerPort: 9443,
	}

	h := NewWebhookHelperOrDie(opt)
	assert.Equal(t, "svc.ns.svc", h.opt.DnsName)
	assert.NotNil(t, h.opt.kubeClient)
	assert.NotNil(t, h.opt.dynamicClient)
}

func TestWebhookHelper_ensureCertReady_paths(t *testing.T) {
	base := &cert.WebhookCert{}

	t.Run("ensure error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(cert.NewWebhookCert, func(_ cert.CertOption, _ []cert.WebhookInfo, _ kubernetes.Interface, _ dynamic.Interface) *cert.WebhookCert {
			return base
		})
		errC := make(chan error, 1)
		w := &WebhookHelper{
			opt:                Option{TimeoutForEnsureCertReady: time.Second},
			ensureCertFinished: make(chan struct{}),
		}
		patches.ApplyMethod(reflect.TypeOf(base), "EnsureCertReady", func(_ *cert.WebhookCert, _ context.Context) error {
			return errors.New("ensure fail")
		})
		patches.ApplyMethod(reflect.TypeOf(base), "WatchAndEnsureWebhooksCA", func(_ *cert.WebhookCert, _ context.Context) error {
			return nil
		})

		w.ensureCertReady(context.Background(), errC)
		assert.EqualError(t, <-errC, "ensure fail")
		select {
		case <-w.ensureCertFinished:
			assert.Fail(t, "ensureCertFinished should not be closed on error")
		default:
		}
	})

	t.Run("watch error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(cert.NewWebhookCert, func(_ cert.CertOption, _ []cert.WebhookInfo, _ kubernetes.Interface, _ dynamic.Interface) *cert.WebhookCert {
			return base
		})
		errC := make(chan error, 1)
		w := &WebhookHelper{
			opt:                Option{TimeoutForEnsureCertReady: time.Second},
			ensureCertFinished: make(chan struct{}),
		}
		patches.ApplyMethod(reflect.TypeOf(base), "EnsureCertReady", func(_ *cert.WebhookCert, _ context.Context) error {
			return nil
		})
		patches.ApplyMethod(reflect.TypeOf(base), "WatchAndEnsureWebhooksCA", func(_ *cert.WebhookCert, _ context.Context) error {
			return errors.New("watch fail")
		})

		w.ensureCertReady(context.Background(), errC)
		select {
		case err := <-errC:
			assert.EqualError(t, err, "watch fail")
		case <-time.After(time.Second):
			assert.Fail(t, "expected error from watch")
		}
		select {
		case <-w.ensureCertFinished:
		case <-time.After(time.Second):
			assert.Fail(t, "ensureCertFinished should be closed")
		}
	})

	t.Run("success", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(cert.NewWebhookCert, func(_ cert.CertOption, _ []cert.WebhookInfo, _ kubernetes.Interface, _ dynamic.Interface) *cert.WebhookCert {
			return base
		})
		errC := make(chan error, 1)
		w := &WebhookHelper{
			opt:                Option{TimeoutForEnsureCertReady: time.Second},
			ensureCertFinished: make(chan struct{}),
		}
		patches.ApplyMethod(reflect.TypeOf(base), "EnsureCertReady", func(_ *cert.WebhookCert, _ context.Context) error {
			return nil
		})
		patches.ApplyMethod(reflect.TypeOf(base), "WatchAndEnsureWebhooksCA", func(_ *cert.WebhookCert, _ context.Context) error {
			return nil
		})

		w.ensureCertReady(context.Background(), errC)
		select {
		case err := <-errC:
			assert.NoError(t, err)
		case <-time.After(200 * time.Millisecond):
			// no error expected
		}
		select {
		case <-w.ensureCertFinished:
		case <-time.After(time.Second):
			assert.Fail(t, "ensureCertFinished should be closed")
		}
	})
}
