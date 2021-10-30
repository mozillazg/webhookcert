package cert

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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