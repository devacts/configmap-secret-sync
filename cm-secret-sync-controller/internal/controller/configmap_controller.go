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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ConfigMapReconciler reconciles a ConfigMap object
type ConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigMap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var cm corev1.ConfigMap

	err := r.Client.Get(ctx, req.NamespacedName, &cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	labels := cm.Labels
	val, ok := labels["sync-me"]

	var secret corev1.Secret
	err = r.Client.Get(
		ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name + "-secret"}, &secret,
	)

	if err == nil {
		logger.Info("Listed secret", "name", secret.Name)
	}

	secretExists := (err == nil)
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Secret Get/List err")
		return ctrl.Result{}, err
	}

	logger.Info("debug vals:", "secretExists", secretExists, "ok", ok, "val", val)
	switch {
	case secretExists && (!ok || val == "false"):
		logger.Info("Deleting secret", "name", secret.Name)
		err = r.Delete(ctx, &secret)
	case ok && val == "true":
		err = r.syncSecret(ctx, &cm, &secret, logger)
	}

	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Reconcile Failed")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ConfigMapReconciler) syncSecret(ctx context.Context, cm *corev1.ConfigMap, secret *corev1.Secret, logger logr.Logger) error {
	logger.Info("Syncing configmap", "name", cm.Name, "UID", cm.UID)
	labels := map[string]string{
		"synced": "true",
	}
	var err error

	data := cm.Data
	byteData := make(map[string][]byte)
	for k, v := range data {
		b := []byte(v)
		byteData[k] = b
	}

	if secret.UID == "" {
		secret.Name = cm.Name + "-secret"
		secret.Namespace = cm.Namespace
		secret.Labels = labels
		secret.Data = byteData
		ctrl.SetControllerReference(cm, secret, r.Scheme)
		err = r.Create(ctx, secret)
	} else {
		base := secret.DeepCopy()
		secret.Labels = labels
		secret.Data = byteData
		err = r.Patch(ctx, secret, client.MergeFrom(base))
	}

	if err == nil {
		logger.Info("Applied cm to secret transform successfully", "cmName", cm.Name, "secretName", secret.Name)
	}
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("configmap").
		Owns(&corev1.Secret{}).
		Complete(r)
}
