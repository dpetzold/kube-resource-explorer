package main

func (k *KubeClient) resourceUsage(namespace, sort string, reverse bool) {

	resources, err := k.GetContainerResources(namespace)
	if err != nil {
		panic(err.Error())
	}

	capacity, err := k.GetClusterCapacity()
	if err != nil {
		panic(err.Error())
	}

	rows := FormatResourceUsage(capacity, resources, sort, reverse)
	PrintResourceUsage(rows)
}
