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

func stringInSlice(s string, slice []string) bool {
	for _, v := range slice {
		if s == v {
			return true
		}
	}
	return false
}

func main() {

	var (
		namespace  = flag.String("namespace", "", "filter by namespace (defaults to all)")
		field      = flag.String("field", "CpuReq", "field to sort by (defaults to CpuReq)")
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

	r := ResourceUsage{}
	validFields := r.getFields()

	if !stringInSlice(*field, validFields) {
		fmt.Printf("\"%s\" is not a valid field. Possible values are:\n\n%s\n", *field, strings.Join(validFields, ", "))
		os.Exit(1)
	}

	rl := NewResourceLister(clientset)
	rl.ListResources(*namespace, *field, *reverse)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
