package updatechannel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	v1 "github.com/rancher/k3os/pkg/apis/k3os.io/v1"
	k3osctlv1 "github.com/rancher/k3os/pkg/generated/controllers/k3os.io/v1"
	"github.com/rancher/k3os/pkg/mode"
	"github.com/rancher/k3os/pkg/system"
	"github.com/rancher/k3os/pkg/upgrade"
	batchctlv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/batch/v1"
	corectlv1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func NewHandler(ctx context.Context, upchans k3osctlv1.UpdateChannelController, nodes corectlv1.NodeController, jobs batchctlv1.JobController) *Handler {
	return &Handler{
		ctx:      ctx,
		upchans:  upchans,
		nodes:    nodes,
		jobs:     jobs,
		nodename: os.Getenv("K3OS_OPERATOR_NODE"),
	}
}

type Handler struct {
	ctx      context.Context
	upchans  k3osctlv1.UpdateChannelController
	nodes    corectlv1.NodeController
	jobs     batchctlv1.JobController
	nodename string
}

func (h *Handler) JobOnRemove(key string, job *batchv1.Job) (*batchv1.Job, error) {
	if job == nil {
		return nil, nil
	}
	// remove finalizers, or will wrangler do that?
	return job, nil
}

func (h *Handler) JobOnChange(key string, job *batchv1.Job) (*batchv1.Job, error) {
	if job == nil {
		return nil, nil
	}
	if job.Status.CompletionTime != nil {
		uchans, err := h.upchans.Cache().List(job.Namespace, labels.Everything())
		if err != nil {
			return job, err
		}
		rebooting := false
		for _, uchan := range uchans {
			for _, unjob := range uchan.Status.Upgrading {
				// any controller can clear the slot
				if strings.HasSuffix(unjob, string(job.GetUID())) {
					if _, err = k3osctlv1.UpdateUpdateChannelOnChange(h.upchans.Updater(), h.ClearUpgrading)(key, uchan); err != nil {
						logrus.Warn(err)
					}
				}
				// reboot job node if success
				if job.Spec.Template.Spec.NodeName == h.nodename && job.Status.Succeeded > 0 && !rebooting {
					rebooting = true
					defer func(delay time.Duration) {
						logrus.Infof("upgrade has finished, rebooting in %s", delay.String())
						go system.Reboot(delay)
					}(5 * time.Second)
				}
			}
		}
	}
	return job, nil
}

func (h *Handler) OnChange(key string, res *v1.UpdateChannel) (*v1.UpdateChannel, error) {
	logrus.Debugf("### K3OS::UPDATE-CHANNEL >> OnChange >> node=%s >> key=%q >> spec=%+v >> status=%+v", h.nodename, key, res.Spec, res.Status)
	defer logrus.Debugf("### K3OS::UPDATE-CHANNEL << OnChange << node=%s << key=%q << spec=%+v << status=%+v", h.nodename, key, res.Spec, res.Status)

	if res == nil {
		return nil, nil
	}

	u := h.upchans.Updater()
	// discrete state changes ftw
	if h.ShouldPoll(key, res) && !h.IsPolling(key, res) && h.CanPoll(key, res) {
		return k3osctlv1.UpdateUpdateChannelOnChange(u, h.SetPolling)(key, res)
	}
	if h.ShouldPoll(key, res) && h.IsPolling(key, res) {
		return k3osctlv1.UpdateUpdateChannelOnChange(u, h.PollLatest)(key, res)
	}
	if h.IsPolling(key, res) {
		return k3osctlv1.UpdateUpdateChannelOnChange(u, h.ClearPolling)(key, res)
	}

	if res.Status.Polling == "" {
		if h.ShouldUpgrade(key, res) && !h.IsUpgrading(key, res) && h.CanUpgrade(key, res) {
			return k3osctlv1.UpdateUpdateChannelOnChange(u, h.SetUpgrading)(key, res)
		}
		if h.ShouldUpgrade(key, res) && h.IsUpgrading(key, res) {
			return k3osctlv1.UpdateUpdateChannelOnChange(u, h.UpgradeNode)(key, res)
		}
		if h.IsUpgrading(key, res) {
			return k3osctlv1.UpdateUpdateChannelOnChange(u, h.ClearUpgrading)(key, res)
		}
	}

	return res, nil
}

func (h *Handler) OnRemove(key string, res *v1.UpdateChannel) (*v1.UpdateChannel, error) {
	logrus.Debugf("### K3OS::UPDATE-CHANNEL >> OnRemove >> node=%s >> key=%q >> spec=%+v >> status=%+v", h.nodename, key, res.Spec, res.Status)
	defer logrus.Debugf("### K3OS::UPDATE-CHANNEL << OnRemove << node=%s << key=%q << spec=%+v << status=%+v", h.nodename, key, res.Spec, res.Status)

	return res, nil
}

func (h *Handler) CanPoll(key string, res *v1.UpdateChannel) bool {
	return res.Status.Polling == ""
}
func (h *Handler) IsPolling(key string, res *v1.UpdateChannel) bool {
	return res.Status.Polling == h.nodename
}
func (h *Handler) ShouldPoll(key string, res *v1.UpdateChannel) bool {
	v := strings.TrimSpace(strings.ToLower(res.Spec.Version))
	return (v == "" || v == "latest")
}

func (h *Handler) CanUpgrade(key string, res *v1.UpdateChannel) bool {
	return res.Spec.Concurrency > len(res.Status.Upgrading)
}
func (h *Handler) ShouldUpgrade(key string, res *v1.UpdateChannel) bool {
	mode, err := mode.Get()
	if err != nil {
		return false
	}
	if strings.TrimSpace(mode) == "live" {
		return false
	}
	current, err := os.Readlink("/k3os/system/k3os/current")
	if err != nil {
		return false
	}
	return filepath.Base(current) != res.Spec.Version
}
func (h *Handler) IsUpgrading(key string, res *v1.UpdateChannel) bool {
	u := append(res.Status.Upgrading[:0:0], res.Status.Upgrading...)
	sort.Strings(u)
	x := sort.SearchStrings(u, h.nodename)
	if x >= len(u) {
		return false
	}
	return strings.HasPrefix(u[x], h.nodename)
}

func (h *Handler) SetPolling(key string, res *v1.UpdateChannel) (*v1.UpdateChannel, error) {
	logrus.Debugf("### K3OS::UPDATE-CHANNEL >> UpdatePolling >> node=%s >> key=%q >> spec=%+v >> status=%+v", h.nodename, key, res.Spec, res.Status)
	defer logrus.Debugf("### K3OS::UPDATE-CHANNEL << UpdatePolling << node=%s << key=%q << spec=%+v << status=%+v", h.nodename, key, res.Spec, res.Status)

	res.Status.Polling = h.nodename
	return res, nil
}

func (h *Handler) ClearPolling(key string, res *v1.UpdateChannel) (*v1.UpdateChannel, error) {
	logrus.Debugf("### K3OS::UPDATE-CHANNEL >> ClearPolling >> node=%s >> key=%q >> spec=%+v >> status=%+v", h.nodename, key, res.Spec, res.Status)
	defer logrus.Debugf("### K3OS::UPDATE-CHANNEL << ClearPolling << node=%s << key=%q << spec=%+v << status=%+v", h.nodename, key, res.Spec, res.Status)

	res.Status.Polling = ""
	return res, nil
}

func (h *Handler) SetUpgrading(key string, res *v1.UpdateChannel) (*v1.UpdateChannel, error) {
	logrus.Debugf("### K3OS::UPDATE-CHANNEL >> UpdateUpgrading >> node=%s >> key=%q >> spec=%+v >> status=%+v", h.nodename, key, res.Spec, res.Status)
	defer logrus.Debugf("### K3OS::UPDATE-CHANNEL << UpdateUpgrading << node=%s << key=%q << spec=%+v << status=%+v", h.nodename, key, res.Spec, res.Status)

	res.Status.Upgrading = append(res.Status.Upgrading, h.nodename)
	return res, nil
}

func (h *Handler) ClearUpgrading(key string, res *v1.UpdateChannel) (*v1.UpdateChannel, error) {
	logrus.Debugf("### K3OS::UPDATE-CHANNEL >> ClearUpgrading >> node=%s >> key=%q >> spec=%+v >> status=%+v", h.nodename, key, res.Spec, res.Status)
	defer logrus.Debugf("### K3OS::UPDATE-CHANNEL << ClearUpgrading << node=%s << key=%q << spec=%+v << status=%+v", h.nodename, key, res.Spec, res.Status)

	upgrading := []string{}
	for _, u := range res.Status.Upgrading {
		if !strings.HasPrefix(u, h.nodename) {
			upgrading = append(upgrading, u)
		}
	}
	res.Status.Upgrading = upgrading
	return res, nil
}

func (h *Handler) PollLatest(key string, res *v1.UpdateChannel) (*v1.UpdateChannel, error) {
	logrus.Debugf("### K3OS::UPDATE-CHANNEL >> PollLatest >> node=%s >> key=%q >> spec=%+v >> status=%+v", h.nodename, key, res.Spec, res.Status)
	defer logrus.Debugf("### K3OS::UPDATE-CHANNEL << PollLatest << node=%s << key=%q << spec=%+v << status=%+v", h.nodename, key, res.Spec, res.Status)

	channel, err := upgrade.NewChannel(res.Spec.URL)
	if err != nil {
		return res, err
	}
	latest, err := channel.Latest()
	if err != nil {
		return res, err
	}
	res.Spec.Version = latest.Name
	return res, nil
}

func (h *Handler) UpgradeNode(key string, res *v1.UpdateChannel) (*v1.UpdateChannel, error) {
	logrus.Debugf("### K3OS::UPDATE-CHANNEL >> UpgradeNode >> node=%s >> key=%q >> spec=%+v >> status=%+v", h.nodename, key, res.Spec, res.Status)
	defer logrus.Debugf("### K3OS::UPDATE-CHANNEL << UpgradeNode << node=%s << key=%q << spec=%+v << status=%+v", h.nodename, key, res.Spec, res.Status)

	for i, upgrading := range res.Status.Upgrading {
		if upgrading == h.nodename {
			job, err := h.createUpgradeJob(res)
			if err != nil {
				return res, err
			}
			res.Status.Upgrading[i] = fmt.Sprintf("%s:%s", upgrading, job.GetUID())
			return res, nil
		}
	}
	return res, nil
}

func (h *Handler) createUpgradeJob(res *v1.UpdateChannel) (*batchv1.Job, error) {
	var (
		deadlineSeconds   = int64(180)
		hostPathDirectory = corev1.HostPathDirectory
		hostPathFile      = corev1.HostPathFile
		privileged        = true
		upgradeKernel     = false
		upgradeRootFS     = true
		name              = h.nodename + `-upgrade`
	)
	debug, _ := strconv.ParseBool(os.Getenv("K3OS_DEBUG"))
	if inf, err := os.Stat("/k3os/system/kernel"); err == nil && inf.IsDir() {
		upgradeKernel = true
	}
	return h.jobs.Create(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "k3os-system",
			Annotations: map[string]string{
				"k3os.io/version": res.Spec.Version,
			},
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &deadlineSeconds,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeName:           h.nodename,
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: `k3os-operator`,
					Containers: []corev1.Container{
						{
							Image: "k8s.gcr.io/pause",
							Name:  name,
							Command: []string{
								"k3os", "operator", "upgrade",
								"--channel=" + res.Spec.URL,
								"--version=" + res.Spec.Version,
								"--remount",
								"--kernel=" + strconv.FormatBool(upgradeKernel),
								"--rootfs=" + strconv.FormatBool(upgradeRootFS),
								"--sync",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "K3OS_DEBUG",
									Value: fmt.Sprintf("%v", debug),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										corev1.Capability("CAP_SYS_BOOT"),
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "etc-os-release",
									MountPath: "/etc/os/release",
									ReadOnly:  true,
								},
								{
									Name:      "etc-ssl",
									MountPath: "/etc/ssl",
									ReadOnly:  true,
								},
								{
									Name:      "k3os-exe",
									MountPath: "/sbin/k3os",
									ReadOnly:  true,
								},
								{
									Name:      "k3os-system",
									MountPath: "/k3os/system",
									ReadOnly:  false,
								},
								{
									Name:      "k3os-temp",
									MountPath: "/tmp",
									ReadOnly:  false,
								},
								{
									Name:      "run-k3os",
									MountPath: "/run/k3os",
									ReadOnly:  false,
								},
								{
									Name:      "var-lib-rancher",
									MountPath: "/var/lib/rancher",
									ReadOnly:  false,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: `etc-os-release`,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/os-release",
									Type: &hostPathFile,
								},
							},
						},
						{
							Name: `etc-ssl`,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/ssl",
									Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: `k3os-exe`,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sbin/k3os",
									Type: &hostPathFile,
								},
							},
						},
						{
							Name: `k3os-system`,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/k3os/system",
									Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: `k3os-temp`,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/tmp",
									Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: `run-k3os`,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/k3os",
									Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: `var-lib-rancher`,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/rancher",
									Type: &hostPathDirectory,
								},
							},
						},
					},
				},
			},
		},
	})
}
