## k8s webhook template, you can use it by simple wirte logical code

- need cert-manager generate tls cert/key

### validationg webhook
- 实现逻辑在`k8s/validation.go`
- pod 打了标签`allow-delete=false`, cannot be delete
### mutating webhook
- 实现逻辑在`k8s/mutating.go`
- 创建 pod 时，自动添加标签`k8s-webhook=test`
### usage
- 快速实现webhook功能
- copy this repo, and write your own logic
- Makefile部署,change VERSION_TAG|REGISTRY_HOST,`make deploy-k8s`


## License

Copyright 2023 changqings.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.