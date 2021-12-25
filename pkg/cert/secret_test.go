package cert

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type FakeSecretInterface struct {
	gotCreateSecret *corev1.Secret
	getSecret       *corev1.Secret
	getSecretErr    error

	gotUpdateSecret *corev1.Secret
}

func (f *FakeSecretInterface) Create(ctx context.Context, secret *corev1.Secret, opts metav1.CreateOptions) (*corev1.Secret, error) {
	f.gotCreateSecret = secret
	return f.gotCreateSecret, nil
}

func (f *FakeSecretInterface) Update(ctx context.Context, secret *corev1.Secret, opts metav1.UpdateOptions) (*corev1.Secret, error) {
	f.gotUpdateSecret = secret
	return f.gotUpdateSecret, nil
}

func (f *FakeSecretInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error) {
	if f.getSecretErr != nil {
		return nil, f.getSecretErr
	}
	if f.getSecret != nil {
		return f.getSecret, nil
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "test")
}

func TestCertManager_ensureSecret(t *testing.T) {
	secretClient := &FakeSecretInterface{}
	c := certManager{
		secretInfo: SecretInfo{
			Name:          "test",
			Namespace:     "",
			dontSaveCaKey: true,
		},
		certOpt: CertOption{
			CAName:               "ca",
			CAOrganizations:      []string{"ca"},
			Hosts:                []string{"example.com"},
			CommonName:           "test",
			CertDir:              "",
			CertValidityDuration: 0,
		},
		secretClient: secretClient,
	}
	s, err := c.ensureSecret(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, s, secretClient.gotCreateSecret)
	assert.Equal(t, s.Name, "test")
	assert.GreaterOrEqual(t, len(s.Data), 3)

	assert.Nil(t, s.Data["ca.key"])
	assert.NotNil(t, s.Data["tls.key"])
	assert.NotNil(t, s.Data["tls.crt"])
	assert.NotNil(t, s.Data["ca.crt"])
}

func TestCertManager_ensureSecret_use_exist_secret(t *testing.T) {
	secretClient := &FakeSecretInterface{}
	c := certManager{
		secretInfo: SecretInfo{
			Name:      "test",
			Namespace: "",
		},
		certOpt: CertOption{
			CAName:               "ca",
			CAOrganizations:      []string{"ca"},
			Hosts:                []string{"example.com"},
			CommonName:           "test",
			CertDir:              "",
			CertValidityDuration: 0,
		},
		secretClient: secretClient,
	}
	s, err := c.ensureSecret(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, s, secretClient.gotCreateSecret)

	secretClient.getSecret = s
	secretClient.gotCreateSecret = nil
	newS, err := c.ensureSecret(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, s, newS)
}

func TestCertManager_ensureSecret_update_exist_secret(t *testing.T) {
	secretClient := &FakeSecretInterface{}
	c := certManager{
		secretInfo: SecretInfo{
			Name:      "test",
			Namespace: "",
		},
		certOpt: CertOption{
			CAName:               "ca",
			CAOrganizations:      []string{"ca"},
			Hosts:                []string{"example.com"},
			CommonName:           "test",
			CertDir:              "",
			CertValidityDuration: 0,
		},
		secretClient: secretClient,
	}
	s, err := c.ensureSecret(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, s, secretClient.gotCreateSecret)

	secretClient.getSecret = s.DeepCopy()
	secretClient.getSecret.Data = nil
	secretClient.gotCreateSecret = nil
	newS, err := c.ensureSecret(context.TODO())
	assert.NoError(t, err)
	assert.NotNil(t, newS)
	assert.NotEqual(t, s, newS)
}

func Test_certManager_certSecretIsValid(t *testing.T) {
	secretClient := &FakeSecretInterface{}
	c := certManager{
		secretInfo: SecretInfo{
			Name:      "test",
			Namespace: "",
		},
		certOpt: CertOption{
			CAName:               "ca",
			CAOrganizations:      []string{"ca"},
			Hosts:                []string{"example.com"},
			CommonName:           "test",
			CertDir:              "",
			CertValidityDuration: 0,
		},
		secretClient: secretClient,
	}
	type args struct {
		secret *corev1.Secret
		now    time.Time
	}
	validSecret, _ := c.newSecret()

	invalidSecretWithoutCa := validSecret.DeepCopy()
	delete(invalidSecretWithoutCa.Data, c.secretInfo.getCACertName())
	invalidSecretWithInvalidCa := validSecret.DeepCopy()
	invalidSecretWithInvalidCa.Data[c.secretInfo.getCACertName()] = []byte("xxx")

	invalidSecretWithoutCert := validSecret.DeepCopy()
	delete(invalidSecretWithoutCert.Data, c.secretInfo.getCertName())
	invalidSecretWithInvalidCert := validSecret.DeepCopy()
	invalidSecretWithInvalidCert.Data[c.secretInfo.getCertName()] = []byte("xxx")

	invalidSecretWithoutKey := validSecret.DeepCopy()
	delete(invalidSecretWithoutKey.Data, c.secretInfo.getKeyName())
	invalidSecretWithInvalidKey := validSecret.DeepCopy()
	invalidSecretWithInvalidKey.Data[c.secretInfo.getKeyName()] = []byte("xxx")

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				secret: validSecret,
			},
			wantErr: false,
		},
		{
			name: "invalid: expired",
			args: args{
				secret: validSecret,
				now:    time.Now().Add(certValidityDuration),
			},
			wantErr: true,
		},
		{
			name: "invalid: no ca",
			args: args{
				secret: invalidSecretWithoutCa,
			},
			wantErr: true,
		},
		{
			name: "invalid: invalid ca",
			args: args{
				secret: invalidSecretWithInvalidCa,
			},
			wantErr: true,
		},
		{
			name: "invalid: no cert",
			args: args{
				secret: invalidSecretWithoutCert,
			},
			wantErr: true,
		},
		{
			name: "invalid: invalid cert",
			args: args{
				secret: invalidSecretWithInvalidCert,
			},
			wantErr: true,
		},
		{
			name: "invalid: no key",
			args: args{
				secret: invalidSecretWithoutKey,
			},
			wantErr: true,
		},
		{
			name: "invalid: invalid key",
			args: args{
				secret: invalidSecretWithInvalidKey,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &certManager{}
			err := c.certSecretIsValid(tt.args.secret, tt.args.now)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
