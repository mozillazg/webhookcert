package cert

import (
	"context"
	"testing"

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
			saveCaKey: true,
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
