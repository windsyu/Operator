package main

import (
	"bytes"
	"context"
	"html/template"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

type PodSpec struct {
	Name          string `json:"name"`
	Image         string `json:"image"`
	Namespace     string `json:"namespace"`
	ContainerName string `json:"container_name"`
}

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", "C:/Users/27380/.kube/config")
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	spec := PodSpec{
		Name:          "nginx-pod-demo",
		Image:         "nginx",
		Namespace:     "default",
		ContainerName: "nginx",
	}
	var pod corev1.Pod
	tmpl, err := ParseTemplate("./template/nginx.yaml", &spec)
	if err != nil {
		panic(err)
	}
	//将tmpl字节流按yaml解析放入pod变量中
	err = yaml.Unmarshal(tmpl, &pod)
	if err != nil {
		panic(err)
	}

	//4.创建pod
	_, err = clientset.CoreV1().Pods("default").Create(context.Background(), &pod, metav1.CreateOptions{})
	if err != nil {
		log.Printf("create pod error:%v\n", err)
		return
	}
	log.Printf("create pod success")
}

func ParseTemplate(name string, item *PodSpec) ([]byte, error) {
	tmpl, err := template.ParseFiles(name)
	if err != nil {
		return nil, err
	}
	buf := bytes.Buffer{}
	//Execute write parsed template to a buf（tmpl是template文件，item是动态装载的PodSpec项，两者会被对应装配后写入buf）
	err = tmpl.Execute(&buf, item)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
