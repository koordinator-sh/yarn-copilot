/*
Copyright 2013 The Cloudera Inc.
Copyright 2023 The Koordinator Authors.

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

package ipc

import (
	hadoop_common "github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopcommon"
	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/security"
)

type TokenAuth struct {
	userToken *hadoop_common.TokenProto
	protocol  *string
	serverID  *string
}

var _ SASLClient = &TokenAuth{}

func CreateTokenAuth(token *hadoop_common.TokenProto, auth *hadoop_common.RpcSaslProto_SaslAuth) *TokenAuth {
	return &TokenAuth{
		userToken: token,
		protocol:  auth.Protocol,
		serverID:  auth.ServerId,
	}
}

func (t *TokenAuth) EvaluateChallenge(challenge []byte) ([]byte, error) {
	response, err := security.GetDigestMD5ChallengeResponse(*t.protocol, *t.serverID, challenge, t.userToken)

	return []byte(response), err
}
