package ctlrhelper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mozillazg/webhookcert/pkg/cert"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctlrlog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	log                                 = ctlrlog.Log.WithName("webhookcert/pkg/ctrlhelper")
	defaultTimeoutForEnsureCertReady    = time.Minute * 5
	defaultTimeoutForCheckServerCert    = time.Second * 3
	defaultTimeoutForCheckServerStarted = time.Second * 3
	defaultHealthzCheckName             = "webhook"
	defaultReadyzCheckName              = "webhook"
	defaultBackoffForCheckServerStarted = wait.Backoff{
		Steps:    10,
		Duration: 500 * time.Millisecond,
		Factor:   3.0,
		Jitter:   0.1,
		Cap:      time.Second * 5,
	}
)

type Option struct {
	// required
	Namespace string
	// required
	SecretName string
	// required
	ServiceName string
	// required
	CertDir string
	// required
	WebhookServerPort int

	DnsName       string
	Organizations []string
	Hosts         []string
	// SkipSecretReadWrite skips Kubernetes Secret get/create/update and reads CA from CertDir instead.
	SkipSecretReadWrite bool
	CACertName          string
	CAKeyName           string
	CertName            string
	KeyName             string

	Webhooks                     []cert.WebhookInfo
	TimeoutForEnsureCertReady    time.Duration
	TimeoutForCheckServerStarted time.Duration
	TimeoutForCheckServerCert    time.Duration
	HealthzCheckName             string
	ReadyzCheckName              string

	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
}

type WebhookHelper struct {
	opt Option

	ensureCertFinished chan struct{}
	webhookReady       chan struct{}
}

func NewNewWebhookHelper(opt Option) (*WebhookHelper, error) {
	err := opt.ValidateAndFillDefaultValues()
	if err != nil {
		return nil, err
	}
	return &WebhookHelper{
		opt:                opt,
		ensureCertFinished: make(chan struct{}),
		webhookReady:       make(chan struct{}),
	}, nil
}

func NewNewWebhookHelperOrDie(opt Option) *WebhookHelper {
	w, err := NewNewWebhookHelper(opt)
	if err != nil {
		log.Error(err, "unable creates a new WebhookHelper for the option")
		os.Exit(1)
	}
	return w
}

func NewWebhookHelperOrDie(opt Option) *WebhookHelper {
	return NewNewWebhookHelperOrDie(opt)
}

func (w *WebhookHelper) EnsureCertFinished() <-chan struct{} {
	return w.ensureCertFinished
}

func (w *WebhookHelper) WebhookReady() <-chan struct{} {
	return w.webhookReady
}

// Setup is a non-block method
// * ensure cert exist and mounted
// * setup healthz and readyz
// * registry webhooks
func (w *WebhookHelper) Setup(ctx context.Context, mgr manager.Manager, registry func(webhook.Server), errC chan<- error) {
	webhookcert := w.ensureCertReady(ctx, errC)
	w.setupHealthzAndReadyz(mgr, webhookcert)
	go w.setupControllers(mgr, webhookcert, registry, errC)
	return
}

// EnsureCertReady ensure cert exist and mounted
// it will block util successes or failed
func (w *WebhookHelper) EnsureCertReady(ctx context.Context) error {
	errC := make(chan error, 1)
	w.ensureCertReady(ctx, errC)
	select {
	case <-w.ensureCertFinished:
		return nil
	case err := <-errC:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *WebhookHelper) ensureCertReady(ctx context.Context, errC chan<- error) *cert.WebhookCert {
	webhookcert := cert.NewWebhookCert(cert.CertOption{
		CAName:              w.opt.ServiceName,
		Organizations:       w.opt.Organizations,
		Hosts:               w.opt.Hosts,
		CommonName:          w.opt.DnsName,
		CertDir:             w.opt.CertDir,
		SkipSecretReadWrite: w.opt.SkipSecretReadWrite,
		SecretInfo: cert.SecretInfo{
			Name:       w.opt.SecretName,
			Namespace:  w.opt.Namespace,
			CACertName: w.opt.CACertName,
			CAKeyName:  w.opt.CAKeyName,
			CertName:   w.opt.CertName,
			KeyName:    w.opt.KeyName,
		},
	}, w.opt.Webhooks, w.opt.kubeClient, w.opt.dynamicClient)

	go func() {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, w.opt.TimeoutForEnsureCertReady)
		defer cancel()

		if err := webhookcert.EnsureCertReady(ctxWithTimeout); err != nil {
			log.Error(err, "ensure cert ready")
			errC <- err
			return
		}
		close(w.ensureCertFinished)

		if err := webhookcert.WatchAndEnsureWebhooksCA(ctx); err != nil {
			log.Error(err, "watch and ensure webhooks CA")
			errC <- err
			return
		}
	}()

	return webhookcert
}

func (w *WebhookHelper) setupControllers(mgr manager.Manager, webhookcert *cert.WebhookCert, registry func(webhook.Server), errC chan<- error) {
	<-w.ensureCertFinished

	log.Info("registering webhooks to the webhook server")
	s := mgr.GetWebhookServer()
	registry(s)
	addr := fmt.Sprintf("127.0.0.1:%d", w.opt.WebhookServerPort)

	w.markWebhookReadyWhenStarted(webhookcert, addr, errC)
}

func (w *WebhookHelper) markWebhookReadyWhenStarted(webhookcert *cert.WebhookCert, addr string, errC chan<- error) {
	if err := w.waitWebhookServerStarted(webhookcert, addr); err != nil {
		log.Error(err, "check webhook server started failed")
		errC <- err
		return
	}

	close(w.webhookReady)
}

func (w *WebhookHelper) waitWebhookServerStarted(webhookcert *cert.WebhookCert, addr string) error {
	return retry.OnError(defaultBackoffForCheckServerStarted, func(err error) bool {
		return true
	}, func() error {
		return webhookcert.CheckServerStartedWithTimeout(addr, w.opt.TimeoutForCheckServerStarted)
	})
}

func (w *WebhookHelper) setupHealthzAndReadyz(mgr manager.Manager, webhookcert *cert.WebhookCert) {
	addr := fmt.Sprintf("127.0.0.1:%d", w.opt.WebhookServerPort)
	_ = mgr.AddHealthzCheck(w.opt.HealthzCheckName, func(_ *http.Request) error {
		select {
		case <-w.webhookReady:
		default:
			return nil
		}

		err := webhookcert.CheckServerCertValidWithTimeout(addr, w.opt.TimeoutForCheckServerCert)
		if err != nil {
			log.Error(err, "check server cert failed")
		}
		return err
	})

	_ = mgr.AddReadyzCheck(w.opt.ReadyzCheckName, func(_ *http.Request) error {
		select {
		case <-w.webhookReady:
			err := webhookcert.CheckServerStartedWithTimeout(addr, w.opt.TimeoutForCheckServerStarted)
			return err
		default:
			return errors.New("webhook is not ready")
		}
	})
}

func (o *Option) ValidateAndFillDefaultValues() error {
	if !o.SkipSecretReadWrite && o.SecretName == "" {
		return errors.New("the SecretName field can not be empty")
	}
	if o.Namespace == "" {
		return errors.New("the Namespace field can not be empty")
	}
	if o.ServiceName == "" {
		return errors.New("the ServiceName field can not be empty")
	}
	if o.CertDir == "" {
		return errors.New("the CertDir field can not be empty")
	}
	if o.WebhookServerPort <= 0 {
		return errors.New("the WebhookServerPort field can not be empty")
	}
	if o.DnsName == "" {
		dnsName := fmt.Sprintf("%s.%s.svc", o.ServiceName, o.Namespace)
		o.DnsName = dnsName
	}
	if len(o.Organizations) == 0 {
		o.Organizations = append(o.Organizations, o.ServiceName)
	}
	if len(o.Hosts) == 0 {
		o.Hosts = append(o.Hosts, o.DnsName)
	}
	if o.TimeoutForEnsureCertReady == 0 {
		o.TimeoutForEnsureCertReady = defaultTimeoutForEnsureCertReady
	}
	if o.TimeoutForCheckServerCert == 0 {
		o.TimeoutForCheckServerCert = defaultTimeoutForCheckServerCert
	}
	if o.TimeoutForCheckServerStarted == 0 {
		o.TimeoutForCheckServerStarted = defaultTimeoutForCheckServerStarted
	}
	if o.HealthzCheckName == "" {
		o.HealthzCheckName = defaultHealthzCheckName
	}
	if o.ReadyzCheckName == "" {
		o.ReadyzCheckName = defaultReadyzCheckName
	}

	var conf *rest.Config
	var err error
	confNeeded := (!o.SkipSecretReadWrite && o.kubeClient == nil) || o.dynamicClient == nil
	if confNeeded {
		conf, err = config.GetConfig()
		if err != nil {
			log.Error(err, "unable to get kubeconfig")
			return err
		}
	}
	if !o.SkipSecretReadWrite && o.kubeClient == nil {
		o.kubeClient, err = kubernetes.NewForConfig(conf)
		if err != nil {
			log.Error(err, "unable creates a new kubernetes.Interface for the given config")
			return err
		}
	}
	if o.dynamicClient == nil {
		o.dynamicClient, err = dynamic.NewForConfig(conf)
		if err != nil {
			log.Error(err, "unable creates a new dynamic.Interface for the given config")
			return err
		}
	}

	return nil
}
