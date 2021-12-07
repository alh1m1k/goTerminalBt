module GoConsoleBT

go 1.16

replace GoConsoleBT/controller => ./controller

replace GoConsoleBT/collider => ./collider

replace GoConsoleBT/output => ./output

require (
	GoConsoleBT/collider v0.0.0-00010101000000-000000000000
	GoConsoleBT/controller v0.0.0-00010101000000-000000000000
	GoConsoleBT/output v0.0.0-00010101000000-000000000000
	github.com/buger/goterm v1.0.1
	github.com/buger/jsonparser v1.1.1
	github.com/eiannone/keyboard v0.0.0-20200508000154-caf4b762e807
	github.com/pkg/profile v1.6.0
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/tanema/ump v0.0.0-20171018143447-d5f261ac1de3
	github.com/xarg/gopathfinding v0.0.0-20170223193223-aefc81ce6658 // indirect
	github.com/xiaonanln/go-lockfree-pool v0.0.0-20181017030802-53ecc7b8f637
	github.com/xiaonanln/go-lockfree-queue v0.0.0-20181015150615-23113b463d4f // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
)
