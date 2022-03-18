module github.com/networkservicemesh/sdk-vpp

go 1.16

require (
	git.fd.io/govpp.git v0.3.6-0.20210927044411-385ccc0d8ba9
	github.com/edwarnicke/govpp v0.0.0-20220311182453-f32f292e0e91
	github.com/edwarnicke/serialize v1.0.7
	github.com/golang/protobuf v1.5.2
	github.com/hashicorp/go-multierror v1.1.1
	github.com/networkservicemesh/api v1.2.1-0.20220315001249-f33f8c3f2feb
	github.com/networkservicemesh/sdk v0.5.1-0.20220316101237-288caa7bbc1c
	github.com/networkservicemesh/sdk-kernel v0.0.0-20220316101641-0103343013f0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/thanhpk/randstr v1.0.4
	github.com/vishvananda/netlink v1.1.1-0.20220118170537-d6b03fdeb845
	github.com/vishvananda/netns v0.0.0-20211101163701-50045581ed74
	go.uber.org/goleak v1.1.12
	golang.org/x/sys v0.0.0-20220307203707-22a9840ba4d7
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200609130330-bd2cb7843e1b
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
)
