module github.com/rueian/aerial

go 1.14

require (
	github.com/cilium/cilium v1.8.5
	github.com/googleapis/gnostic v0.5.3 // indirect
	github.com/spf13/cobra v1.0.0
	k8s.io/api v0.19.3 // indirect
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.0
	k8s.io/utils v0.0.0-20201027101359-01387209bb0d // indirect
)

replace github.com/optiopay/kafka => github.com/cilium/kafka v0.0.0-20180809090225-01ce283b732b
