# Implementation notes

There are a number of ways to implement kernel interfaces with vpp.

If /dev/vhost-net is available, tapv2 is likely to be the most efficient, but its not always available.

If tapv2 is not an option, some variation of binding vpp to one end of a vethpair can be used.
Currently, using af_packet for this purpose is implemented here, but af_xdp is coming up fast,
and so the implementation was done in a way that leaves options for it.

There is a current division of responsiblity between the packages:

- **github.com/networkservicemesh/sdk-vpp/pkg/mechanism/kernel** - provides the selection between tapv2 
  and veth pair strategies depending on what is available.  It also moves the resulting kernel interface
  into the correct netns, names it correctly, applies an 'alias' to it, and turns the interface 'up'
  
  - **github.com/networkservicemesh/sdk-vpp/pkg/mechanism/kernelvap** - configures vpp to provide a kernel interface via tapv2.
  - **github.com/networkservicemesh/sdk-vpp/pkg/mechanism/kernelvethpair** - creates a veth pair, names the peerLink 
    side, applies an 'alias' to the peerLink side, and 'up's the peerLink side
    - **github.com/networkservicemesh/sdk-vpp/pkg/mechanism/kernelvethpair/afpacket** - attaches vpp using af_packet
      to the peerLink side of the veth pair

In the future, a **github.com/networkservicemesh/sdk-vpp/pkg/mechanism/kernelvethpair/af_xdp** is anticipated
to implement using af_xdp to bind vpp to a kernelvethpair.
