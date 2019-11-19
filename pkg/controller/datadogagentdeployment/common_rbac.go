package datadogagentdeployment

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	datadoghqv1alpha1 "github.com/DataDog/datadog-operator/pkg/apis/datadoghq/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// roleBindingInfo contains the required information to build a Cluster Role Binding
type roleBindingInfo struct {
	name               string
	roleName           string
	serviceAccountName string
}

// buildRoleBinding creates a RoleBinding object
func buildRoleBinding(dad *datadoghqv1alpha1.DatadogAgentDeployment, info roleBindingInfo, agentVersion string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    getDefaultLabels(dad, info.name, agentVersion),
			Name:      info.name,
			Namespace: dad.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: datadoghqv1alpha1.RbacAPIGroup,
			Kind:     datadoghqv1alpha1.RoleKind,
			Name:     info.roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      datadoghqv1alpha1.ServiceAccountKind,
				Name:      info.serviceAccountName,
				Namespace: dad.Namespace,
			},
		},
	}
}

// buildServiceAccount creates a ServiceAccount object
func buildServiceAccount(dad *datadoghqv1alpha1.DatadogAgentDeployment, name, agentVersion string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    getDefaultLabels(dad, name, agentVersion),
			Name:      name,
			Namespace: dad.Namespace,
		},
	}
}

// getEventCollectionPolicyRule returns the policy rule for event collection
func getEventCollectionPolicyRule() rbacv1.PolicyRule {
	return rbacv1.PolicyRule{
		APIGroups:     []string{datadoghqv1alpha1.CoreAPIGroup},
		Resources:     []string{datadoghqv1alpha1.ConfigMapsResource},
		ResourceNames: []string{datadoghqv1alpha1.DatadogTokenResourceName},
		Verbs:         []string{datadoghqv1alpha1.GetVerb, datadoghqv1alpha1.UpdateVerb},
	}
}

// getLeaderElectionPolicyRule returns the policy rules for leader election
func getLeaderElectionPolicyRule() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups:     []string{datadoghqv1alpha1.CoreAPIGroup},
			Resources:     []string{datadoghqv1alpha1.ConfigMapsResource},
			ResourceNames: []string{datadoghqv1alpha1.DatadogLeaderElectionResourceName},
			Verbs:         []string{datadoghqv1alpha1.GetVerb, datadoghqv1alpha1.UpdateVerb},
		},
		{
			APIGroups: []string{datadoghqv1alpha1.CoreAPIGroup},
			Resources: []string{datadoghqv1alpha1.ConfigMapsResource},
			Verbs:     []string{datadoghqv1alpha1.CreateVerb},
		},
	}
}

func (r *ReconcileDatadogAgentDeployment) createClusterRoleBinding(logger logr.Logger, dad *datadoghqv1alpha1.DatadogAgentDeployment, info roleBindingInfo, agentVersion string) (reconcile.Result, error) {
	clusterRoleBinding := buildClusterRoleBinding(dad, info, agentVersion)
	if err := controllerutil.SetControllerReference(dad, clusterRoleBinding, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	logger.V(1).Info("createClusterRoleBinding", "clusterRoleBinding.name", clusterRoleBinding.Name)
	return reconcile.Result{Requeue: true}, r.client.Create(context.TODO(), clusterRoleBinding)
}

func cleanupClusterRole(client client.Client, name string) (reconcile.Result, error) {
	clusterRole := &rbacv1.ClusterRole{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name}, clusterRole)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if !ownedByDatadogOperator(clusterRole.OwnerReferences) {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, client.Delete(context.TODO(), clusterRole)
}

func cleanupClusterRoleBinding(client client.Client, name string) (reconcile.Result, error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name}, clusterRoleBinding)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if !ownedByDatadogOperator(clusterRoleBinding.OwnerReferences) {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, client.Delete(context.TODO(), clusterRoleBinding)
}

func cleanupServiceAccount(client client.Client, name string) (reconcile.Result, error) {
	serviceAccount := &corev1.ServiceAccount{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name}, serviceAccount)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if !ownedByDatadogOperator(serviceAccount.OwnerReferences) {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, client.Delete(context.TODO(), serviceAccount)
}