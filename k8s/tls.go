package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/cert-manager/cert-manager/pkg/apis/certmanager"
	certmanager_v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cm_metav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8s_error "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/homedir"
)

var (
	TLSCertDir        = "k8s-tls"
	CertName          = "cert.crt"
	KeyName           = "key.crt"
	WebhookValidPath  = "/webhook/validate"
	WebhookMutatePath = "/webhook/mutate"
	webhookNamespace  = "default"

	TLSPort int32 = 9443
	//ca
	caIssuerName  = "selfsigned-issuer"
	caName        = "selfsigned-ca"
	caSecretName  = "root-ca-secret"
	clusterIssuer = "webhook-issuer"
	// webhook-cert
	webhookCertName    = "webhook-cert"
	webhookSecretName  = "webhook-tls"
	webhookServiceName = "pod-webhook.default.svc"
)

func SetUpCertManager(k8sClient *kubernetes.Clientset, certClient *versioned.Clientset) error {

	// selfsigned-issuer
	caIssuer := certmanager_v1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: caIssuerName,
		},
		Spec: certmanager_v1.IssuerSpec{
			IssuerConfig: certmanager_v1.IssuerConfig{
				SelfSigned: &certmanager_v1.SelfSignedIssuer{},
			},
		},
	}

	// ca certificate
	caCert := certmanager_v1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caName,
			Namespace: "cert-manager", // cert-manager root ns, for cluster-scope clusterIsser use
		},
		Spec: certmanager_v1.CertificateSpec{
			IsCA:       true,
			CommonName: caName,
			SecretName: caSecretName,
			PrivateKey: &certmanager_v1.CertificatePrivateKey{
				Algorithm: certmanager_v1.ECDSAKeyAlgorithm,
				Size:      256,
			},
			IssuerRef: cm_metav1.ObjectReference{
				Name:  caIssuerName,
				Kind:  certmanager_v1.ClusterIssuerKind,
				Group: certmanager.GroupName,
			},
		},
	}

	// create clusterIssuer with ca secret
	myClusterIssuer := certmanager_v1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterIssuer,
		},
		Spec: certmanager_v1.IssuerSpec{
			IssuerConfig: certmanager_v1.IssuerConfig{
				CA: &certmanager_v1.CAIssuer{
					SecretName: caSecretName,
				},
			},
		},
	}
	// create webhook cert
	myCert := certmanager_v1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookCertName,
			Namespace: webhookNamespace,
		},
		Spec: certmanager_v1.CertificateSpec{
			SecretName: webhookSecretName,
			Duration:   &metav1.Duration{Duration: 365 * 10 * 24 * time.Hour},
			IssuerRef: cm_metav1.ObjectReference{
				Name:  clusterIssuer,
				Kind:  certmanager_v1.ClusterIssuerKind,
				Group: certmanager.GroupName,
			},
			CommonName: webhookServiceName,
			DNSNames:   []string{webhookServiceName},
		},
	}

	_, err := certClient.CertmanagerV1().ClusterIssuers().Create(context.Background(), &caIssuer, metav1.CreateOptions{})
	if err != nil {
		if k8s_error.IsAlreadyExists(err) {
			slog.Info("caIssuer already exists", "name", caIssuer.Name)
		} else {
			return err
		}
	}
	slog.Info("caIssuer create success", "name", caIssuer.Name)

	// the order create is sort of important, so add sleep for safe
	<-time.After(time.Second * 2)
	_, err = certClient.CertmanagerV1().Certificates(caCert.Namespace).Create(context.Background(), &caCert, metav1.CreateOptions{})
	if err != nil {
		if k8s_error.IsAlreadyExists(err) {
			slog.Info("caCert already exists", "name", caCert.Name, "namespace", caCert.Namespace)
		} else {
			return err
		}
	}
	slog.Info("caCert create success", "name", caCert.Name, "namespace", caCert.Namespace)

	<-time.After(time.Second * 5)
	_, err = certClient.CertmanagerV1().ClusterIssuers().Create(context.Background(), &myClusterIssuer, metav1.CreateOptions{})
	if err != nil {
		if k8s_error.IsAlreadyExists(err) {
			slog.Info("clusterIssuer already exists", "name", myClusterIssuer.Name)
		} else {
			return err
		}
	}
	slog.Info("clusterIssuer create success", "name", myClusterIssuer.Name)

	<-time.After(time.Second * 5)
	_, err = certClient.CertmanagerV1().Certificates(myCert.Namespace).Create(context.Background(), &myCert, metav1.CreateOptions{})
	if err != nil {
		if k8s_error.IsAlreadyExists(err) {
			slog.Info("myCert already exists", "name", myCert.Name, "namespace", myCert.Namespace)
		} else {
			return err
		}
	}
	slog.Info("myCert create success", "name", myCert.Name, "namespace", myCert.Namespace)

	<-time.After(time.Second * 5)
	// create certfile in cert dir
	if err := genCertFile(k8sClient, webhookSecretName, webhookNamespace); err != nil {
		return err
	}

	return nil
}

func genCertFile(k8sClient *kubernetes.Clientset, secretName, secretNamespace string) error {

	s, err := k8sClient.CoreV1().Secrets(secretNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	tlsCrt, ok := s.Data["tls.crt"]
	if !ok {
		return fmt.Errorf("tls.crt not found in %s/%s.Data", s.Namespace, s.Name)
	}
	tlsKey, ok := s.Data["tls.key"]
	if !ok {
		return fmt.Errorf("tls.key not found in %s/%s.Data", s.Namespace, s.Name)
	}

	err = genCertFileFromBytes(tlsCrt, tlsKey)
	if err != nil {
		return err
	}

	return nil
}

func genCertFileFromBytes(tlsCrt, tlsKey []byte) error {
	certCrtPath := filepath.Join(homedir.HomeDir(), TLSCertDir)
	certKeyPath := filepath.Join(homedir.HomeDir(), TLSCertDir)

	// check file or create
	if err := checkDirOrCreate(certCrtPath); err != nil {
		return err
	}
	if err := checkDirOrCreate(certKeyPath); err != nil {
		return err
	}

	// write date
	err := os.WriteFile(filepath.Join(certCrtPath, CertName), tlsCrt, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(certKeyPath, KeyName), tlsKey, 0644)
	if err != nil {
		return err
	}

	return nil
}

func checkDirOrCreate(path string) error {
	_, err := os.Lstat(path)
	if err != nil {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetCaBundle(k8sClient *kubernetes.Clientset, secretName, secretNamespace string) ([]byte, error) {

	s, err := k8sClient.CoreV1().Secrets(secretNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	d, ok := s.Data["ca.crt"]
	if !ok {
		return nil, fmt.Errorf("ca.crt not found in secret %s/%s.Data", secretName, secretNamespace)
	}
	return d, nil
}

func CheckCertCrdExits(client *apiextv1.Clientset) bool {
	_, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "certificates.cert-manager.io", metav1.GetOptions{})
	if err != nil {
		slog.Error("CheckCertCrdExits failed", "error", err)
	}
	return err == nil
}
