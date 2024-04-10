package main

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

func main() {
	// 1. 构造访问config的配置，从文件中加载，将 home目录下的 .kube/config拷贝到当前./conf/下
	config, err := clientcmd.BuildConfigFromFlags("", "C:/Users/27380/.kube/config")
	if err != nil {
		panic(err)
	}
	config.GroupVersion = &v1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.APIPath = "/api"

	// 2. 创建rest client
	client, err := rest.RESTClientFor(config)
	if err != nil {
		panic(err)
	}

	// 3. 查找命名空间dev下的pod
	var podList v1.PodList
	err = client.Get().Namespace("default").Resource("pods").Do(context.Background()).Into(&podList)
	if err != nil {
		log.Printf("get pods error:%v\n", err)
		return
	}

	fmt.Println("default pod count:", len(podList.Items))
	for _, pod := range podList.Items {
		fmt.Printf("name: %s\n", pod.Name)
	}
}
