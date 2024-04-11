package main

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsresv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

func main() {
	//1. create configuration
	config, err := clientcmd.BuildConfigFromFlags("", "C:/Users/27380/.kube/config")
	if err != nil {
		panic(err)
	}
	//2.create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	//3.create deployment
	deployClient := clientset.AppsV1().Deployments(corev1.NamespaceDefault)
	CreateDeploy(deployClient)

	//4.update strategy
	UpdateDeploy(deployClient)
	prompt()

	//5.view deployment
	ListDeploy(deployClient)

	prompt()

	//6.Delete Deployment
	DeleteDeploy(deployClient)
}

func CreateDeploy(client appsresv1.DeploymentInterface) {
	klog.Info("CreateDeploy.........")
	replicas := int32(2)
	deploy := appsv1.Deployment{
		TypeMeta: v1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "deploy-nginx-demo",
			Namespace: corev1.NamespaceDefault,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Name: "nginx",
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []corev1.ContainerPort{
								{
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	dep, err := client.Create(context.Background(), &deploy, v1.CreateOptions{})
	if err != nil {
		klog.Errorf("create deployment error:%v", err)
		return
	}
	klog.Infof("create deployment success,name:%s", dep.Name)
}

func UpdateDeploy(client appsresv1.DeploymentInterface) {
	klog.Info("Updating Deploy..................")
	//多个客户端对同一个资源操作时会发生错误，所以使用RetryOnConflict来重试
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		//查询要更新的deploy
		deploy, err := client.Get(context.Background(), "deploy-nginx-demo", v1.GetOptions{})
		if err != nil {
			klog.Errorf("can't get deployment, err:%v", err)
			return nil
		}
		replicas := int32(1)
		deploy.Spec.Replicas = &replicas
		deploy.Spec.Template.Spec.Containers[0].Image = "nginx:1.13"

		_, err = client.Update(context.Background(), deploy, v1.UpdateOptions{})
		if err != nil {
			klog.Errorf("update deployment error,err:%v", err)
		}
		return err
	})

	if err != nil {

	}
}

func prompt() {

}

func ListDeploy(client appsresv1.DeploymentInterface) {

}

func DeleteDeploy(client appsresv1.DeploymentInterface) {

}
