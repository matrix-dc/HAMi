# install scheduler use hwameistor scheduler as default
# 
helm upgrade --install  hami-scheduler  hami \
 --set scheduler.kubeScheduler.image=ghcr.m.daocloud.io/hwameistor/scheduler  \
 --set scheduler.kubeScheduler.imageTag=v1.24.0 \
 --set scheduler.enabled=true --set scheduler.kubeScheduler.plugins.enabled=true  \
 --set devicePlugin.enabled=false   --set version=v2.3.27 \
 -n kube-system

# install scheduler use kubernetes default scheduler
# 
helm upgrade --install  hami-scheduler  hami \
 --set scheduler.kubeScheduler.image=registry.aliyuncs.com/google_containers/kube-scheduler  \
 --set scheduler.kubeScheduler.imageTag=v1.29.5 \
 --set scheduler.enabled=true --set scheduler.kubeScheduler.plugins.enabled=false  \
 --set devicePlugin.enabled=false   --set version=v2.3.27 \
 -n kube-system

# device-plugin h100
# first, search "need edit when apply device-plugin", use H100
helm upgrade --install  hami-device-plugin-h100  hami \
  --set scheduler.enabled=false \
  --set devicePlugin.enabled=true  --set version=v2.3.27 \
  --set resourceName=nvidia.com/gpu-h100  -n kube-system


# device-plugin h20
# first, search "need edit when apply device-plugin", use H20
helm upgrade --install  hami-device-plugin-h20  hami \
  --set scheduler.enabled=false \
  --set devicePlugin.enabled=true  --set version=v2.3.27 \
  --set resourceName=nvidia.com/gpu-h20  -n kube-system


# device-plugin 4090
# first, search "need edit when apply device-plugin", use 4090
helm upgrade --install  hami-device-plugin-4090  hami \
  --set scheduler.enabled=false \
  --set devicePlugin.enabled=true  --set version=v2.3.27 \
  --set resourceName=nvidia.com/gpu-4090  -n kube-system