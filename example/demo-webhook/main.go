package main

import (
	"context"
	"flag"
	"os"

	jsonpatch "gomodules.xyz/jsonpatch/v2"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	port    = flag.Int("port", webhook.DefaultPort, "port for the server")
	certDir = flag.String("cert-dir", "/cert", "The directory where certs are stored")
)

func init() {
	log.SetLogger(zap.New())
}

var entryLog = log.Log.WithName("entrypoint")

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		LeaderElection:         false,
		Port:                   *port,
		CertDir:                *certDir,
		HealthProbeBindAddress: "0",
		MetricsBindAddress:     "0",
	})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	entryLog.Info("setting up webhook")
	ctx := signals.SetupSignalHandler()

	setupWebhook(ctx, mgr)

	entryLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}

func setupWebhook(ctx context.Context, mgr manager.Manager) {
	s := mgr.GetWebhookServer()
	h1 := &DemoValidatingWebhook{}
	s.Register("/validating-webhook", &webhook.Admission{Handler: h1})

	h2 := &DemoMutatingWebhook{}
	s.Register("/mutating-webhook", &webhook.Admission{Handler: h2})
}

type DemoValidatingWebhook struct{}

func (w *DemoValidatingWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	return admission.Denied("denied by demo validating webhook")
}

type DemoMutatingWebhook struct{}

func (w *DemoMutatingWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	return admission.Patched("patched by demo mutating webhook", jsonpatch.JsonPatchOperation{
		Operation: "add",
		Path:      "/metadata/annotations/patched-by-demo-mutating-webhook",
		Value:     "added",
	})
}
