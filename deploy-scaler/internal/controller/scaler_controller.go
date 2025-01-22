/*
Copyright 2025.

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
	"encoding/json"
	"fmt"
	"time"

	apiv1alpha1 "github.com/windsyu/deploy-scaler/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("scaler_controller")

var originalDeployInfo = make(map[string]apiv1alpha1.DeployInfo)
var annotaions = make(map[string]string)

// ScalerReconciler reconciles a Scaler object
type ScalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=api.deploy.scaler.com,resources=scalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.deploy.scaler.com,resources=scalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=api.deploy.scaler.com,resources=scalers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Scaler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	log.Info("Reconciling Called")

	scaler := &apiv1alpha1.Scaler{}
	err := r.Get(ctx, req.NamespacedName, scaler)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	startTime := scaler.Spec.Start
	endTime := scaler.Spec.End
	replicas := scaler.Spec.Replicas

	currentHour := time.Now().Local().Hour()
	log.Info(fmt.Sprintf("Current Time: %d", currentHour))

	if currentHour >= startTime && currentHour < endTime {
		if scaler.Status.Status == "" {
			log.Info("Status is nil; Transition to PENDING")
			scaler.Status.Status = apiv1alpha1.PENDING
			if err := r.Status().Update(ctx, scaler); err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		} else if scaler.Status.Status == apiv1alpha1.PENDING {
			log.Info("ScalerDeploy Function Called")
			err := r.ScalerDeploy(scaler, replicas, ctx)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else if scaler.Status.Status == apiv1alpha1.SCALED {
			log.Info("restoreDeployment Function Called")
			if err := r.restoreDeployment(scaler, ctx); err != nil {
				return ctrl.Result{RequeueAfter: time.Duration(10 * time.Second)}, client.IgnoreNotFound(err)
			}
		} else if scaler.Status.Status == apiv1alpha1.RESTORED {
			log.Info("Status RESTORED")
			return ctrl.Result{RequeueAfter: time.Duration(10 * time.Second)}, nil
		}

	}

	// TODO(user): your logic here

	return ctrl.Result{RequeueAfter: time.Duration(10 * time.Second)}, nil
}

func (r *ScalerReconciler) restoreDeployment(scaler *apiv1alpha1.Scaler, ctx context.Context) error {
	for name, originInfo := range originalDeployInfo {
		deployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: originInfo.Namespace,
		}, deployment); err != nil {
			return err
		}

		if *deployment.Spec.Replicas != int32(originInfo.Replicas) {
			*deployment.Spec.Replicas = int32(originInfo.Replicas)
			if err := r.Update(ctx, deployment); err != nil {
				return err
			}
		}
	}
	scaler.Status.Status = apiv1alpha1.RESTORED
	if err := r.Status().Update(ctx, scaler); err != nil {
		return err
	}

	return nil
}

func (r *ScalerReconciler) ScalerDeploy(scaler *apiv1alpha1.Scaler, replicas int, ctx context.Context) error {
	for _, deploy := range scaler.Spec.Deploys {
		deployment := &appsv1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      deploy.Name,
			Namespace: deploy.Namespace,
		}, deployment)
		if err != nil {
			return err
		}
		//更新副本数
		if *deployment.Spec.Replicas != int32(replicas) {
			*deployment.Spec.Replicas = int32(replicas)
			err := r.Update(ctx, deployment)
			if err != nil {
				return err
			}
		}
		scaler.Status.Status = apiv1alpha1.SCALED
		if err := r.Status().Update(ctx, scaler); err != nil {
			return err
		}
	}
	return nil
}

func (r *ScalerReconciler) AddAnotations(scaler *apiv1alpha1.Scaler, ctx context.Context) error {
	//record the original replicas
	for _, deploy := range scaler.Spec.Deploys {
		deployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      deploy.Name,
			Namespace: deploy.Namespace,
		}, deployment); err != nil {
			return err
		}
		if *deployment.Spec.Replicas != int32(scaler.Spec.Replicas) {
			logger.Info("add origin state to originalDeployInfo")
			originalDeployInfo[deployment.Name] = apiv1alpha1.DeployInfo{
				Replicas:  *deployment.Spec.Replicas,
				Namespace: deployment.Namespace,
			}
		}

	}

	//record the original replicas to annotations
	for deployName, info := range originalDeployInfo {
		//encode info to json
		infoJson, err := json.Marshal(info)
		if err != nil {
			return err
		}

		//将infoJson存储到Annotation中
		annotaions[deployName] = string(infoJson)

	}
	//update scaler的annotations
	scaler.ObjectMeta.Annotations = annotaions
	r.Update(ctx, scaler)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Scaler{}).
		Named("scaler").
		Complete(r)
}
