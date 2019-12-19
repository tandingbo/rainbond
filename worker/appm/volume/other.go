// RAINBOND, Application Management Platform
// Copyright (C) 2014-2017 Goodrain Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package volume

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/goodrain/rainbond/db"
	"github.com/goodrain/rainbond/node/nodem/client"
	workerutil "github.com/goodrain/rainbond/worker/util"
	corev1 "k8s.io/api/core/v1"
)

// OtherVolume ali cloud volume struct
type OtherVolume struct {
	Base
}

// CreateVolume ceph rbd volume create volume
func (v *OtherVolume) CreateVolume(define *Define) error {
	volumeType, err := db.GetManager().VolumeTypeDao().GetVolumeTypeByType(v.svm.VolumeType)
	if err != nil {
		logrus.Errorf("get volume type by type error: %s", err.Error())
		return fmt.Errorf("validate volume capacity error")
	}
	if err := workerutil.ValidateVolumeCapacity(volumeType.CapacityValidation, v.svm.VolumeCapacity); err != nil {
		logrus.Errorf("validate volume capacity[%v] error: %s", v.svm.VolumeCapacity, err.Error())
		return err
	}
	volumeMountName := fmt.Sprintf("manual%d", v.svm.ID)
	volumeMountPath := v.svm.VolumePath
	volumeReadOnly := v.svm.IsReadOnly
	labels := v.as.GetCommonLabels(map[string]string{"volume_name": v.svm.VolumeName, "version": v.as.DeployVersion, "reclaim_policy": v.svm.ReclaimPolicy})
	annotations := map[string]string{"volume_name": v.svm.VolumeName}
	claim := newVolumeClaim(volumeMountName, volumeMountPath, v.svm.AccessMode, v.svm.VolumeType, v.svm.VolumeCapacity, labels, annotations)
	logrus.Debugf("storage class is : %s, claim value is : %s", v.svm.VolumeType, claim.GetName())
	claim.Annotations = map[string]string{
		client.LabelOS: func() string {
			if v.as.IsWindowsService {
				return "windows"
			}
			return "linux"
		}(),
	}
	v.as.SetClaim(claim)                 // store claim to appService
	statefulset := v.as.GetStatefulSet() //有状态组件
	vo := corev1.Volume{Name: volumeMountName}
	if statefulset != nil {
		statefulset.Spec.VolumeClaimTemplates = append(statefulset.Spec.VolumeClaimTemplates, *claim)
	} else {
		vo.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claim.GetName(), ReadOnly: volumeReadOnly}
		logrus.Warnf("service[%s] is not stateful, mount volume by k8s volume.PersistenVolumeClaim[%s]", v.svm.ServiceID, claim.GetName())
	}
	define.volumes = append(define.volumes, vo)

	vm := corev1.VolumeMount{
		Name:      volumeMountName,
		MountPath: volumeMountPath,
		ReadOnly:  volumeReadOnly,
	}
	define.volumeMounts = append(define.volumeMounts, vm)
	return nil
}

// CreateDependVolume create depend volume
func (v *OtherVolume) CreateDependVolume(define *Define) error {
	return nil
}
