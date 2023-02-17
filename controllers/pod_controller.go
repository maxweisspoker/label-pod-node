/*
Copyright 2023.

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

// Most of this was taken from:
// https://kubernetes.io/blog/2021/06/21/writing-a-controller-for-pod-labels/
// which is copyrighted by the Kubernetes Authors under the CC BY 4.0 license,
// which can be found here: https://github.com/kubernetes/website/blob/main/LICENSE

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logpkg "sigs.k8s.io/controller-runtime/pkg/log"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	addPodNodeLabelAnnotation = "k8s.maxweiss.io/label-pod-node"
	podNodeNameLabel          = "node-name"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logpkg.FromContext(ctx)

	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Pod")
		return ctrl.Result{}, err
	}

	if pod.Spec.NodeName == "" {
		log.Info("pod not yet assigned to node")
		return ctrl.Result{}, nil
	}

	labelShouldBePresent := pod.Annotations[addPodNodeLabelAnnotation] == "true"
	labelIsPresent := pod.Labels[podNodeNameLabel] == pod.Spec.NodeName

	if labelShouldBePresent {
		if pod.Labels == nil {
			pod.Labels = make(map[string]string)
		}
		pod.Labels[podNodeNameLabel] = pod.Spec.NodeName
		log.Info("adding nodeName pod label")
	} else {
		if labelIsPresent {
			delete(pod.Labels, podNodeNameLabel)
			log.Info("annotation requesting label does not exist; removing existing pod label")
		} else {
			log.Info("annotation requesting label does not exist; not labeling pod")
		}
	}

	if err := r.Update(ctx, &pod); err != nil {
		if apierrors.IsConflict(err) {
			// The Pod has been updated since we read it.
			// Requeue the Pod to try to reconciliate again.
			return ctrl.Result{Requeue: true}, nil
		}
		if apierrors.IsNotFound(err) {
			// The Pod has been deleted since we read it.
			// Requeue the Pod to try to reconciliate again.
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "unable to update Pod")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
