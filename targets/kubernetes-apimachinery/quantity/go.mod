module pbtlab/quantity

go 1.27.0

require (
	gopkg.in/inf.v0 v0.9.1
	k8s.io/apimachinery v0.0.0
	pgregory.net/rapid v1.2.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
)

replace k8s.io/apimachinery => ../apimachinery
