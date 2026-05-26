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
		// 集群外运行：尝试 kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil || home == "" {
				// systemd 环境下 HOME 可能未设置，尝试常见路径
				candidates := []string{"/root/.kube/config", "/home/" + os.Getenv("USER") + "/.kube/config"}
				for _, c := range candidates {
					if _, statErr := os.Stat(c); statErr == nil {
						kubeconfig = c
						break
					}
				}
			} else {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}
		if kubeconfig == "" {
			log.Printf("Warning: K8s not available, cluster features disabled: kubeconfig not found")
			available = false
			return
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
