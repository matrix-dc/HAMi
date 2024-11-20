# 如何操作submodule

## 清理老的submodule
- rm -rf libvgpu 删除子模块目录及源码
- vi .gitmodules  删除项目目录下.gitmodules文件中libvgpu子模块相关条目
- vi .git/config 删除配置项中libvgpu子模块相关条目
- rm  -rf .git/modules/libvgpu 删除模块下的子模块目录，每个子模块对应一个目录，注意只删除对应的子模块目录即可
- rm -rf .gitmodules
- git rm --cached libvgpu 

## 重建
- touch .gitmodules
- git submodule add -b release-2.3.12 https://github.com/carolove/HAMi-core.git libvgpu