/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	secretmirrorv1alpha1 "github.com/devacts/cm-secret-mirror-operator/api/v1alpha1"
)

// SecretMirrorReconciler reconciles a SecretMirror object
type SecretMirrorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=secretmirror.example.com,resources=secretmirrors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secretmirror.example.com,resources=secretmirrors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=secretmirror.example.com,resources=secretmirrors/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *SecretMirrorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	// SecretMirror.spec.sourceConfigMap → fetch the ConfigMap
	// SecretMirror.name + "-secret"     → target secret name

	// 1. Fetch SecretMirror
	// 2. Fetch ConfigMap (name from spec.sourceConfigMap)
	// 3. Sync secret from ConfigMap data
	// 4. Update SecretMirror status

	var sm secretmirrorv1alpha1.SecretMirror
	err := r.Client.Get(ctx, req.NamespacedName, &sm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch configMap Name from SecretMirror Spec
	sourceConfigMapName := sm.Spec.SourceConfigMap
	var cm corev1.ConfigMap
	err = r.Client.Get(
		ctx,
		client.ObjectKey{Namespace: req.Namespace, Name: sourceConfigMapName}, &cm,
	)

	if err != nil {
		var condErr error
		if apierrors.IsNotFound(err) {
			condErr = r.SetCondition(ctx, &sm, metav1.ConditionFalse, "SourceNotFound", "The Source ConfigMap was not found")
		} else {
			condErr = r.SetCondition(ctx, &sm, metav1.ConditionFalse, "SyncFailed", err.Error())
		}
		if condErr != nil {
			logger.Error(condErr, "failed to update status condition")
		}
		return ctrl.Result{}, err
	}

	labels := map[string]string{
		"synced": "true",
	}
	data := cm.Data
	byteData := make(map[string][]byte)
	for k, v := range data {
		byteData[k] = []byte(v)
	}

	// Fetch secret and sync
	var secret corev1.Secret
	secretName := sourceConfigMapName + "-secret"
	err = r.Client.Get(
		ctx, client.ObjectKey{Namespace: req.Namespace, Name: secretName}, &secret,
	)

	var sErr error
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create secret and update condition
			// return create call error
			secret.Name = secretName
			secret.Namespace = req.Namespace
			secret.Labels = labels
			secret.Data = byteData
			ctrl.SetControllerReference(&sm, &secret, r.Scheme)
			sErr = r.Client.Create(ctx, &secret)
			if condErr := r.SetCondition(ctx, &sm, metav1.ConditionTrue, "SecretSynced", "Secret created"); condErr != nil {
				logger.Error(condErr, "failed to update status condition")
			}
			return ctrl.Result{}, sErr
		}

		logger.Error(err, "Secret Fetch err")
		if condErr := r.SetCondition(ctx, &sm, metav1.ConditionFalse, "SyncFailed", err.Error()); condErr != nil {
			logger.Error(condErr, "failed to update status condition")
		}
		return ctrl.Result{}, err
	}

	// Patch secret
	base := secret.DeepCopy()
	secret.Labels = labels
	secret.Data = byteData
	sErr = r.Patch(ctx, &secret, client.MergeFrom(base))
	if sErr == nil {
		sErr = r.SetCondition(ctx, &sm, metav1.ConditionTrue, "SecretSynced", "Secret Patched")
	}
	return ctrl.Result{}, sErr
}

func (r *SecretMirrorReconciler) SetCondition(
	ctx context.Context,
	sm *secretmirrorv1alpha1.SecretMirror,
	status metav1.ConditionStatus, reason, msg string,
) error {
	meta.SetStatusCondition(&sm.Status.Conditions, metav1.Condition{
		Type:               "Synced",
		Status:             status,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: sm.Generation,
	})
	sm.Status.SecretName = sm.Spec.SourceConfigMap + "-secret"
	err := r.Client.Status().Update(ctx, sm)
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretMirrorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretmirrorv1alpha1.SecretMirror{}).
		Named("secretmirror").
		Owns(&corev1.Secret{}).
		Watches(
			&corev1.ConfigMap{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				var smList secretmirrorv1alpha1.SecretMirrorList
				if err := r.Client.List(ctx, &smList, client.InNamespace(obj.GetNamespace())); err != nil {
					return nil
				}
				var requests []reconcile.Request
				for _, sm := range smList.Items {
					if sm.Spec.SourceConfigMap == obj.GetName() {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      sm.Name,
								Namespace: sm.Namespace,
							},
						})
					}
				}
				return requests
			}),
		).
		Complete(r)
}
