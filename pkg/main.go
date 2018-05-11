package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/api/core/v1"
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

	default_duration, err := time.ParseDuration("4h")
	if err != nil {
		panic(err.Error())
	}

	var (
		namespace  = flag.String("namespace", "", "filter by namespace (defaults to all)")
		sort       = flag.String("sort", "CpuReq", "field to sort by")
		reverse    = flag.Bool("reverse", false, "reverse sort output")
		historical = flag.Bool("historical", false, "show historical info")
		duration   = flag.Duration("duration", default_duration, "specify the duration")
		mem_only   = flag.Bool("mem", false, "show historical memory info")
		cpu_only   = flag.Bool("cpu", false, "show historical cpu info")
		project    = flag.String("project", "", "Project id")
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

	k := NewKubeClient(clientset)

	if *historical {

		if *project == "" {
			fmt.Printf("-project is required for historical data\n")
			os.Exit(1)
		}

		var resourceName v1.ResourceName

		if *mem_only {
			resourceName = v1.ResourceMemory
		} else if *cpu_only {
			resourceName = v1.ResourceCPU
		} else {
			panic("Unknown metric type")
		}

		k.historical(*project, *namespace, resourceName, *duration, *sort, *reverse)

	} else {

		r := ContainerResources{}

		if !r.Validate(*sort) {
			fmt.Printf("\"%s\" is not a valid field. Possible values are:\n\n%s\n", *sort, strings.Join(getFields(r), ", "))
			os.Exit(1)
		}

		k.resourceUsage(*namespace, *sort, *reverse)
	}
}
