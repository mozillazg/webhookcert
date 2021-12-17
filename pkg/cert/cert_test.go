package cert

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebhookCert_ensureCert(t *testing.T) {
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
			want: []string{"c", "d", "a", "b"},
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
