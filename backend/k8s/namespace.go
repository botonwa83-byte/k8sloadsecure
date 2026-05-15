package k8s

import (
	"context"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ListNamespaces() ([]string, error) {
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
