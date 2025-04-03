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

package security

import (
	"os/user"
	"sync"

	"k8s.io/klog/v2"

	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/auth"
	hadoop_common "github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopcommon"
	yarn_conf "github.com/koordinator-sh/yarn-copilot/pkg/yarn/config"
)

/*
a (very) basic UserGroupInformation implementation for storing user data/tokens,
This implementation is currently *not* thread-safe
*/
type UserGroupInformation struct {
	conf yarn_conf.YarnConfiguration

	// rwMutex    sync.RWMutex
	userInfo   *hadoop_common.UserInformationProto
	userTokens map[string]*hadoop_common.TokenProto

	authentication string
	keytabFilePath string
	principal      string
}

func (ugi *UserGroupInformation) GetAuthentication() string {
	return ugi.authentication
}

func (ugi *UserGroupInformation) GetKeytabFilePath() string {
	return ugi.keytabFilePath
}

func (ugi *UserGroupInformation) GetPrincipal() string {
	return ugi.principal
}

func (ugi *UserGroupInformation) GetEffectiveUser() string {
	return ugi.userInfo.GetEffectiveUser()
}

func (ugi *UserGroupInformation) GetRealUser() string {
	return ugi.userInfo.GetRealUser()
}

func (ugi *UserGroupInformation) GetUserInfoProto() *hadoop_common.UserInformationProto {
	if ugi.userInfo == nil {
		klog.Warningf("UserGroupInformation is nil")
		return nil
	}

	return ugi.userInfo
}

func (ugi *UserGroupInformation) IsSecurityEnabled() bool {
	// In order to turn on RPC authentication in hadoop, set the value of hadoop.security.authentication property to "kerberos"
	// https://hadoop.apache.org/docs/stable/hadoop-project-dist/hadoop-common/SecureMode.html
	return ugi.authentication != "simple"
}

var once sync.Once
var currentUserGroupInformation *UserGroupInformation
var maxTokens = 16

func CreateCurrentUserInfoProto() (*hadoop_common.UserInformationProto, error) {
	// Figure the current user-name
	var username string
	if currentUser, err := user.Current(); err != nil {
		klog.Warningf("user.Current", err)
		return nil, err
	} else {
		username = currentUser.Username
	}

	return &hadoop_common.UserInformationProto{EffectiveUser: nil, RealUser: &username}, nil
}

func CreateUserGroupInformation(conf yarn_conf.YarnConfiguration) (*UserGroupInformation, error) {
	ugi := &UserGroupInformation{
		conf: conf,
	}

	authentication, err := conf.GetSecurityAuthentication()
	if err != nil {
		return nil, err
	}
	ugi.authentication = authentication

	if ugi.authentication == "kerberos" {
		// TODO: get the user name from the configuration
		ugiProto, err := auth.CreateSimpleUGIProto()
		if err != nil {
			return nil, err
		}
		ugi.userInfo = ugiProto

		keytabFilePath, err := conf.GetResourceManagerKeytab()
		if err != nil {
			return nil, err
		}
		ugi.keytabFilePath = keytabFilePath

		principal, err := conf.GetResourceManagerPrincipal()
		if err != nil {
			return nil, err
		}
		ugi.principal = principal
	} else {
		ugiProto, err := auth.CreateSimpleUGIProto()
		if err != nil {
			return nil, err
		}
		ugi.userInfo = ugiProto
	}

	return ugi, nil
}

func Allocate(userInfo *hadoop_common.UserInformationProto, userTokens map[string]*hadoop_common.TokenProto) *UserGroupInformation {
	ugi := new(UserGroupInformation)

	if userInfo != nil {
		ugi.userInfo = userInfo
	} else {
		currentUserInfo, _ := CreateCurrentUserInfoProto()
		ugi.userInfo = currentUserInfo
	}

	if userTokens != nil {
		ugi.userTokens = userTokens
	} else {
		ugi.userTokens = make(map[string]*hadoop_common.TokenProto) //empty, with room for maxTokens tokens.
	}

	return ugi
}

func initializeCurrentUser() {
	once.Do(func() {
		currentUserGroupInformation = Allocate(nil, nil)
	})
}

func (ugi *UserGroupInformation) GetUserInformation() *hadoop_common.UserInformationProto {
	return ugi.userInfo
}

func (ugi *UserGroupInformation) GetUserTokens() map[string]*hadoop_common.TokenProto {
	return ugi.userTokens
}

func (ugi *UserGroupInformation) AddUserTokenWithAlias(alias string, token *hadoop_common.TokenProto) {
	if token == nil {
		klog.Warningf("supplied token is nil!")
		return
	}

	if length := len(ugi.userTokens); length < maxTokens {
		ugi.userTokens[alias] = token
	} else {
		klog.Warningf("user already has maxTokens:", maxTokens)
	}
}

func (ugi *UserGroupInformation) AddUserToken(token *hadoop_common.TokenProto) {
	if token == nil {
		klog.Warningf("supplied token is nil!")
		return
	}

	ugi.AddUserTokenWithAlias(token.GetService(), token)
}

func GetCurrentUser() *UserGroupInformation {
	initializeCurrentUser()

	return currentUserGroupInformation
}
