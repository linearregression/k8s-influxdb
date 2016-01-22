/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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

// Package install installs the experimental API group, making it available as
// an option to all of the API encoding/decoding machinery.
package install

import (
	"fmt"

	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/github.com/golang/glog"

	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/api"
	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/api/latest"
	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/api/meta"
	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/api/registered"
	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/api/unversioned"
	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/apis/extensions"
	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/runtime"
	"revision.aeip.apigee.net/spatel/k8s-influxdb/influxdb/Godeps/_workspace/src/k8s.io/kubernetes/pkg/util/sets"
)

const importPrefix = "k8s.io/kubernetes/pkg/apis/extensions"

var accessor = meta.NewAccessor()

// availableVersions lists all known external versions for this group from most preferred to least preferred
var availableVersions = []unversioned.GroupVersion{v1beta1.SchemeGroupVersion}

func init() {
	registered.RegisterVersions(availableVersions...)

	externalVersions := []unversioned.GroupVersion{}
	for _, v := range availableVersions {
		if registered.IsAllowedVersion(v) {
			externalVersions = append(externalVersions, v)
		}
	}
	if len(externalVersions) == 0 {
		glog.V(4).Infof("No version is registered for group %v", extensions.GroupName)
		return
	}
	if err := registered.EnableVersions(externalVersions...); err != nil {
		glog.V(4).Infof("%v", err)
		return
	}
	if err := enableVersions(externalVersions); err != nil {
		glog.V(4).Infof("%v", err)
		return
	}
}

// TODO: enableVersions should be centralized rather than spread in each API
// group.
func enableVersions(externalVersions []unversioned.GroupVersion) error {
	addVersionsToScheme(externalVersions...)
	preferredExternalVersion := externalVersions[0]

	groupMeta := latest.GroupMeta{
		GroupVersion:  preferredExternalVersion,
		GroupVersions: externalVersions,
		Codec:         runtime.CodecFor(api.Scheme, preferredExternalVersion),
		RESTMapper:    newRESTMapper(externalVersions),
		SelfLinker:    runtime.SelfLinker(accessor),
		InterfacesFor: interfacesFor,
	}

	if err := latest.RegisterGroup(groupMeta); err != nil {
		return err
	}
	api.RegisterRESTMapper(groupMeta.RESTMapper)
	return nil
}

func newRESTMapper(externalVersions []unversioned.GroupVersion) meta.RESTMapper {
	// the list of kinds that are scoped at the root of the api hierarchy
	// if a kind is not enumerated here, it is assumed to have a namespace scope
	rootScoped := sets.NewString()

	ignoredKinds := sets.NewString()

	return api.NewDefaultRESTMapper(externalVersions, interfacesFor, importPrefix, ignoredKinds, rootScoped)
}

// interfacesFor returns the default Codec and ResourceVersioner for a given version
// string, or an error if the version is not known.
func interfacesFor(version unversioned.GroupVersion) (*meta.VersionInterfaces, error) {
	switch version {
	case v1beta1.SchemeGroupVersion:
		return &meta.VersionInterfaces{
			Codec:            v1beta1.Codec,
			ObjectConvertor:  api.Scheme,
			MetadataAccessor: accessor,
		}, nil
	default:
		g, _ := latest.Group(extensions.GroupName)
		return nil, fmt.Errorf("unsupported storage version: %s (valid: %v)", version, g.GroupVersions)
	}
}

func addVersionsToScheme(externalVersions ...unversioned.GroupVersion) {
	// add the internal version to Scheme
	extensions.AddToScheme(api.Scheme)
	// add the enabled external versions to Scheme
	for _, v := range externalVersions {
		if !registered.IsEnabledVersion(v) {
			glog.Errorf("Version %s is not enabled, so it will not be added to the Scheme.", v)
			continue
		}
		switch v {
		case v1beta1.SchemeGroupVersion:
			v1beta1.AddToScheme(api.Scheme)
		}
	}
}
