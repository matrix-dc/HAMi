/*
Copyright 2024 The HAMi Authors.

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

package nvidia

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/Project-HAMi/HAMi/pkg/api"
	"github.com/Project-HAMi/HAMi/pkg/scheduler/config"
	"github.com/Project-HAMi/HAMi/pkg/util"
	"github.com/Project-HAMi/HAMi/pkg/util/nodelock"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

const (
	HandshakeAnnos      = "hami.io/node-handshake"
	RegisterAnnos       = "hami.io/node-nvidia-register"
	NvidiaGPUDevice     = "NVIDIA"
	NvidiaGPUCommonWord = "GPU"
	GPUInUse            = "nvidia.com/use-gputype"
	GPUNoUse            = "nvidia.com/nouse-gputype"
	NumaBind            = "nvidia.com/numa-bind"
	NodeLockNvidia      = "hami.io/mutex.lock"
	// GPUUseUUID is user can use specify GPU device for set GPU UUID.
	GPUUseUUID = "nvidia.com/use-gpuuuid"
	// GPUNoUseUUID is user can not use specify GPU device for set GPU UUID.
	GPUNoUseUUID = "nvidia.com/nouse-gpuuuid"
)

var (
	ResourceName          string
	ResourceMem           string
	ResourceCores         string
	ResourceMemPercentage string
	ResourcePriority      string
	DebugMode             bool
	OverwriteEnv          bool
)

type NvidiaGPUDevices struct {
}

func InitNvidiaDevice() *NvidiaGPUDevices {
	util.InRequestDevices[NvidiaGPUDevice] = "hami.io/vgpu-devices-to-allocate"
	util.SupportDevices[NvidiaGPUDevice] = "hami.io/vgpu-devices-allocated"
	util.HandshakeAnnos[NvidiaGPUDevice] = HandshakeAnnos
	return &NvidiaGPUDevices{}
}

func (dev *NvidiaGPUDevices) ParseConfig(fs *flag.FlagSet) {
	fs.StringVar(&ResourceName, "resource-name", "nvidia.com/gpu", "resource name")
	fs.StringVar(&ResourceMem, "resource-mem", "nvidia.com/gpumem", "gpu memory to allocate")
	fs.StringVar(&ResourceMemPercentage, "resource-mem-percentage", "nvidia.com/gpumem-percentage", "gpu memory fraction to allocate")
	fs.StringVar(&ResourceCores, "resource-cores", "nvidia.com/gpucores", "cores percentage to use")
	fs.StringVar(&ResourcePriority, "resource-priority", "vgputaskpriority", "vgpu task priority 0 for high and 1 for low")
	fs.BoolVar(&OverwriteEnv, "overwrite-env", false, "If set NVIDIA_VISIBLE_DEVICES=none to pods with no-gpu allocation")
}

func (dev *NvidiaGPUDevices) NodeCleanUp(nn string) error {
	return util.MarkAnnotationsToDelete(NvidiaGPUDevice, nn)
}

func (dev *NvidiaGPUDevices) CheckHealth(devType string, n *corev1.Node) (bool, bool) {
	return util.CheckHealth(devType, n)
}

func (dev *NvidiaGPUDevices) LockNode(n *corev1.Node, p *corev1.Pod) error {
	found := false
	for _, val := range p.Spec.Containers {
		if (dev.GenerateResourceRequests(&val).Nums) > 0 {
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	return nodelock.LockNode(n.Name, NodeLockNvidia)
}

func (dev *NvidiaGPUDevices) ReleaseNodeLock(n *corev1.Node, p *corev1.Pod) error {
	found := false
	for _, val := range p.Spec.Containers {
		if (dev.GenerateResourceRequests(&val).Nums) > 0 {
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	return nodelock.ReleaseNodeLock(n.Name, NodeLockNvidia)
}

func (dev *NvidiaGPUDevices) GetNodeDevices(n corev1.Node) ([]*api.DeviceInfo, error) {
	devEncoded, ok := n.Annotations[RegisterAnnos]
	if !ok {
		return []*api.DeviceInfo{}, errors.New("annos not found " + RegisterAnnos)
	}
	nodedevices, err := util.DecodeNodeDevices(devEncoded)
	if err != nil {
		klog.ErrorS(err, "failed to decode node devices", "node", n.Name, "device annotation", devEncoded)
		return []*api.DeviceInfo{}, err
	}
	if len(nodedevices) == 0 {
		klog.InfoS("no gpu device found", "node", n.Name, "device annotation", devEncoded)
		return []*api.DeviceInfo{}, errors.New("no gpu found on node")
	}

	devDecoded := util.EncodeNodeDevices(nodedevices)
	klog.V(5).InfoS("nodes device information", "node", n.Name, "nodedevices", devDecoded)
	return nodedevices, nil
}

// TODO 暂时只支持nvidia.com/gpu-xxx: 2，这一种配置
// 当 nvidia.com/gpu-xxx: 0的时候，env要设置NVIDIA_VISIBLE_DEVICES: none
func (dev *NvidiaGPUDevices) MutateAdmission(ctr *corev1.Container) (bool, error) {
	var resourceNameOK bool
	var quantity resource.Quantity
	/*gpu related */
	priority, ok := ctr.Resources.Limits[corev1.ResourceName(ResourcePriority)]
	if ok {
		ctr.Env = append(ctr.Env, corev1.EnvVar{
			Name:  api.TaskPriority,
			Value: fmt.Sprint(priority.Value()),
		})
	}
	// scheduler 的ResourceName是包含多种device resourceName信息
	// 需要逐一从pod container的limits中去匹配
	// 从scheduler找到任意满足的resourceName则认为true
	resourceNames := strings.Split(ResourceName, ";")
	for _, resName := range resourceNames {
		quantity, resourceNameOK = ctr.Resources.Limits[corev1.ResourceName(resName)]
		if resourceNameOK {
			nvidiaVisibleDevices := false
			if n, ok := quantity.AsInt64(); ok && n > 0 {
				nvidiaVisibleDevices = true
			}
			if !nvidiaVisibleDevices && OverwriteEnv {
				ctr.Env = append(ctr.Env, corev1.EnvVar{
					Name:  "NVIDIA_VISIBLE_DEVICES",
					Value: "none",
				})
			}
			return resourceNameOK, nil
		}
	}

	// 由于scheduler实现为多deviceType支持，因此不能用scheduler的ResourceName来给pod container resource填入默认值
	// 因此这部分代码注释掉，意味着没有任何resourceName的调度都无效
	// _, resourceCoresOK := ctr.Resources.Limits[corev1.ResourceName(ResourceCores)]
	// _, resourceMemOK := ctr.Resources.Limits[corev1.ResourceName(ResourceMem)]
	// _, resourceMemPercentageOK := ctr.Resources.Limits[corev1.ResourceName(ResourceMemPercentage)]

	// if resourceCoresOK || resourceMemOK || resourceMemPercentageOK {
	// 	if config.DefaultResourceNum > 0 {
	// 		ctr.Resources.Limits[corev1.ResourceName(ResourceName)] = *resource.NewQuantity(int64(config.DefaultResourceNum), resource.BinarySI)
	// 		resourceNameOK = true
	// 	}
	// }

	// 这段代码主要是为了解决当nvidia.com/gpu-<type> 是 0，设置container内GPU全部不可见 MatrixDC - 2024 July - Sprint 3 task 84
	if !resourceNameOK && OverwriteEnv {
		ctr.Env = append(ctr.Env, corev1.EnvVar{
			Name:  "NVIDIA_VISIBLE_DEVICES",
			Value: "none",
		})
	}
	return resourceNameOK, nil
}

func checkGPUtype(annos map[string]string, resourceGpuType, cardtype string) bool {
	cardtype = strings.ToUpper(cardtype)
	if inuse, ok := annos[GPUInUse]; ok {
		useTypes := strings.Split(inuse, ",")
		if !ContainsSliceFunc(useTypes, func(useType string) bool {
			return strings.Contains(cardtype, strings.ToUpper(useType))
		}) {
			return false
		}
	}
	if unuse, ok := annos[GPUNoUse]; ok {
		unuseTypes := strings.Split(unuse, ",")
		if ContainsSliceFunc(unuseTypes, func(unuseType string) bool {
			return strings.Contains(cardtype, strings.ToUpper(unuseType))
		}) {
			return false
		}
	}
	// resourceGpuType来源于pod container resource的limits/requests信息
	// eg: "nvidia.com/gpu-h100", 则resourceGpuType=h100
	// 在目标node的device cardtype中 查到了resourceGpuType，则判定node满足条件
	if resourceGpuType != "" {
		return strings.Contains(cardtype, strings.ToUpper(resourceGpuType))
	}
	return true
}

func ContainsSliceFunc[S ~[]E, E any](s S, match func(E) bool) bool {
	for _, e := range s {
		if match(e) {
			return true
		}
	}
	return false
}

func assertNuma(annos map[string]string) bool {
	numabind, ok := annos[NumaBind]
	if ok {
		enforce, err := strconv.ParseBool(numabind)
		if err == nil && enforce {
			return true
		}
	}
	return false
}

func (dev *NvidiaGPUDevices) CheckType(annos map[string]string, d util.DeviceUsage, n util.ContainerDeviceRequest) (bool, bool, bool) {
	if strings.Compare(n.Type, NvidiaGPUDevice) == 0 {
		return true, checkGPUtype(annos, n.GpuType, d.Type), assertNuma(annos)
	}
	return false, false, false
}

func (dev *NvidiaGPUDevices) CheckUUID(annos map[string]string, d util.DeviceUsage) bool {
	userUUID, ok := annos[GPUUseUUID]
	if ok {
		klog.V(5).Infof("check uuid for nvidia user uuid [%s], device id is %s", userUUID, d.ID)
		// use , symbol to connect multiple uuid
		userUUIDs := strings.Split(userUUID, ",")
		for _, uuid := range userUUIDs {
			if d.ID == uuid {
				return true
			}
		}
		return false
	}

	noUserUUID, ok := annos[GPUNoUseUUID]
	if ok {
		klog.V(5).Infof("check uuid for nvidia not user uuid [%s], device id is %s", noUserUUID, d.ID)
		// use , symbol to connect multiple uuid
		noUserUUIDs := strings.Split(noUserUUID, ",")
		for _, uuid := range noUserUUIDs {
			if d.ID == uuid {
				return false
			}
		}
		return true
	}

	return true
}

func (dev *NvidiaGPUDevices) PatchAnnotations(annoinput *map[string]string, pd util.PodDevices) map[string]string {
	devlist, ok := pd[NvidiaGPUDevice]
	if ok && len(devlist) > 0 {
		deviceStr := util.EncodePodSingleDevice(devlist)
		(*annoinput)[util.InRequestDevices[NvidiaGPUDevice]] = deviceStr
		(*annoinput)[util.SupportDevices[NvidiaGPUDevice]] = deviceStr
		klog.V(5).Infof("pod add notation key [%s], values is [%s]", util.InRequestDevices[NvidiaGPUDevice], deviceStr)
		klog.V(5).Infof("pod add notation key [%s], values is [%s]", util.SupportDevices[NvidiaGPUDevice], deviceStr)
	}
	return *annoinput
}

func (dev *NvidiaGPUDevices) GenerateResourceRequests(ctr *corev1.Container) util.ContainerDeviceRequest {

	// var
	var v resource.Quantity
	var ok bool
	var gpuType string

	// scheduler 的ResourceName是包含多种device resourceName信息
	// 需要逐一从pod container的Limits/Requests中去匹配
	// 并找出对应的gpuType，eg: h100/h20等或者为空
	resourceNames := strings.Split(ResourceName, ";")
	for _, resName := range resourceNames {
		resourceName := corev1.ResourceName(resName)
		v, ok = ctr.Resources.Limits[resourceName]
		if ok {
			gpuType = util.GetGpuTypeFromResourceName(resName)
			break
		} else {
			v, ok = ctr.Resources.Requests[resourceName]
			if ok {
				gpuType = util.GetGpuTypeFromResourceName(resName)
				break
			}
		}
	}

	resourceMem := corev1.ResourceName(ResourceMem)
	resourceMemPercentage := corev1.ResourceName(ResourceMemPercentage)
	resourceCores := corev1.ResourceName(ResourceCores)
	if ok {
		if n, ok := v.AsInt64(); ok {
			memnum := 0
			mem, ok := ctr.Resources.Limits[resourceMem]
			if !ok {
				mem, ok = ctr.Resources.Requests[resourceMem]
			}
			if ok {
				memnums, ok := mem.AsInt64()
				if ok {
					memnum = int(memnums)
				}
			}
			mempnum := int32(101)
			mem, ok = ctr.Resources.Limits[resourceMemPercentage]
			if !ok {
				mem, ok = ctr.Resources.Requests[resourceMemPercentage]
			}
			if ok {
				mempnums, ok := mem.AsInt64()
				if ok {
					mempnum = int32(mempnums)
				}
			}
			if mempnum == 101 && memnum == 0 {
				if config.DefaultMem != 0 {
					memnum = int(config.DefaultMem)
				} else {
					mempnum = 100
				}
			}
			corenum := config.DefaultCores
			core, ok := ctr.Resources.Limits[resourceCores]
			if !ok {
				core, ok = ctr.Resources.Requests[resourceCores]
			}
			if ok {
				corenums, ok := core.AsInt64()
				if ok {
					corenum = int32(corenums)
				}
			}
			return util.ContainerDeviceRequest{
				Nums:             int32(n),
				Type:             NvidiaGPUDevice,
				GpuType:          gpuType,
				Memreq:           int32(memnum),
				MemPercentagereq: int32(mempnum),
				Coresreq:         int32(corenum),
			}
		}
	}
	return util.ContainerDeviceRequest{}
}
