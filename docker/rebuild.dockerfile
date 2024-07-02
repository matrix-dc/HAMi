FROM  projecthami/hami:v2.3.12

COPY scheduler /k8s-vgpu/bin
COPY nvidia-device-plugin /k8s-vgpu/bin