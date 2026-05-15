package k8s

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SystemNamespace   = "k8sgate-system"
	ViewerClusterRole = "k8sgate-viewer"
	DevClusterRole    = "k8sgate-developer"
	AdminSAName       = "k8sgate-admin"
)

// EnsureClusterRoles 确保系统所需的 ClusterRole 存在
func EnsureClusterRoles() error {
	ctx := context.Background()

	viewerRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: ViewerClusterRole},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"", "apps", "batch", "networking.k8s.io"},
				Resources: []string{
					"pods", "pods/log", "services", "deployments", "replicasets",
					"statefulsets", "daemonsets", "jobs", "cronjobs", "configmaps",
					"secrets", "ingresses", "endpoints", "events",
					"persistentvolumeclaims", "namespaces",
				},
				Verbs: []string{"get", "list", "watch"},
			},
		},
	}

	devRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: DevClusterRole},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"", "apps", "batch", "networking.k8s.io"},
				Resources: []string{
					"pods", "pods/log", "pods/exec", "services", "deployments",
					"replicasets", "statefulsets", "daemonsets", "jobs", "cronjobs",
					"configmaps", "secrets", "ingresses", "endpoints",
					"persistentvolumeclaims",
				},
				Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{"", "apps", "batch", "networking.k8s.io"},
				Resources: []string{"namespaces", "events"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	for _, role := range []*rbacv1.ClusterRole{viewerRole, devRole} {
		existing, err := clientset.RbacV1().ClusterRoles().Get(ctx, role.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = clientset.RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create ClusterRole %s: %v", role.Name, err)
			}
		} else if err == nil {
			existing.Rules = role.Rules
			_, err = clientset.RbacV1().ClusterRoles().Update(ctx, existing, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update ClusterRole %s: %v", role.Name, err)
			}
		} else {
			return err
		}
	}

	return nil
}

// EnsureServiceAccount 确保用户的 ServiceAccount 存在
func EnsureServiceAccount(userID uint) error {
	ctx := context.Background()
	saName := fmt.Sprintf("k8sgate-%d", userID)

	_, err := clientset.CoreV1().ServiceAccounts(SystemNamespace).Get(ctx, saName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: SystemNamespace,
				Labels:    map[string]string{"app": "k8sgate", "user-id": fmt.Sprintf("%d", userID)},
			},
		}
		_, err = clientset.CoreV1().ServiceAccounts(SystemNamespace).Create(ctx, sa, metav1.CreateOptions{})
		return err
	}
	return err
}

// SyncRoleBindings 同步用户的 RoleBinding
func SyncRoleBindings(userID uint, role string, namespaces []string) error {
	ctx := context.Background()
	saName := fmt.Sprintf("k8sgate-%d", userID)

	if err := EnsureServiceAccount(userID); err != nil {
		return err
	}

	// 清除旧的 RoleBinding
	if err := cleanRoleBindings(ctx, saName); err != nil {
		return err
	}

	if role == "admin" {
		return nil // admin 使用 admin SA
	}

	// global_viewer: 全局只读，使用 ClusterRoleBinding
	if role == "global_viewer" {
		return ensureGlobalViewerBinding(ctx, saName)
	}

	// developer: 在指定命名空间创建 RoleBinding
	clusterRole := DevClusterRole
	for _, ns := range namespaces {
		rb := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: ns,
				Labels:    map[string]string{"app": "k8sgate", "user-id": fmt.Sprintf("%d", userID)},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      saName,
					Namespace: SystemNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     clusterRole,
			},
		}

		existing, err := clientset.RbacV1().RoleBindings(ns).Get(ctx, saName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = clientset.RbacV1().RoleBindings(ns).Create(ctx, rb, metav1.CreateOptions{})
		} else if err == nil {
			existing.Subjects = rb.Subjects
			existing.RoleRef = rb.RoleRef
			_, err = clientset.RbacV1().RoleBindings(ns).Update(ctx, existing, metav1.UpdateOptions{})
		}
		if err != nil {
			return fmt.Errorf("failed to create RoleBinding in %s: %v", ns, err)
		}
	}

	return nil
}

// ensureGlobalViewerBinding 为 global_viewer 创建 ClusterRoleBinding
func ensureGlobalViewerBinding(ctx context.Context, saName string) error {
	crbName := saName
	_, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, crbName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:   crbName,
				Labels: map[string]string{"app": "k8sgate"},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      saName,
					Namespace: SystemNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     ViewerClusterRole,
			},
		}
		_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	}
	return err
}

// GetTokenForUser 获取用户 ServiceAccount 的 Token
func GetTokenForUser(userID uint, role string) (string, error) {
	if !available {
		return "mock-token-for-dev", nil
	}
	ctx := context.Background()

	saName := AdminSAName
	if role != "admin" {
		saName = fmt.Sprintf("k8sgate-%d", userID)
	}

	expSeconds := int64(3600)
	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: &expSeconds,
		},
	}

	result, err := clientset.CoreV1().ServiceAccounts(SystemNamespace).
		CreateToken(ctx, saName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token for %s: %v", saName, err)
	}

	return result.Status.Token, nil
}

func cleanRoleBindings(ctx context.Context, saName string) error {
	// 列出所有命名空间中该用户的 RoleBinding
	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range nsList.Items {
		err := clientset.RbacV1().RoleBindings(ns.Name).Delete(ctx, saName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// EnsureAdminSA 确保管理员 ServiceAccount 存在
func EnsureAdminSA() error {
	ctx := context.Background()

	// 创建 ServiceAccount
	_, err := clientset.CoreV1().ServiceAccounts(SystemNamespace).Get(ctx, AdminSAName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      AdminSAName,
				Namespace: SystemNamespace,
				Labels:    map[string]string{"app": "k8sgate"},
			},
		}
		_, err = clientset.CoreV1().ServiceAccounts(SystemNamespace).Create(ctx, sa, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// 创建 ClusterRoleBinding
	crbName := "k8sgate-admin"
	_, err = clientset.RbacV1().ClusterRoleBindings().Get(ctx, crbName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:   crbName,
				Labels: map[string]string{"app": "k8sgate"},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      AdminSAName,
					Namespace: SystemNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
		}
		_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
		return err
	}
	return err
}

// EnsureNamespace 确保系统命名空间存在
func EnsureNamespace() error {
	ctx := context.Background()
	_, err := clientset.CoreV1().Namespaces().Get(ctx, SystemNamespace, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   SystemNamespace,
				Labels: map[string]string{"app": "k8sgate"},
			},
		}
		_, err = clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		return err
	}
	return err
}

// InitK8sResources 初始化所有 K8s 资源
func InitK8sResources() error {
	if !available {
		return fmt.Errorf("K8s not available, skipping resource init")
	}
	if err := EnsureNamespace(); err != nil {
		return fmt.Errorf("ensure namespace: %v", err)
	}
	if err := EnsureClusterRoles(); err != nil {
		return fmt.Errorf("ensure cluster roles: %v", err)
	}
	if err := EnsureAdminSA(); err != nil {
		return fmt.Errorf("ensure admin SA: %v", err)
	}
	return nil
}
