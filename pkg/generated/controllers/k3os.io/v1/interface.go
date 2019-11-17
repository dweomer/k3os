/*
Copyright 2019 Rancher Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by main. DO NOT EDIT.

package v1

import (
	v1 "github.com/rancher/k3os/pkg/apis/k3os.io/v1"
	clientset "github.com/rancher/k3os/pkg/generated/clientset/versioned/typed/k3os.io/v1"
	informers "github.com/rancher/k3os/pkg/generated/informers/externalversions/k3os.io/v1"
	"github.com/rancher/wrangler/pkg/generic"
)

type Interface interface {
	UpdateChannel() UpdateChannelController
}

func New(controllerManager *generic.ControllerManager, client clientset.K3osV1Interface,
	informers informers.Interface) Interface {
	return &version{
		controllerManager: controllerManager,
		client:            client,
		informers:         informers,
	}
}

type version struct {
	controllerManager *generic.ControllerManager
	informers         informers.Interface
	client            clientset.K3osV1Interface
}

func (c *version) UpdateChannel() UpdateChannelController {
	return NewUpdateChannelController(v1.SchemeGroupVersion.WithKind("UpdateChannel"), c.controllerManager, c.client, c.informers.UpdateChannels())
}