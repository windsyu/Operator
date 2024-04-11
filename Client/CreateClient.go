package main

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

// clientset 的演示
func main() {
	// 1. 构造访问config的配置，从文件中加载，将 home目录下的 .kube/config拷贝到当前./conf/下
	config, err := clientcmd.BuildConfigFromFlags("", "C:/Users/27380/.kube/config")
	if err != nil {
		panic(err)
	}

	// 2. 创建clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	//3.operate resource objects
	PodList, err := clientset.CoreV1().Pods("default").List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Print("list pods err:%v\n", err)
		return
	}
	fmt.Println("default pod count:", len(PodList.Items))
	for _, pod := range PodList.Items {
		fmt.Printf("name:%s\n", pod.Name)
	}
}
