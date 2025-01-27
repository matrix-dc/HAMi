# install scheduler use hwameistor scheduler as default
# 
helm upgrade --install  hami-scheduler  hami  --set scheduler.kubeScheduler.useDefault=false --set scheduler.kubeScheduler.image=harbor.43.143.130.168.nip.io:30443/system_containers/hwameistor_scheduler  --set scheduler.kubeScheduler.imageTag=v1.24.0 --set scheduler.enabled=true --set scheduler.kubeScheduler.plugins.enabled=true  --set devicePlugin.enabled=false   --set version=v2.3.29  --set scheduler.extender.image=harbor.43.143.130.168.nip.io:30443/system_containers/hami  -n kube-system

# install scheduler use kubernetes default scheduler
# 
helm upgrade --install  hami-scheduler  hami  --set scheduler.kubeScheduler.image=registry.aliyuncs.com/google_containers/kube-scheduler   --set scheduler.kubeScheduler.imageTag=v1.29.5  --set scheduler.enabled=true --set scheduler.kubeScheduler.plugins.enabled=false   --set devicePlugin.enabled=false   --set version=v2.3.27  --set scheduler.extender.image=harbor.43.143.130.168.nip.io:30443/system_containers/hami -n kube-system

# device-plugin h100
# first, search "need edit when apply device-plugin", use H100
helm upgrade --install  hami-device-plugin-h100  hami   --set scheduler.enabled=false   --set devicePlugin.enabled=true  --set version=v2.3.27   --set devicePlugin.image=harbor.43.143.130.168.nip.io:30443/system_containers/hami --set devicePlugin.monitorimage=harbor.43.143.130.168.nip.io:30443/system_containers/hami   --set resourceName=nvidia.com/gpu-h100  -n kube-system


# device-plugin h20
# first, search "need edit when apply device-plugin", use H20
helm upgrade --install  hami-device-plugin-h20  hami   --set scheduler.enabled=false   --set devicePlugin.enabled=true  --set version=v2.3.27   --set devicePlugin.image=harbor.43.143.130.168.nip.io:30443/system_containers/hami --set devicePlugin.monitorimage=harbor.43.143.130.168.nip.io:30443/system_containers/hami   --set resourceName=nvidia.com/gpu-h20  -n kube-system


# device-plugin 4090
# first, search "need edit when apply device-plugin", use 4090
helm upgrade --install  hami-device-plugin-4090  hami   --set scheduler.enabled=false   --set devicePlugin.enabled=true  --set version=v2.3.27   --set devicePlugin.image=harbor.43.143.130.168.nip.io:30443/system_containers/hami --set devicePlugin.monitorimage=harbor.43.143.130.168.nip.io:30443/system_containers/hami   --set resourceName=nvidia.com/gpu-4090  -n kube-system