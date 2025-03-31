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
