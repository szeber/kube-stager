//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2023.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StagingSite) DeepCopyInto(out *StagingSite) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StagingSite.
func (in *StagingSite) DeepCopy() *StagingSite {
	if in == nil {
		return nil
	}
	out := new(StagingSite)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StagingSite) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StagingSiteList) DeepCopyInto(out *StagingSiteList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]StagingSite, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StagingSiteList.
func (in *StagingSiteList) DeepCopy() *StagingSiteList {
	if in == nil {
		return nil
	}
	out := new(StagingSiteList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StagingSiteList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StagingSiteService) DeepCopyInto(out *StagingSiteService) {
	*out = *in
	if in.ResourceOverrides != nil {
		in, out := &in.ResourceOverrides, &out.ResourceOverrides
		*out = make(map[string]corev1.ResourceRequirements, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.ExtraEnvs != nil {
		in, out := &in.ExtraEnvs, &out.ExtraEnvs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.CustomTemplateValues != nil {
		in, out := &in.CustomTemplateValues, &out.CustomTemplateValues
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StagingSiteService.
func (in *StagingSiteService) DeepCopy() *StagingSiteService {
	if in == nil {
		return nil
	}
	out := new(StagingSiteService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StagingSiteServiceStatus) DeepCopyInto(out *StagingSiteServiceStatus) {
	*out = *in
	in.DeploymentStatus.DeepCopyInto(&out.DeploymentStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StagingSiteServiceStatus.
func (in *StagingSiteServiceStatus) DeepCopy() *StagingSiteServiceStatus {
	if in == nil {
		return nil
	}
	out := new(StagingSiteServiceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StagingSiteSpec) DeepCopyInto(out *StagingSiteSpec) {
	*out = *in
	out.DisableAfter = in.DisableAfter
	out.DeleteAfter = in.DeleteAfter
	if in.DailyBackupWindowHour != nil {
		in, out := &in.DailyBackupWindowHour, &out.DailyBackupWindowHour
		*out = new(int32)
		**out = **in
	}
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make(map[string]StagingSiteService, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StagingSiteSpec.
func (in *StagingSiteSpec) DeepCopy() *StagingSiteSpec {
	if in == nil {
		return nil
	}
	out := new(StagingSiteSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StagingSiteStatus) DeepCopyInto(out *StagingSiteStatus) {
	*out = *in
	if in.LastAppliedConfiguration != nil {
		in, out := &in.LastAppliedConfiguration, &out.LastAppliedConfiguration
		*out = (*in).DeepCopy()
	}
	if in.DisableAt != nil {
		in, out := &in.DisableAt, &out.DisableAt
		*out = (*in).DeepCopy()
	}
	if in.DeleteAt != nil {
		in, out := &in.DeleteAt, &out.DeleteAt
		*out = (*in).DeepCopy()
	}
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make(map[string]StagingSiteServiceStatus, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.LastBackupTime != nil {
		in, out := &in.LastBackupTime, &out.LastBackupTime
		*out = (*in).DeepCopy()
	}
	if in.NextBackupTime != nil {
		in, out := &in.NextBackupTime, &out.NextBackupTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StagingSiteStatus.
func (in *StagingSiteStatus) DeepCopy() *StagingSiteStatus {
	if in == nil {
		return nil
	}
	out := new(StagingSiteStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TimeInterval) DeepCopyInto(out *TimeInterval) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TimeInterval.
func (in *TimeInterval) DeepCopy() *TimeInterval {
	if in == nil {
		return nil
	}
	out := new(TimeInterval)
	in.DeepCopyInto(out)
	return out
}