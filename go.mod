module GoConsoleBT

go 1.16

replace GoConsoleBT/controller => ./controller

replace GoConsoleBT/collider => ./collider

replace GoConsoleBT/output => ./output

require (
	GoConsoleBT/collider v0.0.0-00010101000000-000000000000
	GoConsoleBT/controller v0.0.0-00010101000000-000000000000
	GoConsoleBT/output v0.0.0-00010101000000-000000000000
	github.com/buger/goterm v1.0.1 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/eiannone/keyboard v0.0.0-20200508000154-caf4b762e807 // indirect
	github.com/mmatczuk/go_generics v0.0.0-20181212143635-0aaa050f9bab // indirect
	github.com/pkg/profile v1.6.0 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/xiaonanln/go-lockfree-pool v0.0.0-20181017030802-53ecc7b8f637 // indirect
	github.com/xiaonanln/go-lockfree-queue v0.0.0-20181015150615-23113b463d4f // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
)
