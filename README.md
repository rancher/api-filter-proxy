api-filter-proxy
========

A microservice that acts as a proxy to Rancher API server, intercepting the API calls configured in the config.json. For each API call intercepted, the proxy will call the specified endpoint(s) and then forward the API to the destination specified.

## Building

`make`


## Running

`./bin/api-filter-proxy`

## License
Copyright (c) 2014-2016 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
