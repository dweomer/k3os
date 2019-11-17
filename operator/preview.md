# Introducing the k3OS Operator
# :construction: Work-In-Progress :construction:

Initially focused with providing a kubernetes-native upgrade experience, this change-set enhances the k3os multi-call binary to provide:
- `k3os ops agent` custom resource and node controller
- `k3os ops upgrade` perform rootfs and kernel upgrades

The `k3os-upgrade-rootfs` and `k3os-upgrade-kernel` scripts have been updated to leverage `k3os ops upgrade` CLI.

## The Custom Resource
This change-set also introduces a new custom resource, the `UpdateChannel` a.k.a. `upchan`:
```sh
$ kubectl describe upchan -A
Name:         github-releases
Namespace:    k3os-system
Labels:       <none>
Annotations:  k3os.io/node: k3os-21702
API Version:  k3os.io/v1
Kind:         UpdateChannel
Metadata:
  Creation Timestamp:  2019-11-17T02:25:56Z
  Finalizers:
    wrangler.cattle.io/k3os-operator
  Generation:        1
  Resource Version:  377
  Self Link:         /apis/k3os.io/v1/namespaces/k3os-system/updatechannels/github-releases
  UID:               9bd4ecf3-44ea-4d69-b720-09e01b20ad76
Spec:
  Concurrency:  1
  URL:          github-releases://dweomer/k3os
  Version:      v0.7.0-dweomer1
Status:
Events:  <none>
```

## The Resource Controller
Controlling this resource is a DaemonSet running on every `k3os` node in the cluster. It watches for changes on `UpdateChannel.Spec.Version` and if a node's installed version differs the controller will attempt to take up one of the `UpdateChannel.Status.Updating` slots (max of `UpdateChannel.Spec.Concurrency`) and when successful will schedule a batch `Job` that invokes `k3os ops upgrade`. The controller will watch for this job to finish, when it does it will free up the `UpdateChannel.Status.Updating` slot and schedule a reboot, via goroutine, on a delay of 5 seconds.

Additionally, the controller will notice when `UpdateChannel.Spec.Version` is `latest` (or empty) and attempt to poll for the latest release. As there is only one `UpdateChannel.Status.Polling` slot, only one node will poll at a time and if there are any polling nodes updates will not be triggered.

# WARNING, PREVIEW, EXPECT TO RE-IMAGE NODES
## This preview should be considered early alpha quality at best.
### Do not run it on nodes that you cannot stand to lose any and all data from.

# :construction: Work-In-Progress :construction:

# Kicking the Tires

So, you still want to give it a spin? See below for the requirements to replicate the embedded asciinema cast above.

## Requirements
- Linux or Mac
- Docker (18.09 or better)
- Go 1.12.x
- Qemu (tested with 4.1 on macOS Mojave, 2.11.1 on Ubuntu Bionic)

  On Linux, make sure you have permissions to the `/dev/kvm` device.
  If you do not you will get a fully emulated VM which is very slow.

## Steps
- clone the repo at the first preview release:
  ```
  git clone -b v0.7.0-dweomer1 https://github.com/dweomer/k3os.git
  ```
- build it
  ```
  cd k3os
  git tag --delete v0.7.0-dweomer2 # prevents improper versioning
  make
  ```
- create a new vm (if you prefer to boot from the iso, that works too)
  ```
  # this script specifies the kernel and initrd to use (super fast boot, avoids grub)
  # which means that you will not see your kernel upgraded after rebooting even though
  # the kernel was updated on disk. see /k3os/system/kernel
  ./scripts/run-qemu \
    k3os.mode=install \
    k3os.install.device=/dev/vda \
    k3os.install.silent \
    k3os.password=rancher
  ```
- run the vm (if you prefer to boot from the iso, that works too)
  ```
  # login with username rancher, password rancher
  ./scripts/run-qemu k3os.password=rancher
  ```
- operating the operator
  ```
  # after logging in, you can poke around with kubectl
  kubectl -n k3os-system get all

  # you should see the daemonset first, followed by it's pod which will create the update channel
  # this can take anywhere from 10-90 seconds after boot
  kubectl -n k3os-system describe upchan

  # to see the logs for the pod
  kubectl -n k3os-system logs -l name=k3os-operator

  # let's force an update
  kubectl -n k3os-system \
  patch upchan github-releases --type='json' \
  --patch='[{"op":"replace","path":"/spec/version","value":"latest"}]'

  # check the update channel
  kubectl -n k3os-system describe upchans
  ```
