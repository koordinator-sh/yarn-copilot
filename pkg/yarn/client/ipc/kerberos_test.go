/*
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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEnableKerberos(t *testing.T) {
	cases := map[string]bool{
		"1":     true,
		"true":  true,
		"yes":   true,
		"y":     true,
		"on":    true,
		"0":     false,
		"false": false,
		"no":    false,
		"n":     false,
		"off":   false,
		"":      false, // default
	}

	for val, expected := range cases {
		os.Setenv(EnvEnableKerberos, val)
		assert.Equal(t, expected, IsEnableKerberos())
	}
	os.Unsetenv(EnvEnableKerberos)
}

func TestGetKerberosConf_Defaults(t *testing.T) {
	conf := GetKerberosConf()
	assert.Equal(t, DefaultKrb5ConfPath, conf.Krb5ConfPath)
	assert.Equal(t, DefaultKeytabPath, conf.KeytabPath)
	assert.Equal(t, DefaultClientUserName, conf.ClientUserName)
	assert.Equal(t, DefaultRealm, conf.ClientRealm)
}

func TestGetKerberosConf_EnvOverride(t *testing.T) {
	os.Setenv(EnvKrb5ConfPath, "/tmp/krb5.conf")
	os.Setenv(EnvKeytabPath, "/tmp/test.keytab")
	os.Setenv(EnvClientUserName, "testuser")
	os.Setenv(EnvClientRealm, "TEST.COM")
	conf := GetKerberosConf()
	assert.Equal(t, "/tmp/krb5.conf", conf.Krb5ConfPath)
	assert.Equal(t, "/tmp/test.keytab", conf.KeytabPath)
	assert.Equal(t, "testuser", conf.ClientUserName)
	assert.Equal(t, "TEST.COM", conf.ClientRealm)

	os.Unsetenv(EnvKrb5ConfPath)
	os.Unsetenv(EnvKeytabPath)
	os.Unsetenv(EnvClientUserName)
	os.Unsetenv(EnvClientRealm)
}

func TestEncodeASN1Length(t *testing.T) {
	assert.Equal(t, []byte{0x7F}, encodeASN1Length(127))
	assert.Equal(t, []byte{0x81, 0x80}, encodeASN1Length(128))
	assert.Equal(t, []byte{0x82, 0x01, 0x00}, encodeASN1Length(256))
	assert.Equal(t, []byte{0x83, 0x01, 0x00, 0x00}, encodeASN1Length(65536))
}
