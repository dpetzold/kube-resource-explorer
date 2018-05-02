package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

var GitCommit string

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func main() {

	var (
		namespace  = flag.String("namespace", "", "filter by namespace (defaults to all)")
		sort       = flag.String("sort", "CpuReq", "field to sort by")
		reverse    = flag.Bool("reverse", false, "reverse sort output")
		kubeconfig *string
	)

	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	r := ResourceAllocation{}

	if !r.Validate(*sort) {
		fmt.Printf("\"%s\" is not a valid field. Possible values are:\n\n%s\n", *sort, strings.Join(r.getFields(), ", "))
		os.Exit(1)
	}

	rl := NewResourceLister(clientset)
	resources, err := rl.ListResources(*namespace)
	if err != nil {
		panic(err.Error())
	}

	rl.Print(resources, *sort, *reverse)
}
