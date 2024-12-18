## k8s webhook template, you can use it by simply write logical code

- need cert-manager generate tls cert/key

### validationg webhook
- logical in `k8s/validation.go`, change it whth your need
### mutating webhook
- logical in `k8s/mutating.go`, change with your need

### usage
- as template, you can copy it and modify logical
- copy this repo, and write your own logic
- change VERSION_TAG|REGISTRY_HOST of Makefile `make deploy-k8s`


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
