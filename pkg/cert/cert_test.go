package cert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

func TestWebhookCert_EnsureCert(t *testing.T) {
	certOpt := CertOption{
		CAName:          "",
		CAOrganizations: nil,
		Hosts:           nil,
		CommonName:      "",
		CertDir:         "",
		SecretInfo:      SecretInfo{},
	}
	secretClient := &FakeSecretInterface{}
	w := &WebhookCert{
		certOpt: certOpt,
		certmanager: &certManager{
			secretInfo:   certOpt.SecretInfo,
			certOpt:      certOpt,
			secretClient: secretClient,
		},
		webhookmanager: &webhookManager{
			webhooks:             nil,
			resourceClientGetter: nil,
		},
	}

	err := w.ensureCert(context.TODO())
	assert.NoError(t, err)
	assert.NotNil(t, secretClient.gotCreateSecret)

	ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Second)
	defer cancelFunc()
	err = w.EnsureCert(ctx)
	assert.NoError(t, err)
}

func TestWebhookCert_EnsureCertReady(t *testing.T) {
	certOpt := CertOption{
		CAName:          "",
		CAOrganizations: nil,
		Hosts:           nil,
		CommonName:      "",
		CertDir:         "",
		SecretInfo:      SecretInfo{},
	}
	secretClient := &FakeSecretInterface{}
	w := &WebhookCert{
		certOpt: certOpt,
		certmanager: &certManager{
			secretInfo:   certOpt.SecretInfo,
			certOpt:      certOpt,
			secretClient: secretClient,
		},
		webhookmanager: &webhookManager{
			webhooks:             nil,
			resourceClientGetter: nil,
		},
	}

	err := w.ensureCert(context.TODO())
	assert.NoError(t, err)
	assert.NotNil(t, secretClient.gotCreateSecret)

	ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Second)
	defer cancelFunc()
	err = w.EnsureCertReady(ctx)
	assert.Error(t, err)
}

func TestCertOption_getCertValidityDuration(t *testing.T) {
	type fields struct {
		CertValidityDuration time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   time.Duration
	}{
		{
			name:   "default",
			fields: fields{},
			want:   certValidityDuration,
		},
		{
			name: "with value",
			fields: fields{
				CertValidityDuration: time.Hour,
			},
			want: time.Hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CertOption{
				CertValidityDuration: tt.fields.CertValidityDuration,
			}
			if got := c.getCertValidityDuration(); got != tt.want {
				t.Errorf("getCertValidityDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertOption_getHots(t *testing.T) {
	type fields struct {
		Hosts    []string
		DNSNames []string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "no",
			fields: fields{},
			want:   []string{},
		},
		{
			name: "only hosts",
			fields: fields{
				Hosts: []string{"a", "b"},
			},
			want: []string{"a", "b"},
		},
		{
			name: "merge DNSNames",
			fields: fields{
				Hosts:    []string{"a", "b"},
				DNSNames: []string{"c", "d"},
			},
			want: []string{"a", "b", "c", "d"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CertOption{
				Hosts:    tt.fields.Hosts,
				DNSNames: tt.fields.DNSNames,
			}
			if got := c.getHots(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHots() = %v, want %v", got, tt.want)
			}
		})
	}
}

func getCertForTesting(t *testing.T) (*WebhookCert, tls.Certificate) {
	certDir, err := os.MkdirTemp(os.TempDir(), "test-webhookcert")
	assert.NoError(t, err)

	c := &certManager{
		secretInfo: SecretInfo{
			Name:      "test",
			Namespace: "",
		},
		certOpt: CertOption{
			CAName:               "ca",
			CAOrganizations:      []string{"ca"},
			Hosts:                []string{"example.com"},
			CommonName:           "test",
			CertDir:              certDir,
			CertValidityDuration: 0,
		},
	}
	secret, err := c.newSecret()
	assert.NoError(t, err)

	cert := secret.Data[c.secretInfo.getCertName()]
	err = ioutil.WriteFile(path.Join(certDir, c.secretInfo.getCertName()), cert, 0644)
	assert.NoError(t, err)
	key := secret.Data[c.secretInfo.getKeyName()]
	err = ioutil.WriteFile(path.Join(certDir, c.secretInfo.getKeyName()), key, 0644)
	assert.NoError(t, err)
	tlsCert, err := tls.X509KeyPair(cert, key)
	assert.NoError(t, err)

	return &WebhookCert{
		certOpt:        c.certOpt,
		certmanager:    c,
		webhookmanager: &webhookManager{},
		checkerClient:  nil,
	}, tlsCert
}

type mockCheckerClientInterface struct {
	resp *http.Response
	err  error
}

func (m *mockCheckerClientInterface) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

func TestWebhookCert_CheckServerStarted_success(t *testing.T) {
	c, _ := getCertForTesting(t)
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer s.Close()

	addr := s.URL[len("http://")+1:]
	err := c.CheckServerStarted(context.TODO(), addr)
	assert.NoError(t, err)

	err = c.CheckServerStartedWithTimeout(addr, time.Second)
	assert.NoError(t, err)
}

func TestWebhookCert_CheckServerStarted_failed(t *testing.T) {
	c, _ := getCertForTesting(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	s.Close()

	addr := s.URL[len("http://")+1:]
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	err := c.CheckServerStarted(ctx, addr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook server is not reachable")
	cancel()

	err = c.CheckServerStartedWithTimeout(addr, time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook server is not reachable")
}

func TestWebhookCert_CheckServerCertValid_success(t *testing.T) {
	c, tlsCert := getCertForTesting(t)
	var ps []*x509.Certificate
	for _, c := range tlsCert.Certificate {
		ps = append(ps, &x509.Certificate{Raw: c})
	}
	resp := &http.Response{
		TLS: &tls.ConnectionState{
			PeerCertificates: ps,
		},
		Body: ioutil.NopCloser(strings.NewReader("")),
	}
	m := &mockCheckerClientInterface{resp: resp}
	c.checkerClient = m

	err := c.CheckServerCertValid(context.TODO(), "127.0.0.1")
	assert.NoError(t, err)

	err = c.CheckServerCertValidWithTimeout("127.0.0.1", time.Second)
	assert.NoError(t, err)
}

func TestWebhookCert_CheckServerCertValid_error_resp_err(t *testing.T) {
	c, _ := getCertForTesting(t)
	m := &mockCheckerClientInterface{err: errors.New("resp error")}
	c.checkerClient = m

	err := c.CheckServerCertValid(context.TODO(), "127.0.0.1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connect webhook server")
	assert.Contains(t, err.Error(), "resp error")

	err = c.CheckServerCertValidWithTimeout("127.0.0.1", time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connect webhook server")
	assert.Contains(t, err.Error(), "resp error")
}

func TestWebhookCert_CheckServerCertValid_error_resp_no_tls(t *testing.T) {
	c, _ := getCertForTesting(t)
	resp := &http.Response{
		Body: ioutil.NopCloser(strings.NewReader("")),
	}
	m := &mockCheckerClientInterface{resp: resp}
	c.checkerClient = m

	err := c.CheckServerCertValid(context.TODO(), "127.0.0.1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook server does not serve TLS certificate")

	err = c.CheckServerCertValidWithTimeout("127.0.0.1", time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook server does not serve TLS certificate")
}

func TestWebhookCert_CheckServerCertValid_error_cert_value_not_match(t *testing.T) {
	c, tlsCert := getCertForTesting(t)
	var ps []*x509.Certificate
	for _, c := range tlsCert.Certificate {
		ps = append(ps, &x509.Certificate{Raw: c})
	}
	ps[len(ps)-1].Raw = append(ps[len(ps)-1].Raw, '1')
	resp := &http.Response{
		TLS: &tls.ConnectionState{
			PeerCertificates: ps,
		},
		Body: ioutil.NopCloser(strings.NewReader("")),
	}
	m := &mockCheckerClientInterface{resp: resp}
	c.checkerClient = m

	err := c.CheckServerCertValid(context.TODO(), "127.0.0.1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate chain mismatch")

	err = c.CheckServerCertValidWithTimeout("127.0.0.1", time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate chain mismatch")
}

func TestWebhookCert_WatchAndEnsureWebhooksCA(t *testing.T) {
	certOpt := CertOption{
		CAName:          "",
		CAOrganizations: nil,
		Hosts:           nil,
		CommonName:      "",
		CertDir:         "",
		SecretInfo:      SecretInfo{},
	}
	secretClient := &FakeSecretInterface{}
	object := &v1.ValidatingWebhookConfiguration{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Webhooks: []v1.ValidatingWebhook{
			{
				Name: "test1",
			},
		},
	}
	obj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	wh := &unstructured.Unstructured{Object: obj}

	watcher := &mockWatchInterface{}
	res := &mockResourceInterface{
		getData: &mockResourceInterfaceData{
			data: wh,
		},
		updateData: &mockResourceInterfaceData{
			data: wh,
		},
		w: watcher,
	}
	m := &webhookManager{
		webhooks: []WebhookInfo{
			{
				Type: ValidatingV1,
				Name: "test",
			},
		},
		resourceClientGetter: func(resource schema.GroupVersionResource) resourceInterface {
			return res
		},
	}
	w := &WebhookCert{
		certOpt: certOpt,
		certmanager: &certManager{
			secretInfo:   certOpt.SecretInfo,
			certOpt:      certOpt,
			secretClient: secretClient,
		},
		webhookmanager: m,
	}
	watchSendEvents := make(chan watch.Event, 10)
	watcher.events = watchSendEvents
	ctx, cancel := context.WithCancel(context.Background())
	go w.WatchAndEnsureWebhooksCA(ctx)

	watchSendEvents <- watch.Event{Type: watch.Added, Object: object}
	watchSendEvents <- watch.Event{Type: watch.Modified, Object: object}
	watchSendEvents <- watch.Event{Type: watch.Error}

	time.Sleep(time.Second)
	cancel()
	time.Sleep(time.Second)
}
