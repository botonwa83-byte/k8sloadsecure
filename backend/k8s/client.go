package k8s

import (
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var clientset *kubernetes.Clientset
var restConfig *rest.Config
var available bool

func InitClient() {
	var err error

	// 优先使用集群内配置
	restConfig, err = rest.InClusterConfig()
	if err != nil {
		// 本地开发：尝试 kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Printf("Warning: K8s not available, cluster features disabled: %v", err)
			available = false
			return
		}
	}

	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Printf("Warning: K8s client init failed, cluster features disabled: %v", err)
		available = false
		return
	}
	available = true
	log.Println("K8s client initialized successfully")
}

func IsAvailable() bool {
	return available
}

func GetClientset() *kubernetes.Clientset {
	return clientset
}
