#!/bin/bash
#
# Copyright 2022-2023 The Koordinator Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -e

SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
LICENSE_HEADER_PATH="./hack/boilerplate/boilerplate.go.txt"

cd $GOPATH/src/github.com/koordinator-sh/yarn-copilot

# generates gomock files
mockgen -source pkg/yarn/client/factory.go \
  -destination pkg/yarn/client/mockclient/mock_factory.go \
  -copyright_file ${LICENSE_HEADER_PATH}
mockgen -source pkg/yarn/client/client.go \
  -destination pkg/yarn/client/mockclient/mock_client.go \
  -copyright_file ${LICENSE_HEADER_PATH}
