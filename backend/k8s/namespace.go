package k8s

import (
	"context"
	"errors"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ListNamespaces() ([]string, error) {
	if !available {
		return []string{"default", "kube-system", "demo-project"}, nil
	}
	nsList, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	names := make([]string, len(nsList.Items))
	for i, ns := range nsList.Items {
		names[i] = ns.Name
	}
	sort.Strings(names)
	return names, nil
}

func EnsureAvailable() error {
	if !available {
		return errors.New("K8s not available")
	}
	return nil
}
