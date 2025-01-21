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
	// "errors"
	"fmt"
	"reflect"

	pkgerror "github.com/pkg/errors"
	v1 "github.com/windsyu/i-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	AppFinalizer = "application.crd.xiaofeng.com/finalizer"
)

var logger = log.Log.WithName("application")

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.crd.xiaofeng.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.crd.xiaofeng.com,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.crd.xiaofeng.com,resources=applications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile

// Reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	log.Info("Reconciling Called")

	var app v1.Application
	//是否存在CR对应的deployment
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		log.Error(err, "unable to fetch Application")

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// examine DeletionTimestamp to determine if CR is under deletion
	if app.ObjectMeta.DeletionTimestamp.IsZero() {
		// CR is not being deleted, add finalizer and update CR to register our finalizer
		if !controllerutil.ContainsFinalizer(&app, AppFinalizer) {
			controllerutil.AddFinalizer(&app, AppFinalizer)
			if err := r.Update(ctx, &app); err != nil {
				log.Error(err, "unable to add finalizer to Application")
				return ctrl.Result{}, err
			}
		}

	} else {
		//CR is being deleted
		if controllerutil.ContainsFinalizer(&app, AppFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(&app); err != nil {
				log.Error(err, "unable to cleanup before Finalizer")
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&app, AppFinalizer)
			if err := r.Update(ctx, &app); err != nil {
				log.Error(err, "unable to remove finalizer from Application")
				return ctrl.Result{}, err
			}
		}

		//Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil

	}

	log.Info("run reconcile logic")
	if err := r.syncApp(ctx, &app); err != nil {
		log.Error(err, "unable to sync Application")
		return ctrl.Result{}, nil
	}
	//sync status
	var deploy appsv1.Deployment
	objKey := client.ObjectKey{Namespace: app.Namespace, Name: deploymentName(app.Name)}
	if err := r.Get(ctx, objKey, &deploy); err != nil {
		log.Error(err, "unable to fetch Deployment", "deployment", objKey.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	copyApp := app.DeepCopy()
	//if ready replicas is greater than 1 ,set status to true
	copyApp.Status.Ready = deploy.Status.ReadyReplicas > 1
	if !reflect.DeepEqual(app, copyApp) {
		//update when change
		log.Info("app changed ,update app status")
		if err := r.Client.Status().Update(ctx, copyApp); err != nil {
			log.Error(err, "unable to update Application status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func deploymentName(Name string) string {
	return fmt.Sprintf("app-%s", Name)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Application{}).
		Named("application").
		Complete(r)
}

func (r *ApplicationReconciler) deleteExternalResources(app *v1.Application) error {
	// TODO(user): delete external resources defined for Application
	logger.Info(fmt.Sprintf("deleting %v", app.Name))
	return nil
}

func (r *ApplicationReconciler) syncApp(ctx context.Context, app *v1.Application) error {
	if app.Spec.Enable {
		return r.syncAppEnabled(ctx, app)
	}
	return r.syncAppDisabled(ctx, app)
}

func (r *ApplicationReconciler) syncAppDisabled(ctx context.Context, app *v1.Application) error {
	var deploy appsv1.Deployment
	objKey := client.ObjectKey{Namespace: app.Namespace, Name: deploymentName(app.Name)}
	err := r.Get(ctx, objKey, &deploy)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return pkgerror.WithMessagef(err, "unable to fetch Deployment [%s]", objKey.String())
	}

	log.Log.Info("reconcile application delete deployment", "app", app.Namespace, "deployment", objKey.Name)
	if err := r.Delete(ctx, &deploy); err != nil {
		return pkgerror.WithMessage(err, "unable to delete Deployment")
	}
	return nil
}

func (r *ApplicationReconciler) syncAppEnabled(ctx context.Context, app *v1.Application) error {
	var deploy appsv1.Deployment
	objKey := client.ObjectKey{Namespace: app.Namespace, Name: deploymentName(app.Name)}
	err := r.Get(ctx, objKey, &deploy)
	//create deployment if needed
	if err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("Application creates deployment", "app", app.Namespace, "deployment", objKey.Name)
			deploy = r.genergateDeployment(app)
			if err := r.Create(ctx, &deploy); err != nil {
				return pkgerror.WithMessage(err, "unable to create deployment")
			}
		}

		return pkgerror.WithMessagef(err, "uable to fetch deployment [%s]", objKey.String())
	}

	//update deployment if needed
	if !equal(app, deploy) {
		log.Log.Info("Application update deployment", "app", app.Namespace, "deployment", objKey.Name)
		deploy.Spec.Template.Spec.Containers[0].Image = app.Spec.Image
		if err := r.Update(ctx, &deploy); err != nil {
			return pkgerror.WithMessage(err, "unable to update deployment")
		}
	}
	return nil
}

func equal(app *v1.Application, deploy appsv1.Deployment) bool {
	return deploy.Spec.Template.Spec.Containers[0].Image == app.Spec.Image
}

func (r *ApplicationReconciler) genergateDeployment(app *v1.Application) appsv1.Deployment {
	deploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName(app.Name),
			Namespace: app.Namespace,
			Labels: map[string]string{
				"app": app.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": app.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  app.Name,
							Image: app.Spec.Image,
						},
					},
				},
			},
		},
	}
	_ = controllerutil.SetControllerReference(app, &deploy, r.Scheme)
	return deploy
}
