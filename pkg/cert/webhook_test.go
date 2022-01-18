package cert

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

var (
	caPemForTestA = `
-----BEGIN CERTIFICATE-----
MIIDITCCAgmgAwIBAgIRALa+7m4Hx1iyV1LHKW08luYwDQYJKoZIhvcNAQELBQAw
HjENMAsGA1UEChMEdGVzdDENMAsGA1UEAxMEdGVzdDAeFw0yMTEwMjQwOTE5MDla
Fw0yMTEwMjQwOTIwMDlaMB4xDTALBgNVBAoTBHRlc3QxDTALBgNVBAMTBHRlc3Qw
ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCfQMs82gOc131cx3f7WQ+w
9EjrfvhAH8aMALVbMU5iBPsNEEspdy5KOXLtWs0UA5H5b2MMcxDxItn27HP3hm9V
aKjvHgYkUyfR55I+76aGVqTOEnKV+KqS47bIRRKCwPMof9acBjo8BqjveN9uj9kb
LNQlOzqca9i1COXxrhlJopPAwzXxtdfb6tnkJMucM4DvCKAejNZx/XDxeJlZlnmU
hS2nSxc56uc3R+rl/0gSsktpX++k724H0h8aZPNxln4FswGQmi0qK8fVfEn6vX1C
3K9ry6tQ0y05vhZfzdpdbJ6cKXBSL8B5Oe7D89WR3+bETjfldIDtyrDID+jw+CRt
AgMBAAGjWjBYMA4GA1UdDwEB/wQEAwICpDAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud
DgQWBBR9v97Cv22Ji84LPyZx84wDPkQmljAWBgNVHREEDzANggtleGFtcGxlLmNv
bTANBgkqhkiG9w0BAQsFAAOCAQEAnmhDVY7kQnpcXL2y34iTQYUSG1OF/0zBrm5H
3mAWuvMruFFGtZfqzzqtNh8kPEmMuctUhRurN09HFmjcnz0RE80J9nubqlFHlRQ3
Wta3jwM0Jb+Ij2EspVDj1QI4otAPINVL2jXqX4/hCW0hcnAb+blBL7XbkV/pDNJa
V3wa0XeaNo0oojOZyQM2XLoSFeGVFA7x9mZIsIQAuLvRz6IReQYHqKI3MxzkDSnJ
SQzSEpuOtTQtYRRXXz6lJ0vhU/T3rYXRdZ2fGdw8l+MpmbfB4wks3aF5TmMQYJ0I
YMKSSC0azBfzS8i6s3Is11VgTqSO4N0sCNxrf14uk3fbLeiXiA==
-----END CERTIFICATE-----
`
	caPemForTestB = `
-----BEGIN CERTIFICATE-----
MIIDJzCCAg+gAwIBAgIQPgTIfUFASngyfJd27ebNgzANBgkqhkiG9w0BAQsFADAf
MRAwDgYDVQQKEwdhYmMgaW5jMQswCQYDVQQDEwJjYTAeFw0yMTEwMjQxMjE0MTBa
Fw0yMjAxMjIxMjE0MTBaMCgxEDAOBgNVBAoTB2FiYyBpbmMxFDASBgNVBAMTC2Ns
aWVudC1uYW1lMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0q+E02Ht
+xgCTwFBu+zH7aY0vB1Orda3pfTMb0uSSevxQ79aB2Oyfz/ZdWIOIFDEFryOwha1
6EH24znCgW7mW4wmKKRfEUl/L9sE+atmiogxZXpBy5CxQnQQJ6oP7FMfwLIiBUE4
9LdQtDrolyjO94S3QQJ/EdS8xpjSdvRyZS323W4A4L+YRkyOD7v8M4kZsYbba3Qf
rW44TX2L/uHRznVHiYdt4JkmcfHRXk4dO/VmR8COvc64tfqtRpiXvQVGjZPrQaDm
BSEhPF8/zQTdCwF7EU7qlU4bZpzxlbGPwSR5eiVqu8ORRzNSdiHRZ1R4mdl18tuj
flIlVg+Dmdp5HQIDAQABo1YwVDAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYI
KwYBBQUHAwIwDAYDVR0TAQH/BAIwADAfBgNVHSMEGDAWgBT9WZbv325bWdSR9wWd
mqEv6wia8jANBgkqhkiG9w0BAQsFAAOCAQEAHkUQGMBFWru91FpkejLFT4uXsiL+
0zqenVA6LOQXFD0hxvKrn/Fy4tJjZaiwB7e7mWFR2u5B7Y8UMwV67+KQypqGBS0E
lWiYsZA7EwCE7z7RasMXMB1jgj/I9fbHWVOhwSuFW+hYG+3dkKLW7zvdDdlvcVlX
FUpSaZBLg80t7yPHJC3RJMxoqdIeMq5xWzdR6TE6/SH9Pp8iRoHFB3hsN0eAtRLW
S3J4qo6C+00W5FTQLaaewomhgphdqIzzE35Le8P5yEuY1FKgjR+2ZzvClY59CiLm
u6q4FrpUnbTjgFcXm5hHHMvA/4rT6+//X5VM5qZ+0dxcYYgDTOLJ53kq2g==
-----END CERTIFICATE-----
`
)

func Test_injectCertToWebhook(t *testing.T) {
	type args struct {
		object *v1.ValidatingWebhookConfiguration
		caPem  []byte
	}
	tests := []struct {
		name        string
		args        args
		wantChanged bool
		wantErr     bool
		wantCaPems  []string
	}{
		{
			name: "wehooks changed",
			args: args{
				object: &v1.ValidatingWebhookConfiguration{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Webhooks: []v1.ValidatingWebhook{
						{
							Name: "test1",
						},
						{
							Name: "test2",
						},
					},
				},
				caPem: []byte(caPemForTestA),
			},
			wantChanged: true,
			wantErr:     false,
			wantCaPems:  []string{caPemForTestA, caPemForTestA},
		},
		{
			name: "wehooks merge certs",
			args: args{
				object: &v1.ValidatingWebhookConfiguration{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Webhooks: []v1.ValidatingWebhook{
						{
							Name: "test1",
						},
						{
							Name: "test2",
							ClientConfig: v1.WebhookClientConfig{
								CABundle: []byte(caPemForTestB),
							},
						},
					},
				},
				caPem: []byte(caPemForTestA),
			},
			wantChanged: true,
			wantErr:     false,
			wantCaPems: []string{caPemForTestA,
				fmt.Sprintf("%s\n%s", strings.TrimSpace(caPemForTestA), strings.TrimSpace(caPemForTestB)),
			},
		},
		{
			name: "wehooks merge certs ignore invalid cert",
			args: args{
				object: &v1.ValidatingWebhookConfiguration{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Webhooks: []v1.ValidatingWebhook{
						{
							Name: "test1",
							ClientConfig: v1.WebhookClientConfig{
								CABundle: []byte("Cg=="),
							},
						},
						{
							Name: "test2",
							ClientConfig: v1.WebhookClientConfig{
								CABundle: []byte(caPemForTestB),
							},
						},
					},
				},
				caPem: []byte(caPemForTestA),
			},
			wantChanged: true,
			wantErr:     false,
			wantCaPems: []string{caPemForTestA,
				fmt.Sprintf("%s\n%s", strings.TrimSpace(caPemForTestA), strings.TrimSpace(caPemForTestB)),
			},
		},
		{
			name: "wehooks merge certs: dont merge same cert",
			args: args{
				object: &v1.ValidatingWebhookConfiguration{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Webhooks: []v1.ValidatingWebhook{
						{
							Name: "test1",
							ClientConfig: v1.WebhookClientConfig{
								CABundle: []byte(caPemForTestA),
							},
						},
						{
							Name: "test2",
							ClientConfig: v1.WebhookClientConfig{
								CABundle: []byte(caPemForTestB),
							},
						},
					},
				},
				caPem: []byte(caPemForTestA),
			},
			wantChanged: true,
			wantErr:     false,
			wantCaPems: []string{caPemForTestA,
				fmt.Sprintf("%s\n%s", strings.TrimSpace(caPemForTestA), strings.TrimSpace(caPemForTestB)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tt.args.object)
			wh := &unstructured.Unstructured{Object: obj}
			assert.NoError(t, err)
			changed, err := injectCertToWebhook(wh, tt.args.caPem)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantChanged, changed)

			finalObj := &v1.ValidatingWebhookConfiguration{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(wh.Object, finalObj)
			assert.NoError(t, err)
			caPems := []string{}
			for _, w := range finalObj.Webhooks {
				caPems = append(caPems, string(w.ClientConfig.CABundle))
			}
			if len(tt.wantCaPems) != 0 {
				assert.Len(t, caPems, len(finalObj.Webhooks))
				for i, p := range caPems {
					assert.Equal(t, strings.TrimSpace(tt.wantCaPems[i]), strings.TrimSpace(p))
				}
			}
		})
	}
}

type mockResourceInterfaceData struct {
	inputName string
	inputData *unstructured.Unstructured
	data      *unstructured.Unstructured
	err       error
	callCount int
}

type mockWatchInterface struct {
	callStop int
	events   chan watch.Event
}

func (m *mockWatchInterface) Stop() {
	m.callStop++
}

func (m *mockWatchInterface) ResultChan() <-chan watch.Event {
	return m.events
}

type mockResourceInterface struct {
	getData    *mockResourceInterfaceData
	updateData *mockResourceInterfaceData

	w  *mockWatchInterface
	we error
}

func (m *mockResourceInterface) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	m.getData.inputName = name
	m.getData.callCount++
	return m.getData.data.DeepCopy(), m.getData.err
}

func (m *mockResourceInterface) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	m.updateData.inputData = obj
	m.updateData.callCount++
	return m.updateData.data.DeepCopy(), m.updateData.err
}

func (m *mockResourceInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return m.w, m.we
}

func Test_webhookManager_ensureCA_success(t *testing.T) {
	object := &v1.ValidatingWebhookConfiguration{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Webhooks: []v1.ValidatingWebhook{
			{
				Name: "test1",
			},
			{
				Name: "test2",
			},
		},
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	wh := &unstructured.Unstructured{Object: obj}
	assert.NoError(t, err)
	res := &mockResourceInterface{
		getData: &mockResourceInterfaceData{
			data: wh,
			err:  nil,
		},
		updateData: &mockResourceInterfaceData{
			data: wh,
			err:  nil,
		},
	}
	m := webhookManager{
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
	err = m.ensureCA(context.TODO(), []byte(caPemForTestA))
	assert.NoError(t, err)

	assert.Equal(t, "test", res.getData.inputName)
	assert.Equal(t, 1, res.getData.callCount)
	assert.Equal(t, 1, res.updateData.callCount)
	assert.NotNil(t, res.updateData.inputData)
}

func Test_webhookManager_ensureCA_skip_get_not_found(t *testing.T) {
	res := &mockResourceInterface{
		getData: &mockResourceInterfaceData{
			err: apierrors.NewNotFound(schema.GroupResource{}, "test"),
		},
		updateData: &mockResourceInterfaceData{},
	}
	m := webhookManager{
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
	err := m.ensureCA(context.TODO(), []byte(caPemForTestA))
	assert.NoError(t, err)

	assert.Equal(t, "test", res.getData.inputName)
	assert.Equal(t, 1, res.getData.callCount)
	assert.Equal(t, 0, res.updateData.callCount)
	assert.Nil(t, res.updateData.inputData)
}

func Test_webhookManager_ensureCA_skip_update_not_found(t *testing.T) {
	object := &v1.ValidatingWebhookConfiguration{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Webhooks: []v1.ValidatingWebhook{
			{
				Name: "test1",
			},
			{
				Name: "test2",
			},
		},
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	wh := &unstructured.Unstructured{Object: obj}
	res := &mockResourceInterface{
		getData: &mockResourceInterfaceData{
			data: wh,
		},
		updateData: &mockResourceInterfaceData{
			err: apierrors.NewNotFound(schema.GroupResource{}, "test"),
		},
	}
	m := webhookManager{
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
	err = m.ensureCA(context.TODO(), []byte(caPemForTestA))
	assert.NoError(t, err)

	assert.Equal(t, "test", res.getData.inputName)
	assert.Equal(t, 1, res.getData.callCount)
	assert.Equal(t, 1, res.updateData.callCount)
	assert.NotNil(t, res.updateData.inputData)
}

func Test_webhookManager_ensureCA_retry_when_get_error(t *testing.T) {
	res := &mockResourceInterface{
		getData: &mockResourceInterfaceData{
			err: apierrors.NewInternalError(errors.New("error from get")),
		},
		updateData: &mockResourceInterfaceData{},
	}
	m := webhookManager{
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
	err := m.ensureCA(context.TODO(), []byte(caPemForTestA))
	assert.Error(t, err)
	t.Log(err)

	assert.Equal(t, "test", res.getData.inputName)
	assert.Greater(t, res.getData.callCount, 2)
	assert.Equal(t, 0, res.updateData.callCount)
	assert.Nil(t, res.updateData.inputData)
}

func Test_webhookManager_ensureCA_retry_when_update_error(t *testing.T) {
	object := &v1.ValidatingWebhookConfiguration{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Webhooks: []v1.ValidatingWebhook{
			{
				Name: "test1",
			},
			{
				Name: "test2",
			},
		},
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	wh := &unstructured.Unstructured{Object: obj}
	res := &mockResourceInterface{
		getData: &mockResourceInterfaceData{
			data: wh,
		},
		updateData: &mockResourceInterfaceData{
			err: apierrors.NewInternalError(errors.New("error from update")),
		},
	}
	m := webhookManager{
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
	err = m.ensureCA(context.TODO(), []byte(caPemForTestA))
	assert.Error(t, err)
	t.Log(err)

	assert.Equal(t, "test", res.getData.inputName)
	assert.Greater(t, res.getData.callCount, 2)
	assert.Greater(t, res.updateData.callCount, 2)
	assert.NotNil(t, res.updateData.inputData)
}

func Test_webhookManager_watchChanges_receive_event(t *testing.T) {
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
			err: apierrors.NewInternalError(errors.New("error from update")),
		},
		w: watcher,
	}
	m := webhookManager{
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

	watchSendEvents := make(chan watch.Event, 10)
	watcher.events = watchSendEvents
	events := make(chan watch.Event)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.watchChanges(ctx, events, m.webhooks[0], time.Minute)

	// receive event
	e1 := watch.Event{Type: watch.Added}
	watchSendEvents <- e1
	select {
	case receivedE1 := <-events:
		assert.Equal(t, e1, receivedE1)
	case <-time.After(time.Second):
		assert.Fail(t, "no event")
	}

	// receive error
	close(watchSendEvents)
	select {
	case <-events:
	case <-time.After(time.Second):
	}

	// after close canceled watch
	assert.Equal(t, 1, watcher.callStop)
	select {
	case <-events:
		assert.Fail(t, "should no event")
	case <-time.After(time.Second):
	}
}
