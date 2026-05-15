package k8s

import (
	"log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var clientset *kubernetes.Clientset
var restConfig *rest.Config

func InitClient() {
	var err error
	restConfig, err = rest.InClusterConfig()
	if err != nil {
		log.Printf("Warning: not running in cluster, using default config: %v", err)
		// 本地开发时可以使用 kubeconfig，生产环境使用 InClusterConfig
		restConfig = &rest.Config{
			Host: "https://kubernetes.default.svc",
		}
	}

	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("failed to create kubernetes client: %v", err)
	}
}

func GetClientset() *kubernetes.Clientset {
	return clientset
}
